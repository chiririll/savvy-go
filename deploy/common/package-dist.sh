#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
OUT_DIR="${OUT_DIR:-$ROOT/dist}"
VERSION="${APP_VERSION:-$(git -C "$ROOT" rev-parse --short HEAD)}"
BIN="${OUT_DIR}/savvy-go"

if [[ ! -f "$BIN" ]]; then
    echo "missing $BIN" >&2
    exit 1
fi

if [[ ! -d "$ROOT/public/build" ]]; then
    echo "public/build is missing" >&2
    exit 1
fi

STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT

mkdir -p "$STAGE/public"
cp -a "$BIN" "$STAGE/savvy-go"
chmod +x "$STAGE/savvy-go"
# Static files (index.html, icons, manifest, robots.txt) plus the Vite build.
find "$ROOT/public" -mindepth 1 -maxdepth 1 -type f -exec cp -a {} "$STAGE/public/" \;
cp -a "$ROOT/public/build" "$STAGE/public/build"
printf '%s\n' "$VERSION" > "$STAGE/VERSION"

rm -rf "$OUT_DIR/public"
mkdir -p "$OUT_DIR"
cp -a "$STAGE/public" "$OUT_DIR/public"

TARBALL="$OUT_DIR/savvy-go.tar.gz"
tar -C "$STAGE" -czf "$TARBALL" .
echo "Wrote $TARBALL (VERSION=$VERSION)"
