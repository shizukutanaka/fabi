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

func ledgerDir() (string, error) {
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
