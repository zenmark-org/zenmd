#!/bin/sh
# Installs zenmd for the docs CI action (specs/packaging/docs-ci).
#
# A release version (ZM_VERSION, else the action's ref v<x.y.z>): downloads
# zenmd_<v>_linux_<arch>.tar.gz and SHA256SUMS from that release on github.com
# only, checks the archive, and puts zenmd in $RUNNER_TEMP/zenmd/bin and on
# PATH for later steps. Any other ref (uses: ./ or @main): builds ./cmd/zenmd
# from the action's own source with go (setup-go first).
#
# Environment: ZM_VERSION, ACTION_REF, RUNNER_TEMP, GITHUB_PATH, GITHUB_ACTION_PATH.
# Tests only: ZENMD_TEST_RELEASE_BASE (a file:// folder standing in for the
# release URL), ZENMD_TEST_OS, ZENMD_TEST_ARCH (uname -s, -m). action.yml never
# passes these.
set -eu

die() { echo "$*" >&2; exit 1; }

os="${ZENMD_TEST_OS:-}"
[ -n "$os" ] || os="$(uname -s)"
[ "$os" = Linux ] || die "zenmd's action runs on Linux runners"
machine="${ZENMD_TEST_ARCH:-}"
[ -n "$machine" ] || machine="$(uname -m)"
case "$machine" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) die "zenmd's action runs on amd64 and arm64 runners, not $machine" ;;
esac

v="${ZM_VERSION:-}"
ref="${ACTION_REF:-}"
if [ -z "$v" ]; then
  case "$ref" in
    v[0-9]*) v="$ref" ;;
  esac
fi

bindir="${RUNNER_TEMP:?RUNNER_TEMP is not set}/zenmd/bin"

if [ -z "$v" ]; then
  # A branch, a SHA or ./: build from the checked-out source.
  command -v go >/dev/null 2>&1 || die "building zenmd from source needs Go: add actions/setup-go before this action"
  src="$(cd "${GITHUB_ACTION_PATH:?GITHUB_ACTION_PATH is not set}" && pwd)"
  mkdir -p "$bindir"
  (cd "$src" && CGO_ENABLED=0 go build -trimpath -o "$bindir/zenmd" ./cmd/zenmd) || die "could not build zenmd from source"
  echo "$bindir" >> "${GITHUB_PATH:?GITHUB_PATH is not set}"
  exit 0
fi

v="${v#v}"
case "$v" in
  *[!0-9A-Za-z.-]*) die "zenmd's version must be an exact version like 0.8.0, not latest or a range" ;;
esac
printf '%s' "$v" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$' ||
  die "zenmd's version must be an exact version like 0.8.0, not latest or a range"

name="zenmd_${v}_linux_${arch}"
base="https://github.com/zenmark-org/zenmd/releases/download"
proto="=https"
if [ -n "${ZENMD_TEST_RELEASE_BASE:-}" ]; then
  base="$ZENMD_TEST_RELEASE_BASE"
  proto="=file"
fi

work="$RUNNER_TEMP/zenmd/download"
rm -rf "$work"
mkdir -p "$work" "$bindir"
fetch() { curl --fail --silent --location --proto "$proto" --proto-redir "$proto" -o "$work/$1" "$base/v$v/$1"; }
fetch "$name.tar.gz" && fetch SHA256SUMS ||
  die "could not download zenmd $v for Linux (Linux builds start at 0.8.0)"

want="$(awk -v n="$name.tar.gz" '$2 == n { print $1; exit }' "$work/SHA256SUMS")"
[ -n "$want" ] || die "$name.tar.gz is not in the release's SHA256SUMS"
if command -v sha256sum >/dev/null 2>&1; then
  got="$(sha256sum "$work/$name.tar.gz" | awk '{ print $1 }')"
else
  got="$(shasum -a 256 "$work/$name.tar.gz" | awk '{ print $1 }')"
fi
[ "$got" = "$want" ] || die "checksum mismatch for $name.tar.gz: not installing"

mkdir -p "$work/x"
tar -xzf "$work/$name.tar.gz" -C "$work/x"
[ -f "$work/x/$name/zenmd" ] || die "$name.tar.gz has no $name/zenmd"
cp "$work/x/$name/zenmd" "$bindir/zenmd"
chmod 755 "$bindir/zenmd"
rm -rf "$work"
echo "$bindir" >> "${GITHUB_PATH:?GITHUB_PATH is not set}"
