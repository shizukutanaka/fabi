# Changelog

## v0.0.2-dev — 2026-11-07 (unreleased)

Real fixes verified by the falsification loop:

- **doctor: sha256-verify dup candidates** — name+size heuristic
  produced a false-positive "160MB reclaimable" (vae vs vae_1_0
  are separate revisions). `remove` now requires content equality. L14
- **install: Startup-folder launcher** — schtasks needed admin;
  Startup `.cmd` needs none. install/uninstall verified end-to-end
- **.gitattributes eol=lf** — fixes gofmt on Windows runners/users
- CI green on all 3 OSes

## v0.0.1-dev — 2026-11-07

First measured build. Phase 0–1 complete; Phase 2 seeded.

### Commands (11)

| Command | What it does |
|---|---|
| `scan [--json]` | 6 probes → ledger → estate report |
| `watch [interval]` | periodic scan, timeseries (default 5m) |
| `doctor` | evidence-backed recommendations (remove/idle/investigate) |
| `report` | statement — trend + reclaimable + verified |
| `selftest` | falsification engine — every claim gets a test |
| `ledger` | query events (stats / events / export) |
| `route <task>` | physics: tok/s = bandwidth ÷ resident bytes |
| `goal "<text>"` | goal → tasks → local feasibility |
| `install`/`uninstall` | logon task, dry-run, `--yes` required |
| `bench` | real inference measurement on a live API |
| `status` | one-line health from the ledger |

### Measured (Windows, RTX A2000 12GB, 2026-11-07)

binary 7.6 MB · scan 0.1–1.5 s · RSS ~14 MB · 0 config · offline · tests 5/5

### Real findings on the reference machine

3 idle tools · 2 hidden services (LM Studio + dead gateway)
· 1 dead WSL distro · 10.5 GB unattributed VRAM · 2 unknown
agent tools caught by generic patterns · 1 false-positive
dup detected and corrected (sha256 verify — L14)

### Design laws now in code

Ledger = single truth (views only) · patterns = data (`patterns.json`)
· missing provenance → inference note · deletion always manual ·
per-probe timeouts · dependency deleted when cost > value
