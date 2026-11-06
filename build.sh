#!/usr/bin/env bash
# build.sh — cross-compile fabi for all 3 OSes + sha256 checksums.
set -euo pipefail
cd "$(dirname "$0")"

OUT="release/$(go version | awk '{print $3}' | tr -d go)-$(date +%Y%m%d)"
mkdir -p "$OUT"

targets=(windows/amd64 linux/amd64 darwin/amd64 darwin/arm64)
for t in "${targets[@]}"; do
  os="${t%/*}"; arch="${t#*/}"
  bin="fabi-${os}-${arch}"
  [ "$os" = windows ] && bin="${bin}.exe"
  echo "→ $os/$arch"
  GOOS="$os" GOARCH="$arch" go build -ldflags="-s -w" -o "$OUT/$bin" .
done

cd "$OUT"
sha256sum fabi-* > SHA256SUMS.txt
echo
echo "== release =="
ls -la fabi-* SHA256SUMS.txt
cat SHA256SUMS.txt
