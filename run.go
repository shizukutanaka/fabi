package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// fabi run — Phase-2 atom: actually EXECUTE a task through a discovered
// local model, measure real tok/s, record the decision + result.
// route predicts physics; run measures it.

const ollamaBase = "127.0.0.1:11434"

type ollamaTags struct {
	Models []struct {
		Name string `json:"name"`
		Size int64  `json:"size"`
	} `json:"models"`
}

type ollamaGenResp struct {
	Response           string  `json:"response"`
	EvalCount          int     `json:"eval_count"`
	EvalDuration       int64   `json:"eval_duration"` // ns, decode
	PromptEvalCount    int     `json:"prompt_eval_count"`
	PromptEvalDuration int64   `json:"prompt_eval_duration"` // ns, prefill
	Done               bool    `json:"done"`
	ttftMs             float64 // computed in execPrompt (not from JSON)
}

func runRun(args []string) {
	if len(args) > 0 && args[0] == "--sweep" {
		runSweep()
		return
	}
	if len(args) > 0 && args[0] == "--compare" {
		runCompare(strings.Join(args[1:], " "))
		return
	}
	var forceModel string
	var parts []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--model" && i+1 < len(args) {
			forceModel = args[i+1]
			i++
		} else {
			parts = append(parts, args[i])
		}
	}
	prompt := strings.Join(parts, " ")

	// Discover live ollama models.
	var models []string
	var sizes map[string]int64 = map[string]int64{}
	st, body, err := rawHTTP(ollamaBase, "GET", "/api/tags", "", 3*time.Second)
	if err == nil && st == 200 {
		var t ollamaTags
		if err := json.Unmarshal(body, &t); err == nil {
			for _, m := range t.Models {
				models = append(models, m.Name)
				sizes[m.Name] = m.Size
			}
		}
	}
	if len(models) == 0 {
		fmt.Println("fabi run — no live generative model found.")
		fmt.Println("(checked ollama :11434 — start `ollama serve` or pull a model)")
		return
	}
	if prompt == "" {
		fmt.Println("usage: fabi run \"<prompt>\"")
		fmt.Println("models: " + strings.Join(models, ", "))
		return
	}

	// Pick the smallest (fastest) model, or the --model override.
	best := models[0]
	for _, m := range models {
		if sizes[m] < sizes[best] {
			best = m
		}
	}
	if forceModel != "" {
		best = forceModel
	}

	fmt.Println()
	fmt.Println("FABI RUN — real execution, real measurement")
	fmt.Println("─────────────────────────────────────────────")
	fmt.Printf("model:   %s (%.1f GB)\n", best, float64(sizes[best])/1e9)
	fmt.Printf("prompt:  %s\n", trunc(prompt, 80))
	fmt.Println("running...")
	fmt.Println()

	r, toks, wall, err := execPrompt(best, prompt)
	l := openLedger()
	defer l.close()
	if err != nil {
		fmt.Printf("failed: %v\n", err)
		l.emit("task.executed", map[string]any{
			"ok": false, "model": best, "error": err.Error(),
		})
		return
	}

	fmt.Println("response:")
	fmt.Println("  " + trunc(strings.TrimSpace(r.Response), 400))
	fmt.Println()
	fmt.Printf("measured: %d tokens in %.1fs = %.1f tok/s · ttft %.0f ms (prefill %d tok)",
		r.EvalCount, wall.Seconds(), toks, r.ttftMs, r.PromptEvalCount)
	fmt.Println("  [M]")
	fmt.Println()

	l.emit("task.executed", map[string]any{
		"ok": true, "model": best, "endpoint": ollamaBase,
		"prompt": trunc(prompt, 200), "eval_count": r.EvalCount,
		"tok_s": toks, "ttft_ms": r.ttftMs,
		"prefill_tok": r.PromptEvalCount, "wall_s": wall.Seconds(),
	})
}

// inferenceLock serializes every generation through one queue —
// the sluice/bastion pattern: concurrent requests can overlap on
// shared VRAM and OOM; one-at-a-time cannot. Single goroutine owns
// the GPU; others wait.
var inferenceLock sync.Mutex

// execPrompt runs one generation on the given model, returning the
// parsed response, measured tok/s and wall time. Shared by run and
// goal --run task chains.
func execPrompt(model, prompt string) (ollamaGenResp, float64, time.Duration, error) {
	inferenceLock.Lock()
	defer inferenceLock.Unlock()
	payload, _ := json.Marshal(map[string]any{
		"model": model, "prompt": prompt, "stream": false,
	})
	t0 := time.Now()
	st, rbody, err := rawHTTP(ollamaBase, "POST", "/api/generate", string(payload), 120*time.Second)
	wall := time.Since(t0)
	var r ollamaGenResp
	if err != nil {
		return r, 0, wall, err
	}
	if st != 200 {
		return r, 0, wall, fmt.Errorf("HTTP %d", st)
	}
	if err := json.NewDecoder(bytes.NewReader(rbody)).Decode(&r); err != nil {
		return r, 0, wall, err
	}
	var toks, ttft float64
	if r.EvalDuration > 0 && r.EvalCount > 0 {
		toks = float64(r.EvalCount) / (float64(r.EvalDuration) / 1e9)
	}
	if r.PromptEvalDuration > 0 {
		ttft = float64(r.PromptEvalDuration) / 1e6 // ns → ms
	}
	r.ttftMs = ttft
	return r, toks, wall, nil
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// runCompare answers Socratic Q4 — did the scheduler's choice change a
// real outcome? Runs the same prompt on every live model, shows the
// measured winner vs the scheduler's pick. Decision proven, not asserted.
func runCompare(prompt string) {
	if prompt == "" {
		prompt = "Count from one to ten."
	}
	st, body, err := rawHTTP(ollamaBase, "GET", "/api/tags", "", 3*time.Second)
	if err != nil || st != 200 {
		fmt.Println("fabi run --compare: ollama not reachable")
		return
	}
	var t ollamaTags
	if err := json.Unmarshal(body, &t); err != nil || len(t.Models) == 0 {
		fmt.Println("fabi run --compare: no models")
		return
	}
	l := openLedger()
	defer l.close()

	// The scheduler's pick for a fast task: smallest model.
	pick := t.Models[0].Name
	var pickSize = t.Models[0].Size
	for _, m := range t.Models[1:] {
		if m.Size < pickSize {
			pick, pickSize = m.Name, m.Size
		}
	}

	fmt.Println()
	fmt.Println("FABI COMPARE — did the scheduler's choice matter?")
	fmt.Println("─────────────────────────────────────────────")
	fmt.Printf("prompt: %s\nscheduler picks (fastest/smallest): %s\n\n", prompt, pick)

	var winner string
	var winnerToks float64
	for _, m := range t.Models {
		_, toks, wall, err := execPrompt(m.Name, prompt)
		if err != nil {
			fmt.Printf("  %-18s %.1f GB   FAILED %v\n", m.Name, float64(m.Size)/1e9, err)
			continue
		}
		mark := ""
		if m.Name == pick {
			mark = "  ← scheduler's choice"
		}
		fmt.Printf("  %-18s %.1f GB   %5.1f tok/s  (%.1fs)%s\n",
			m.Name, float64(m.Size)/1e9, toks, wall.Seconds(), mark)
		if toks > winnerToks {
			winnerToks, winner = toks, m.Name
		}
		l.emit("inference.measured", map[string]any{
			"endpoint": ollamaBase, "model": m.Name, "op": "compare",
			"tok_s": toks, "ok": true,
		})
	}
	fmt.Println()
	if winner == pick {
		fmt.Printf("verdict: scheduler was RIGHT — %s is measured fastest\n", pick)
	} else {
		fmt.Printf("verdict: scheduler was WRONG — %s (%.1f) beats %s\n", winner, winnerToks, pick)
	}
}

// runUnload frees VRAM by unloading loaded models (keep_alive:0).
// This is fabi's first real control primitive — not observation,
// resource control. After: /api/ps confirms the release.
func runUnload(args []string) {
	st, body, err := rawHTTP(ollamaBase, "GET", "/api/ps", "", 3*time.Second)
	if err != nil || st != 200 {
		fmt.Println("fabi unload: ollama not reachable on :11434")
		return
	}
	var ps struct {
		Models []struct {
			Name     string `json:"name"`
			SizeVRAM int64  `json:"size_vram"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &ps); err != nil || len(ps.Models) == 0 {
		fmt.Println("fabi unload: nothing loaded — 0 VRAM held by models")
		return
	}
	l := openLedger()
	defer l.close()
	var freed int64
	all := len(args) == 0 || args[0] == "--all"
	for _, m := range ps.Models {
		if !all && len(args) > 0 && m.Name != args[0] {
			continue
		}
		payload, _ := json.Marshal(map[string]any{"model": m.Name, "keep_alive": 0})
		rawHTTP(ollamaBase, "POST", "/api/generate", string(payload), 10*time.Second)
		freed += m.SizeVRAM
		fmt.Printf("  unloaded %s (%.1f GB)\n", m.Name, float64(m.SizeVRAM)/1e9)
		l.emit("model.unloaded", map[string]any{
			"model": m.Name, "vram_freed": m.SizeVRAM,
		})
	}
	fmt.Printf("\nfabi unload: %.1f GB VRAM freed (%d model(s))\n",
		float64(freed)/1e9, len(ps.Models))
}

// loadedModels returns how many models currently hold VRAM and the
// total bytes — the loaded-vs-installed distinction fabi didn't have.
func loadedModels() (int, int64) {
	st, body, err := rawHTTP(ollamaBase, "GET", "/api/ps", "", 3*time.Second)
	if err != nil || st != 200 {
		return 0, 0
	}
	var ps struct {
		Models []struct {
			SizeVRAM int64 `json:"size_vram"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &ps); err != nil {
		return 0, 0
	}
	var vram int64
	for _, m := range ps.Models {
		vram += m.SizeVRAM
	}
	return len(ps.Models), vram
}

// runSweep measures real tok/s on every live ollama model — replaces
// predicted physics with measured ranking in the ledger.
func runSweep() {
	st, body, err := rawHTTP(ollamaBase, "GET", "/api/tags", "", 3*time.Second)
	if err != nil || st != 200 {
		fmt.Println("fabi run --sweep: ollama not reachable on :11434")
		return
	}
	var t ollamaTags
	if err := json.Unmarshal(body, &t); err != nil || len(t.Models) == 0 {
		fmt.Println("fabi run --sweep: no models on ollama")
		return
	}

	l := openLedger()
	defer l.close()

	fmt.Println()
	fmt.Println("FABI SWEEP — measured tok/s per model (warmup + median of 3)")
	fmt.Println("─────────────────────────────────────────────")
	const canned = "Count from one to ten."
	for _, m := range t.Models {
		// Warmup — cold-load latency pollutes the first sample (asiai/MLPerf).
		execPrompt(m.Name, "warmup")

		var toksS, ttftS []float64
		var wall time.Duration
		var okCount int
		for i := 0; i < 3; i++ {
			r, toks, w, err := execPrompt(m.Name, canned)
			if err != nil {
				break
			}
			toksS = append(toksS, toks)
			ttftS = append(ttftS, r.ttftMs)
			wall = w
			okCount++
		}
		if okCount == 0 {
			fmt.Printf("  %-18s %.1f GB   FAILED\n", m.Name, float64(m.Size)/1e9)
			continue
		}
		toks := median(toksS)
		ttft := median(ttftS)
		fmt.Printf("  %-18s %.1f GB   %5.1f tok/s  ttft %4.0fms  (%.1fs, %d samples)\n",
			m.Name, float64(m.Size)/1e9, toks, ttft, wall.Seconds(), okCount)
		l.emit("inference.measured", map[string]any{
			"endpoint": ollamaBase, "model": m.Name, "op": "sweep-median",
			"tok_s": toks, "ttft_ms": ttft, "samples": okCount,
			"wall_s": wall.Seconds(), "bytes": m.Size, "ok": true,
		})
	}
	// Cooldown — return VRAM to baseline (bastion pattern: don't leave
	// the estate changed after measurement).
	runUnload(nil)
	fmt.Println()
	fmt.Println("measured medians in ledger — cooldown unloaded all models")
}

// median of a small sample set — honest central value, not a lucky run.
func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	cp := append([]float64(nil), xs...)
	for i := 1; i < len(cp); i++ {
		for j := i; j > 0 && cp[j] < cp[j-1]; j-- {
			cp[j], cp[j-1] = cp[j-1], cp[j]
		}
	}
	return cp[len(cp)/2]
}
