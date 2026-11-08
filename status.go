package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// fabi status — one-line machine health from the ledger.
// The fastest possible answer: estate, gaps, and how old the data is.

func runStatus() {
	l := openLedger()
	defer l.close()

	events := localEvents(l.allEvents(), l.importedRanges())
	if len(events) == 0 {
		fmt.Println(tr("status.empty.ledger"))
		return
	}
	windows := loadScanWindows(events)
	if len(windows) == 0 {
		fmt.Println(tr("status.empty.scans"))
		return
	}
	latest := windows[len(windows)-1]

	var running, installed, services, hidden, deadWSL, models int
	var gpuUtil, gpuTemp, vramUsed, vramTotal float64
	for _, f := range latest.findings {
		switch {
		case f.kind == "tool" && f.running:
			running++
		case f.kind == "tool":
			installed++
		case f.kind == "service":
			services++
			if strings.Contains(f.detail, "owner not visible") {
				hidden++
			}
		case f.kind == "model":
			models++
		case f.kind == "runtime" && strings.Contains(f.detail, "vhdx MISSING"):
			deadWSL++
		}
	}
	// GPU util from latest measurement.sampled
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		if e.kind == "measurement.sampled" && strings.Contains(e.payload, "gpu_util") {
			var p map[string]any
			if err := json.Unmarshal([]byte(e.payload), &p); err == nil {
				gpuUtil = numOf(p, "gpu_util")
				gpuTemp = numOf(p, "gpu_temp_c")
				vramUsed = numOf(p, "vram_used_mib")
				vramTotal = numOf(p, "vram_total_mib")
			}
			break
		}
	}

	// Latest measured inference (the heartbeat of execution).
	var lastToks float64
	var lastModel string
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		if e.kind == "inference.measured" {
			var p map[string]any
			if err := json.Unmarshal([]byte(e.payload), &p); err == nil {
				if v := numOf(p, "tok_s"); v > 0 {
					lastToks = v
					lastModel = strOf(p, "model")
				}
			}
			break
		}
	}

	age := time.Since(time.UnixMilli(latest.startTs))
	stale := ""
	if age > 24*time.Hour {
		stale = trf("status.stale", age.Hours())
	}

	// --json: the one-line health becomes a machine-readable object —
	// same numbers, scriptable.
	if jsonOut {
		n, vram := loadedModels()
		out, _ := json.MarshalIndent(map[string]any{
			"running": running, "installed_idle": installed,
			"services": services, "hidden_services": hidden,
			"models": models, "dead_wsl": deadWSL,
			"gpu_util_pct": gpuUtil, "gpu_temp_c": gpuTemp,
			"vram_used_mib": vramUsed, "vram_total_mib": vramTotal,
			"last_tok_s": lastToks, "last_model": lastModel,
			"loaded_models": n, "loaded_vram_bytes": vram,
			"scan_age_s": age.Seconds(),
		}, "", "  ")
		fmt.Println(string(out))
		return
	}

	fmt.Print(trf("status.line",
		running, installed, services, hidden, models,
		gpuUtil, gpuTemp, vramUsed/1024, vramTotal/1024))
	if deadWSL > 0 {
		fmt.Printf(" | dead-wsl %d", deadWSL)
	}
	if lastToks > 0 {
		fmt.Printf(" | %.0f tok/s %s", lastToks, lastModel)
	}
	// Loaded-model state — installed ≠ loaded. Which models hold VRAM?
	if n, vram := loadedModels(); n > 0 {
		fmt.Printf(" | loaded %d (%.1f GB)", n, float64(vram)/1e9)
	}
	fmt.Println(stale)
	fmt.Println(tr("status.hint"))
}
