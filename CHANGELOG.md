# Changelog

## v0.0.5-dev — unreleased

- **`fabi auto`** — Automation Engine seed: applies only safe
  recommendations (idle unload >15 min); user-triggered,
  non-destructive, keeps recently-used models warm
- **`OLLAMA_HOST`** — standard env var routes run/goal/serve to
  remote/cloud ollama-compatible endpoints; route's
  "cloud or downgrade" verdict now points to it

## v0.0.4 — 2026-11-07

Research-driven hardening (Qiita/Zenn/papers/GitHub analysis):

- **`fabi unload`** — first real control action: `keep_alive:0` frees
  VRAM; verified via `/api/ps` (2.0 GB freed on reference hw)
- **Loaded-model observation** — `/api/ps` → `status` shows
  `loaded N (X GB)`; installed ≠ loaded
- **GPU queue** — all inference serialized via `execPrompt` mutex
  (sluice/bastion pattern — no concurrent GPU access, no OOM)
- **MLPerf-lite sweep** — warmup + median-of-3 + auto-unload;
  honest central values, VRAM returns to baseline
- **CARROT-lite scheduler** — `modelForTask`: largest model meeting
  the latency constraint wins (measured tok/s, physics fallback)
- **TTFT** — `prompt_eval_duration` → prefill/decode separated
  in `run`, `--sweep`, and `inference.measured` events
- **`--ctx`** — num_ctx control per request (verified: 8k ctx
  loads 0.6 GB vs 4k's 0.5 GB)
- **doctor Rule 7** — idle loaded models holding VRAM → actionable
  `fabi unload` recommendation (observation→diagnosis→control loop)
- **Ghost-VRAM attribution** — `/api/ps` correlates loaded models
  into the unattributed-VRAM recommendation (461 MiB explained)
- **selftest** — resource-control claim falsified via
  `model.unloaded` events (10 PASS)
- **`--compare`** — prove scheduler decisions with measured outcomes

## v0.0.3-dev — 2026-11-07

Phase 2 became live code — execution primitives, measured:

- **`fabi run "<prompt>"`** — execute on a live local model,
  measure real tok/s, record `task.executed`
- **`fabi run --sweep`** — measured tok/s across all live models
  (qwen 0.5b = 193.8, llama3.2 1b = 75.9 on reference hw)
- **`fabi run --model <name>`** — pick a specific model
- **`fabi goal "..." --run`** — goal → task decomposition →
  per-task model scheduling (3b→fastest, 8b→largest) →
  chained execution with threaded outputs
- **`fabi route`** — measured local models ranked next to the
  physics prediction
- **Ledger completeness fix (L15)** — every chain step emits
  `task.executed` with goal/task/step/model/tok_s
- **httpio: chunked transfer-encoding decode** — ollama replies
  chunked; rawHTTP now handles it (raw TCP, still no net/http)

## v0.0.2-dev — 2026-11-07

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
