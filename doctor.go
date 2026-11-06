package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// doctor turns ledger history into evidence-backed recommendations.
// Rule (SPEC v0.4 #78): no evidence → no recommendation.
// Fabi never deletes user files autonomously — apply is always manual.

type findingRec struct {
	id      int64
	ts      int64
	kind    string
	name    string
	detail  string
	size    int64
	running bool
	path    string
}

type scanWindow struct {
	startTs  int64
	findings []findingRec
}

type recommendation struct {
	class      string // remove | idle | investigate | info
	target     string
	detail     string
	confidence string
	reclaim    int64
	evidence   []int64
}

// loadScanWindows partitions resource.found events into scan windows.
// Shared by doctor and report — one query, many views (#73).
func loadScanWindows(events []ledgerEvent) []scanWindow {
	var scanStarts []int64
	for _, e := range events {
		if e.kind == "scan.started" {
			scanStarts = append(scanStarts, e.ts)
		}
	}
	windows := make([]scanWindow, len(scanStarts))
	for i, ts := range scanStarts {
		windows[i].startTs = ts
	}
	windowOf := func(ts int64, payloadScan int64) int {
		for i, w := range windows {
			if payloadScan != 0 && payloadScan == w.startTs {
				return i
			}
		}
		wi := 0
		for i, ts2 := range scanStarts {
			if ts2 <= ts {
				wi = i
			}
		}
		return wi
	}
	for _, e := range events {
		if e.kind != "resource.found" {
			continue
		}
		var p map[string]any
		if err := json.Unmarshal([]byte(e.payload), &p); err != nil {
			continue
		}
		f := findingRec{
			id:     e.id,
			ts:     e.ts,
			kind:   strOf(p, "kind"),
			name:   strOf(p, "name"),
			detail: strOf(p, "detail"),
			path:   strOf(p, "path"),
		}
		f.size = int64(numOf(p, "size_bytes"))
		f.running = p["running"] == true
		var ps int64
		if v, ok := p["scan"].(float64); ok {
			ps = int64(v)
		}
		wi := windowOf(e.ts, ps)
		windows[wi].findings = append(windows[wi].findings, f)
	}
	return windows
}

func runDoctor() {
	l := openLedger()
	defer l.close()

	events := l.allEvents()
	if len(events) == 0 {
		fmt.Println("FABI DOCTOR — ledger is empty. Run `fabi scan` first.")
		return
	}

	windows := loadScanWindows(events)
	if len(windows) == 0 || len(windows[len(windows)-1].findings) == 0 {
		fmt.Println("FABI DOCTOR — scans exist but no findings recorded yet.")
		return
	}
	latest := windows[len(windows)-1]

	var recs []recommendation

	// --- Rule 1: duplicate models in latest scan ---
	dupGroups := map[string][]findingRec{}
	for _, f := range latest.findings {
		if f.kind == "model" && f.size > 0 {
			dupGroups[strings.ToLower(f.name)] = append(dupGroups[strings.ToLower(f.name)], f)
		}
	}
	var dupReclaim int64
	for name, group := range dupGroups {
		if len(group) < 2 {
			continue
		}
		sort.Slice(group, func(i, j int) bool {
			if group[i].size != group[j].size {
				return group[i].size > group[j].size
			}
			return group[i].path < group[j].path
		})
		keep := group[0]
		var ev []int64
		var detail strings.Builder
		fmt.Fprintf(&detail, "keep: %s (%s)\n", orPath(keep.path), formatBytes(keep.size))
		for _, g := range group[1:] {
			fmt.Fprintf(&detail, "remove: %s (%s)\n", orPath(g.path), formatBytes(g.size))
			dupReclaim += g.size
			ev = append(ev, g.id)
		}
		ev = append(ev, keep.id)
		recs = append(recs, recommendation{
			class: "remove", target: name, detail: strings.TrimSpace(detail.String()),
			confidence: "high — identical name+size in latest scan",
			reclaim:    dupReclaim, evidence: ev,
		})
	}

	// --- Rule 2: installed tools never observed running ---
	type toolStat struct {
		installedWindows int
		runningWindows   int
		latest           *findingRec
		ids              []int64
	}
	toolStats := map[string]*toolStat{}
	toolOrder := []string{}
	for wi, w := range windows {
		seen := map[string]bool{}
		for i := range w.findings {
			f := &w.findings[i]
			if f.kind != "tool" {
				continue
			}
			key := normalizeToolName(f.name)
			if seen[key] {
				continue
			}
			seen[key] = true
			st, ok := toolStats[key]
			if !ok {
				st = &toolStat{}
				toolStats[key] = st
				toolOrder = append(toolOrder, key)
			}
			if !f.running {
				st.installedWindows++
			} else {
				st.runningWindows++
			}
			if wi == len(windows)-1 {
				cp := *f
				st.latest = &cp
			}
			st.ids = append(st.ids, f.id)
		}
	}
	for _, key := range toolOrder {
		st := toolStats[key]
		if st.latest == nil || st.latest.running {
			continue
		}
		if st.installedWindows < 2 || st.runningWindows > 0 {
			continue
		}
		conf := "low"
		if st.installedWindows >= 5 {
			conf = "medium"
		}
		detail := fmt.Sprintf("installed in %d/%d scans, running 0/%d — %s",
			st.installedWindows, len(windows), len(windows), st.latest.detail)
		recs = append(recs, recommendation{
			class: "idle", target: key, detail: detail,
			confidence: conf, evidence: lastN(st.ids, 5),
		})
	}

	// --- Rule 3: hidden compute ---
	var hidden []int64
	var hiddenDetail []string
	for _, f := range latest.findings {
		if f.kind == "service" && strings.Contains(f.detail, "owner not visible") {
			hidden = append(hidden, f.id)
			hiddenDetail = append(hiddenDetail, fmt.Sprintf("%s @ %s", f.name, f.detail))
		}
	}
	if len(hidden) > 0 {
		recs = append(recs, recommendation{
			class: "investigate", target: "hidden compute",
			detail: strings.Join(hiddenDetail, "\n    ") +
				"\n    likely WSL2 / VM / container — inspect: wsl -l -v, docker ps",
			confidence: "info — provenance gap, not an error",
			evidence:   hidden,
		})
	}

	// --- Rule 4: dead WSL installs ---
	for _, f := range latest.findings {
		if strings.Contains(f.detail, "vhdx MISSING") {
			recs = append(recs, recommendation{
				class:  "investigate",
				target: strings.ToLower(f.name),
				detail: fmt.Sprintf("%s\n    cleanup: wsl --unregister %s",
					f.detail, strings.TrimPrefix(f.name, "WSL distro: ")),
				confidence: "medium — registration exists, disk image gone",
				evidence:   []int64{f.id},
			})
		}
	}

	// --- Rule 5: GPU burn suspects ---
	// VRAM used but nobody attributed → list running AI workloads as candidates.
	var gapMiB float64
	var gapEv int64
	for _, f := range latest.findings {
		if f.kind == "gpu" && strings.Contains(f.name, "unattributed") {
			gapEv = f.id
			fmt.Sscanf(f.detail, "%f MiB", &gapMiB)
		}
	}
	if gapMiB > 0 {
		var suspects []string
		var susEv []int64
		for _, f := range latest.findings {
			if f.kind == "tool" && f.running && strings.Contains(f.name, "[") {
				suspects = append(suspects, f.name+" — "+f.detail)
				susEv = append(susEv, f.id)
			}
		}
		detail := fmt.Sprintf("%.0f MiB unattributed VRAM", gapMiB)
		if len(suspects) > 0 {
			detail += "\n    suspects (running ai-workloads):\n      " +
				strings.Join(suspects, "\n      ")
		}
		recs = append(recs, recommendation{
			class:      "investigate",
			target:     "gpu burn",
			detail:     detail,
			confidence: "info — attribution gap, suspects are circumstantial",
			evidence:   append([]int64{gapEv}, susEv...),
		})
	}

	// --- Emit recommendations back to the ledger ---
	// Dedupe: don't re-emit a recommendation identical to the latest open one.
	latestRec := map[string]map[string]any{}
	for _, e := range l.events("doctor.recommendation") {
		var p map[string]any
		if err := json.Unmarshal([]byte(e.payload), &p); err == nil {
			if t, ok := p["target"].(string); ok {
				latestRec[t] = p
			}
		}
	}
	var emitted, unchanged, reclaimTotal int64
	for _, r := range recs {
		if prev, ok := latestRec[r.target]; ok &&
			strOf(prev, "class") == r.class &&
			strOf(prev, "detail") == r.detail {
			unchanged++
			continue
		}
		l.emit("doctor.recommendation", map[string]any{
			"class":         r.class,
			"target":        r.target,
			"detail":        r.detail,
			"confidence":    r.confidence,
			"reclaim_bytes": r.reclaim,
			"evidence_ids":  r.evidence,
		})
		emitted++
		if r.class == "remove" {
			reclaimTotal += r.reclaim
		}
	}

	printDoctor(recs, len(events), len(windows), reclaimTotal, latest, emitted, unchanged)
}

func strOf(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}

func numOf(m map[string]any, k string) float64 {
	if v, ok := m[k].(float64); ok {
		return v
	}
	return 0
}

func orPath(p string) string {
	if p == "" {
		return "(path not recorded — pre-path ledger event)"
	}
	return p
}

func lastN(ids []int64, n int) []int64 {
	if len(ids) <= n {
		return ids
	}
	return ids[len(ids)-n:]
}

func printDoctor(recs []recommendation, nEvents, nScans int, reclaim int64, latest scanWindow, emitted, unchanged int64) {
	fmt.Println()
	fmt.Println("FABI DOCTOR — evidence-based recommendations")
	fmt.Println("─────────────────────────────────────────────")
	fmt.Printf("ledger: %d events across %d scans | latest scan: %d findings\n",
		nEvents, nScans, len(latest.findings))
	fmt.Println()

	if len(recs) == 0 {
		fmt.Println("No recommendations. Every finding has provenance.")
	}

	var remove, idle, investigate []recommendation
	for _, r := range recs {
		switch r.class {
		case "remove":
			remove = append(remove, r)
		case "idle":
			idle = append(idle, r)
		case "investigate":
			investigate = append(investigate, r)
		}
	}

	if len(remove) > 0 {
		fmt.Printf("REMOVE — duplicate content (~%s reclaimable)\n", formatBytes(reclaim))
		for _, r := range remove {
			fmt.Printf("  [%s] %s\n", r.confidence, r.target)
			for _, line := range strings.Split(r.detail, "\n") {
				fmt.Printf("      %s\n", line)
			}
			fmt.Printf("      evidence: events %v\n", r.evidence)
		}
		fmt.Println("  note: fabi never deletes user files autonomously.")
		fmt.Println("        remove candidates manually after review.")
		fmt.Println()
	}

	if len(idle) > 0 {
		fmt.Printf("IDLE — installed, never observed running (%d scans of history)\n", nScans)
		for _, r := range idle {
			fmt.Printf("  [%s] %s — %s\n", r.confidence, r.target, r.detail)
		}
		fmt.Println()
	}

	if len(investigate) > 0 {
		fmt.Println("INVESTIGATE — provenance gaps")
		for _, r := range investigate {
			fmt.Printf("  [%s] %s\n", r.confidence, r.target)
			for _, line := range strings.Split(r.detail, "\n") {
				fmt.Printf("      %s\n", line)
			}
		}
		fmt.Println()
	}

	fmt.Println("─────────────────────────────────────────────")
	fmt.Printf("recommendations: %d new, %d unchanged\n", emitted, unchanged)
	fmt.Println("deletion is always manual — evidence first, action never automatic")
}
