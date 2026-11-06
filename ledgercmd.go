package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	default:
		fmt.Println("usage: fabi ledger [stats | events [-k kind] [-n N] | export]")
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
