#!/usr/bin/env bash
# Usage: package-linux.sh <deb|rpm>
set -euo pipefail

FORMAT="${1:-}"
case "$FORMAT" in
    deb) export APP_USER=www-data; GLOB='savvy-go_*.deb' ;;
    rpm) export APP_USER=savvy-go; GLOB='savvy-go-*.rpm' ;;
    *) echo "usage: $0 <deb|rpm>" >&2; exit 1 ;;
esac

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

# nfpm does not expand env vars in content sources or file owners, so the unit
# and the config are rendered with the account name substituted.
mkdir -p "$OUT_DIR/stage"
trap 'rm -rf "$OUT_DIR/stage"' EXIT
sed "s/@APP_USER@/$APP_USER/g" "$ROOT/deploy/nfpm/savvy-go.service" > "$OUT_DIR/stage/savvy-go.service"
sed "s/@APP_USER@/$APP_USER/g" "$ROOT/deploy/nfpm/nfpm.yaml" > "$OUT_DIR/stage/nfpm.yaml"

export APP_VERSION="$VERSION"
(
    cd "$ROOT/deploy/nfpm"
    nfpm package --config "$OUT_DIR/stage/nfpm.yaml" --packager "$FORMAT" --target "$OUT_DIR"
)

pkg="$(ls -1t "$OUT_DIR"/$GLOB | head -n1)"
cp -f "$pkg" "$OUT_DIR/savvy-go.$FORMAT"

echo "Wrote $pkg and $OUT_DIR/savvy-go.$FORMAT (VERSION=$VERSION)"
