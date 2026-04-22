#!/bin/sh
set -eu

# install.sh — Install ralph CLI binary from GitHub Releases.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/yoshpy-dev/ralph/main/scripts/install.sh | sh
#   curl -fsSL ... | sh -s -- --version 0.1.0

REPO="yoshpy-dev/ralph"
BINARY="ralph"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Parse args.
VERSION=""
while [ $# -gt 0 ]; do
  case "$1" in
    --version) shift; VERSION="$1" ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
  shift
done

# Detect OS and architecture.
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  darwin|linux) ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Determine version.
if [ -z "$VERSION" ]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"v\(.*\)".*/\1/')"
  if [ -z "$VERSION" ]; then
    echo "Error: could not determine latest version."
    exit 1
  fi
fi

# Sanitize version string — allow only digits and dots.
case "$VERSION" in
  *[!0-9.]*) echo "Error: unexpected version format: $VERSION"; exit 1 ;;
  "") echo "Error: empty version string."; exit 1 ;;
esac

# Download and verify.
FILENAME="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${FILENAME}"
CHECKSUM_URL="https://github.com/${REPO}/releases/download/v${VERSION}/checksums.txt"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading ralph v${VERSION} for ${OS}/${ARCH}..."
curl -fsSL -o "${TMPDIR}/${FILENAME}" "$URL"
curl -fsSL -o "${TMPDIR}/checksums.txt" "$CHECKSUM_URL"

# Verify checksum.
cd "$TMPDIR"
if command -v sha256sum >/dev/null 2>&1; then
  grep "$FILENAME" checksums.txt | sha256sum -c --quiet
elif command -v shasum >/dev/null 2>&1; then
  grep "$FILENAME" checksums.txt | shasum -a 256 -c --quiet
else
  echo "Error: no checksum verification tool found (sha256sum or shasum required)."
  echo "Install one of these tools or verify the binary manually."
  exit 1
fi

# Extract and install.
tar xzf "$FILENAME"
if [ -w "$INSTALL_DIR" ]; then
  mv "$BINARY" "$INSTALL_DIR/"
else
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv "$BINARY" "$INSTALL_DIR/"
fi

echo "Installed ralph v${VERSION} to ${INSTALL_DIR}/${BINARY}"
echo "Run 'ralph version' to verify."
