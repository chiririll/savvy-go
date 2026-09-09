#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
OUT_DIR="${OUT_DIR:-$ROOT/dist}"
VERSION="${APP_VERSION:-$(git -C "$ROOT" rev-parse --short HEAD)}"
BIN="${OUT_DIR}/savvy"

if [[ ! -f "$BIN" ]]; then
    echo "missing $BIN — build with: go build -o dist/savvy ./cmd/savvy" >&2
    exit 1
fi

if [[ ! -d "$ROOT/public/build" ]]; then
    echo "public/build is missing. Run: npm ci && npm run build" >&2
    exit 1
fi

STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT

mkdir -p "$STAGE/public"
cp -a "$BIN" "$STAGE/savvy"
chmod +x "$STAGE/savvy"
cp -a "$ROOT/public/build" "$STAGE/public/build"
for f in favicon.svg robots.txt site.webmanifest; do
    if [[ -f "$ROOT/public/$f" ]]; then
        cp -a "$ROOT/public/$f" "$STAGE/public/$f"
    fi
done
printf '%s\n' "$VERSION" > "$STAGE/VERSION"

rm -rf "$OUT_DIR/public"
mkdir -p "$OUT_DIR"
cp -a "$STAGE/public" "$OUT_DIR/public"

TARBALL="$OUT_DIR/savvy.tar.gz"
tar -C "$STAGE" -czf "$TARBALL" .
echo "Wrote $TARBALL (VERSION=$VERSION)"
