# winget submission notes

PR [microsoft/winget-pkgs#472788](https://github.com/microsoft/winget-pkgs/pull/472788)
was **closed unmerged** (2026-11-06). Two blockers, both now understood:

## Blocker 1 — CLA not signed (user action)

The Microsoft CLA was never signed. Before resubmitting:

1. Go to <https://opensource.microsoft.com/cla/>
2. Sign in with the GitHub account that will open the PR
3. Agree to the CLA — it applies to all Microsoft OSS repos

## Blocker 2 — LicenseUrl 404 (fixed)

`LicenseUrl` pointed at `blob/main/LICENSE`; the default branch is
`master`. Fixed in `Fabi.Fabi.locale.en-US.yaml` (now
`blob/master/LICENSE`).

Verified good:

- `InstallerUrl` → `releases/download/v0.0.5/fabi-windows-amd64.exe` resolves
- `InstallerSha256` `b560cd04…93310` matches the published asset

## Resubmitting

```sh
# 1. sign the CLA (above)
# 2. copy the three manifests into a winget-pkgs fork under:
#    manifests/f/Fabi/Fabi/0.0.5/
#    (or use `wingetcreate` to generate a fresh manifest)
# 3. open the PR — validation should pass: URLs resolve, hash matches
# 4. reply `@microsoft-github-policy-service agree` if the CLA bot asks again
```

For a new version, bump `PackageVersion` in all three files and point
`InstallerUrl`/`InstallerSha256` at the new release asset — the release
workflow publishes `fabi-windows-amd64.exe` + `SHA256SUMS.txt`
automatically on tag push.
