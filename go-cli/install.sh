#!/bin/sh
set -e

# gocode installer
# Usage: curl -fsSL https://raw.githubusercontent.com/channyeintun/go-code/main/go-cli/install.sh | sh

REPO="channyeintun/go-code"
BINARY_NAME="gocode"
ENGINE_NAME="gocode-engine"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  darwin|linux) ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

PLATFORM="${OS}-${ARCH}"
ARCHIVE="${BINARY_NAME}-${PLATFORM}.tar.gz"

echo "Detected platform: ${PLATFORM}"

# Get latest release URL
LATEST_URL="https://github.com/${REPO}/releases/latest/download/${ARCHIVE}"
echo "Downloading ${LATEST_URL}..."

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -fsSL "$LATEST_URL" -o "$TMPDIR/$ARCHIVE"
tar -xzf "$TMPDIR/$ARCHIVE" -C "$TMPDIR"

# Install binaries
echo "Installing to ${INSTALL_DIR}..."
if [ -w "$INSTALL_DIR" ]; then
  cp "$TMPDIR/${BINARY_NAME}-${PLATFORM}/${BINARY_NAME}" "$INSTALL_DIR/"
  cp "$TMPDIR/${BINARY_NAME}-${PLATFORM}/${ENGINE_NAME}" "$INSTALL_DIR/"
else
  sudo cp "$TMPDIR/${BINARY_NAME}-${PLATFORM}/${BINARY_NAME}" "$INSTALL_DIR/"
  sudo cp "$TMPDIR/${BINARY_NAME}-${PLATFORM}/${ENGINE_NAME}" "$INSTALL_DIR/"
fi

chmod +x "$INSTALL_DIR/$BINARY_NAME" "$INSTALL_DIR/$ENGINE_NAME"

echo ""
echo "gocode installed successfully!"
echo ""
echo "Set your API key and start:"
echo "  export ANTHROPIC_API_KEY=\"sk-ant-...\""
echo "  gocode"
echo ""
echo "Or use a different provider:"
echo "  gocode --model openai/gpt-4o"
echo "  gocode --model ollama/gemma3"
