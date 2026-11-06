package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// fabi route — physics-derived routing (SPEC #48, #30).
// tok/s ≈ bandwidth ÷ resident bytes. Not a planner — a physics check.

// task profiles: what each job needs. Output tokens + a latency target
// turn "reasonable response time" into a number.
type taskProfile struct {
	outTokens  int
	latencyS   float64
	modelClass string // 3b|8b|70b — smallest class that plausibly does it well
	desc       string
}

var taskProfiles = map[string]taskProfile{
	"ocr":         {200, 1.5, "3b", "extract text from 1 page"},
	"translate":   {800, 4.0, "3b", "translate ~1000 words"},
	"summarize":   {600, 4.0, "8b", "summarize a document"},
	"reasoning":   {1000, 5.0, "8b", "1000 output tokens in 5 s"},
	"code_review": {2000, 12.0, "8b", "review ~2000 LOC"},
	"search":      {300, 0.3, "3b", "one semantic query"},
	"image_gen":   {0, 6.0, "img", "one 1024px image"},
	"video":       {1500, 25.0, "70b", "understand 60 s clip"},
	"planning":    {800, 1.2, "8b", "produce a 12-step task graph"},
}

// residentBytes per model class at Q4 (#48 table, conservative).
var modelBytes = map[string]int64{
	"3b":  1_900_000_000,  // ~1.9 GB
	"8b":  4_700_000_000,  // ~4.7 GB
	"70b": 40_000_000_000, // ~40 GB
	"img": 7_000_000_000,  // SDXL-class ~7 GB (diffusion, not token-bound)
}

// bandwidth guess from GPU name (GB/s). Real table lives in data later;
// this is the measured-physics fallback.
func bandwidthForGPU(name string) float64 {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "a2000"):
		return 288
	case strings.Contains(n, "4090"):
		return 1008
	case strings.Contains(n, "3090"):
		return 936
	case strings.Contains(n, "a100"):
		return 1555
	case strings.Contains(n, "h100"):
		return 3350
	case strings.Contains(n, "3060"), strings.Contains(n, "3070"):
		return 448
	case strings.Contains(n, "rtx"):
		return 360 // conservative RTX-class default
	default:
		return 200
	}
}

func runRoute(taskName string) {
	if taskName == "" {
		fmt.Println("usage: fabi route <task>")
		fmt.Println("tasks: " + strings.Join(taskNames(), ", "))
		return
	}
	tp, ok := taskProfiles[strings.ToLower(taskName)]
	if !ok {
		fmt.Printf("unknown task %q — available: %s\n", taskName, strings.Join(taskNames(), ", "))
		return
	}

	// Get measured hardware: VRAM from the ledger's latest GPU finding.
	l := openLedger()
	defer l.close()
	var vramMiB float64
	var gpuName string
	for _, e := range l.events("resource.found") {
		if strings.Contains(e.payload, `"kind":"gpu"`) &&
			!strings.Contains(e.payload, "unattributed") {
			var p struct {
				Name    string             `json:"name"`
				Metrics map[string]float64 `json:"metrics"`
			}
			if err := json.Unmarshal([]byte(e.payload), &p); err == nil && p.Metrics != nil {
				if t := p.Metrics["vram_total_mib"]; t > 0 {
					vramMiB = t
					gpuName = p.Name
				}
			}
		}
	}

	latTok := float64(tp.outTokens) / tp.latencyS // required tok/s
	resident := modelBytes[tp.modelClass]
	bw := bandwidthForGPU(gpuName)
	toks := bw * 1e9 / float64(resident)
	latency := float64(tp.outTokens) / toks
	fitVRAM := float64(resident)/1048576 <= vramMiB

	fmt.Println()
	fmt.Printf("FABI ROUTE — %s\n", taskName)
	fmt.Println("─────────────────────────────────────────────")
	fmt.Printf("task:        %s → %d out tokens in %.1f s = %.0f tok/s needed\n",
		tp.desc, tp.outTokens, tp.latencyS, latTok)
	fmt.Printf("model class: %s (%.1f GB resident)\n", tp.modelClass, float64(resident)/1e9)
	if gpuName == "" {
		fmt.Println("hardware:    no GPU found in ledger — run `fabi scan` first")
	} else {
		fmt.Printf("hardware:    %s — %.0f GiB VRAM, ~%.0f GB/s\n",
			gpuName, vramMiB/1024, bw)
	}
	fmt.Println()
	fmt.Println("physics:")
	fmt.Printf("  %.0f GB/s ÷ %.1f GB = %.0f tok/s (decode, #48)\n",
		bw, float64(resident)/1e9, toks)
	if gpuName == "" {
		fmt.Println("verdict:     NO LOCAL GPU — cloud or smaller model required")
		return
	}
	fmt.Printf("  latency:   %d tok ÷ %.0f tok/s = %.1f s (target %.1f s)\n",
		tp.outTokens, toks, latency, tp.latencyS)
	fmt.Printf("  VRAM:      %.1f GB needed vs %.0f GiB available → %v\n",
		float64(resident)/1e9, vramMiB/1024, fitVRAM)
	fmt.Println()

	switch {
	case !fitVRAM:
		fmt.Println("verdict:     DOESN'T FIT — smaller model or cloud required")
		fmt.Println("             → OLLAMA_HOST=<host>:11434 routes to a remote engine")
	case latency <= tp.latencyS:
		fmt.Printf("verdict:     LOCAL OK — %.0f tok/s ≥ %.0f needed, %.1fs ≤ %.1fs\n",
			toks, latTok, latency, tp.latencyS)
	case latency <= tp.latencyS*3:
		fmt.Printf("verdict:     LOCAL POSSIBLE — %.1fs vs %.1fs target (%.0f× slow);\n"+
			"             speculative decoding / draft model may close the gap\n",
			latency, tp.latencyS, latency/tp.latencyS)
	default:
		fmt.Printf("verdict:     LOCAL TOO SLOW — %.1fs vs %.1fs; cloud or downgrade\n"+
			"             → OLLAMA_HOST=<host>:11434 routes to a remote engine\n",
			latency, tp.latencyS)
	}

	l.emit("route.evaluated", map[string]any{
		"task": taskName, "model_class": tp.modelClass,
		"gpu": gpuName, "vram_mib": vramMiB, "bandwidth_gbps": bw,
		"tok_s": toks, "latency_s": latency, "fit_vram": fitVRAM,
	})

	// Measured reality: if the ledger holds real tok/s, prefer them over
	// the physics table. Prediction is a guess; measurement is a fact.
	type mrow struct {
		model string
		toks  float64
		ts    int64
	}
	latestM := map[string]mrow{}
	for _, e := range l.events("inference.measured") {
		var p map[string]any
		if err := json.Unmarshal([]byte(e.payload), &p); err == nil &&
			strOf(p, "op") == "generate" {
			if v, ok := p["tok_s"].(float64); ok && v > 0 {
				m := mrow{strOf(p, "model"), v, e.ts}
				if cur, ok := latestM[m.model]; !ok || m.ts > cur.ts {
					latestM[m.model] = m
				}
			}
		}
	}
	if len(latestM) > 0 {
		fmt.Println()
		fmt.Println("measured local models (real tok/s, `run --sweep`):")
		var rows []mrow
		for _, m := range latestM {
			rows = append(rows, m)
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].toks > rows[j].toks })
		for _, m := range rows {
			ml := float64(tp.outTokens) / m.toks
			v := "FEASIBLE"
			if ml > tp.latencyS {
				v = "slow"
			}
			fmt.Printf("  %-18s %5.1f tok/s → %4.1fs vs %.1fs | %s\n",
				m.model, m.toks, ml, tp.latencyS, v)
		}
		fmt.Println("  (measured — not predicted; `fabi run --sweep` to refresh)")
	}
}

func taskNames() []string {
	var out []string
	for k := range taskProfiles {
		out = append(out, k)
	}
	return out
}
