# Fabi

**Universal AI Control Plane** — scans your machine's AI estate, writes an
append-only ledger, and tells you the truth with numbers.

Not an LLM. Not a launcher. The layer that makes every AI tool visible.

## Commands

```text
fabi scan       detect AI tools, models, hardware → ledger → estate report
fabi watch      periodic scan, timeseries (default 5 min) + inference heartbeat
fabi doctor     evidence-based recommendations (remove / idle / investigate)
fabi report     statement — estate trend + measured inference stats
fabi selftest   falsification engine — every spec claim gets a test
fabi ledger     query the event log (stats | events | export)
fabi route      physics + measured model verdict for a task
fabi goal       purpose → task chain → per-task model scheduling (--run)
fabi install    register fabi watch in Startup (fully reversible)
fabi bench      hardware capability measurement
fabi status     one-line health: estate + GPU + tok/s + loaded models
fabi run        execute on a live model (--model, --ctx, --sweep, --compare)
fabi serve      JSON API :11435 (GET /status|/models|/ledger|/doctor · POST /run|/goal)
fabi unload     free VRAM — unload loaded models (real control action)
fabi auto       apply safe recs only — idle unload, user-triggered (Automation seed)
```

Remote engines: `OLLAMA_HOST=<host>:11434` routes run/goal/serve to a
remote or cloud ollama-compatible endpoint — the "cloud or downgrade"
path route recommends.

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
