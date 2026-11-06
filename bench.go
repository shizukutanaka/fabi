package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// fabi bench — real inference measurement (SPEC #48 physics vs reality).
// Discovers a live local API, POSTs one request, measures wall time,
// writes inference.measured to the ledger. Physics assumed; bench measures.

func runBench() {
	l := openLedger()
	defer l.close()

	endpoints := []string{"http://127.0.0.1:1234", "http://127.0.0.1:11434", "http://127.0.0.1:8080"}
	for _, base := range endpoints {
		if result := tryBench(l, base); result != "" {
			fmt.Println()
			fmt.Println("FABI BENCH — real inference measurement")
			fmt.Println("─────────────────────────────────────────────")
			fmt.Println(result)
			return
		}
	}
	fmt.Println("FABI BENCH — no live inference API found on this machine.")
	fmt.Println("(checked :1234, :11434, :8080 — none responded with /v1/models)")
}

// tryBench probes an endpoint for OpenAI-compatible API and measures one call.
func tryBench(l *ledger, base string) string {
	client := &http.Client{Timeout: 4 * time.Second}

	// /v1/models — does a generative/embedding service live here?
	resp, err := client.Get(base + "/v1/models")
	if err != nil {
		return ""
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	var models struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &models); err != nil || len(models.Data) == 0 {
		return ""
	}
	model := models.Data[0].ID

	// Try embeddings (works for embedding + generative servers).
	req := map[string]any{"input": "physics says bandwidth divided by resident bytes", "model": model}
	b, _ := json.Marshal(req)
	t0 := time.Now()
	r, err := client.Post(base+"/v1/embeddings", "application/json", bytes.NewReader(b))
	lat := time.Since(t0)
	if err != nil {
		return ""
	}
	rbody, _ := io.ReadAll(r.Body)
	r.Body.Close()

	ok := r.StatusCode == 200
	l.emit("inference.measured", map[string]any{
		"endpoint": base, "model": model, "op": "embed",
		"latency_ms": lat.Milliseconds(), "status": r.StatusCode, "ok": ok,
	})
	status := "FAILED"
	if ok {
		status = fmt.Sprintf("OK — %d ms", lat.Milliseconds())
	}
	return fmt.Sprintf("endpoint: %s\nmodel:    %s\nop:       /v1/embeddings\nstatus:   HTTP %d %s\nbytes:    %d response\n\nfirst real inference measured [M] — ledger: inference.measured",
		base, model, r.StatusCode, status, len(rbody))
}
