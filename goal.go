package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// fabi goal — Goal → Task → Capability → Model → Hardware (SPEC #6, #30).
// The user states a purpose; physics decides what's feasible.

// goalTaskMap: purpose keywords → task profile names.
var goalTaskMap = []struct {
	kws   []string
	tasks []string
}{
	{[]string{"homepage", "website", "site", "ホームページ", "ページ"}, []string{"planning", "code_review", "image_gen"}},
	{[]string{"translate", "翻訳"}, []string{"translate"}},
	{[]string{"image", "picture", "画像", "写真", "generate"}, []string{"image_gen"}},
	{[]string{"video", "動画"}, []string{"video"}},
	{[]string{"summarize", "summary", "要約", "まとめ"}, []string{"summarize"}},
	{[]string{"research", "search", "研究", "調査", "find"}, []string{"search", "reasoning"}},
	{[]string{"code", "program", "build", "make", "作成", "作る", "実装"}, []string{"planning", "code_review"}},
	{[]string{"accounting", "invoice", "経理", "請求", "receipt"}, []string{"ocr", "reasoning"}},
	{[]string{"meeting", "minutes", "議事録"}, []string{"summarize", "reasoning"}},
	{[]string{"email", "mail", "メール", "letter"}, []string{"summarize", "translate"}},
}

type evalResult struct {
	task    string
	profile taskProfile
	toks    float64
	latency float64
	fitVRAM bool
	verdict string
}

func evalTask(tp taskProfile, gpuName string, vramMiB float64) evalResult {
	resident := modelBytes[tp.modelClass]
	fitVRAM := float64(resident)/1048576 <= vramMiB

	// Image gen isn't token-decode bound — verdict is VRAM fit only.
	if tp.modelClass == "img" {
		v := "VRAM FIT (gen time varies)"
		if !fitVRAM {
			v = "DOESN'T FIT"
		}
		return evalResult{"", tp, 0, 0, fitVRAM, v}
	}

	bw := bandwidthForGPU(gpuName)
	toks := bw * 1e9 / float64(resident)
	latency := float64(tp.outTokens) / toks
	verdict := "LOCAL OK"
	switch {
	case !fitVRAM:
		verdict = "DOESN'T FIT"
	case latency <= tp.latencyS:
		verdict = "LOCAL OK"
	case latency <= tp.latencyS*3:
		verdict = "POSSIBLE (slow)"
	default:
		verdict = "TOO SLOW"
	}
	return evalResult{"", tp, toks, latency, fitVRAM, verdict}
}

func runGoal(goal string) {
	if goal == "" {
		fmt.Println("usage: fabi goal \"<purpose>\"")
		fmt.Println("example: fabi goal \"build a homepage\"")
		return
	}

	// Goal → tasks via keyword match (first match wins; default is generic).
	var tasks []string
	gl := strings.ToLower(goal)
	for _, m := range goalTaskMap {
		for _, kw := range m.kws {
			if strings.Contains(gl, kw) {
				tasks = m.tasks
				break
			}
		}
		if tasks != nil {
			break
		}
	}
	if tasks == nil {
		tasks = []string{"planning", "reasoning"}
	}

	// Hardware from ledger (latest gpu finding with metrics).
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

	fmt.Println()
	fmt.Printf("FABI GOAL — %q\n", goal)
	fmt.Println("─────────────────────────────────────────────")
	fmt.Printf("tasks (%d): %s\n", len(tasks), strings.Join(tasks, ", "))
	if gpuName != "" {
		fmt.Printf("hardware: %s — %.0f GiB VRAM\n", gpuName, vramMiB/1024)
	} else {
		fmt.Println("hardware: no GPU in ledger — local verdicts are pessimistic")
	}
	fmt.Println()

	ok := 0
	for _, tn := range tasks {
		tp, has := taskProfiles[tn]
		if !has {
			continue
		}
		r := evalTask(tp, gpuName, vramMiB)
		r.task = tn
		if tp.modelClass == "img" {
			fmt.Printf("  %-12s %3s class | diffusion | %s\n", tn, tp.modelClass, r.verdict)
		} else {
			fmt.Printf("  %-12s %3s class | %.0f tok/s → %.1fs vs %.1fs | %s\n",
				tn, tp.modelClass, r.toks, r.latency, tp.latencyS, r.verdict)
		}
		if r.verdict == "LOCAL OK" {
			ok++
		}
	}
	fmt.Println()
	fmt.Printf("verdict: %d/%d tasks feasible locally", ok, len(tasks))
	switch {
	case ok == len(tasks):
		fmt.Println(" — GO")
	case ok == 0:
		fmt.Println(" — needs cloud or downgrade")
	default:
		fmt.Println(" — partial local; hybrid route")
	}
	fmt.Println()

	l.emit("goal.evaluated", map[string]any{
		"goal": goal, "tasks": tasks,
		"local_ok": ok, "gpu": gpuName, "vram_mib": vramMiB,
	})
}
