# Changelog

## Unreleased

Development since v0.0.7 (`main.go` version is `0.0.8-dev`):
graceful Ctrl+C for `fabi serve` and `fabi tui`, the version shown
in the TUI header, and added route/goal physics test coverage.

## v0.0.7 — 2026-11-08

Corresponds to git tag `v0.0.7` and the
[GitHub Release v0.0.7](https://github.com/shizukutanaka/fabi/releases/tag/v0.0.7)
(prebuilt binaries for windows/linux/darwin × amd64/arm64, plus
`SHA256SUMS.txt`). The entries below — previously listed under
Unreleased — shipped in this release, together with the three
front ends (CLI, `fabi tui`, and the `fabi serve` `/ui` dashboard):

- **`fabi tui`** — live terminal dashboard (status/tools/models/
  recommendations) sharing the doctor engine with the CLI and `/ui`
- **serve** — SSE `GET /events` (dashboards refresh on ledger events
  instead of polling), `GET /findings`, and watch-daemon liveness
  shown in `fabi status`, the TUI, and `/ui`
- **full i18n** — 233 keys × 10 languages
  (en/ja/zh/es/de/fr/ko/pt/ru/it), audited to zero unlocalized
  literals: `FABI_LANG`/`LANG`/`LC_ALL` + Windows
  `GetUserDefaultUILanguage`; per-language `lang_<code>.go` data
  files; every user-facing surface localized — bare fabi, all
  23 help descriptions, update, status, doctor, scan, report,
  ledger, diff, why, run, sweep, compare, unload, auto, install,
  uninstall, goal, route (incl. verdicts), watch, serve, lang;
  missing keys surface as `tr:key` rather than empty output (L2)
- **`FABI_HOME`** — relocates the ledger+state dir (the disk that
  holds the binary needn't hold the data — survives a full
  system drive)
- **`GET /ui`** — embedded single-file dashboard on `fabi serve`
  (status/tools/models/doctor + control buttons; loopback only,
  zero external assets, confirm-before-POST)
- **`fabi lang`** — shows detected language, its source
  (env/OS), the supported list and how to override
- **`--json` everywhere** — every output-producing command emits
  pure JSON: scan, report, doctor, status, diff, why, update,
  goal, route, run, sweep, compare, unload, auto, pattern,
  install, uninstall, selftest, ledger stats/events/import;
  **18/18 falsified** in `fabi selftest` (selftest --json itself
  excluded — recursion)
- **API parity complete** — `GET /route?task=` (14 endpoints);
  every data command reachable via HTTP
- **6-arch** — windows/arm64 + linux/arm64; CI cross-compiles
  all 6 release targets every push; `fabi version` shows
  os/arch + detected language
- **patterns expanded** — +global AI tools: Msty, Cherry Studio,
  llamafile, Xinference, SillyTavern, Chatbox, Fooocus, Jan,
  FastChat, localAI (204 detection entries)
- **docs** — README.ja.md, USAGE.ja.md; `fabi help` exits 0
- **selftest** — 31 PASS: +i18n honest fallback, +language
  detection, +every --json flag valid (18 commands)

## v0.0.6 — 2026-11-07

- **`fabi update` + `GET /update`** — honest version check via OS
  curl (no auto-install, deny-by-default holds); the only outbound
  call fabi makes, user-invoked only
- **`--json` contract everywhere** — scan/status/report/doctor/diff/
  why/ledger all emit machine-readable JSON; selftest verifies all 7
- **packaging** — `packaging/gen.sh` regenerates winget/scoop/
  homebrew manifests per release; release workflow auto-builds
  4 targets + SHA256 + commits manifests back
- **winget fix** — LicenseUrl `main`→`master` (PR #472788 was closed:
  unsigned CLA + 404); resubmission guide in `packaging/winget/`
- **docs** — SECURITY.md, USAGE.md, CONTRIBUTING.md, issue/PR
  templates
- **selftest** — 29 PASS: +semverLess

- **`fabi ledger import` / `POST /import`** — multi-PC estate merging:
  dedupe on ts|kind|payload, `ledger.imported` provenance events with
  `first_id..last_id` ranges; derived views (doctor/report/why/diff)
  exclude imported ranges so foreign estates can't pollute local recs
- **route calibration** — `fabi route` learns the measured/predicted
  ratio from the ledger and prints a calibrated estimate under the
  physics ceiling
- **auto observability** — `auto.skipped` events record safe
  no-action decisions (ollama down / nothing loaded / all in use)
- **`ledger stats` scans-by-os** — merged estates show origin OS
- **UX** — bare `fabi` is context-aware (first-run intro vs status);
  doctor ends with next-step guidance; honest tagline everywhere
- **selftest coverage** — multi-engine discovery, honest-eye notes,
  auto safety, goal engine, marketplace merge, API shapes, route
  physics upper bound, explainability resolution, serve daemon,
  diff delta, watch heartbeat, append-order (pre-import scope),
  deny-by-default lifecycle, estate import — 23 PASS
- **CI** — `fabi selftest` runs on ubuntu/windows/macos
- **docs parity** — API.md, README, AGENTS.md synced; MIT LICENSE

## v0.0.5 — 2026-11-07

- **`fabi auto`** — Automation Engine seed: applies only safe
  recommendations (idle unload >15 min); user-triggered,
  non-destructive, keeps recently-used models warm
- **`OLLAMA_HOST`** — standard env var routes run/goal/serve to
  remote/cloud ollama-compatible endpoints; route's
  "cloud or downgrade" verdict now points to it
- **serve API complete** — 15 endpoints + /ui: added POST /unload (VRAM
  release) and POST /scan (trigger observation over API)
- **report control line** — counts VRAM releases in the statement
- **selftest race fix** — ledger-append check accepts concurrent
  writers (resident watch could make +1 flaky)

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
