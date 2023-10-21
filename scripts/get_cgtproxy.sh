#!/bin/sh

# Modify from https://goreleaser.com/static/run

set -e
set -x

# https://gist.github.com/lukechilds/a83e1d7127b78fef38c2914c4ececc3c
get_latest_release() {
	curl --silent "https://api.github.com/repos/$1/releases/latest" |
		grep '"tag_name":' |
		sed -E 's/.*"([^"]+)".*/\1/' |
		sed -E 's/v(.*)/\1/'
}

LATEST="$(get_latest_release black-desk/cgtproxy)"

test -z "$VERSION" && VERSION="$LATEST"

test -z "$VERSION" && {
	echo "Unable to get cgtproxy version." >&2
	exit 1
}

TMP_DIR="$(mktemp -d)"
# shellcheck disable=SC2064 # intentionally expands here
trap "rm -rf \"$TMP_DIR\"" EXIT INT TERM

ARCH="$(uname -m)"
test "$ARCH" = "aarch64" && ARCH="arm64"
test "$ARCH" = "x86_64" && ARCH="amd64"
TAR_FILE="cgtproxy_${VERSION}_linux_${ARCH}.tar.gz"

cd "$TMP_DIR"
echo "Downloading cgtproxy $VERSION..."
curl -fLO "https://github.com/black-desk/cgtproxy/releases/download/v$VERSION/$TAR_FILE"
tar -xf "$TMP_DIR/$TAR_FILE" -C "$TMP_DIR"

curl -fLO "https://raw.githubusercontent.com/black-desk/cgtproxy/v$VERSION/misc/systemd/cgtproxy.service"

SUDO=sudo

if command pkexec >/dev/null 2>&1; then
	SUDO=pkexec
fi

test -z "$PREFIX" && PREFIX="/usr/local"

$SUDO install -m755 -D "$TMP_DIR/cgtproxy" "$PREFIX/bin/cgtproxy"
$SUDO install -m644 -D "$TMP_DIR/cgtproxy.service" "$PREFIX/lib/systemd/system/cgtproxy.service"
