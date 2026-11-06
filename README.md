# Fabi

**Universal AI Control Plane** — scans your machine's AI estate, writes an
append-only ledger, and tells you the truth with numbers.

Not an LLM. Not a launcher. The layer that makes every AI tool visible.

## Commands

```text
fabi scan       detect AI tools, models, hardware → ledger → estate report
fabi watch      periodic scan, timeseries (default 5 min)
fabi doctor     evidence-based recommendations (remove / idle / investigate)
fabi report     statement — estate trend + reclaimable, measured
fabi selftest   falsification engine — every spec claim gets a test
fabi ledger     query the event log (stats | events | export)
```

## Build

```sh
go build -ldflags="-s -w" -o fabi.exe .
./build.sh      # all 3 OSes + sha256 checksums
```

## Design

- **One truth**: everything is an event in `~/.fabi/ledger.db` (append-only SQLite)
- **Patterns are data**: `patterns.json` embedded + `~/.fabi/patterns.json` merge —
  extend detection with zero code changes
- **No promises without numbers**: every claim has a budget and a measured value
  (`fabi selftest` verifies live)
- **Deletion is manual**: doctor never deletes; it shows evidence, you decide

## Measured (Windows, RTX A2000, 2026-11-07)

binary 7.5 MB · scan ~0.1–1.5 s · RSS ~15 MB · 0 config · offline core

See `SPEC_v0.6.md` for the full design + measured results.
