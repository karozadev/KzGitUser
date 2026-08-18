#!/bin/sh
# Install script for kzgit (KzGitUser).
#
# Usage:
#   wget -qO- https://raw.githubusercontent.com/karozadev/KzGitUser/main/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/karozadev/KzGitUser/main/install.sh | sh
#
# Or download and inspect first:
#   wget https://raw.githubusercontent.com/karozadev/KzGitUser/main/install.sh
#   chmod +x install.sh
#   ./install.sh

set -eu

REPO="karozadev/KzGitUser"
BINARY_NAME="kzgit"
INSTALL_DIR="${KZGIT_INSTALL_DIR:-/usr/local/bin}"

info() {
    printf '\033[1;34m==>\033[0m %s\n' "$1"
}

error() {
    printf '\033[1;31mError:\033[0m %s\n' "$1" >&2
    exit 1
}

detect_os() {
    os=$(uname -s)
    case "$os" in
        Linux) echo "linux" ;;
        Darwin) echo "darwin" ;;
        MINGW* | MSYS* | CYGWIN*) echo "windows" ;;
        *) error "Unsupported operating system: $os" ;;
    esac
}

detect_arch() {
    arch=$(uname -m)
    case "$arch" in
        x86_64 | amd64) echo "amd64" ;;
        arm64 | aarch64) echo "arm64" ;;
        *) error "Unsupported architecture: $arch" ;;
    esac
}

downloader() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$1"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$1"
    else
        error "Neither curl nor wget is available. Please install one of them and try again."
    fi
}

download_file() {
    url="$1"
    dest="$2"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "$dest" "$url" || error "Failed to download $url"
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O "$dest" "$url" || error "Failed to download $url"
    else
        error "Neither curl nor wget is available. Please install one of them and try again."
    fi
}

main() {
    os=$(detect_os)
    arch=$(detect_arch)
    info "Detected platform: ${os}/${arch}"

    info "Fetching latest release information..."
    latest_tag=$(downloader "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name":' \
        | sed -E 's/.*"tag_name":[[:space:]]*"([^"]+)".*/\1/') \
        || error "Failed to fetch the latest release information from GitHub"

    if [ -z "$latest_tag" ]; then
        error "Could not determine the latest kzgit release"
    fi
    info "Latest release: ${latest_tag}"

    version="${latest_tag#v}"

    if [ "$os" = "windows" ]; then
        archive_ext="zip"
    else
        archive_ext="tar.gz"
    fi

    archive_name="kzgit_${version}_${os}_${arch}.${archive_ext}"
    download_url="https://github.com/${REPO}/releases/download/${latest_tag}/${archive_name}"

    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT

    info "Downloading ${archive_name}..."
    download_file "$download_url" "${tmp_dir}/${archive_name}"

    info "Extracting archive..."
    case "$archive_ext" in
        tar.gz)
            tar -xzf "${tmp_dir}/${archive_name}" -C "$tmp_dir"
            ;;
        zip)
            command -v unzip >/dev/null 2>&1 || error "unzip is required to install kzgit on Windows"
            unzip -q "${tmp_dir}/${archive_name}" -d "$tmp_dir"
            ;;
    esac

    binary_path="${tmp_dir}/${BINARY_NAME}"
    if [ "$os" = "windows" ]; then
        binary_path="${binary_path}.exe"
    fi
    [ -f "$binary_path" ] || error "Could not find the ${BINARY_NAME} binary in the downloaded archive"

    if [ -w "$INSTALL_DIR" ]; then
        install_cmd=""
    elif command -v sudo >/dev/null 2>&1; then
        install_cmd="sudo"
    else
        error "No write permission on ${INSTALL_DIR} and sudo is not available. Set KZGIT_INSTALL_DIR to a writable directory and retry."
    fi

    info "Installing ${BINARY_NAME} to ${INSTALL_DIR}..."
    mkdir -p "$INSTALL_DIR" 2>/dev/null || true
    if [ -n "${install_cmd:-}" ]; then
        $install_cmd mkdir -p "$INSTALL_DIR"
        $install_cmd cp "$binary_path" "${INSTALL_DIR}/${BINARY_NAME}"
        $install_cmd chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    else
        cp "$binary_path" "${INSTALL_DIR}/${BINARY_NAME}"
        chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        info "kzgit installed successfully!"
        "$BINARY_NAME" version
    elif [ -x "${INSTALL_DIR}/${BINARY_NAME}" ]; then
        info "kzgit installed successfully to ${INSTALL_DIR}/${BINARY_NAME}"
        info "Make sure ${INSTALL_DIR} is in your PATH to use the 'kzgit' command directly."
        "${INSTALL_DIR}/${BINARY_NAME}" version
    else
        error "Installation verification failed: ${BINARY_NAME} was not found after installation"
    fi
}

main "$@"
