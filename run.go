package main

import (
	"encoding/json"
	"fmt"
	"strings"
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
	Response     string `json:"response"`
	EvalCount    int    `json:"eval_count"`
	EvalDuration int64  `json:"eval_duration"` // ns
	Done         bool   `json:"done"`
}

func runRun(args []string) {
	prompt := strings.Join(args, " ")

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

	// Pick the smallest (fastest) model.
	best := models[0]
	for _, m := range models {
		if sizes[m] < sizes[best] {
			best = m
		}
	}

	fmt.Println()
	fmt.Println("FABI RUN — real execution, real measurement")
	fmt.Println("─────────────────────────────────────────────")
	fmt.Printf("model:   %s (%.1f GB)\n", best, float64(sizes[best])/1e9)
	fmt.Printf("prompt:  %s\n", trunc(prompt, 80))
	fmt.Println("running...")
	fmt.Println()

	payload, _ := json.Marshal(map[string]any{
		"model":  best,
		"prompt": prompt,
		"stream": false,
	})
	t0 := time.Now()
	st, rbody, err := rawHTTP(ollamaBase, "POST", "/api/generate", string(payload), 60*time.Second)
	wall := time.Since(t0)

	l := openLedger()
	defer l.close()

	if err != nil || st != 200 {
		fmt.Printf("failed: HTTP %d err=%v\n", st, err)
		l.emit("task.executed", map[string]any{
			"ok": false, "model": best, "status": st,
			"error": fmt.Sprint(err),
		})
		return
	}

	var r ollamaGenResp
	if err := json.Unmarshal(rbody, &r); err != nil {
		fmt.Println("bad response:", err)
		return
	}

	var toks float64
	if r.EvalDuration > 0 && r.EvalCount > 0 {
		toks = float64(r.EvalCount) / (float64(r.EvalDuration) / 1e9)
	}

	fmt.Println("response:")
	fmt.Println("  " + trunc(strings.TrimSpace(r.Response), 400))
	fmt.Println()
	fmt.Printf("measured: %d tokens in %.1fs = %.1f tok/s", r.EvalCount,
		wall.Seconds(), toks)
	fmt.Println("  [M — real, not predicted]")
	fmt.Println()

	l.emit("task.executed", map[string]any{
		"ok": true, "model": best, "endpoint": ollamaBase,
		"prompt": trunc(prompt, 200), "eval_count": r.EvalCount,
		"tok_s": toks, "wall_s": wall.Seconds(), "status": st,
	})
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
