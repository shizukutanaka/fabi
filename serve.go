package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
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
	fmt.Printf("fabi serve — http://127.0.0.1:%s\n", servePort)
	fmt.Println("endpoints: GET /status · GET /models · POST /run {prompt, model?}")
	fmt.Println("(loopback only — other agents can call fabi now)")
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
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
	body := ""
	if cl > 0 && cl < 1<<20 {
		b := make([]byte, cl)
		if _, err := io.ReadFull(br, b); err == nil {
			body = string(b)
		}
	}

	switch {
	case method == "GET" && path == "/status":
		writeJSON(conn, 200, statusJSON())
	case method == "GET" && path == "/models":
		writeJSON(conn, 200, modelsJSON())
	case method == "POST" && path == "/run":
		serveRun(conn, body)
	case method == "POST" && path == "/goal":
		serveGoal(conn, body)
	case method == "GET" && path == "/ledger":
		serveLedger(conn, query)
	case method == "GET" && path == "/doctor":
		serveDoctor(conn)
	default:
		writeJSON(conn, 404, map[string]string{"error": "unknown endpoint"})
	}
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
		"tok_s": toks, "wall_s": wall.Seconds(),
	})
	writeJSON(conn, 200, map[string]any{
		"model": model, "response": strings.TrimSpace(r.Response),
		"tok_s": toks, "wall_s": wall.Seconds(), "eval_count": r.EvalCount,
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
				continue
			}
			if prev != "" {
				p += "\n\nContext from previous step:\n" + trunc(prev, 600)
			}
			model := modelForClass(taskProfiles[tn].modelClass)
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
				"wall_s": wall.Seconds(),
				"output": trunc(strings.TrimSpace(r.Response), 500),
			})
			prev = r.Response
		}
		resp["steps"] = steps
		resp["executed"] = stepNum
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

func statusJSON() map[string]any {
	l := openLedger()
	defer l.close()
	events := l.allEvents()
	windows := loadScanWindows(events)
	if len(windows) == 0 {
		return map[string]any{"error": "no scans yet"}
	}
	latest := windows[len(windows)-1]
	var running, installed, services, models int
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
		}
	}
	return map[string]any{
		"running": running, "installed_idle": installed,
		"services": services, "models": models,
		"scans": len(windows), "version": version,
	}
}

func modelsJSON() map[string]any {
	st, body, err := rawHTTP(ollamaBase, "GET", "/api/tags", "", 3*time.Second)
	if err != nil || st != 200 {
		return map[string]any{"models": []string{}}
	}
	var t ollamaTags
	json.Unmarshal(body, &t)
	names := []string{}
	for _, m := range t.Models {
		names = append(names, m.Name)
	}
	return map[string]any{"models": names}
}

func writeJSON(conn net.Conn, status int, v any) {
	b, _ := json.Marshal(v)
	fmt.Fprintf(conn, "HTTP/1.1 %d OK\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
		status, len(b), b)
}
