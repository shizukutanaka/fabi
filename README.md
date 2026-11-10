# Fabi

[日本語](README.ja.md)

**The honest eye on your PC's AI estate** — sees what exists, what's
idle, what holds VRAM, what changed, and why it says what it says —
all on one append-only ledger.

Not an LLM. Not a launcher. The layer that makes every AI tool visible —
including the ones hiding.

## Download (prebuilt binaries)

No Go toolchain needed — grab a single binary from
[GitHub Releases](https://github.com/shizukutanaka/fabi/releases)
(latest: [v0.0.7](https://github.com/shizukutanaka/fabi/releases/tag/v0.0.7)):

| OS / arch | Asset |
|---|---|
| Windows x64 | `fabi-windows-amd64.exe` |
| Windows ARM64 | `fabi-windows-arm64.exe` |
| Linux x64 | `fabi-linux-amd64` |
| Linux ARM64 | `fabi-linux-arm64` |
| macOS Intel | `fabi-darwin-amd64` |
| macOS Apple Silicon | `fabi-darwin-arm64` |

1. Download the asset for your OS/arch plus `SHA256SUMS.txt` from the
   same release.
2. Verify the checksum (below) before running it.
3. Run it: on Windows, rename to `fabi.exe` if you like and run
   `fabi.exe scan`. On Linux/macOS, `chmod +x fabi-<os>-<arch>` and
   rename it to `fabi` (e.g. `mv fabi-linux-amd64 fabi`), then
   `./fabi scan`.

Verify with SHA-256 — the hash printed must match the line for your
asset in `SHA256SUMS.txt`:

```sh
# Linux
sha256sum fabi-linux-amd64
# macOS
shasum -a 256 fabi-darwin-arm64
```

```powershell
# Windows (PowerShell)
Get-FileHash .\fabi-windows-amd64.exe -Algorithm SHA256
```

Already have fabi? `fabi update` checks GitHub Releases for a newer
version (check only — no auto-install). To build from source instead,
see [Build](#build) below.

## Commands

```text
fabi scan       detect AI tools, models, hardware → ledger → estate report
fabi watch      periodic scan, timeseries (default 5 min) + inference heartbeat
fabi doctor     evidence-based recommendations (remove / idle / investigate)
fabi report     statement — estate trend + measured inference stats
fabi selftest   falsification engine — every spec claim gets a test
fabi ledger     query the event log (stats | events | export | import)
fabi diff       estate delta between scans (+appeared / -gone; N for N back)
fabi why        explain a recommendation — show its evidence chain
fabi route      physics + measured model verdict for a task
fabi goal       purpose → task chain → per-task model scheduling (--run)
fabi install    register fabi watch in Startup (fully reversible)
fabi status     one-line health: estate + GPU + tok/s + loaded models
fabi run        execute on a live model (--model, --ctx, --sweep, --compare)
fabi serve      JSON API :11435 (GET /status|/models|/ledger|/doctor|/diff|/why|/pattern · POST /run|/goal|/unload|/scan|/pattern|/import) + /ui dashboard
fabi tui        live terminal dashboard (status/tools/models/recommendations)
fabi unload     free VRAM — unload loaded models (real control action)
fabi auto       apply safe recs only — idle unload, user-triggered (Automation seed)
fabi pattern    extend detection via data — add/list user patterns (Marketplace seed)
fabi update     honest update check — compares to github releases (no auto-install)
fabi lang       detected language + supported list + how to override

Localized output: 10 languages (en/ja/zh/es/de/fr/ko/pt/ru/it) via
`FABI_LANG`/`LANG`/`LC_ALL` env or Windows UI language — all user-facing
surfaces; add a language by adding a `lang_<code>.go` data file.
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
