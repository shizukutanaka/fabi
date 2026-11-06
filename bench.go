package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// fabi bench — real inference measurement (SPEC #48 physics vs reality).
// Discovers a live local API, POSTs one request, measures wall time,
// writes inference.measured to the ledger. Physics assumed; bench measures.

func runBench() {
	l := openLedger()
	defer l.close()

	endpoints := []string{"127.0.0.1:1234", "127.0.0.1:11434", "127.0.0.1:8080"}
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
	status, body, err := rawHTTP(base, "GET", "/v1/models", "", 4*time.Second)
	if err != nil || status != 200 {
		return ""
	}
	var models struct {
		Data []struct{ ID string `json:"id"` } `json:"data"`
	}
	if err := json.Unmarshal(body, &models); err != nil || len(models.Data) == 0 {
		return ""
	}
	model := models.Data[0].ID

	payload, _ := json.Marshal(map[string]any{
		"input": "physics says bandwidth divided by resident bytes",
		"model": model,
	})
	t0 := time.Now()
	st, rbody, err := rawHTTP(base, "POST", "/v1/embeddings", string(payload), 4*time.Second)
	lat := time.Since(t0)
	if err != nil {
		return ""
	}

	ok := st == 200
	l.emit("inference.measured", map[string]any{
		"endpoint": "http://" + base, "model": model, "op": "embed",
		"latency_ms": lat.Milliseconds(), "status": st, "ok": ok,
	})
	statusTxt := "FAILED"
	if ok {
		statusTxt = fmt.Sprintf("OK — %d ms", lat.Milliseconds())
	}
	return fmt.Sprintf("endpoint: http://%s\nmodel:    %s\nop:       /v1/embeddings\nstatus:   HTTP %d %s\nbytes:    %d response\n\nfirst real inference measured [M] — ledger: inference.measured",
		base, model, st, statusTxt, len(rbody))
}

var _ = strings.Contains
