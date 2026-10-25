#!/usr/bin/env bash
# Builds the signed apt and dnf repositories into a directory, ready for GitHub Pages.
# The packages come from the last KEEP stable GitHub releases, so every run rebuilds
# the whole tree. Needs: gh, gpg, apt-ftparchive, createrepo_c.
# Signs with the default secret key of the gpg keyring, which must have no passphrase.
#
# Usage: deploy/repo/build.sh [out-dir]
# Env:   GITHUB_REPOSITORY  owner/name, as in Actions
#        SITE_URL           public URL of the site, no trailing slash
#        KEEP               stable releases to keep (default 5)
set -euo pipefail

OUT=${1:-site}
KEEP=${KEEP:-5}
: "${GITHUB_REPOSITORY:?}" "${SITE_URL:?}"
here=$(cd "$(dirname "$0")" && pwd)

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

sign() { gpg --batch --yes "$@"; }

rm -rf "$OUT"
pool="$OUT/apt/pool/main"
dist="$OUT/apt/dists/stable"
mkdir -p "$pool" "$dist/main/binary-amd64" "$OUT/rpm"

# Stable tags are plain vX.Y.Z; anything with a suffix (v1.2.0-rc1) is a pre-release.
tags=$(gh release list --repo "$GITHUB_REPOSITORY" --exclude-drafts --limit 50 --json tagName --jq '.[].tagName' \
  | grep -E '^v[0-9]+(\.[0-9]+)*$' | head -n "$KEEP" || true)
[ -n "$tags" ] || { echo "no stable releases found" >&2; exit 1; }

for tag in $tags; do
  v=${tag#v}
  if ! gh release download "$tag" --repo "$GITHUB_REPOSITORY" --pattern 'savvy-go.deb' --pattern 'savvy-go.rpm' --dir "$work/$tag"; then
    echo "::warning::$tag has no packages, skipped"
    continue
  fi
  # Release assets share one name; the repos need a distinct file per version.
  mv "$work/$tag/savvy-go.deb" "$pool/savvy-go_${v}_amd64.deb"
  mv "$work/$tag/savvy-go.rpm" "$OUT/rpm/savvy-go-${v}-1.x86_64.rpm"
done

# apt: pool + dists/stable, Release signed as InRelease and Release.gpg.
(
  cd "$OUT/apt"
  apt-ftparchive packages pool > dists/stable/main/binary-amd64/Packages
  gzip -9kf dists/stable/main/binary-amd64/Packages
  apt-ftparchive -c "$here/apt-release.conf" release dists/stable > dists/stable/Release
)
sign --armor --detach-sign --output "$dist/Release.gpg" "$dist/Release"
sign --clearsign --output "$dist/InRelease" "$dist/Release"

# dnf: the metadata is signed; the packages themselves are not.
createrepo_c --quiet "$OUT/rpm"
sign --armor --detach-sign --output "$OUT/rpm/repodata/repomd.xml.asc" "$OUT/rpm/repodata/repomd.xml"

gpg --armor --export > "$OUT/savvy-go.asc"

sed "s#@SITE_URL@#$SITE_URL#g" "$here/savvy-go.repo" > "$OUT/savvy-go.repo"
sed -e "s#@REPO@#$GITHUB_REPOSITORY#g" -e "s#@SITE_URL@#$SITE_URL#g" "$here/index.html" > "$OUT/index.html"

echo "Built $OUT with: $(echo $tags)"
