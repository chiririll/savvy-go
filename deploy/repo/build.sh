#!/usr/bin/env bash
# Builds the signed apt, dnf and apk repositories into a directory, ready for GitHub Pages.
# The packages come from the last KEEP stable GitHub releases, so every run rebuilds
# the whole tree. Needs: gh, gpg, apt-ftparchive, createrepo_c, and docker + openssl for apk.
# Signs apt and dnf with the default secret key of the gpg keyring, which must have no passphrase.
# apk has its own RSA key (apk cannot use gpg): APK_KEY is the path of the PEM private key.
# Without it the apk repository is skipped.
#
# Usage: deploy/repo/build.sh [out-dir]
# Env:   GITHUB_REPOSITORY  owner/name, as in Actions
#        SITE_URL           public URL of the site, no trailing slash
#        KEEP               stable releases to keep (default 5)
#        APK_KEY            path of the RSA private key (PEM) that signs the apk index; optional
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
apk_dir="$OUT/apk/x86_64"
mkdir -p "$pool" "$dist/main/binary-amd64" "$OUT/rpm" "$apk_dir"

# Stable tags are plain vX.Y.Z; anything with a suffix (v1.2.0-rc1) is a pre-release.
tags=$(gh release list --repo "$GITHUB_REPOSITORY" --exclude-drafts --limit 50 --json tagName --jq '.[].tagName' \
  | grep -E '^v[0-9]+(\.[0-9]+)*$' | head -n "$KEEP" || true)
[ -n "$tags" ] || { echo "no stable releases found" >&2; exit 1; }

# gh lists newest first, so the first tag that has packages is the latest version.
latest=
for tag in $tags; do
  v=${tag#v}
  if ! gh release download "$tag" --repo "$GITHUB_REPOSITORY" --pattern 'savvy-go.deb' --pattern 'savvy-go.rpm' --dir "$work/$tag"; then
    echo "::warning::$tag has no packages, skipped"
    continue
  fi
  latest=${latest:-$tag}
  # Releases from before the apk existed have none, so it is optional.
  if gh release download "$tag" --repo "$GITHUB_REPOSITORY" --pattern 'savvy-go.apk' --dir "$work/$tag" 2>/dev/null; then
    mv "$work/$tag/savvy-go.apk" "$apk_dir/savvy-go-${v}-r1.apk"
  fi
  # Release assets share one name; the repos need a distinct file per version.
  mv "$work/$tag/savvy-go.deb" "$pool/savvy-go_${v}_amd64.deb"
  mv "$work/$tag/savvy-go.rpm" "$OUT/rpm/savvy-go-${v}-1.x86_64.rpm"
done
[ -n "$latest" ] || { echo "no release has packages" >&2; exit 1; }

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

# apk: APKINDEX.tar.gz signed with the RSA key; the index holds a checksum of every package, so
# the packages themselves are not signed. apk tools exist only on Alpine, hence the container.
if ! ls "$apk_dir"/*.apk >/dev/null 2>&1; then
  echo "::warning::no release has an apk, apk repository skipped"
  rm -rf "$OUT/apk"
elif [ -z "${APK_KEY:-}" ]; then
  echo "::warning::APK_KEY is not set, apk repository skipped"
  rm -rf "$OUT/apk"
else
  # The signature names the public key after the private one, so the name is fixed here.
  cp "$APK_KEY" "$work/savvy-go.rsa"
  docker run --rm -e "OWNER=$(id -u):$(id -g)" \
    -v "$(realpath "$apk_dir"):/repo" -v "$work/savvy-go.rsa:/keys/savvy-go.rsa:ro" -w /repo alpine:3.22 sh -ec '
      apk add -q --no-cache abuild apk-tools
      apk index -q --allow-untrusted -o APKINDEX.tar.gz ./*.apk
      abuild-sign -q -k /keys/savvy-go.rsa APKINDEX.tar.gz
      chown -R "$OWNER" /repo'
  openssl rsa -in "$work/savvy-go.rsa" -pubout -out "$OUT/savvy-go.rsa.pub" 2>/dev/null
fi

gpg --armor --export > "$OUT/savvy-go.asc"

cp "$here/../../public/favicon.svg" "$here/../../public/favicon.ico" "$OUT/"

sed "s#@SITE_URL@#$SITE_URL#g" "$here/savvy-go.repo" > "$OUT/savvy-go.repo"
sed "s#@SITE_URL@#$SITE_URL#g" "$here/install.sh" > "$OUT/install.sh"
# The landing page: static assets as they are, index.html with the placeholders filled in.
cp "$here/site/style.css" "$here/site/app.js" "$here/site/icons.svg" "$OUT/"
sed -e "s#@REPO@#$GITHUB_REPOSITORY#g" -e "s#@SITE_URL@#$SITE_URL#g" -e "s#@VERSION@#$latest#g" \
  "$here/site/index.html" > "$OUT/index.html"

echo "Built $OUT with: $(echo $tags), latest $latest"
