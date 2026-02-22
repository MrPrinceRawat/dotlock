#!/bin/sh
set -e

REPO="mrprincerawat/dotlock"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux)  OS="linux" ;;
    darwin) OS="darwin" ;;
    *)      echo "Unsupported OS: $OS" && exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)             echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

# Get latest version
VERSION=$(curl -sI "https://github.com/${REPO}/releases/latest" | grep -i "location:" | sed 's/.*tag\///' | tr -d '\r\n')
if [ -z "$VERSION" ]; then
    echo "Failed to get latest version"
    exit 1
fi

FILENAME="dotlock_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"

echo "Installing dotlock ${VERSION} (${OS}/${ARCH})..."

# Download and extract
TMPDIR=$(mktemp -d)
curl -sL "$URL" -o "${TMPDIR}/${FILENAME}"
tar -xzf "${TMPDIR}/${FILENAME}" -C "$TMPDIR"

# Install
if [ -w "$INSTALL_DIR" ]; then
    mv "${TMPDIR}/dotlock" "${INSTALL_DIR}/dotlock"
else
    sudo mv "${TMPDIR}/dotlock" "${INSTALL_DIR}/dotlock"
fi

chmod +x "${INSTALL_DIR}/dotlock"
rm -rf "$TMPDIR"

echo "dotlock installed to ${INSTALL_DIR}/dotlock"
echo "Run 'dotlock --help' to get started"
