# Fabi — Project Conventions

Universal AI Control Plane. Go, single binary, zero config, offline core.

## Build / Test

```sh
go build -ldflags="-s -w" -o fabi.exe .   # release binary
go test ./...                              # unit tests
bash build.sh                              # 4-target release + SHA256
```

## Laws (enforced in code — read SPEC_v0.7 first)

- **Ledger is the only truth**: `~/.fabi/ledger.db`, append-only events.
  All state is a view. No state may live outside the ledger.
- **Patterns are data**: detection rules live in `patterns.json`
  (embedded) + `~/.fabi/patterns.json` (user merge). Never hardcode new
  tool names in probes — add patterns.
- **Missing provenance → inference note, never silence** (L2).
- **Deletion is always manual** (L9). No `--apply` ever.
- **Every claim gets a test** (L11). `fabi selftest` is the contract;
  CI has a ≤10 MB binary-size gate.
- **Permission default: deny** — install/uninstall are dry-run without `--yes`.

## Layout

```
main.go      18 commands + usage         report.go    statement + --json + control
scan.go      orchestrator                doctor.go    8 evidence-based rules
probes.go    6 probes                    ledger.go    SQLite events + import
watch.go     periodic + inference beat   goal/route.go physics + calibrated routing
run.go       exec/sweep/compare/unload/auto  httpio.go raw TCP (no net/http)
serve.go     Universal API (14 endpoints)  install.go Startup-folder launcher
selftest.go  falsification engine        ledgercmd.go stats/events/export/import
status.go    one-line health (+loaded)   patterns.json detection data
diff.go      estate delta (N scans)      why.go      evidence chain
packages_*.go OS package lists           wsl_*.go     WSL distro detection
sysinfo.go   shared helpers              patterns.go  Marketplace seed
lang.go      i18n — strings are data (10 langs, FABI_LANG/OS detect)
SPEC_v0.*    the spec — v0.7, Parts 1–17 API.md       serve endpoint docs
```

## Control surface (Phase 2/3 primitives)

```
fabi run/sweep/compare  measure inference    fabi unload   free VRAM
fabi goal --run         task chains          fabi auto     safe recs only
fabi serve              14-endpoint API      OLLAMA_HOST   remote engines
fabi ledger import      multi-PC merge       fabi diff     estate delta
fabi update             honest update check
fabi update             honest update check
```

## Numbers that bind (do not regress)

binary ≤10 MB · scan <3 s · RSS ≤150 MB · 0 config · offline core
