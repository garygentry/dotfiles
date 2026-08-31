#!/usr/bin/env bash
# nodejs/install.sh - Install Node.js (sudo-free, official prebuilt tarball)
#
# apt-based node needs sudo, so on a no-sudo host it degrades to installing
# nothing. This installs the official Node.js prebuilt tarball into ~/.local
# (the same deterministic, sudo-free pattern as Go and claude-code),
# checksum-verified against nodejs.org's SHASUMS256.txt. Works on every host,
# with or without sudo. Pin the LTS line with DOTFILES_NODE_MAJOR (default 22).
set -euo pipefail

_home="${DOTFILES_HOME:-$HOME}"
_bin="${_home}/.local/bin"
_share="${_home}/.local/share/node"

# On WSL the Windows PATH bleeds in, so `node` may resolve to a Windows install
# under /mnt/c. Treat that as absent — we want a real Linux/macOS node.
_node_path="$(command -v node 2>/dev/null || true)"
if [[ -n "$_node_path" && "$_node_path" != /mnt/* ]]; then
    log_info "Node.js already installed: $(node --version 2>/dev/null)"
    exit 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install Node.js (sudo-free tarball) to ${_share}"
    exit 0
fi

# Platform (node's naming: linux/darwin, x64/arm64).
case "$(uname -s)" in
    Linux)  _os="linux" ;;
    Darwin) _os="darwin" ;;
    *) log_error "Node.js: unsupported OS $(uname -s)"; exit 1 ;;
esac
case "$(uname -m)" in
    x86_64|amd64)  _arch="x64" ;;
    arm64|aarch64) _arch="arm64" ;;
    *) log_error "Node.js: unsupported architecture $(uname -m)"; exit 1 ;;
esac
_platform="${_os}-${_arch}"

_major="${DOTFILES_NODE_MAJOR:-22}"
_base="https://nodejs.org/dist/latest-v${_major}.x"

# SHASUMS256.txt gives us both the exact filename (newest patch of this LTS line)
# and its checksum, so we never hardcode a patch version.
_shasums="$(curl -fsSL "${_base}/SHASUMS256.txt" 2>/dev/null || true)"
if [[ -z "$_shasums" ]]; then
    log_error "Node.js: could not reach ${_base}/SHASUMS256.txt (network or bad DOTFILES_NODE_MAJOR=${_major}?)"
    exit 1
fi
_file="$(printf '%s\n' "$_shasums" | grep -oE "node-v[0-9]+\.[0-9]+\.[0-9]+-${_platform}\.tar\.gz" | head -1 || true)"
if [[ -z "$_file" ]]; then
    log_error "Node.js: no ${_platform} tarball in latest-v${_major}.x"
    exit 1
fi
_sum="$(printf '%s\n' "$_shasums" | awk -v f="$_file" '$2==f{print $1}' | head -1 || true)"
_ver="$(printf '%s' "$_file" | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' || true)"
if [[ ! "$_sum" =~ ^[a-f0-9]{64}$ || -z "$_ver" ]]; then
    log_error "Node.js: could not resolve checksum/version for ${_file}"
    exit 1
fi

_work="$(mktemp -d)"
trap 'rm -rf "$_work"' EXIT

log_info "Installing Node.js ${_ver} (${_platform}, sudo-free)..."
curl -fsSL -o "${_work}/node.tar.gz" "${_base}/${_file}"
_actual="$(sha256sum "${_work}/node.tar.gz" | cut -d' ' -f1)"
if [[ "$_actual" != "$_sum" ]]; then
    log_error "Node.js: checksum mismatch (expected ${_sum}, got ${_actual})"
    exit 1
fi

_dir="node-${_ver}-${_platform}"   # the tarball's top-level directory
mkdir -p "$_share" "$_bin"
rm -rf "${_share:?}/${_dir}"
tar -xzf "${_work}/node.tar.gz" -C "$_share"

# Symlink the user-facing binaries into ~/.local/bin. npm/npx inside the tarball
# are relative symlinks into lib/node_modules, so they resolve through our link.
for _b in node npm npx; do
    if [[ -e "${_share}/${_dir}/bin/${_b}" ]]; then
        ln -sf "${_share}/${_dir}/bin/${_b}" "${_bin}/${_b}"
    fi
done

log_success "Node.js installed: $("${_bin}/node" --version 2>/dev/null) (npm $("${_bin}/npm" --version 2>/dev/null))"
