#!/bin/sh
set -e

REPO="nnstt1/hcpt"
BINARY="hcpt"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Check required commands
for cmd in curl tar; do
    if ! command -v "$cmd" > /dev/null 2>&1; then
        echo "Error: '$cmd' is required but not installed." >&2
        exit 1
    fi
done

# Detect OS
OS="$(uname -s)"
case "$OS" in
    Darwin) OS="darwin" ;;
    Linux)  OS="linux"  ;;
    *)
        echo "Error: Unsupported OS: $OS" >&2
        exit 1
        ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    arm64)   ARCH="arm64" ;;
    aarch64) ARCH="arm64" ;;
    *)
        echo "Error: Unsupported architecture: $ARCH" >&2
        exit 1
        ;;
esac

# Fetch the latest release version from GitHub API
echo "Fetching latest release..."
VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed 's/.*"tag_name": *"v\([^"]*\)".*/\1/')"

if [ -z "$VERSION" ]; then
    echo "Error: Failed to fetch the latest version." >&2
    exit 1
fi

echo "Latest version: v${VERSION}"

# Build download URL
ARCHIVE="hcpt_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${ARCHIVE}"

# Download archive to a temp directory
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading ${ARCHIVE}..."
if ! curl -fsSL "$URL" -o "${TMP_DIR}/${ARCHIVE}"; then
    echo "Error: Failed to download ${URL}" >&2
    exit 1
fi

# Extract archive
tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "$TMP_DIR"

# Install binary
BINARY_PATH="${TMP_DIR}/${BINARY}"
if [ ! -f "$BINARY_PATH" ]; then
    echo "Error: Binary '${BINARY}' not found in archive." >&2
    exit 1
fi

chmod +x "$BINARY_PATH"

if [ -w "$INSTALL_DIR" ]; then
    mv "$BINARY_PATH" "${INSTALL_DIR}/${BINARY}"
else
    echo "Installing to ${INSTALL_DIR} (sudo required)..."
    sudo mv "$BINARY_PATH" "${INSTALL_DIR}/${BINARY}"
fi

echo "Successfully installed ${BINARY} v${VERSION} to ${INSTALL_DIR}/${BINARY}"
