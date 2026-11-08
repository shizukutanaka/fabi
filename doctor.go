package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
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

	events := localEvents(l.allEvents(), l.importedRanges())
	if len(events) == 0 {
		fmt.Println("FABI DOCTOR — ledger is empty. Run `fabi scan` first.")
		return
	}

	windows := loadScanWindows(events)
	if len(windows) == 0 || len(windows[len(windows)-1].findings) == 0 {
		fmt.Println(tr("doctor.empty.findings"))
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
		allSame := true
		for _, g := range group[1:] {
			if sameContent(keep.path, g.path) {
				fmt.Fprintf(&detail, "remove: %s (%s)\n", orPath(g.path), formatBytes(g.size))
				dupReclaim += g.size
			} else {
				allSame = false
				fmt.Fprintf(&detail, "different content — NOT a duplicate: %s\n", orPath(g.path))
			}
			ev = append(ev, g.id)
		}
		ev = append(ev, keep.id)
		if allSame {
			recs = append(recs, recommendation{
				class: "remove", target: name, detail: strings.TrimSpace(detail.String()),
				confidence: "high — sha256 identical content",
				reclaim:    dupReclaim, evidence: ev,
			})
		} else {
			recs = append(recs, recommendation{
				class: "investigate", target: name,
				detail:     strings.TrimSpace(detail.String()),
				confidence: "high — same name+size but sha256 differs (separate revisions)",
				reclaim:    0, evidence: ev,
			})
		}
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
		// Correlate with loaded models — some of the "ghost" burn is
		// just resident weights. /api/ps knows what ollama holds.
		var loadedVRAM int64
		var loadedNames []string
		if st, b, err := rawHTTP(ollamaBase, "GET", "/api/ps", "", 3*time.Second); err == nil && st == 200 {
			var ps struct {
				Models []struct {
					Name     string `json:"name"`
					SizeVRAM int64  `json:"size_vram"`
				} `json:"models"`
			}
			if json.Unmarshal(b, &ps) == nil {
				for _, m := range ps.Models {
					loadedVRAM += m.SizeVRAM
					loadedNames = append(loadedNames, m.Name)
				}
			}
		}
		if loadedVRAM > 0 {
			explained := float64(loadedVRAM) / 1048576
			pct := explained / gapMiB * 100
			detail += fmt.Sprintf(
				"\n    explained: %.0f MiB (%.0f%%) is loaded models — %s",
				explained, pct, strings.Join(loadedNames, ", "))
		}
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

	// --- Rule 6: inference throughput regression ---
	// Watch's heartbeat exists for this: real tok/s over time. If a model's
	// measured throughput fell >30% (first-half vs second-half average),
	// something real changed — thermal, contention, or a swapped engine.
	type meas struct {
		id   int64
		toks float64
	}
	byModel := map[string][]meas{}
	for _, e := range events {
		if e.kind != "inference.measured" {
			continue
		}
		var p map[string]any
		if err := json.Unmarshal([]byte(e.payload), &p); err != nil {
			continue
		}
		if v, ok := p["tok_s"].(float64); ok && v > 0 {
			m := strOf(p, "model")
			byModel[m] = append(byModel[m], meas{e.id, v})
		}
	}
	for model, ms := range byModel {
		if len(ms) < 4 {
			continue
		}
		half := len(ms) / 2
		var oldAvg, newAvg float64
		for _, m := range ms[:half] {
			oldAvg += m.toks
		}
		oldAvg /= float64(half)
		var ev []int64
		for _, m := range ms[half:] {
			newAvg += m.toks
			ev = append(ev, m.id)
		}
		newAvg /= float64(len(ms) - half)
		if oldAvg > 0 && newAvg < oldAvg*0.7 {
			recs = append(recs, recommendation{
				class:  "investigate",
				target: "inference " + model,
				detail: fmt.Sprintf(
					"tok/s fell from ~%.0f to ~%.0f (-%.0f%%) across %d samples\n"+
						"    causes: thermal throttle, memory/CPU contention, engine swap",
					oldAvg, newAvg, (1-newAvg/oldAvg)*100, len(ms)),
				confidence: "medium — measured regression in ledger timeseries",
				evidence:   ev[:min(3, len(ev))],
			})
		}
	}

	// --- Rule 7: idle loaded models holding VRAM ---
	// /api/ps knows what's resident; the ledger knows when it was last
	// used. A model loaded but unused for a while is reclaimable VRAM —
	// fabi can actually free it (unload), making this an actionable rec.
	lastUse := map[string]int64{}
	for _, e := range events {
		if e.kind != "task.executed" && e.kind != "inference.measured" {
			continue
		}
		var p map[string]any
		if err := json.Unmarshal([]byte(e.payload), &p); err != nil {
			continue
		}
		if m := strOf(p, "model"); m != "" && e.ts > lastUse[m] {
			lastUse[m] = e.ts
		}
	}
	st, psBody, err := rawHTTP(ollamaBase, "GET", "/api/ps", "", 3*time.Second)
	if err == nil && st == 200 {
		var ps struct {
			Models []struct {
				Name     string `json:"name"`
				SizeVRAM int64  `json:"size_vram"`
			} `json:"models"`
		}
		if err := json.Unmarshal(psBody, &ps); err == nil {
			cutoff := time.Now().Add(-15 * time.Minute).UnixMilli()
			for _, m := range ps.Models {
				if lastUse[m.Name] > cutoff {
					continue // recently used — leave it warm
				}
				recs = append(recs, recommendation{
					class:  "idle",
					target: "model " + m.Name,
					detail: fmt.Sprintf(
						"loaded, holding %.1f GB VRAM, unused >15min\n"+
							"    action: fabi unload %s",
						float64(m.SizeVRAM)/1e9, m.Name),
					confidence: "high — resident but idle, reclaimable",
				})
			}
		}
	}

	// --- Rule 8: installed but never-invoked models ---
	// The estate's biggest waste is model files that were never used.
	// "Never invoked through a fabi-tracked path" — honest framing;
	// the model may predate fabi. Recommendation is manual review (L9).
	var usedModels = map[string]bool{}
	for _, e := range events {
		if e.kind != "task.executed" && e.kind != "inference.measured" {
			continue
		}
		var p map[string]any
		if json.Unmarshal([]byte(e.payload), &p) == nil {
			if m := strOf(p, "model"); m != "" {
				usedModels[m] = true
			}
		}
	}
	// Installed models come from ollama /api/tags — those that never
	// appear in task/inference events are reclaim candidates.
	st, tagsBody, err := rawHTTP(ollamaBase, "GET", "/api/tags", "", 3*time.Second)
	if err == nil && st == 200 {
		var t ollamaTags
		if err := json.Unmarshal(tagsBody, &t); err == nil {
			for _, m := range t.Models {
				if usedModels[m.Name] || len(usedModels) == 0 {
					continue // used, or fabi has no usage data yet
				}
				recs = append(recs, recommendation{
					class:  "idle",
					target: "model " + m.Name,
					detail: fmt.Sprintf(
						"installed (%.1f GB) but never invoked via fabi — "+
							"reclaim disk by removing manually if unneeded",
						float64(m.Size)/1e9),
					confidence: "medium — untracked, may predate fabi",
				})
			}
		}
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

	// --json: recommendations as a machine-readable array — the
	// actionable output scripts consume.
	if jsonOut {
		var arr []map[string]any
		for _, r := range recs {
			arr = append(arr, map[string]any{
				"class":         r.class,
				"target":        r.target,
				"detail":        r.detail,
				"confidence":    r.confidence,
				"reclaim_bytes": r.reclaim,
				"evidence_ids":  r.evidence,
			})
		}
		out, _ := json.MarshalIndent(map[string]any{
			"recommendations": arr,
			"emitted":         emitted,
			"unchanged":       unchanged,
			"reclaim_total":   reclaimTotal,
		}, "", "  ")
		fmt.Println(string(out))
		return
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

// sameContent verifies two paths hold byte-identical files via sha256.
// Empty paths or unreadable files → not identical (never recommend removal).
func sameContent(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	ha, hb := fileSHA256(a), fileSHA256(b)
	return ha != "" && ha == hb
}

func fileSHA256(p string) string {
	f, err := os.Open(p)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

func printDoctor(recs []recommendation, nEvents, nScans int, reclaim int64, latest scanWindow, emitted, unchanged int64) {
	fmt.Println()
	fmt.Println(tr("doctor.title"))
	fmt.Println("─────────────────────────────────────────────")
	fmt.Printf("ledger: %d events across %d scans | latest scan: %d findings\n",
		nEvents, nScans, len(latest.findings))
	fmt.Println()

	if len(recs) == 0 {
		fmt.Println(tr("doctor.none"))
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
		fmt.Println(tr("doctor.note"))
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
		fmt.Println(tr("doctor.investigate"))
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
	fmt.Println(tr("doctor.footer"))
	fmt.Println()
	fmt.Println("next:   fabi why <target> — see the evidence behind a recommendation")
	fmt.Println("        fabi report       — the full estate statement")
	fmt.Println("        fabi auto         — apply only safe idle-model unloads")
}
