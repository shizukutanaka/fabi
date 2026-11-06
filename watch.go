package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

// fabi watch — periodic scans, timeseries history (SPEC: iteration 5 min).
// Every pass writes events to the ledger; history feeds doctor + report.

// printScanLine is the quiet-mode summary for watch.
func printScanLine(all []Finding, vm vmStat, elapsed time.Duration) {
	var running, installed, services, models int
	var modelBytes int64
	var gpuLine string
	for _, f := range all {
		switch f.Kind {
		case "tool":
			if f.Running {
				running++
			} else {
				installed++
			}
		case "service":
			services++
		case "model":
			models++
			modelBytes += f.SizeBytes
		case "gpu":
			if f.Metrics != nil {
				gpuLine = fmt.Sprintf("gpu %.0f%% %.0f°C", f.Metrics["gpu_util"], f.Metrics["gpu_temp_c"])
			}
		}
	}
	loadedStr := ""
	if n, vram := loadedModels(); n > 0 {
		loadedStr = fmt.Sprintf(" | loaded %d/%.1fGB", n, float64(vram)/1e9)
	}
	fmt.Printf("[%s] %4d ms | tools %d↑%d↓ | svc %d | models %d/%s | %s%s\n",
		time.Now().Format("15:04:05"),
		elapsed.Milliseconds(),
		running, installed, services,
		models, formatBytes(modelBytes),
		gpuLine, loadedStr)
}

// runWatch loops: scan → sleep → scan. Interval default 5 min (spec).
func runWatch(args []string) {
	interval := 5 * time.Minute
	for i, a := range args {
		switch a {
		case "-i", "--interval":
			if i+1 < len(args) {
				if d, err := time.ParseDuration(args[i+1]); err == nil {
					interval = d
				}
			}
		default:
			if !strings.HasPrefix(a, "-") {
				if d, err := time.ParseDuration(a); err == nil {
					interval = d
				}
			}
		}
	}
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}

	fmt.Printf("fabi watch — scan every %s (Ctrl+C to stop)\n", interval)

	// Graceful exit: close cleanly on Ctrl+C so the ledger isn't left mid-write.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	tick := time.NewTicker(interval)

	runScan(time.Now(), true)
	measureLiveInference()
	for {
		select {
		case <-tick.C:
			runScan(time.Now(), true)
			measureLiveInference()
		case <-sig:
			fmt.Println("\nwatch stopped — ledger complete.")
			return
		}
	}
}

// measureLiveInference adds real tok/s to the timeseries — a canned
// generation on the fastest local model. If nothing's live, it skips
// silently; heartbeat data only exists when inference is possible.
func measureLiveInference() {
	model := bestLocalModel()
	if model == "" {
		return
	}
	l := openLedger()
	defer l.close()
	r, toks, wall, err := execPrompt(model, "Count: 1 2 3")
	if err != nil {
		return
	}
	l.emit("inference.measured", map[string]any{
		"endpoint": ollamaBase, "model": model, "op": "watch-generate",
		"tok_s": toks, "eval_count": r.EvalCount, "wall_s": wall.Seconds(), "ok": true,
	})
	fmt.Printf("[inference] %s %.1f tok/s (%.1fs)\n", model, toks, wall.Seconds())
}

var _ = strconv.Itoa
