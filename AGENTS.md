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
main.go      commands + usage            report.go    statement + --json
scan.go      orchestrator                doctor.go    evidence-based recs
probes.go    6 probes                    ledger.go    SQLite events
watch.go     periodic scan               goal/route.go physics routing
install.go   task scheduler (dry-run)    bench.go     real inference measure
selftest.go  falsification engine        ledgercmd.go stats/events/export
status.go    one-line health             patterns.json detection data
packages_*.go OS-specific package lists   wsl_*.go     WSL distro detection
SPEC_v0.*    the spec — v0.7 is current
```

## Numbers that bind (do not regress)

binary ≤10 MB · scan <3 s · RSS ≤150 MB · 0 config · offline core
