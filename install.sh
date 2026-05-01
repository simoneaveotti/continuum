#!/usr/bin/env bash
set -euo pipefail

REPO="simoneaveotti/continuum"
BINARY="ctx"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | sed 's/.*"tag_name": *"\(.*\)".*/\1/')"

if [[ -z "$VERSION" ]]; then
  echo "Unable to determine latest release version" >&2
  exit 1
fi

ARCHIVE_EXT="tar.gz"
if [[ "$OS" == "mingw"* || "$OS" == "msys"* || "$OS" == "cygwin"* ]]; then
  OS="windows"
  ARCHIVE_EXT="zip"
fi

URL="https://github.com/${REPO}/releases/download/${VERSION}/ctx_${VERSION}_${OS}_${ARCH}.${ARCHIVE_EXT}"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

curl -fsSL "$URL" -o "$TMP/archive.$ARCHIVE_EXT"

if [[ "$ARCHIVE_EXT" == "zip" ]]; then
  unzip -q "$TMP/archive.$ARCHIVE_EXT" -d "$TMP"
else
  tar -xzf "$TMP/archive.$ARCHIVE_EXT" -C "$TMP"
fi

install -m 755 "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"

if command -v xattr >/dev/null 2>&1; then
  xattr -c "$INSTALL_DIR/$BINARY" 2>/dev/null || true
fi

echo "ctx ${VERSION} installed to ${INSTALL_DIR}/${BINARY}"
