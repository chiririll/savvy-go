#!/bin/sh
# Adds the Go Savvy package repository for this distribution and installs savvy-go.
# Served from the repository site; build.sh fills in @SITE_URL@.
#
# Usage: curl -fsSL @SITE_URL@/install.sh | sudo sh
#        wget -qO- @SITE_URL@/install.sh | sudo sh
set -eu

SITE=@SITE_URL@

die() { echo "savvy-go: $*" >&2; exit 1; }

# fetch URL FILE, with whichever of curl and wget is installed.
fetch() {
  if command -v curl >/dev/null 2>&1; then curl -fsSL -o "$2" "$1"
  elif command -v wget >/dev/null 2>&1; then wget -qO "$2" "$1"
  else die "needs curl or wget"
  fi
}

[ "$(id -u)" -eq 0 ] || die "run as root: curl -fsSL $SITE/install.sh | sudo sh"
[ "$(uname -m)" = x86_64 ] || die "packages are built for x86_64 only, this machine is $(uname -m)"
[ -r /etc/os-release ] || die "cannot detect the distribution: no /etc/os-release"
# shellcheck source=/dev/null
. /etc/os-release

family=
for id in $ID ${ID_LIKE:-}; do
  case $id in
    debian | ubuntu) family=apt ;;
    fedora | rhel | centos) family=dnf ;;
    suse | opensuse*) family=zypper ;;
  esac
  [ -z "$family" ] || break
done

case $family in
  apt)
    install -d -m 0755 /etc/apt/keyrings
    fetch "$SITE/savvy-go.asc" /etc/apt/keyrings/savvy-go.asc
    chmod 0644 /etc/apt/keyrings/savvy-go.asc
    echo "deb [signed-by=/etc/apt/keyrings/savvy-go.asc] $SITE/apt stable main" > /etc/apt/sources.list.d/savvy-go.list
    apt-get update
    DEBIAN_FRONTEND=noninteractive apt-get install -y savvy-go
    ;;
  dnf)
    fetch "$SITE/savvy-go.repo" /etc/yum.repos.d/savvy-go.repo
    if command -v dnf >/dev/null 2>&1; then dnf install -y savvy-go; else yum install -y savvy-go; fi
    ;;
  zypper)
    zypper --non-interactive removerepo savvy-go >/dev/null 2>&1 || true
    zypper --non-interactive addrepo --refresh "$SITE/savvy-go.repo"
    zypper --non-interactive --gpg-auto-import-keys install savvy-go
    ;;
  *)
    die "${PRETTY_NAME:-this distribution} is not supported; see $SITE for the packages"
    ;;
esac

echo
echo "Go Savvy is installed and runs as the savvy-go service on http://127.0.0.1:8080"
