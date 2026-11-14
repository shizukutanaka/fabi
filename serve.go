package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

// fabi serve — minimal JSON API over raw sockets (no net/http — keeps
// the binary under budget). Universal API v0: other tools call fabi.
//
//   GET  /status  → one-line health JSON
//   GET  /models  → live local models (ollama passthrough)
//   POST /run     → {prompt, model?} → {response, tok_s, model}
//
// Loopback only, no auth — same permission model as probes.

const servePort = "11435"

func runServe() {
	ln, err := net.Listen("tcp", "127.0.0.1:"+servePort)
	if err != nil {
		fmt.Println("fabi serve:", err)
		return
	}
	fmt.Println(trf("serve.listen", servePort))
	fmt.Println("endpoints: GET /status|/models|/ledger|/doctor|/pattern|/diff|/why|/route|/findings|/lang|/events · POST /run|/goal|/unload|/scan|/pattern|/import · /ui dashboard")
	fmt.Println(tr("serve.loopback"))
	// The control surface starting is itself a ledgered fact — audit.
	l := openLedger()
	l.emit("serve.started", map[string]any{
		"addr": "127.0.0.1:" + servePort, "ts": time.Now().UnixMilli(),
	})
	l.close()
	// Graceful Ctrl+C: stop accepting, ledger the stop, exit cleanly.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		l2 := openLedger()
		l2.emit("serve.stopped", map[string]any{"ts": time.Now().UnixMilli()})
		l2.close()
		ln.Close()
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			return // listener closed — graceful stop
		}
		go serveConn(conn)
	}
}

func serveConn(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(120 * time.Second))
	br := bufio.NewReader(conn)

	line, err := br.ReadString('\n')
	if err != nil {
		return
	}
	parts := strings.Fields(line)
	if len(parts) < 2 {
		writeJSON(conn, 400, map[string]string{"error": "bad request"})
		return
	}
	method, path := parts[0], parts[1]
	var query string
	if i := strings.IndexByte(path, '?'); i >= 0 {
		query = path[i+1:]
		path = path[:i]
	}

	// Headers — we only need Content-Length for POST bodies.
	cl := 0
	for {
		h, err := br.ReadString('\n')
		if err != nil || h == "\r\n" || h == "\n" {
			break
		}
		if strings.HasPrefix(strings.ToLower(h), "content-length:") {
			fmt.Sscanf(strings.TrimSpace(h[15:]), "%d", &cl)
		}
	}
	maxCL := 1 << 20
	if path == "/import" {
		maxCL = 16 << 20 // estates are bulk — a ledger export is ~MB
	}
	if cl >= maxCL {
		conn.Write([]byte("HTTP/1.1 413 Payload Too Large\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"))
		return
	}
	body := ""
	if cl > 0 {
		b := make([]byte, cl)
		if _, err := io.ReadFull(br, b); err == nil {
			body = string(b)
		}
	}

	switch {
	case method == "GET" && (path == "/ui" || path == "/ui/"):
		writeHTML(conn, 200, uiHTML)
	case method == "GET" && path == "/status":
		writeJSON(conn, 200, statusJSON())
	case method == "GET" && path == "/models":
		writeJSON(conn, 200, modelsJSON())
	case method == "GET" && path == "/findings":
		writeJSON(conn, 200, findingsJSON())
	case method == "GET" && path == "/lang":
		writeJSON(conn, 200, uiLabelsJSON())
	case method == "GET" && path == "/events":
		serveSSE(conn)
	case method == "POST" && path == "/run":
		serveRun(conn, body)
	case method == "POST" && path == "/goal":
		serveGoal(conn, body)
	case method == "GET" && path == "/ledger":
		serveLedger(conn, query)
	case method == "GET" && path == "/doctor":
		serveDoctor(conn)
	case method == "GET" && path == "/diff":
		serveDiff(conn)
	case method == "GET" && path == "/why":
		serveWhy(conn, query)
	case method == "POST" && path == "/unload":
		serveUnload(conn)
	case method == "POST" && path == "/scan":
		serveScan(conn)
	case method == "GET" && path == "/pattern":
		servePattern(conn, body, query, false)
	case method == "POST" && path == "/pattern":
		servePattern(conn, body, query, true)
	case method == "POST" && path == "/import":
		serveImport(conn, body)
	case method == "GET" && path == "/update":
		serveUpdate(conn)
	case method == "GET" && path == "/route":
		serveRoute(conn, query)
	default:
		writeJSON(conn, 404, map[string]string{"error": "unknown endpoint"})
	}
}

// serveRoute — GET /route?task=summarize: physics verdict as JSON.
func serveRoute(conn net.Conn, query string) {
	task := ""
	for _, kv := range strings.Split(strings.TrimPrefix(query, "?"), "&") {
		if p := strings.SplitN(kv, "=", 2); len(p) == 2 && p[0] == "task" {
			task = p[1]
		}
	}
	if task == "" {
		writeJSON(conn, 400, map[string]string{"error": "need ?task= — tasks: " + strings.Join(taskNames(), ", ")})
		return
	}
	d, ok := routeData(task)
	if !ok {
		writeJSON(conn, 400, map[string]string{"error": "unknown task — tasks: " + strings.Join(taskNames(), ", ")})
		return
	}
	writeJSON(conn, 200, d)
}

// serveUpdate — GET /update: honest version check over the API.
func serveUpdate(conn net.Conn) {
	cur, latest, url, behind, err := updateCheck()
	if err != nil {
		writeJSON(conn, 502, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(conn, 200, map[string]any{
		"current": cur, "latest": latest, "behind": behind, "url": url,
	})
}

func serveRun(conn net.Conn, body string) {
	var req struct {
		Prompt string `json:"prompt"`
		Model  string `json:"model"`
	}
	if err := json.Unmarshal([]byte(body), &req); err != nil || req.Prompt == "" {
		writeJSON(conn, 400, map[string]string{"error": `need {"prompt":"..."}`})
		return
	}
	model := req.Model
	if model == "" {
		model = bestLocalModel()
	}
	l := openLedger()
	defer l.close()
	r, toks, wall, err := execPrompt(model, req.Prompt)
	if err != nil {
		l.emit("task.executed", map[string]any{
			"ok": false, "model": model, "via": "serve", "error": err.Error(),
		})
		writeJSON(conn, 502, map[string]string{"error": err.Error()})
		return
	}
	l.emit("task.executed", map[string]any{
		"ok": true, "model": model, "via": "serve",
		"prompt": trunc(req.Prompt, 200), "eval_count": r.EvalCount,
		"tok_s": toks, "ttft_ms": r.ttftMs, "wall_s": wall.Seconds(),
	})
	writeJSON(conn, 200, map[string]any{
		"model": model, "response": strings.TrimSpace(r.Response),
		"tok_s": toks, "ttft_ms": r.ttftMs, "wall_s": wall.Seconds(),
		"eval_count": r.EvalCount, "prefill_tok": r.PromptEvalCount,
	})
}

// serveGoal: POST /goal {"goal":"...","run":true} — decompose a purpose
// into tasks, return verdicts; optionally execute the chain and return
// per-step results. Universal API's planning endpoint.
func serveGoal(conn net.Conn, body string) {
	var req struct {
		Goal string `json:"goal"`
		Run  bool   `json:"run"`
	}
	if err := json.Unmarshal([]byte(body), &req); err != nil || req.Goal == "" {
		writeJSON(conn, 400, map[string]string{"error": `need {"goal":"..."}`})
		return
	}

	var tasks []string
	gl := strings.ToLower(req.Goal)
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

	resp := map[string]any{"goal": req.Goal, "tasks": tasks}

	if req.Run {
		l := openLedger()
		defer l.close()
		var steps []map[string]any
		var skipped []string
		var prev string
		stepNum := 0
		for _, tn := range tasks {
			p, ok := map[string]string{
				"planning":    "Produce a short 5-step plan for: " + req.Goal,
				"code_review": "List quality risks for: " + req.Goal,
				"summarize":   "Summarize in one sentence: " + req.Goal,
				"translate":   "Translate to English: " + req.Goal,
				"reasoning":   "Explain briefly: " + req.Goal,
				"search":      "List 3 key questions about: " + req.Goal,
				"ocr":         "Describe how to extract text for: " + req.Goal,
			}[tn]
			if !ok {
				skipped = append(skipped, tn)
				continue
			}
			if prev != "" {
				p += "\n\nContext from previous step:\n" + trunc(prev, 600)
			}
			model := modelForTask(taskProfiles[tn])
			r, toks, wall, err := execPrompt(model, p)
			if err != nil {
				steps = append(steps, map[string]any{
					"task": tn, "error": err.Error(),
				})
				break
			}
			stepNum++
			l.emit("task.executed", map[string]any{
				"ok": true, "model": model, "task": tn, "goal": req.Goal,
				"step": stepNum, "via": "serve",
				"tok_s": toks, "wall_s": wall.Seconds(),
			})
			steps = append(steps, map[string]any{
				"task": tn, "model": model, "tok_s": toks,
				"ttft_ms": r.ttftMs, "wall_s": wall.Seconds(),
				"output": trunc(strings.TrimSpace(r.Response), 500),
			})
			prev = r.Response
		}
		resp["steps"] = steps
		resp["executed"] = stepNum
		if len(skipped) > 0 {
			resp["skipped"] = skipped
			// Actionable: is the estate's image engine reachable?
			st, _, err := rawHTTP("127.0.0.1:8188", "GET", "/", "", 2*time.Second)
			resp["image_engine_up"] = err == nil && st > 0
		}
	}
	writeJSON(conn, 200, resp)
}

// serveLedger: GET /ledger?kind=X&n=N — query events over the API.
// The single truth is now machine-readable by any local agent.
func serveLedger(conn net.Conn, query string) {
	kind, n := "", 20
	for _, kv := range strings.Split(query, "&") {
		p := strings.SplitN(kv, "=", 2)
		if len(p) == 2 {
			switch p[0] {
			case "kind":
				kind = p[1]
			case "n":
				fmt.Sscanf(p[1], "%d", &n)
			}
		}
	}
	if n > 200 {
		n = 200
	}
	l := openLedger()
	defer l.close()
	var out []map[string]any
	events := l.allEvents()
	for i := len(events) - 1; i >= 0 && len(out) < n; i-- {
		e := events[i]
		if kind != "" && e.kind != kind {
			continue
		}
		var payload map[string]any
		json.Unmarshal([]byte(e.payload), &payload)
		out = append(out, map[string]any{
			"id": e.id, "ts": e.ts, "kind": e.kind, "payload": payload,
		})
	}
	writeJSON(conn, 200, map[string]any{
		"count": len(out), "events": out,
	})
}

// serveDoctor: GET /doctor — latest open recommendations per target.
func serveDoctor(conn net.Conn) {
	l := openLedger()
	defer l.close()
	latest := map[string]map[string]any{}
	for _, e := range l.events("doctor.recommendation") {
		var p map[string]any
		if err := json.Unmarshal([]byte(e.payload), &p); err != nil {
			continue
		}
		if t, ok := p["target"].(string); ok {
			p["event_id"] = e.id
			latest[t] = p
		}
	}
	var out []map[string]any
	for _, r := range latest {
		out = append(out, r)
	}
	writeJSON(conn, 200, map[string]any{
		"count": len(out), "recommendations": out,
	})
}

// serveUnload: POST /unload — free VRAM via keep_alive:0 for every
// loaded model. Control over the API, ledgered like everything else.
func serveUnload(conn net.Conn) {
	st, body, err := rawHTTP(ollamaBase, "GET", "/api/ps", "", 3*time.Second)
	if err != nil || st != 200 {
		writeJSON(conn, 502, map[string]string{"error": "ollama not reachable"})
		return
	}
	var ps struct {
		Models []struct {
			Name     string `json:"name"`
			SizeVRAM int64  `json:"size_vram"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &ps); err != nil {
		writeJSON(conn, 500, map[string]string{"error": "bad /api/ps"})
		return
	}
	l := openLedger()
	defer l.close()
	var freed int64
	var names []string
	for _, m := range ps.Models {
		payload, _ := json.Marshal(map[string]any{"model": m.Name, "keep_alive": 0})
		rawHTTP(ollamaBase, "POST", "/api/generate", string(payload), 10*time.Second)
		freed += m.SizeVRAM
		names = append(names, m.Name)
		l.emit("model.unloaded", map[string]any{
			"model": m.Name, "vram_freed": m.SizeVRAM, "via": "serve",
		})
	}
	writeJSON(conn, 200, map[string]any{
		"unloaded": names, "vram_freed_bytes": freed, "count": len(names),
	})
}

// serveSSE: GET /events — Server-Sent Events stream of new ledger
// events. The conn has a 120s deadline; the client (EventSource)
// auto-reconnects, so the stream is effectively continuous.
func serveSSE(conn net.Conn) {
	fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\nContent-Type: text/event-stream\r\n"+
		"Cache-Control: no-cache\r\nConnection: keep-alive\r\n\r\n")
	l := openLedger()
	defer l.close()
	last := int64(0)
	deadline := time.Now().Add(100 * time.Second)
	for time.Now().Before(deadline) {
		var latest []map[string]any
		for _, e := range l.allEvents() {
			if e.id > last {
				var p map[string]any
				json.Unmarshal([]byte(e.payload), &p)
				latest = append(latest, map[string]any{
					"id": e.id, "ts": e.ts, "kind": e.kind, "payload": p,
				})
				last = e.id
			}
		}
		for _, ev := range latest {
			b, _ := json.Marshal(ev)
			if _, err := fmt.Fprintf(conn, "data: %s\n\n", b); err != nil {
				return
			}
		}
		if _, err := fmt.Fprintf(conn, ": keepalive\n\n"); err != nil {
			return
		}
		time.Sleep(2 * time.Second)
	}
}

// uiLabelsJSON — the dashboard's labels in the detected language.
// The embedded /ui is English-only HTML until the JS asks for this
// map — the same tr() engine the CLI uses, served as data.
func uiLabelsJSON() map[string]any {
	return map[string]any{
		"models": tr("sec.models"), "services": tr("sec.services"),
		"runtimes": tr("scan.runtimes"), "packages": tr("sec.packages"),
		"recs": tr("doctor.title"), "action": tr("ui.action"),
		"scan": tr("ui.scan"), "unload": tr("ui.unload"), "auto": tr("ui.auto"),
		"running": tr("ui.running"), "installed": tr("ui.installed"),
		"reclaimable": tr("ui.reclaimable"), "refreshed": tr("ui.refreshed"),
		"watchAlive": tr("ui.watch.alive"), "watchStale": tr("ui.watch.stale"),
		"watchStopped": tr("ui.watch.stopped"),
		"none":         tr("ui.none"), "noScans": tr("ui.noscans"),
		"noMeasure": tr("ui.nomeasure"), "estate": tr("ui.estate"),
		"history": tr("ui.history"), "inference": tr("ui.inference"),
		"control": tr("ui.control"), "detected": langCode, "title": tr("ui.title"),
		"tools": tr("ui.tools"),
	}
}

// serveScan: POST /scan — trigger a fresh observation over the API.
// Agents can ask fabi to look again; the full probe pipeline runs
// and every finding lands in the ledger before we answer.
func serveScan(conn net.Conn) {
	runScan(time.Now(), true) // quiet — full pipeline, one-line to serve's stdout
	writeJSON(conn, 200, statusJSON())
}

// servePattern: GET/POST /pattern — agents extend detection through
// data too. POST {"field":"exe","value":"x.exe"} adds to the user
// file; GET lists user patterns. Same ~/.fabi/patterns.json store.
func servePattern(conn net.Conn, body, query string, write bool) {
	dir, err := ledgerDir()
	if err != nil {
		writeJSON(conn, 500, map[string]string{"error": err.Error()})
		return
	}
	path := filepath.Join(dir, "patterns.json")
	var user patternSet
	if b, err := os.ReadFile(path); err == nil {
		json.Unmarshal(b, &user)
	}
	if !write {
		writeJSON(conn, 200, map[string]any{"user_patterns": user})
		return
	}
	var req struct {
		Field string `json:"field"`
		Value string `json:"value"`
		URL   string `json:"url"`
	}
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		writeJSON(conn, 400, map[string]string{"error": `need {"field":"...","value":"..."} or {"url":"..."}`})
		return
	}
	if req.URL != "" {
		installPatternPack(req.URL, path, &user)
		writeJSON(conn, 200, map[string]any{"installed": req.URL})
		return
	}
	if err := addPatternField(&user, req.Field, req.Value); err != nil {
		writeJSON(conn, 400, map[string]string{"error": err.Error()})
		return
	}
	b, _ := json.MarshalIndent(user, "", "  ")
	if err := os.WriteFile(path, b, 0644); err != nil {
		writeJSON(conn, 500, map[string]string{"error": err.Error()})
		return
	}
	activePatterns = nil
	writeJSON(conn, 200, map[string]any{"added": req.Value, "field": req.Field})
}

// addPatternField appends to a user patternSet by field name —
// shared logic for the CLI and API pattern commands.
func addPatternField(user *patternSet, field, value string) error {
	switch field {
	case "exe":
		user.ExePatterns = append(user.ExePatterns, value)
	case "cmd":
		user.CmdPatterns = append(user.CmdPatterns, value)
	case "scripthost":
		user.ScriptHosts = append(user.ScriptHosts, value)
	case "dir":
		user.ScanDirs = append(user.ScanDirs, value)
	case "modelroot":
		user.ModelRoots = append(user.ModelRoots, value)
	case "pkgtool":
		user.PkgAITools = append(user.PkgAITools, value)
	case "pkgruntime":
		user.PkgRuntimes = append(user.PkgRuntimes, value)
	case "port":
		name, port := value, 0
		if i := strings.LastIndexByte(value, ':'); i >= 0 {
			name = value[:i]
			fmt.Sscanf(value[i+1:], "%d", &port)
		}
		if port <= 0 {
			return fmt.Errorf("port needs name:port, e.g. myservice:9000")
		}
		user.Ports = append(user.Ports, portTarget{Port: port, Name: name})
	default:
		return fmt.Errorf("unknown field %q — exe|cmd|scripthost|dir|modelroot|pkgtool|pkgruntime|port", field)
	}
	return nil
}

// serveDiff: GET /diff — estate delta over API, same logic as
// `fabi diff` but machine-readable for agents.
func serveDiff(conn net.Conn) {
	l := openLedger()
	defer l.close()
	windows := loadScanWindows(l.allEvents())
	if len(windows) < 2 {
		writeJSON(conn, 200, map[string]any{"error": "need 2+ scans"})
		return
	}
	prev, latest := windows[len(windows)-2], windows[len(windows)-1]
	key := func(f findingRec) string {
		if f.kind == "tool" {
			return "tool:" + normalizeToolName(f.name)
		}
		return f.kind + ":" + strings.ToLower(f.name)
	}
	having := map[string]findingRec{}
	for _, f := range prev.findings {
		having[key(f)] = f
	}
	var added, removed []map[string]any
	for _, f := range latest.findings {
		if _, ok := having[key(f)]; !ok {
			added = append(added, map[string]any{
				"kind": f.kind, "name": f.name, "detail": f.detail,
			})
		}
	}
	nowKeys := map[string]bool{}
	for _, f := range latest.findings {
		nowKeys[key(f)] = true
	}
	for k, f := range having {
		if !nowKeys[k] {
			removed = append(removed, map[string]any{
				"kind": f.kind, "name": f.name, "detail": f.detail,
			})
		}
	}
	writeJSON(conn, 200, map[string]any{
		"from": prev.startTs, "to": latest.startTs,
		"added": added, "removed": removed,
	})
}

// serveWhy: GET /why?target=X — evidence chain over API. Agents
// can ask why fabi says something, not just what it says.
func serveWhy(conn net.Conn, query string) {
	target := ""
	for _, kv := range strings.Split(strings.TrimPrefix(query, "?"), "&") {
		if p := strings.SplitN(kv, "=", 2); len(p) == 2 && p[0] == "target" {
			target = p[1]
		}
	}
	if target == "" {
		writeJSON(conn, 400, map[string]string{"error": "need ?target="})
		return
	}
	term := strings.ToLower(target)
	l := openLedger()
	defer l.close()
	events := l.allEvents()
	byID := map[int64]ledgerEvent{}
	for _, e := range events {
		byID[e.id] = e
	}
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		if e.kind != "doctor.recommendation" {
			continue
		}
		var p map[string]any
		if json.Unmarshal([]byte(e.payload), &p) != nil ||
			!strings.Contains(strings.ToLower(strOf(p, "target")), term) {
			continue
		}
		var ev []map[string]any
		if a, ok := p["evidence_ids"].([]any); ok {
			for _, v := range a {
				if f, ok := v.(float64); ok {
					if ev2, ok2 := byID[int64(f)]; ok2 {
						ev = append(ev, map[string]any{
							"id": int64(f), "ts": ev2.ts, "kind": ev2.kind,
						})
					}
				}
			}
		}
		writeJSON(conn, 200, map[string]any{
			"target": strOf(p, "target"), "class": strOf(p, "class"),
			"detail": strOf(p, "detail"), "confidence": strOf(p, "confidence"),
			"evidence": ev,
		})
		return
	}
	writeJSON(conn, 404, map[string]string{"error": "no recommendation for " + target})
}

func statusJSON() map[string]any {
	l := openLedger()
	defer l.close()
	events := l.allEvents()
	windows := loadScanWindows(events)
	if len(windows) == 0 {
		// Keep the documented shape (API.md) in every state: consumers
		// read the count fields unconditionally, so a bare error map
		// renders as undefined values. The error key stays for anyone
		// keying on it, alongside the truthful zero counts.
		return map[string]any{
			"error":   "no scans yet",
			"running": 0, "installed_idle": 0,
			"services": 0, "models": 0,
			"scans": 0, "version": version, "gpu": "",
		}
	}
	latest := windows[len(windows)-1]
	var running, installed, services, models int
	var gpuName string
	for _, f := range latest.findings {
		switch f.kind {
		case "tool":
			if f.running {
				running++
			} else {
				installed++
			}
		case "service":
			services++
		case "model":
			models++
		case "gpu":
			gpuName = f.name
		}
	}
	resp := map[string]any{
		"running": running, "installed_idle": installed,
		"services": services, "models": models,
		"scans": len(windows), "version": version, "gpu": gpuName,
	}
	if n, vram := loadedModels(); n > 0 {
		resp["loaded_models"] = n
		resp["loaded_vram_bytes"] = vram
	}
	return resp
}

// findingsJSON — the latest scan's findings grouped by kind, for the
// /ui dashboard's full estate view (tools/services/models/runtimes/packages).
func findingsJSON() map[string]any {
	l := openLedger()
	defer l.close()
	windows := loadScanWindows(l.allEvents())
	if len(windows) == 0 {
		return map[string]any{"error": "no scans yet"}
	}
	groups := map[string][]map[string]any{}
	for _, f := range windows[len(windows)-1].findings {
		groups[f.kind] = append(groups[f.kind], map[string]any{
			"name": f.name, "detail": f.detail,
			"size_bytes": f.size, "running": f.running,
		})
	}
	return map[string]any{"groups": groups}
}

func modelsJSON() map[string]any {
	names := []string{}
	if st, body, err := rawHTTP(ollamaBase, "GET", "/api/tags", "", 3*time.Second); err == nil && st == 200 {
		var t ollamaTags
		if json.Unmarshal(body, &t) == nil {
			for _, m := range t.Models {
				names = append(names, m.Name)
			}
		}
	}
	// Second engine — agents see the whole estate's live models,
	// not just ollama's (multi-engine parity with `fabi run`).
	if st, b, err := rawHTTP(lmstudioBase, "GET", "/v1/models", "", 3*time.Second); err == nil && st == 200 {
		var lm struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if json.Unmarshal(b, &lm) == nil {
			for _, m := range lm.Data {
				names = append(names, "lmstudio:"+m.ID)
			}
		}
	}
	return map[string]any{"models": names}
}

// serveImport: POST /import — merge an exported estate (JSONL body)
// into this ledger. API parity with `fabi ledger import` — local
// processes can push estates without shell access (loopback only).
func serveImport(conn net.Conn, body string) {
	if strings.TrimSpace(body) == "" {
		writeJSON(conn, 400, map[string]string{"error": "empty body — POST JSONL events"})
		return
	}
	l := openLedger()
	defer l.close()
	if l.db == nil {
		writeJSON(conn, 500, map[string]string{"error": "no ledger"})
		return
	}
	sc := bufio.NewScanner(strings.NewReader(body))
	imported, dup, bad := importJSONL(l, sc, "api:/import")
	writeJSON(conn, 200, map[string]any{
		"imported": imported, "duplicates_skipped": dup, "malformed": bad,
	})
}

// uiHTML — the embedded dashboard. Single file, no external assets,
// loopback-only. Fetches the JSON endpoints and renders them.
const uiHTML = `<!doctype html><meta charset=utf-8><title>fabi</title>
<style>
body{font:14px/1.5 system-ui,sans-serif;background:#0d1117;color:#c9d1d9;margin:0;padding:1.5rem;max-width:52rem}
h1{font-size:1rem;color:#58a6ff;border-bottom:1px solid #21262d;padding-bottom:.4rem}
h2{font-size:.8rem;text-transform:uppercase;letter-spacing:.05em;color:#8b949e;margin:1.4rem 0 .4rem}
.card{background:#161b22;border:1px solid #21262d;border-radius:8px;padding:.8rem 1rem;margin-bottom:.6rem}
.num{font-weight:600;color:#d2a8ff}
.dim{color:#8b949e;font-size:.85rem}
.pill{display:inline-block;padding:.1rem .5rem;border-radius:99px;background:#21262d;font-size:.75rem}
.dot{display:inline-block;width:.6rem;height:.6rem;border-radius:50%;vertical-align:middle}
.ok{color:#3fb950}.warn{color:#d29922}.bad{color:#f85149}
table{width:100%;border-collapse:collapse}td,th{text-align:left;padding:.2rem .4rem;font-size:.85rem;border-bottom:1px solid #21262d}
th{color:#8b949e;font-weight:500}
pre{white-space:pre-wrap;word-break:break-all;margin:0}
button{background:#21262d;color:#c9d1d9;border:1px solid #30363d;border-radius:6px;padding:.3rem .8rem;font:inherit;cursor:pointer}
button:hover{border-color:#58a6ff}
</style>
<h1 id=hl>fabi — the honest eye on this PC's AI estate</h1>
<div id=s></div>
<h2 id=ht>AI tools</h2><div id=t></div>
<h2 id=hm>Models</h2><div id=m></div>
<h2 id=hg>Services / runtimes / packages</h2><div id=g></div>
<h2 id=hr>Recommendations</h2><div id=r></div>
<h2 id=hh>Scan history</h2><div id=h></div>
<h2 id=hi>Inference trend</h2><div id=i></div>
<h2 id=hc>Control</h2>
<div class=card><button id=bs onclick="do('scan')">scan now</button>
<button id=bu onclick="do('unload')">unload idle models</button>
<button id=ba onclick="do('auto')">fabi auto (safe only)</button><div id=c></div></div>
<h2>Raw</h2><div id=x></div>
<script>
const J = async p => (await fetch(p)).json();
function card(t,d){return '<div class=card>'+t+(d?'<div class=dim>'+d+'</div>':'')+'</div>'}
// Localized labels — the same tr() engine the CLI uses, served by /lang.
const L = await (await fetch('/lang')).json();
hl.textContent=L.title;
ht.textContent=L.tools;
hm.textContent=L.models.replace(/%.*/,'').trim();
hg.textContent=L.estate; hr.textContent=L.recs;
hh.textContent=L.history; hi.textContent=L.inference; hc.textContent=L.control;
bs.textContent=L.scan; bu.textContent=L.unload; ba.textContent=L.auto;
const none=L.none;
async function tick(){
 const st=await J('/status');
 if(st.error){s.innerHTML=card(L.noScans,'');}
 else s.innerHTML=card('<span class=num>'+st.running+'</span> '+L.running+' · <span class=num>'+st.installed_idle+'</span> '+L.installed+' · <span class=num>'+st.models+'</span> '+plain(L.models)+(st.gpu?' · <span class=dim>'+st.gpu+'</span>':'')+' <span class=dim>· '+L.refreshed+' '+new Date().toLocaleTimeString()+'</span>', '');
 const ms=await J('/models');
 m.innerHTML=(ms.models||[]).map(x=>card('<b>'+x+'</b>')).join('')||'<div class=dim>'+none+'</div>';
 const d=await J('/doctor');
 const act=(c,t)=>({idle:'fabi unload '+t,remove:'fabi report',investigate:'fabi scan'})[c]||'fabi auto';
 const rec=(d.recommendations||[]).filter(x=>x.class=='remove').reduce((a,x)=>a+(x.reclaim_bytes||0),0);
 if(rec>0)s.innerHTML+=card(L.reclaimable+': <span class=num>~'+(rec/1073741824).toFixed(1)+' GB</span> — '+L.reclaimableNote);
 r.innerHTML=(d.recommendations||[]).map(x=>card('<span class=pill>'+x.class+'</span> <b>'+x.target+'</b>'+(x.reclaim_bytes?' <span class=pill>~'+(x.reclaim_bytes/1073741824).toFixed(1)+'GB</span>':'')+'<pre>'+x.detail+'</pre><div class=dim>'+L.action+': '+act(x.class,x.target)+'</div>')).join('')||'<div class=dim>'+none+'</div>';
 const fd=await J('/findings');
 const grp=(fd.groups||{});
 const sec=(n,items)=>(items&&items.length)?'<div class=dim>'+n+'</div>'+(items||[]).map(x=>'<div class=card><b>'+x.name+'</b><pre class=dim>'+x.detail+'</pre></div>').join(''):'';
 const plain=t=>(t||'').replace(/ *\(%[ds]\)/,'').replace(/ *%[ds].*/,'');
 g.innerHTML=sec(plain(L.services),grp.service)+sec(plain(L.runtimes),grp.runtime)+sec(plain(L.packages),grp.package)||'<div class=dim>'+none+'</div>';
 // Watch-daemon liveness — ui.state events mean fabi watch is alive.
 const w=await J('/ledger?kind=ui.state&n=1');
 const we=(w.events||[])[0];
 if(we){
  const age=(Date.now()-we.ts)/60000;
  const stopped=we.payload&&we.payload.stopped;
  const live=!stopped&&age<10;
  const lbl=stopped?L.watchStopped:(live?L.watchAlive:L.watchStale);
  const col=live?'#3fb950':(stopped?'#d29922':'#8b949e');
  s.innerHTML+=card('<span class=dot style="background:'+col+'"></span> '+lbl+' <span class=dim>'+age.toFixed(0)+'m ago</span>');
 }
 // Scan history — the ledger's time dimension as a bar sparkline.
 const h=await J('/ledger?kind=report.generated&n=30');
 const ev=(h.events||[]).reverse();
 if(ev.length){
  const mx=Math.max(...ev.map(e=>e.payload.findings||0),1);
  h.innerHTML=card(ev.map(e=>{
   const p=Math.round((e.payload.findings||0)/mx*100);
   const d=new Date(e.ts).toLocaleString();
   return '<div class=dim title="'+d+' — '+(e.payload.findings||0)+' findings">'
    +'<span style="display:inline-block;background:#58a6ff;height:.7rem;width:'+p+'%;border-radius:2px;vertical-align:middle"></span>'
    +' <span class=num>'+(e.payload.findings||0)+'</span> <span style="font-size:.7rem">'+d+'</span></div>';
  }).join(''));
 } else h.innerHTML='<div class=dim>'+L.noScans+'</div>';
 // Inference trend — measured tok/s over time from inference.measured.
 const inf=await J('/ledger?kind=inference.measured&n=30');
 const iev=(inf.events||[]).reverse();
 if(iev.length){
  const im=Math.max(...iev.map(e=>e.payload.tok_s||0),1);
  i.innerHTML=card(iev.map(e=>{
   const p=Math.round((e.payload.tok_s||0)/im*100);
   const m=e.payload.model||'?';
   return '<div class=dim title="'+new Date(e.ts).toLocaleString()+' — '+m+'">'
    +'<span style="display:inline-block;background:#3fb950;height:.7rem;width:'+p+'%;border-radius:2px;vertical-align:middle"></span>'
    +' <span class=num>'+(e.payload.tok_s||0).toFixed(0)+'</span> <span style="font-size:.7rem">'+m+'</span></div>';
  }).join(''));
 } else i.innerHTML='<div class=dim>'+L.noMeasure+'</div>';
 x.innerHTML='<div class=dim><pre>'+JSON.stringify(st,null,2)+'</pre></div>';
}
tick(); setInterval(tick, 4000);
// Live updates — new ledger events refresh the view immediately.
try{new EventSource('/events').onmessage=e=>{
 const d=JSON.parse(e.data);
 if(d.kind=='report.generated'||d.kind=='inference.measured'||d.kind=='model.unloaded')tick();
};}catch(_){}
async function do(p){
 if(!confirm(p+' — are you sure?'))return;
 c.innerHTML='<div class=dim>running…</div>';
 const r=await (await fetch('/'+p,{method:'POST'})).json();
 c.innerHTML=card(JSON.stringify(r));
 if(p=='scan')setTimeout(tick,1500); // re-fetch after a scan lands
}
</script>`

func writeHTML(conn net.Conn, status int, html string) {
	fmt.Fprintf(conn, "HTTP/1.1 %d OK\r\nContent-Type: text/html; charset=utf-8\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
		status, len(html), html)
}

func writeJSON(conn net.Conn, status int, v any) {
	b, _ := json.Marshal(v)
	fmt.Fprintf(conn, "HTTP/1.1 %d OK\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
		status, len(b), b)
}
