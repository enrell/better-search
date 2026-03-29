#!/bin/bash
set -e

REPO="enrell/searxng-web-fetch-mcp"
BIN_DIR="${HOME}/.local/bin"
BIN_NAME="searxng-web-fetch-mcp"
INSTALL_PATH="${BIN_DIR}/${BIN_NAME}"
FORCE_UPDATE="${FORCE_UPDATE:-false}"

mkdir -p "${BIN_DIR}"

get_current_version() {
    if [ -f "${INSTALL_PATH}" ]; then
        "${INSTALL_PATH}" 2>&1 | grep -oP 'v\d+\.\d+\.\d+' | head -1 || echo "unknown"
    else
        echo "none"
    fi
}

get_latest_version() {
    curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep -oP '"tag_name":\s*"\K[^"]+' || echo ""
}

get_download_url() {
    local platform=$1
    curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep -oP '"browser_download_url":\s*"\K[^"]+' | grep "${platform}" || echo ""
}

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "${OS}" in
  linux)
    case "${ARCH}" in
      x86_64)
        PLATFORM="linux-x86_64"
        ;;
      aarch64|arm64)
        PLATFORM="linux-arm64"
        ;;
      riscv64)
        PLATFORM="linux-riscv64"
        ;;
      *)
        echo "Unsupported Linux architecture: ${ARCH}"
        exit 1
        ;;
    esac
    ;;
  darwin)
    case "${ARCH}" in
      arm64)
        PLATFORM="darwin-arm64"
        ;;
      x86_64|x86_64h)
        PLATFORM="darwin-x86_64"
        ;;
      *)
        echo "Unsupported macOS architecture: ${ARCH}"
        exit 1
        ;;
    esac
    ;;
  mingw*|msys*|cygwin*|windows)
    case "${ARCH}" in
      x86_64)
        PLATFORM="windows-x86_64"
        ;;
      *)
        echo "Unsupported Windows architecture: ${ARCH}"
        exit 1
        ;;
    esac
    ;;
  *)
    echo "Unsupported platform: ${OS}"
    exit 1
    ;;
esac

if [ -f "${INSTALL_PATH}" ] && [ "${FORCE_UPDATE}" != "true" ]; then
    current=$(get_current_version)
    latest=$(get_latest_version)
    
    if [ -n "${latest}" ] && [ "${current}" != "${latest}" ]; then
        echo "Current version: ${current:-none}"
        echo "Latest version:  ${latest}"
        echo "Update available! Downloading..."
    elif [ "${current}" = "${latest}" ]; then
        echo "Already on latest version: ${latest}"
        exit 0
    else
        echo "Could not determine version. Proceeding with install..."
    fi
fi

echo "Downloading searxng-web-fetch-mcp for ${PLATFORM}..."

DOWNLOAD_URL=$(get_download_url "${PLATFORM}")

if [ -z "${DOWNLOAD_URL}" ]; then
    echo "Error: Could not find download URL for platform ${PLATFORM}"
    exit 1
fi

TEMP_PATH="${INSTALL_PATH}.tmp"
curl -L --max-time 120 "${DOWNLOAD_URL}" -o "${TEMP_PATH}" || {
    echo "Download failed! (timeout or network error)"
    rm -f "${TEMP_PATH}"
    exit 1
}
mv "${TEMP_PATH}" "${INSTALL_PATH}"
chmod +x "${INSTALL_PATH}"

echo "Installed to: ${INSTALL_PATH}"
