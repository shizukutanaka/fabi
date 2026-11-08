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
	ranges := l.importedRanges()
	var vramMiB float64
	var gpuName string
	for _, e := range localEvents(l.events("resource.found"), ranges) {
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
	fmt.Println(tr("route.physics"))
	fmt.Printf("  %.0f GB/s ÷ %.1f GB = %.0f tok/s (decode, #48)\n",
		bw, float64(resident)/1e9, toks)

	// Calibration: physics is an upper bound — real decode pays KV cache,
	// attention and contention overhead. If the ledger has measured runs,
	// learn the measured/predicted ratio and correct the ceiling into an
	// honest estimate.
	var ratios []float64
	for _, e := range localEvents(l.events("inference.measured"), ranges) {
		var p map[string]any
		if err := json.Unmarshal([]byte(e.payload), &p); err == nil &&
			strOf(p, "op") == "generate" {
			sz := numOf(p, "bytes")
			v := numOf(p, "tok_s")
			if sz > 0 && v > 0 {
				pred := bw * 1e9 / sz
				if pred > 0 {
					ratios = append(ratios, v/pred)
				}
			}
		}
	}
	if len(ratios) > 0 {
		sort.Float64s(ratios)
		med := ratios[len(ratios)/2]
		cal := toks * med
		fmt.Printf("  calibrated: ~%.0f tok/s (median measured/predicted %.2f, n=%d)\n",
			cal, med, len(ratios))
	}
	if gpuName == "" {
		fmt.Println(tr("route.nolocal"))
		return
	}
	fmt.Printf("  latency:   %d tok ÷ %.0f tok/s = %.1f s (target %.1f s)\n",
		tp.outTokens, toks, latency, tp.latencyS)
	fmt.Printf("  VRAM:      %.1f GB needed vs %.0f GiB available → %v\n",
		float64(resident)/1e9, vramMiB/1024, fitVRAM)
	fmt.Println()

	switch {
	case !fitVRAM:
		fmt.Println(tr("route.nofit"))
		fmt.Println(tr("route.nofit.hint"))
	case latency <= tp.latencyS:
		fmt.Println(trf("route.ok", toks, latTok, latency, tp.latencyS))
	case latency <= tp.latencyS*3:
		fmt.Println(trf("route.possible", latency, tp.latencyS, latency/tp.latencyS))
		fmt.Println("             speculative decoding / draft model may close the gap")
	default:
		fmt.Println(trf("route.tooslow", latency, tp.latencyS))
		fmt.Println(tr("route.nofit.hint"))
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
	for _, e := range localEvents(l.events("inference.measured"), ranges) {
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
		fmt.Println(tr("route.measured"))
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
