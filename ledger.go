package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// ledger is the append-only event store. One file: ~/.fabi/ledger.db.
// All views (resource/capability/object/knowledge/policy/driver) are queries
// over this log — second truths are forbidden (SPEC v0.4 #73).
type ledger struct {
	db     *sql.DB
	scanTs int64 // correlates events to the scan that produced them
}

type ledgerEvent struct {
	id      int64
	ts      int64
	kind    string
	payload string
}

// ledgerDir resolves the data dir: FABI_HOME env overrides ~/.fabi
// (lets the estate move to a drive with space — the disk where the
// binaries live needn't be the disk where the ledger lives).
func ledgerDir() (string, error) {
	if d := os.Getenv("FABI_HOME"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".fabi"), nil
}

func openLedger() *ledger {
	dir, err := ledgerDir()
	if err != nil {
		return &ledger{}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return &ledger{}
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "ledger.db"))
	if err != nil {
		return &ledger{}
	}
	// Append-only WAL; synchronous=OFF is acceptable for a local telemetry log.
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA synchronous=OFF")
	db.Exec(`CREATE TABLE IF NOT EXISTS events (
		id      INTEGER PRIMARY KEY AUTOINCREMENT,
		ts      INTEGER NOT NULL,
		kind    TEXT NOT NULL,
		payload TEXT NOT NULL
	)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_events_kind ON events(kind)`)
	return &ledger{db: db}
}

func (l *ledger) close() {
	if l.db != nil {
		l.db.Close()
	}
}

// emit appends one event. Payload is JSON. Never fails the scan.
func (l *ledger) emit(kind string, payload map[string]any) {
	if l.db == nil {
		return
	}
	if l.scanTs != 0 {
		payload["scan"] = l.scanTs
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_, err = l.db.Exec(
		"INSERT INTO events (ts, kind, payload) VALUES (?, ?, ?)",
		time.Now().UnixMilli(), kind, string(b),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fabi: ledger write failed: %v\n", err)
	}
}

// events returns all events of a kind, oldest first.
func (l *ledger) events(kind string) []ledgerEvent {
	if l.db == nil {
		return nil
	}
	rows, err := l.db.Query(
		"SELECT id, ts, kind, payload FROM events WHERE kind = ? ORDER BY id", kind)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []ledgerEvent
	for rows.Next() {
		var e ledgerEvent
		if err := rows.Scan(&e.id, &e.ts, &e.kind, &e.payload); err == nil {
			out = append(out, e)
		}
	}
	return out
}

// importedRanges returns the id ranges that arrived via
// `fabi ledger import` — foreign estate inside this ledger.
func (l *ledger) importedRanges() [][2]int64 {
	var out [][2]int64
	for _, e := range l.events("ledger.imported") {
		var p struct {
			FirstID int64 `json:"first_id"`
			LastID  int64 `json:"last_id"`
		}
		if json.Unmarshal([]byte(e.payload), &p) == nil && p.LastID >= p.FirstID && p.FirstID > 0 {
			out = append(out, [2]int64{p.FirstID, p.LastID})
		}
	}
	return out
}

// localEvents drops events inside imported ranges — foreign estate
// must not pollute local recommendations (W13/W14). Raw queries
// (ledger events/stats) still see everything; derived views stay local.
func localEvents(events []ledgerEvent, ranges [][2]int64) []ledgerEvent {
	if len(ranges) == 0 {
		return events
	}
	out := events[:0]
	for _, e := range events {
		foreign := false
		for _, r := range ranges {
			if e.id >= r[0] && e.id <= r[1] {
				foreign = true
				break
			}
		}
		if !foreign {
			out = append(out, e)
		}
	}
	return out
}

// allEvents returns every event, oldest first.
func (l *ledger) allEvents() []ledgerEvent {
	if l.db == nil {
		return nil
	}
	rows, err := l.db.Query("SELECT id, ts, kind, payload FROM events ORDER BY id")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []ledgerEvent
	for rows.Next() {
		var e ledgerEvent
		if err := rows.Scan(&e.id, &e.ts, &e.kind, &e.payload); err == nil {
			out = append(out, e)
		}
	}
	return out
}
