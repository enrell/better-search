#!/bin/bash
set -e

REPO="enrell/searxng-web-fetch-mcp"
BIN_DIR="${HOME}/.local/bin"
BIN_NAME="searxng-web-fetch-mcp"
INSTALL_PATH="${BIN_DIR}/${BIN_NAME}"

mkdir -p "${BIN_DIR}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "${OS}" in
  linux)
    PLATFORM="linux-x86_64"
    ;;
  darwin)
    if [ "${ARCH}" = "arm64" ]; then
      PLATFORM="darwin-arm64"
    else
      PLATFORM="darwin-x86_64"
    fi
    ;;
  *)
    echo "Unsupported platform: ${OS}"
    exit 1
    ;;
esac

echo "Downloading searxng-web-fetch-mcp for ${PLATFORM}..."
curl -sL "https://github.com/${REPO}/releases/latest/download/searxng-web-fetch-mcp-${PLATFORM}" -o "${INSTALL_PATH}"
chmod +x "${INSTALL_PATH}"

echo "Installed to: ${INSTALL_PATH}"