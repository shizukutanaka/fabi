# Release Process — Fabi v0.0.2

Current state: **shipped to review queue** — all steps done except
Microsoft's async validation on the winget PR.

## Done `[M]`

- `bash build.sh` → 4 binaries + SHA256SUMS (7.8–8.0 MB each)
- GitHub release `v0.0.2` — all 4 artifacts + checksums live
- winget manifests: validated clean (`winget validate`, zero warnings)
- PR `microsoft/winget-pkgs#472788` — "New package: Fabi.Fabi version 0.0.2"
  · CLA agreed (individual)
  · updated to 0.0.2 so reviewers see the sha256-fix, not the buggy 0.0.1
  · open, no blocking labels, awaiting WinGetSvc backend validation
- Artifact verified end-to-end: download → SHA256 == manifest → runs

## Install mechanism (changed)

`fabi install` uses the **Startup folder** (`fabi-watch.cmd`), not
`schtasks` — measured reason: Task Scheduler required admin on the
reference machine; a Startup `.cmd` needs none and is trivially
reversible via `fabi uninstall --yes`. Verified end-to-end.

## Code signing — now a measured requirement

`fabi.exe` (unsigned, built locally) was **blocked mid-session** on the
reference machine — unreadable/unlaunchable, quarantine-class behavior.
A port-probing + process-enumerating binary is exactly what SmartScreen
flags. Signing is not optional for public distribution:

- Windows: EV/OV Authenticode → `signtool sign`
- macOS: `codesign` + `notarytool` (Gatekeeper)

## Remaining user actions

1. Watch PR `#472788` for `Validation-Completed` / request labels —
   responds asynchronously (hours–days).
2. Procure a signing cert before the first real public release.
3. `winget install Fabi` once merged.

## Honesty gates (still)

- Numbers in release notes come from `fabi selftest` / ledger only.
- `UNVERIFIED` marks stay until a real macOS/Linux scan ships proof.
- Version claims match the binary's `fabi version` output.
