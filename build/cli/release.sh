#!/bin/sh
# Builds the command-line-only zenmd (cmd/zenmd, no Wails, no cgo) for Linux
# into dist/: zenmd_<version>_linux_<arch>.tar.gz for amd64 and arm64, and
# SHA256SUMS. Used by `task release:cli` and the release workflow's cli job.
# Usage: build/cli/release.sh <version>
set -eu
v="${1:?usage: release.sh <version>}"
root="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$root"
rm -rf dist
mkdir -p dist
for arch in amd64 arm64; do
  name="zenmd_${v}_linux_${arch}"
  mkdir -p "dist/$name"
  CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath \
    -ldflags "-s -w -X github.com/zenmark-org/zenmd/internal/cli.Version=$v" \
    -o "dist/$name/zenmd" ./cmd/zenmd
  cp LICENSE README.md "dist/$name/"
  tar -C dist -czf "dist/$name.tar.gz" "$name"
  rm -rf "dist/$name"
done
cd dist
if command -v sha256sum >/dev/null 2>&1; then sha256sum ./*.tar.gz | sed 's| \./| |' > SHA256SUMS
else shasum -a 256 ./*.tar.gz | sed 's| \./| |' > SHA256SUMS; fi
cat SHA256SUMS
