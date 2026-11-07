package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// fabi ledger — the single truth is queryable (SPEC #60).

func runLedgerCmd(args []string) {
	l := openLedger()
	defer l.close()
	if l.db == nil {
		fmt.Println("FABI LEDGER — no database yet. Run `fabi scan` first.")
		return
	}

	sub := "stats"
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "stats", "":
		ledgerStats(l)
	case "events":
		kind := ""
		n := 10
		rest := args[1:]
		for i := 0; i < len(rest); i++ {
			switch rest[i] {
			case "-n", "--n":
				if i+1 < len(rest) {
					n, _ = strconv.Atoi(rest[i+1])
					i++
				}
			case "--kind", "-k":
				if i+1 < len(rest) {
					kind = rest[i+1]
					i++
				}
			}
		}
		ledgerTail(l, kind, n)
	case "export":
		ledgerExport(l)
	case "import":
		if len(args) < 2 {
			fmt.Println("usage: fabi ledger import <file>")
			return
		}
		ledgerImport(l, args[1])
	default:
		fmt.Println("usage: fabi ledger [stats | events [-k kind] [-n N] | export | import <file>]")
	}
}

func ledgerStats(l *ledger) {
	var total int64
	l.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&total)
	var minTs, maxTs int64
	l.db.QueryRow("SELECT COALESCE(MIN(ts),0), COALESCE(MAX(ts),0) FROM events").Scan(&minTs, &maxTs)

	fmt.Println()
	fmt.Println("FABI LEDGER — the single truth")
	fmt.Println("─────────────────────────────────────────────")
	dir, _ := ledgerDir()
	if st, err := os.Stat(filepath.Join(dir, "ledger.db")); err == nil {
		fmt.Printf("file:     %s (%.1f KB)\n", filepath.Join(dir, "ledger.db"),
			float64(st.Size())/1024)
	}
	fmt.Printf("events:   %d\n", total)
	if minTs > 0 {
		fmt.Printf("range:    %s → %s\n",
			time.UnixMilli(minTs).Format("2006-01-02 15:04"),
			time.UnixMilli(maxTs).Format("2006-01-02 15:04"))
	}
	fmt.Println()
	fmt.Println("by kind:")
	rows, err := l.db.Query("SELECT kind, COUNT(*) FROM events GROUP BY kind ORDER BY COUNT(*) DESC")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var k string
			var c int64
			rows.Scan(&k, &c)
			fmt.Printf("  %-28s %d\n", k, c)
		}
	}

	// Source breakdown: after `fabi ledger import`, events from other
	// machines live here too. scan.started carries goos — count scans
	// per OS so a merged estate still shows where each piece came from.
	fmt.Println()
	fmt.Println("scans by os:")
	srows, err := l.db.Query(
		`SELECT json_extract(payload,'$.goos'), COUNT(*) FROM events
		 WHERE kind='scan.started' GROUP BY 1`)
	if err == nil {
		defer srows.Close()
		var total int64
		for srows.Next() {
			var g string
			var c int64
			srows.Scan(&g, &c)
			tag := ""
			if g == runtime.GOOS {
				tag = " (this machine)"
			}
			fmt.Printf("  %-12s %d scan(s)%s\n", g, c, tag)
			total += c
		}
		if total == 0 {
			fmt.Println("  none")
		}
	}
}

func ledgerTail(l *ledger, kind string, n int) {
	if n <= 0 {
		n = 10
	}
	query := "SELECT ts, kind, payload FROM events "
	args := []any{}
	if kind != "" {
		query += "WHERE kind = ? "
		args = append(args, kind)
	}
	query += "ORDER BY id DESC LIMIT ?"
	args = append(args, n)

	rows, err := l.db.Query(query, args...)
	if err != nil {
		fmt.Println("fabi: ledger query failed:", err)
		return
	}
	defer rows.Close()
	fmt.Println()
	fmt.Printf("FABI LEDGER — last %d events%s\n", n, kindSuffix(kind))
	fmt.Println("─────────────────────────────────────────────")
	for rows.Next() {
		var ts int64
		var k, payload string
		rows.Scan(&ts, &k, &payload)
		fmt.Printf("%s  %-24s %s\n",
			time.UnixMilli(ts).Format("15:04:05.000"),
			k, truncate(payload, 90))
	}
}

func ledgerExport(l *ledger) {
	rows, err := l.db.Query("SELECT ts, kind, payload FROM events ORDER BY id")
	if err != nil {
		fmt.Fprintln(os.Stderr, "fabi: export failed:", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var ts int64
		var k, payload string
		rows.Scan(&ts, &k, &payload)
		var p json.RawMessage = json.RawMessage(payload)
		out, _ := json.Marshal(map[string]any{
			"ts": ts, "kind": k, "payload": p,
		})
		fmt.Println(string(out))
	}
}

// ledgerImport merges another machine's exported JSONL into this ledger —
// the first multi-PC primitive (SPEC: estate merging). Original ts and
// payload are preserved; exact duplicates are skipped; the merge itself is
// recorded so provenance stays honest.
func ledgerImport(l *ledger, path string) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Println("fabi: cannot open", path, "—", err)
		return
	}
	defer f.Close()

	seen := map[string]bool{}
	rows, err := l.db.Query("SELECT ts, kind, payload FROM events")
	if err == nil {
		for rows.Next() {
			var ts int64
			var k, p string
			rows.Scan(&ts, &k, &p)
			seen[fmt.Sprintf("%d|%s|%s", ts, k, p)] = true
		}
		rows.Close()
	}

	// Provenance: remember which id range came from this import, so
	// views can tell local evidence from foreign estate (W13).
	var firstID int64
	l.db.QueryRow("SELECT COALESCE(MAX(id),0)+1 FROM events").Scan(&firstID)

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	var imported, dup, bad int
	ins, _ := l.db.Prepare("INSERT INTO events (ts, kind, payload) VALUES (?, ?, ?)")
	defer ins.Close()
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e struct {
			Ts      int64           `json:"ts"`
			Kind    string          `json:"kind"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal([]byte(line), &e); err != nil || e.Kind == "" || e.Ts == 0 {
			bad++
			continue
		}
		p := string(e.Payload)
		if seen[fmt.Sprintf("%d|%s|%s", e.Ts, e.Kind, p)] {
			dup++
			continue
		}
		if _, err := ins.Exec(e.Ts, e.Kind, p); err == nil {
			imported++
		}
	}
	l.emit("ledger.imported", map[string]any{
		"source": path, "imported": imported,
		"duplicates_skipped": dup, "malformed": bad,
		"first_id": firstID, "last_id": firstID + int64(imported) - 1,
	})
	fmt.Printf("fabi ledger import: %d new event(s), %d duplicate(s) skipped, %d malformed\n",
		imported, dup, bad)
	if imported > 0 {
		fmt.Println("estate merged — `fabi diff` and `fabi report` now see the imported history")
	}
}

func kindSuffix(k string) string {
	if k == "" {
		return ""
	}
	return " (kind=" + k + ")"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
