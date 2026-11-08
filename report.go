package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Report struct {
	ToolsRunning   []Finding
	ToolsInstalled []Finding
	Services       []Finding
	Models         []Finding
	GPUs           []Finding
	Runtimes       []Finding
	Packages       []Finding
	RamUsed        uint64
	RamTotal       uint64
	FabiRSS        uint64
	ModelBytes     int64
	DupModels      int
	DupBytes       int64
}

func buildReport(results []ProbeResult, all []Finding, elapsed time.Duration, vm vmStat, rss uint64) *Report {
	r := &Report{RamUsed: vm.Used, RamTotal: vm.Total, FabiRSS: rss}

	// Merge tool findings by normalized name; running wins over installed.
	tools := map[string]*Finding{}
	toolOrder := []string{}
	mergeTool := func(f Finding) {
		key := normalizeToolName(f.Name)
		if key == "" {
			return
		}
		ex, ok := tools[key]
		if !ok {
			cp := f
			tools[key] = &cp
			toolOrder = append(toolOrder, key)
			return
		}
		if f.Running && !ex.Running {
			ex.Running = true
			ex.Detail = f.Detail
		} else if len(f.Detail) > len(ex.Detail) {
			ex.Detail = f.Detail
		}
	}

	modelNames := map[string][]Finding{}
	for _, f := range all {
		switch f.Kind {
		case "tool":
			mergeTool(f)
		case "service":
			r.Services = append(r.Services, f)
		case "model":
			r.Models = append(r.Models, f)
			r.ModelBytes += f.SizeBytes
			if f.SizeBytes > 0 {
				modelNames[strings.ToLower(f.Name)] = append(modelNames[strings.ToLower(f.Name)], f)
			}
		case "gpu":
			r.GPUs = append(r.GPUs, f)
		case "runtime":
			key := normalizeRuntimeKey(f.Name)
			dup := false
			for _, e := range r.Runtimes {
				if normalizeRuntimeKey(e.Name) == key {
					dup = true
					break
				}
			}
			if !dup {
				r.Runtimes = append(r.Runtimes, f)
			}
		case "package":
			r.Packages = append(r.Packages, f)
		}
	}
	for _, key := range toolOrder {
		f := *tools[key]
		if f.Running {
			r.ToolsRunning = append(r.ToolsRunning, f)
		} else {
			r.ToolsInstalled = append(r.ToolsInstalled, f)
		}
	}

	// duplicate model detection: same filename, multiple copies
	for _, group := range modelNames {
		if len(group) < 2 {
			continue
		}
		r.DupModels++
		var max int64
		var sum int64
		for _, f := range group {
			sum += f.SizeBytes
			if f.SizeBytes > max {
				max = f.SizeBytes
			}
		}
		r.DupBytes += sum - max
	}

	sort.Slice(r.ToolsRunning, func(i, j int) bool { return r.ToolsRunning[i].Name < r.ToolsRunning[j].Name })
	sort.Slice(r.ToolsInstalled, func(i, j int) bool { return r.ToolsInstalled[i].Name < r.ToolsInstalled[j].Name })
	sort.Slice(r.Services, func(i, j int) bool { return r.Services[i].Name < r.Services[j].Name })
	sort.Slice(r.Models, func(i, j int) bool { return r.Models[i].Name < r.Models[j].Name })
	sort.Slice(r.Runtimes, func(i, j int) bool { return r.Runtimes[i].Name < r.Runtimes[j].Name })
	_ = elapsed
	return r
}

// normalizeToolName maps process names, dir names, and package names onto one
// canonical tool key so "ollama.exe", ".ollama", and "Ollama" merge.
func normalizeToolName(name string) string {
	n := strings.ToLower(name)
	n = strings.TrimSuffix(n, " (ai-workload)")
	tag := ""
	if i := strings.Index(n, " ["); i >= 0 && strings.HasSuffix(n, "]") {
		tag = strings.TrimSpace(n[i+2 : len(n)-1])
		n = n[:i]
	}
	n = strings.TrimSuffix(n, ".exe")
	n = strings.TrimPrefix(n, ".")
	base := ""
	switch {
	case strings.Contains(n, "ollama"):
		base = "ollama"
	case strings.Contains(n, "lm-studio"), strings.Contains(n, "lmstudio"),
		strings.Contains(n, "lm studio"):
		base = "lm studio"
	case strings.Contains(n, "comfyui"):
		base = "comfyui"
	case strings.Contains(n, "gpt4all"):
		base = "gpt4all"
	case strings.Contains(n, "anythingllm"):
		base = "anythingllm"
	case strings.Contains(n, "open-webui"), strings.Contains(n, "openwebui"):
		base = "open-webui"
	case n == "jan" || strings.Contains(n, "jan.ai"):
		base = "jan"
	case strings.Contains(n, "dify"):
		base = "dify"
	case strings.Contains(n, "flowise"):
		base = "flowise"
	case strings.Contains(n, "pinokio"):
		base = "pinokio"
	case strings.Contains(n, "koboldcpp"):
		base = "koboldcpp"
	case strings.Contains(n, "docker"):
		base = "docker"
	default:
		base = n
	}
	if tag != "" {
		return base + "|" + tag
	}
	return base
}

// normalizeRuntimeKey collapses installer sub-entries into product families:
// "Python 3.11.9 (64-bit)", "Python 3.11.9 Core Interpreter" → "python@3.11".
func normalizeRuntimeKey(name string) string {
	n := strings.ToLower(name)
	for {
		i := strings.Index(n, "(")
		if i < 0 {
			break
		}
		j := strings.Index(n[i:], ")")
		if j < 0 {
			break
		}
		n = n[:i] + n[i+j+1:]
	}
	fields := strings.Fields(n)
	if len(fields) == 0 {
		return ""
	}
	key := fields[0]
	// attach major.minor from the first version-looking token
	rest := strings.Join(fields[1:], " ")
	if m := versionRe.FindStringSubmatch(rest); m != nil {
		key += "@" + m[1] + "." + m[2]
	}
	return key
}

var versionRe = regexp.MustCompile(`(\d+)\.(\d+)`)

func formatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func printReport(r *Report, results []ProbeResult, elapsed time.Duration) {
	fmt.Println()
	fmt.Println(tr("scan.title"))
	fmt.Println("─────────────────────────────────────────────")
	fmt.Printf("scan time: %d ms | fabi RSS: %s | RAM: %s / %s used\n",
		elapsed.Milliseconds(), formatBytes(int64(r.FabiRSS)),
		formatBytes(int64(r.RamUsed)), formatBytes(int64(r.RamTotal)))
	fmt.Println()

	fmt.Println(trf("sec.probes", len(results)))
	for _, p := range results {
		status := "ok"
		if p.Err != nil {
			status = "err: " + p.Err.Error()
		}
		fmt.Printf("  %-11s %5d ms  %2d findings  %s\n", p.Name, p.Duration.Milliseconds(), len(p.Findings), status)
	}
	fmt.Println()

	fmt.Println(trf("sec.tools.run", len(r.ToolsRunning)))
	for _, f := range r.ToolsRunning {
		fmt.Printf("  %-28s %s\n", f.Name, f.Detail)
	}
	if len(r.ToolsRunning) == 0 {
		fmt.Println(tr("list.none"))
	}
	fmt.Println()

	fmt.Println(trf("sec.tools.inst", len(r.ToolsInstalled)))
	for _, f := range r.ToolsInstalled {
		fmt.Printf("  %-28s %s\n", f.Name, f.Detail)
	}
	if len(r.ToolsInstalled) == 0 {
		fmt.Println(tr("list.none"))
	}
	fmt.Println()

	fmt.Println(trf("sec.services", len(r.Services)))
	for _, f := range r.Services {
		fmt.Printf("  %-28s %s\n", f.Name, f.Detail)
	}
	if len(r.Services) == 0 {
		fmt.Println(tr("list.none"))
	}
	fmt.Println()

	// Model summary: manifests (size 0) + sized files.
	var sized, unsized int
	for _, m := range r.Models {
		if m.SizeBytes > 0 {
			sized++
		} else {
			unsized++
		}
	}
	fmt.Println(trf("sec.models", len(r.Models), formatBytes(r.ModelBytes)))
	fmt.Printf("  sized files: %d | manifests/other: %d\n", sized, unsized)
	if r.DupModels > 0 {
		fmt.Printf("  duplicates: %d names, ~%s reclaimable\n", r.DupModels, formatBytes(r.DupBytes))
	}
	// show first few model names for the "numbers on screen" law
	shown := 0
	for _, m := range r.Models {
		if shown >= 8 {
			break
		}
		if m.SizeBytes > 0 {
			fmt.Printf("    %-40s %s\n", m.Name, formatBytes(m.SizeBytes))
			shown++
		}
	}
	fmt.Println()

	if len(r.GPUs) > 0 {
		fmt.Println("GPU")
		for _, g := range r.GPUs {
			fmt.Printf("  %s — %s\n", g.Name, g.Detail)
		}
		fmt.Println()
	}

	if len(r.Runtimes) > 0 {
		fmt.Printf("RUNTIMES / INFRA (%d)\n", len(r.Runtimes))
		for _, rt := range r.Runtimes {
			fmt.Printf("  %-28s %s\n", rt.Name, rt.Detail)
		}
		fmt.Println()
	}

	fmt.Println("─────────────────────────────────────────────")
	fmt.Println(tr("report.ledger.line"))
	fmt.Println(tr("report.next"))
}

// ---- fabi report — the monthly statement (#87, #76) ----

type scanAgg struct {
	ts                 int64
	modelBytes         int64
	models, services   int
	running, installed int
}

// runReport renders the statement. Numbers come only from the ledger;
// "verified" deltas appear only when scans prove bytes actually moved.
func runReport() {
	l := openLedger()
	defer l.close()

	events := localEvents(l.allEvents(), l.importedRanges())
	if len(events) == 0 {
		fmt.Println(tr("report.empty.ledger"))
		return
	}
	windows := loadScanWindows(events)

	var aggs []scanAgg
	for _, w := range windows {
		agg := scanAgg{ts: w.startTs}
		tools := map[string]bool{}
		for _, f := range w.findings {
			switch f.kind {
			case "model":
				agg.models++
				agg.modelBytes += f.size
			case "service":
				agg.services++
			case "tool":
				k := normalizeToolName(f.name)
				if f.running {
					tools[k+"|r"] = true
				} else {
					tools[k+"|i"] = true
				}
			}
		}
		for k := range tools {
			if strings.HasSuffix(k, "|r") {
				agg.running++
			} else {
				agg.installed++
			}
		}
		aggs = append(aggs, agg)
	}
	if len(aggs) == 0 {
		fmt.Println(tr("report.empty.scans"))
		return
	}
	first, latest := aggs[0], aggs[len(aggs)-1]

	// GPU: latest finding wins.
	gpuLine := "(none detected)"
	for i := len(windows) - 1; i >= 0 && gpuLine == "(none detected)"; i-- {
		for _, f := range windows[i].findings {
			if f.kind == "gpu" {
				gpuLine = f.name + " — " + f.detail
			}
		}
	}

	// Doctor recommendations: dedupe by target, keep latest, sum reclaim.
	type recLite struct {
		class   string
		reclaim int64
		ts      int64
	}
	recLatest := map[string]recLite{}
	for _, e := range events {
		if e.kind != "doctor.recommendation" {
			continue
		}
		var p map[string]any
		if err := json.Unmarshal([]byte(e.payload), &p); err != nil {
			continue
		}
		target := strOf(p, "target")
		rl := recLite{
			class:   strOf(p, "class"),
			reclaim: int64(numOf(p, "reclaim_bytes")),
			ts:      e.ts,
		}
		if cur, ok := recLatest[target]; !ok || rl.ts >= cur.ts {
			recLatest[target] = rl
		}
	}
	recByClass := map[string]int{}
	var reclaimFlagged int64
	for _, r := range recLatest {
		recByClass[r.class]++
		if r.class == "remove" {
			reclaimFlagged += r.reclaim
		}
	}

	// Verified savings: a flagged target that is absent from the latest scan
	// is proven gone — savings measured, not believed (#99, L7).
	present := map[string]bool{}
	for _, f := range windows[len(windows)-1].findings {
		present[strings.ToLower(f.name)] = true
		if f.kind == "tool" {
			present["tool:"+normalizeToolName(f.name)] = true
		}
	}
	var verifiedReclaim int64
	var verifiedCount int
	for target, r := range recLatest {
		if r.class != "remove" && r.class != "idle" {
			continue
		}
		key := strings.ToLower(target)
		if !present[key] && !present["tool:"+key] {
			verifiedCount++
			verifiedReclaim += r.reclaim
		}
	}

	verified := first.modelBytes - latest.modelBytes
	days := float64(latest.ts-first.ts) / 86400000.0

	l.emit("report.viewed", map[string]any{
		"scans": len(aggs), "days": days,
	})

	// --json contract: same numbers, machine-readable. Scripts get the
	// statement without parsing the text layout.
	if jsonOut {
		out, _ := json.MarshalIndent(map[string]any{
			"period_from":      first.ts,
			"period_to":        latest.ts,
			"scans":            len(aggs),
			"days":             days,
			"tools_running":    latest.running,
			"tools_idle":       latest.installed,
			"services":         latest.services,
			"models":           latest.models,
			"model_bytes":      latest.modelBytes,
			"verified_reclaim": verified,
			"recs_by_class":    recByClass,
			"flagged_reclaim":  reclaimFlagged,
			"verified_count":   verifiedCount,
			"verified_savings": verifiedReclaim,
		}, "", "  ")
		fmt.Println(string(out))
		return
	}

	fmt.Println()
	fmt.Println(tr("report.title"))
	fmt.Println("─────────────────────────────────────────────")
	fmt.Printf("period: %s → %s (%d scans, %.1f days)\n",
		time.UnixMilli(first.ts).Format("2006-01-02 15:04"),
		time.UnixMilli(latest.ts).Format("2006-01-02 15:04"),
		len(aggs), days)
	fmt.Println()

	fmt.Println(tr("report.estate"))
	fmt.Println(trf("estate.tools", latest.running, latest.installed))
	fmt.Printf("  services     %d answering on localhost\n", latest.services)
	fmt.Printf("  models       %d entries, %s on disk\n", latest.models, formatBytes(latest.modelBytes))
	fmt.Printf("  GPU          %s\n", gpuLine)
	fmt.Println()

	fmt.Printf("TREND (%d scans)\n", len(aggs))
	delta := ""
	if verified > 0 {
		delta = fmt.Sprintf(" — %s reclaimed, verified", formatBytes(verified))
	} else if verified < 0 {
		delta = fmt.Sprintf(" — %s added", formatBytes(-verified))
	} else {
		delta = " — nothing reclaimed yet"
	}
	fmt.Printf("  model disk   %s → %s%s\n",
		formatBytes(first.modelBytes), formatBytes(latest.modelBytes), delta)
	fmt.Printf("  running tools %d → %d\n", first.running, latest.running)
	fmt.Println(tr("report.deltas.note"))
	fmt.Println()

	// Measured inference throughput per model — the statement records
	// execution reality, not just asset inventory.
	type stat struct {
		n           int
		sum, mn, mx float64
	}
	stats := map[string]*stat{}
	var order []string
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
			s := stats[m]
			if s == nil {
				s = &stat{mn: 1e18}
				stats[m] = s
				order = append(order, m)
			}
			s.n++
			s.sum += v
			if v < s.mn {
				s.mn = v
			}
			if v > s.mx {
				s.mx = v
			}
		}
	}
	if len(order) > 0 {
		fmt.Println(tr("report.inference"))
		for _, m := range order {
			s := stats[m]
			fmt.Printf("  %-18s %3d samples  avg %5.0f  min %5.0f  max %5.0f\n",
				m, s.n, s.sum/float64(s.n), s.mn, s.mx)
		}
		fmt.Println()
	}

	fmt.Println(tr("report.work"))
	fmt.Printf("  recommendations open: %d\n", len(recLatest))
	parts := []string{}
	for _, c := range []string{"remove", "idle", "investigate"} {
		if n := recByClass[c]; n > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", c, n))
		}
	}
	if len(parts) > 0 {
		fmt.Println("    " + strings.Join(parts, " · "))
	}
	if reclaimFlagged > 0 {
		fmt.Printf("  reclaimable flagged: ~%s (awaiting manual review)\n", formatBytes(reclaimFlagged))
	}
	if verifiedCount > 0 {
		fmt.Println(trf("estate.verified",
			verifiedCount, formatBytes(verifiedReclaim)))
	} else {
		fmt.Println(tr("report.verified.none"))
	}

	// Control actions: model.unloaded events = VRAM actually freed —
	// the statement counts what fabi controlled, not just observed.
	var unloadN, unloadVia int
	var vramFreed int64
	for _, e := range events {
		if e.kind != "model.unloaded" {
			continue
		}
		var p map[string]any
		if err := json.Unmarshal([]byte(e.payload), &p); err != nil {
			continue
		}
		unloadN++
		vramFreed += int64(numOf(p, "vram_freed"))
		if strOf(p, "via") == "auto" {
			unloadVia++
		}
	}
	if unloadN > 0 {
		fmt.Printf("  control: %d VRAM release(s) — %s freed%s\n",
			unloadN, formatBytes(vramFreed),
			map[bool]string{true: fmt.Sprintf(" (%d via auto)", unloadVia)}[unloadVia > 0])
	}

	// Executions: goal chains and prompts actually run — the execution
	// pillar's footprint in the statement.
	goals := map[string]bool{}
	var steps int
	for _, e := range events {
		if e.kind != "task.executed" {
			continue
		}
		var p map[string]any
		if json.Unmarshal([]byte(e.payload), &p) == nil {
			if strOf(p, "goal") != "" {
				steps++
				goals[strOf(p, "goal")] = true
			}
		}
	}
	if len(goals) > 0 {
		fmt.Printf("  executions: %d goal(s), %d step(s) run\n", len(goals), steps)
	}
	fmt.Println()

	fmt.Println(tr("report.notyet"))
	fmt.Println(tr("report.cloud.spend"))
	fmt.Println(tr("report.gpu.hours"))
	fmt.Println()
	fmt.Println("─────────────────────────────────────────────")
	fmt.Println(tr("report.numbers.line"))
	fmt.Println(tr("report.savings.line"))
}
