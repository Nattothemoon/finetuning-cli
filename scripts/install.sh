#!/usr/bin/env bash
# Install the latest `ft` binary from GitHub Releases.
#
#   curl -fsSL https://raw.githubusercontent.com/Nattothemoon/finetuning-cli/main/scripts/install.sh | bash
#
# Honors:
#   FT_VERSION=v0.1.0   pin a specific tag (default: latest)
#   FT_INSTALL_DIR=...  override install destination (default: /usr/local/bin or $HOME/.local/bin)
#   FT_REPO=owner/repo  override GitHub repo (default: Nattothemoon/finetuning-cli)

set -euo pipefail

repo="${FT_REPO:-Nattothemoon/finetuning-cli}"
version="${FT_VERSION:-latest}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "unsupported architecture: $arch" >&2; exit 1 ;;
esac
case "$os" in
  darwin|linux) ;;
  *) echo "unsupported OS: $os (use install.ps1 on Windows)" >&2; exit 1 ;;
esac

if [ "$version" = "latest" ]; then
  api_url="https://api.github.com/repos/${repo}/releases/latest"
  version="$(curl -fsSL "$api_url" | grep -E '"tag_name"' | head -n1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"
  if [ -z "$version" ]; then
    echo "could not determine latest release of $repo" >&2
    exit 1
  fi
fi

# Strip leading v for the archive name; keep it for the URL path.
v_clean="${version#v}"

archive="ft_${v_clean}_${os}_${arch}.tar.gz"
url="https://github.com/${repo}/releases/download/${version}/${archive}"

# Decide install dir: prefer /usr/local/bin if writable, else $HOME/.local/bin.
install_dir="${FT_INSTALL_DIR:-}"
if [ -z "$install_dir" ]; then
  if [ -w "/usr/local/bin" ]; then
    install_dir="/usr/local/bin"
  else
    install_dir="$HOME/.local/bin"
    mkdir -p "$install_dir"
  fi
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $url" >&2
curl -fsSL "$url" -o "$tmp/$archive"

tar -xzf "$tmp/$archive" -C "$tmp"
chmod +x "$tmp/ft"

dest="$install_dir/ft"
if [ -w "$install_dir" ]; then
  mv "$tmp/ft" "$dest"
else
  echo "Installing to $dest (requires sudo)..." >&2
  sudo mv "$tmp/ft" "$dest"
fi

echo "Installed ft $version to $dest" >&2
case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) echo "note: $install_dir is not on your PATH — add it to your shell profile." >&2 ;;
esac

"$dest" --version || true
