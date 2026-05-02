#!/usr/bin/env bash
set -euo pipefail

REPO="cy-infamous/purewin"
BINARY="pw"
INSTALL_DIR="/usr/local/bin"
TMP_DIR=$(mktemp -d)

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

info()  { echo -e "${CYAN}▸${NC} $*"; }
ok()    { echo -e "${GREEN}✓${NC} $*"; }
warn()  { echo -e "${YELLOW}!${NC} $*"; }
err()   { echo -e "${RED}✗${NC} $*" >&2; }

cleanup() { rm -rf "$TMP_DIR"; }
trap cleanup EXIT

ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    armv7l)  ARCH="arm" ;;
    *)       err "Unsupported architecture: $ARCH"; exit 1 ;;
esac

OS=$(uname -s | tr '[:upper:]' '[:lower:]')

if [ "$OS" != "linux" ]; then
    err "This script is for Linux only. For Windows, use install.ps1"
    exit 1
fi

info "Installing ${BOLD}PureWin${NC} for ${OS}/${ARCH}..."

LATEST_URL="https://api.github.com/repos/${REPO}/releases/latest"
TAG=$(curl -fsSL "$LATEST_URL" 2>/dev/null | grep '"tag_name"' | head -1 | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$TAG" ]; then
    warn "No GitHub release found. Building from source..."
    
    if ! command -v go &>/dev/null; then
        err "Go is not installed. Install Go 1.24+ from https://go.dev/dl/"
        exit 1
    fi
    
    info "Cloning repository..."
    git clone --depth 1 "https://github.com/${REPO}.git" "$TMP_DIR/purewin"
    
    info "Building..."
    cd "$TMP_DIR/purewin"
    CGO_ENABLED=0 go build -o "$BINARY" .
    
    info "Installing to ${INSTALL_DIR}..."
    mv "$BINARY" "${INSTALL_DIR}/${BINARY}"
    chmod +x "${INSTALL_DIR}/${BINARY}"
    
    ok "Installed PureWin (built from source)"
    ok "Run '${BOLD}pw${NC}' to get started"
    exit 0
fi

FILENAME="pw-${OS}-${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${FILENAME}"

info "Downloading ${TAG}..."
if ! curl -fsSL -o "${TMP_DIR}/${FILENAME}" "$DOWNLOAD_URL"; then
    err "Failed to download ${FILENAME}"
    err "Try installing from source: go install github.com/${REPO}@latest"
    exit 1
fi

info "Extracting..."
tar xzf "${TMP_DIR}/${FILENAME}" -C "$TMP_DIR"

info "Installing to ${INSTALL_DIR}..."
mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
chmod +x "${INSTALL_DIR}/${BINARY}"

ok "Installed PureWin ${TAG}"
ok "Run '${BOLD}pw${NC}' to get started"
