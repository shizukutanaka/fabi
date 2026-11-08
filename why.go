package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// fabi why <target> — explain why fabi says what it says. L4 made
// user-visible: every recommendation carries evidence_ids back to
// the ledger events that justify it. Show that chain.
func runWhy(args []string) {
	var wargs []string
	for _, a := range args {
		if a == "--json" {
			jsonOut = true
		} else {
			wargs = append(wargs, a)
		}
	}
	if len(wargs) == 0 {
		fmt.Println("usage: fabi why <name> [--json] — explain a recommendation/finding")
		return
	}
	term := strings.ToLower(strings.Join(wargs, " "))
	l := openLedger()
	defer l.close()
	events := localEvents(l.allEvents(), l.importedRanges())
	byID := map[int64]ledgerEvent{}
	for _, e := range events {
		byID[e.id] = e
	}

	// Find the latest recommendation whose target matches.
	var best *struct {
		class, target, detail, conf string
		ids                         []int64
	}
	for _, e := range events {
		if e.kind != "doctor.recommendation" {
			continue
		}
		var p map[string]any
		if json.Unmarshal([]byte(e.payload), &p) != nil {
			continue
		}
		t := strOf(p, "target")
		if !strings.Contains(strings.ToLower(t), term) {
			continue
		}
		var ids []int64
		if a, ok := p["evidence_ids"].([]any); ok {
			for _, v := range a {
				if f, ok := v.(float64); ok {
					ids = append(ids, int64(f))
				}
			}
		}
		best = &struct {
			class, target, detail, conf string
			ids                         []int64
		}{strOf(p, "class"), t, strOf(p, "detail"),
			strOf(p, "confidence"), ids}
	}

	// --json: the evidence chain as structured data — explainability
	// becomes machine-auditable.
	if jsonOut && best != nil {
		var ev []map[string]any
		for _, id := range best.ids {
			e, ok := byID[id]
			if !ok {
				continue
			}
			var p map[string]any
			json.Unmarshal([]byte(e.payload), &p)
			ev = append(ev, map[string]any{
				"id": id, "ts": e.ts, "kind": e.kind, "payload": p,
			})
		}
		out, _ := json.MarshalIndent(map[string]any{
			"target": best.target, "class": best.class,
			"detail": best.detail, "confidence": best.conf,
			"evidence": ev,
		}, "", "  ")
		fmt.Println(string(out))
		return
	}
	if jsonOut && best == nil {
		var ev []map[string]any
		var n int
		for i := len(events) - 1; i >= 0 && n < 10; i-- {
			e := events[i]
			if !strings.Contains(strings.ToLower(e.payload), term) {
				continue
			}
			var p map[string]any
			json.Unmarshal([]byte(e.payload), &p)
			ev = append(ev, map[string]any{
				"id": e.id, "ts": e.ts, "kind": e.kind, "payload": p,
			})
			n++
		}
		out, _ := json.MarshalIndent(map[string]any{
			"target": term, "recommendation": nil,
			"observations": ev,
		}, "", "  ")
		fmt.Println(string(out))
		return
	}

	fmt.Println()
	if best != nil {
		fmt.Printf("WHY %s\n", best.target)
		fmt.Println("─────────────────────────────────────────────")
		fmt.Printf("  recommendation: [%s] %s\n", best.class, best.detail)
		fmt.Printf("  confidence: %s\n\n", best.conf)
		if len(best.ids) == 0 {
			fmt.Println(tr("why.heuristic"))
		} else {
			fmt.Println(tr("why.evidence"))
			for _, id := range best.ids {
				e, ok := byID[id]
				if !ok {
					continue
				}
				var p map[string]any
				json.Unmarshal([]byte(e.payload), &p)
				fmt.Printf("    #%d %s  %s  %s\n",
					id, time.UnixMilli(e.ts).Format("01-02 15:04"),
					e.kind, trunc(fmt.Sprint(p), 60))
			}
		}
		return
	}

	// No recommendation — show every event mentioning the term so the
	// user can see fabi's raw observations about it.
	fmt.Printf("WHY %s — no recommendation; raw observations:\n", term)
	fmt.Println("─────────────────────────────────────────────")
	var n int
	for i := len(events) - 1; i >= 0 && n < 10; i-- {
		e := events[i]
		if !strings.Contains(strings.ToLower(e.payload), term) {
			continue
		}
		var p map[string]any
		json.Unmarshal([]byte(e.payload), &p)
		fmt.Printf("  #%d %s  %s  %s\n",
			e.id, time.UnixMilli(e.ts).Format("01-02 15:04"),
			e.kind, trunc(fmt.Sprint(p), 60))
		n++
	}
	if n == 0 {
		fmt.Println(tr("why.nothing"))
	}
}
