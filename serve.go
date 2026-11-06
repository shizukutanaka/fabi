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
