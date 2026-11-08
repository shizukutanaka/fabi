package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// fabi tui — the terminal dashboard: one screen, live refresh.
// Same data as `fabi status`/`scan`, rendered as a bordered screen
// that re-draws on an interval. Ctrl+C exits. Zero deps — ANSI only.
func runTUI(args []string) {
	interval := 2 * time.Second
	if len(args) > 0 {
		if d, err := time.ParseDuration(args[0]); err == nil && d > 0 {
			interval = d
		}
	}
	l := openLedger()
	defer l.close()

	fmt.Print("\033[?25l")       // hide cursor
	defer fmt.Print("\033[?25h") // restore on exit

	for {
		events := localEvents(l.allEvents(), l.importedRanges())
		windows := loadScanWindows(events)
		fmt.Print("\033[2J\033[H") // clear + home
		renderTUI(events, windows)
		fmt.Println("\n" + trf("watch.every", interval) + "  (Ctrl+C)")
		time.Sleep(interval)
	}
}

func renderTUI(events []ledgerEvent, windows []scanWindow) {
	var b strings.Builder
	b.WriteString("FABI TUI — " + tr("tagline") + "\n")
	b.WriteString(strings.Repeat("─", 46) + "\n")

	if len(windows) == 0 {
		b.WriteString(tr("status.empty.ledger") + "\n")
		fmt.Print(b.String())
		return
	}
	latest := windows[len(windows)-1]

	var running, installed, services, hidden, models int
	var gpuUtil, gpuTemp, vramUsed, vramTotal, lastToks float64
	var lastModel string
	var toolsRun, toolsIdle, modelsList []findingRec
	for _, f := range latest.findings {
		switch {
		case f.kind == "tool" && f.running:
			running++
			toolsRun = append(toolsRun, f)
		case f.kind == "tool":
			installed++
			toolsIdle = append(toolsIdle, f)
		case f.kind == "service":
			services++
			if strings.Contains(f.detail, "owner not visible") {
				hidden++
			}
		case f.kind == "model":
			models++
			modelsList = append(modelsList, f)
		}
	}
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		if e.kind == "measurement.sampled" && strings.Contains(e.payload, "gpu_util") {
			var p map[string]any
			if json.Unmarshal([]byte(e.payload), &p) == nil {
				gpuUtil = numOf(p, "gpu_util")
				gpuTemp = numOf(p, "gpu_temp_c")
				vramUsed = numOf(p, "vram_used_mib")
				vramTotal = numOf(p, "vram_total_mib")
			}
			break
		}
	}
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].kind == "inference.measured" {
			var p map[string]any
			if json.Unmarshal([]byte(events[i].payload), &p) == nil {
				if v := numOf(p, "tok_s"); v > 0 {
					lastToks, lastModel = v, strOf(p, "model")
				}
			}
			break
		}
	}

	stale := ""
	if age := time.Since(time.UnixMilli(latest.startTs)); age > 24*time.Hour {
		stale = trf("status.stale", age.Hours())
	}
	b.WriteString(fmt.Sprintf(tr("status.line"), running, installed, services, hidden, models,
		gpuUtil, gpuTemp, vramUsed/1024, vramTotal/1024) + stale + "\n")
	if lastModel != "" {
		b.WriteString(fmt.Sprintf("  last: %.0f tok/s · %s\n", lastToks, lastModel))
	}
	b.WriteString("\n")

	b.WriteString(trf("sec.tools.run", running) + "\n")
	for _, f := range toolsRun {
		b.WriteString("  · " + f.name + "\n")
	}
	if len(toolsIdle) > 0 {
		b.WriteString(trf("sec.tools.inst", installed) + "\n")
		for _, f := range toolsIdle {
			b.WriteString("  · " + f.name + "\n")
		}
	}
	if len(modelsList) > 0 {
		var disk int64
		for _, f := range modelsList {
			disk += f.size
		}
		b.WriteString("\n" + trf("sec.models", models, formatBytes(disk)) + "\n")
		for _, f := range modelsList {
			b.WriteString("  · " + f.name + " — " + f.detail + "\n")
		}
	}

	// Doctor recommendations — the actionable part of the honest eye.
	recs := computeRecommendations(events, windows, latest)
	var reclaim int64
	for _, r := range recs {
		if r.class == "remove" {
			reclaim += r.reclaim
		}
	}
	if reclaim > 0 {
		b.WriteString("  reclaimable: ~" + formatBytes(reclaim) + "\n")
	}
	if len(recs) > 0 {
		b.WriteString("\n" + tr("doctor.title") + "\n")
		for _, r := range recs {
			b.WriteString("  [" + r.class + "] " + r.target +
				"  →  " + whyAction(r.class, r.target) + "\n")
		}
	}
	fmt.Print(b.String())
}
