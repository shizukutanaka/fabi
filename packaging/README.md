# packaging — distribution manifests

Ready-to-submit manifests for package managers. Files live here
because they belong to this repo's release cycle; the *target* repos
(tap/bucket/winget fork) are external.

## Status

| Channel | File | State |
|---|---|---|
| winget | `winget/` | PR #472788 closed (CLA unsigned + LicenseUrl 404 — fixed; resubmit after CLA) |
| Scoop | `scoop/fabi.json` | ready — needs a `scoop-bucket` repo to hold it |
| Homebrew | `homebrew/fabi.rb` | ready — needs a `homebrew-fabi` repo to hold it |

## Scoop

Create a repo named `scoop-bucket`, commit `scoop/fabi.json` at
`bucket/fabi.json`. Users then run:

```powershell
scoop bucket add fabi https://github.com/shizukutanaka/scoop-bucket
scoop install fabi/fabi
```

`checkver`/`autoupdate` fields let `bin/checkver.ps1` +
`auto-pr.ps1` from the Scoop BucketTemplate auto-bump versions.

## Homebrew

Create a repo named `homebrew-fabi`, commit `homebrew/fabi.rb` at
`Formula/fabi.rb`. Users then run:

```sh
brew install shizukutanaka/fabi/fabi
```

## winget

See `winget/README.md` — sign the Microsoft CLA, then resubmit the
three manifests under `manifests/f/Fabi/Fabi/<version>/`.

## On each release

Bump `version`/`PackageVersion`, point URLs at the new tag, and copy
fresh SHA256s from the release's `SHA256SUMS.txt` (the release
workflow publishes it automatically).
