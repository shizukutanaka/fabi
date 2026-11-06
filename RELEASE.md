# Release Process — Fabi v0.0.1

Phase 0 remaining work is distribution, not code (SPEC v0.6 #95).

## 1. Build artifacts

```sh
bash build.sh   # → release/<ver>/  4 binaries + SHA256SUMS.txt
```

Targets: windows/amd64, linux/amd64, darwin/amd64, darwin/arm64.

## 2. GitHub release (owner action)

- Push repo to GitHub (`<owner>/fabi` — set real owner in manifests).
- `gh release create v0.0.1 release/*/fabi-* release/*/SHA256SUMS.txt --title "Fabi v0.0.1"`

## 3. Code signing (owner action)

- Windows: EV/OV Authenticode cert → `signtool sign fabi-windows-amd64.exe`
  (unsigned exe triggers SmartScreen — kills the 60 s / 0-config funnel)
- macOS: `codesign` + `notarytool` — required for Gatekeeper

## 4. winget

- Fill `InstallerUrl` + `InstallerSha256` in `packaging/winget/Fabi.Fabi.installer.yaml`
  from the released binary's SHA256SUMS.txt.
- `winget validate --manifest packaging/winget/`
- `winget install --manifest packaging/winget/` (local test)
- PR to microsoft/winget-pkgs: `manifests/f/Fabi/Fabi/0.0.1/`

## 5. brew / apt (later)

- brew: `brew create` formula pointing at darwin binaries — needs repo first.
- apt: `.deb` packaging is a Phase-2 task (systemd unit for watch mode).

## Honesty gates (don't skip)

- Every number in release notes comes from `fabi selftest` output or ledger — not hope.
- `other OSes: UNVERIFIED` stays on the page until a real macOS/Linux scan ships proof.
