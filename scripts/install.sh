uname_arch() {
  arch=$(uname -m)
  case $arch in
    x86_64) arch="amd64" ;;
    x86) arch="386" ;;
    i686) arch="386" ;;
    i386) arch="386" ;;
    aarch64) arch="arm64" ;;
    armv5*) arch="armv5" ;;
    armv6*) arch="armv6" ;;
    armv7*) arch="armv7" ;;
  esac
  echo ${arch}
}

uname_os() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$os" in
    cygwin_nt*) os="windows" ;;
    mingw*) os="windows" ;;
    msys_nt*) os="windows" ;;
  esac
  echo "$os"
}


GITHUB_OWNER="ddosify"
GITHUB_REPO="ddosify"
TAG="latest"
INSTALL_DIR="/usr/bin/"
ARCH=$(uname_arch)
OS=$(uname_os)
PLATFORM="${OS}/${ARCH}"
NAME=${GITHUB_REPO}_${OS}_${ARCH}
TARBALL=${NAME}.tar.gz
GITHUB_RELEASES_PAGE=https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases

VERSION=$(curl $GITHUB_RELEASES_PAGE/$TAG -sL -H 'Accept:application/json' | tr -s '\n' ' ' | sed 's/.*"tag_name":"//' | sed 's/".*//')
TARBALL_URL=${GITHUB_RELEASES_PAGE}/download/${VERSION}/${TARBALL}

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

if [ "$OS" = "windows" ]; then
  binary="${binary}.exe"
fi
install "$binary" $INSTALL_DIR
echo "Installed $GITHUB_REPO to $INSTALL_DIR"

rm -rf "${tmpdir}"
