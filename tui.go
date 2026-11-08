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
	var toolsRun, toolsIdle, modelsList, svcList, rtList, pkgList []findingRec
	var gpuName string
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
			svcList = append(svcList, f)
			if strings.Contains(f.detail, "owner not visible") {
				hidden++
			}
		case f.kind == "model":
			models++
			modelsList = append(modelsList, f)
		case f.kind == "runtime":
			rtList = append(rtList, f)
		case f.kind == "package":
			pkgList = append(pkgList, f)
		case f.kind == "gpu":
			gpuName = f.name
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
	// Watch-daemon liveness — a recent ui.state event means fabi watch is alive.
	var watchTs int64
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].kind == "ui.state" {
			watchTs = events[i].ts
			break
		}
	}
	if watchTs > 0 {
		if age := time.Since(time.UnixMilli(watchTs)); age < 10*time.Minute {
			b.WriteString(fmt.Sprintf("  "+tr("tui.watch.alive")+"\n", age.Minutes()))
		} else {
			b.WriteString(fmt.Sprintf("  "+tr("tui.watch.stale")+"\n", age.Minutes()))
		}
	}
	if gpuName != "" {
		b.WriteString("  " + trf("tui.gpu", gpuName) + "\n")
	}
	if lastModel != "" {
		b.WriteString("  " + trf("tui.last", lastToks, lastModel) + "\n")
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
	if len(svcList) > 0 {
		b.WriteString(trf("sec.services", len(svcList)) + "\n")
		for _, f := range svcList {
			b.WriteString("  · " + f.name + " — " + f.detail + "\n")
		}
	}
	if len(rtList) > 0 {
		b.WriteString(trf("scan.runtimes", len(rtList)) + "\n")
		for _, f := range rtList {
			b.WriteString("  · " + f.name + " — " + f.detail + "\n")
		}
	}
	if len(pkgList) > 0 {
		b.WriteString(trf("sec.packages", len(pkgList)) + "\n")
		for _, f := range pkgList {
			b.WriteString("  · " + f.name + " — " + f.detail + "\n")
		}
	}
	// Inference trend — measured tok/s over time from inference.measured.
	var toks []float64
	for i := len(events) - 1; i >= 0 && len(toks) < 24; i-- {
		if events[i].kind == "inference.measured" {
			var p map[string]any
			if json.Unmarshal([]byte(events[i].payload), &p) == nil {
				if v := numOf(p, "tok_s"); v > 0 {
					toks = append(toks, v)
				}
			}
		}
	}
	if len(toks) > 0 {
		// reverse so oldest→newest reads left→right
		for i, j := 0, len(toks)-1; i < j; i, j = i+1, j-1 {
			toks[i], toks[j] = toks[j], toks[i]
		}
		mx := toks[0]
		for _, v := range toks {
			if v > mx {
				mx = v
			}
		}
		bars := []rune("▁▂▃▄▅▆▇█")
		var spark strings.Builder
		for _, v := range toks {
			spark.WriteRune(bars[int(v/mx*float64(len(bars)-1))])
		}
		b.WriteString(fmt.Sprintf(tr("tui.inference")+"\n",
			spark.String(), func() float64 {
				m := toks[0]
				for _, v := range toks {
					if v < m {
						m = v
					}
				}
				return m
			}(), mx))
	}

	// Scan history — the ledger's time dimension as an ASCII sparkline.
	if len(windows) > 1 {
		start := len(windows) - 24
		if start < 0 {
			start = 0
		}
		max := 1
		for _, w := range windows[start:] {
			if len(w.findings) > max {
				max = len(w.findings)
			}
		}
		bars := []rune("▁▂▃▄▅▆▇█")
		var spark strings.Builder
		for _, w := range windows[start:] {
			h := len(w.findings) * (len(bars) - 1) / max
			if h < 0 {
				h = 0
			}
			spark.WriteRune(bars[h])
		}
		b.WriteString("\n" + trf("tui.history", spark.String(), len(windows)-start) + "\n")
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
		b.WriteString("  " + trf("tui.reclaimable", formatBytes(reclaim)) + "\n")
	}
	if len(recs) > 0 {
		b.WriteString("\n" + tr("doctor.title") + "\n")
		for _, r := range recs {
			sz := ""
			if r.class == "remove" && r.reclaim > 0 {
				sz = "  ~" + formatBytes(r.reclaim)
			}
			b.WriteString("  [" + r.class + "] " + r.target + sz +
				"  →  " + whyAction(r.class, r.target) + "\n")
		}
		b.WriteString("  " + tr("doctor.next") + "\n")
	}
	fmt.Print(b.String())
}
