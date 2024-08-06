#!/bin/sh

uname_arch() {
  arch=$(uname -m)
  case $arch in
    x86_64) arch="amd64" ;;
    x86) arch="386" ;;
    i686) arch="386" ;;
    i386) arch="386" ;;
    aarch64) arch="arm64" ;;
    armv*) arch="armv6" ;;
    armv*) arch="armv6" ;;
    armv*) arch="armv6" ;;
  esac
  if [ "$(uname_os)" == "darwin" ]; then
    arch="all"
  fi
  echo ${arch}
}

uname_os() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  echo "$os"
}



GITHUB_OWNER="ddosify"
GITHUB_REPO="ddosify"
TAG="latest"
INSTALL_DIR="/usr/local/bin/"
OS=$(uname_os)
ARCH=$(uname_arch)
PLATFORM="${OS}/${ARCH}"
GITHUB_RELEASES_PAGE=https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases
VERSION=$(curl $GITHUB_RELEASES_PAGE/$TAG -sL -H 'Accept:application/json' | tr -s '\n' ' ' | sed 's/.*"tag_name":"//' | sed 's/".*//' | tr -d v)
NAME=${GITHUB_REPO}_${VERSION}_${OS}_${ARCH}

TARBALL=${NAME}.tar.gz
TARBALL_URL=${GITHUB_RELEASES_PAGE}/download/v${VERSION}/${TARBALL}

echo "Downloading latest $GITHUB_REPO binary from $TARBALL_URL"
tmpfolder=$(mktemp -d)
$(curl $TARBALL_URL -sL -o $tmpfolder/$TARBALL)

if [ ! -f $tmpfolder/$TARBALL ]; then
    echo "Can not download. Exiting..."
    exit 14
fi
cd ${tmpfolder} && tar --no-same-owner -xzf "$tmpfolder/$TARBALL"

if [ ! -f $tmpfolder/$GITHUB_REPO ]; then
    echo "Can not find $GITHUB_REPO. Exiting..."
    exit 15
fi

binary=$tmpfolder/$GITHUB_REPO
echo "Installing $GITHUB_REPO to $INSTALL_DIR (sudo access required to write to $INSTALL_DIR)"
sudo install "$binary" $INSTALL_DIR
echo "Installed $GITHUB_REPO to $INSTALL_DIR"
echo "Simple usage: ddosify -t https://testserver.ddosify.com"
rm -rf "${tmpdir}"
