#!/bin/sh
set -eu

REPO="btouchard/herald"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY="herald"

main() {
    os="$(detect_os)"
    arch="$(detect_arch)"
    version="$(fetch_latest_version)"

    printf "Installing %s %s (%s/%s)...\n" "$BINARY" "$version" "$os" "$arch"

    # Version without the leading "v" (goreleaser strips it in archive names)
    ver="${version#v}"
    archive="${BINARY}_${ver}_${os}_${arch}.tar.gz"
    url="https://github.com/${REPO}/releases/download/${version}/${archive}"
    checksums_url="https://github.com/${REPO}/releases/download/${version}/checksums.txt"

    tmpdir="$(mktemp -d)"
    trap 'rm -rf "$tmpdir"' EXIT

    printf "Downloading %s...\n" "$archive"
    download "$url" "$tmpdir/$archive"
    download "$checksums_url" "$tmpdir/checksums.txt"

    printf "Verifying checksum...\n"
    verify_checksum "$tmpdir" "$archive"

    printf "Extracting...\n"
    tar -xzf "$tmpdir/$archive" -C "$tmpdir"

    printf "Installing to %s...\n" "$INSTALL_DIR"
    install_binary "$tmpdir/$BINARY" "$INSTALL_DIR/$BINARY"

    printf "Done. %s %s installed at %s/%s\n" "$BINARY" "$version" "$INSTALL_DIR" "$BINARY"
}

detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       printf "Unsupported OS: %s\n" "$(uname -s)" >&2; exit 1 ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)             printf "Unsupported architecture: %s\n" "$(uname -m)" >&2; exit 1 ;;
    esac
}

fetch_latest_version() {
    url="https://api.github.com/repos/${REPO}/releases/latest"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" | parse_version
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$url" | parse_version
    else
        printf "curl or wget required\n" >&2; exit 1
    fi
}

parse_version() {
    # Extract tag_name without requiring jq
    sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1
}

download() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "$2" "$1"
    else
        wget -qO "$2" "$1"
    fi
}

verify_checksum() {
    dir="$1"
    file="$2"
    expected="$(grep "$file" "$dir/checksums.txt" | awk '{print $1}')"
    if [ -z "$expected" ]; then
        printf "Checksum not found for %s\n" "$file" >&2; exit 1
    fi
    actual="$(sha256sum "$dir/$file" 2>/dev/null || shasum -a 256 "$dir/$file" 2>/dev/null)"
    actual="$(echo "$actual" | awk '{print $1}')"
    if [ "$expected" != "$actual" ]; then
        printf "Checksum mismatch: expected %s, got %s\n" "$expected" "$actual" >&2; exit 1
    fi
}

install_binary() {
    src="$1"
    dst="$2"
    if [ -w "$(dirname "$dst")" ]; then
        mv "$src" "$dst"
        chmod +x "$dst"
    else
        printf "Need elevated permissions for %s\n" "$(dirname "$dst")"
        sudo mv "$src" "$dst"
        sudo chmod +x "$dst"
    fi
}

main
