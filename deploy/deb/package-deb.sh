#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
OUT_DIR="${OUT_DIR:-$ROOT/dist}"
VERSION="${APP_VERSION:-}"

if [[ ! -f "$OUT_DIR/savvy-go" ]]; then
    echo "missing $OUT_DIR/savvy-go — build the Go binary first" >&2
    exit 1
fi
if [[ ! -d "$OUT_DIR/public" ]]; then
    echo "missing $OUT_DIR/public — run deploy/common/package-dist.sh first" >&2
    exit 1
fi

if ! command -v nfpm >/dev/null 2>&1; then
    echo "nfpm is not installed. See https://nfpm.goreleaser.com" >&2
    exit 1
fi

if [[ -z "$VERSION" ]]; then
    echo "APP_VERSION is required" >&2
    exit 1
fi

mkdir -p "$OUT_DIR"
OUT_DIR="$(cd "$OUT_DIR" && pwd)"
if command -v cygpath >/dev/null 2>&1; then
    OUT_DIR="$(cygpath -m "$OUT_DIR")"
fi

export APP_VERSION="$VERSION"
(
    cd "$ROOT/deploy/deb"
    nfpm package --config nfpm.yaml --packager deb --target "$OUT_DIR"
)

deb="$(ls -1t "$OUT_DIR"/savvy-go_*.deb | head -n1)"
cp -f "$deb" "$OUT_DIR/savvy-go.deb"

echo "Wrote $deb and $OUT_DIR/savvy-go.deb (VERSION=$VERSION)"
