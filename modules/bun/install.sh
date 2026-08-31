#!/usr/bin/env bash
# bun/install.sh - Install Bun (sudo-free, official release binary)
#
# Installs the official Bun release binary into ~/.local (sudo-free), verified
# against the release's SHASUMS256.txt when published. The x64 default build
# requires AVX2, so on a CPU without it (common on VMs) we pick the -baseline
# build. Pin a version with DOTFILES_BUN_VERSION=X.Y.Z (default: latest).
set -euo pipefail

_home="${DOTFILES_HOME:-$HOME}"
_bin="${_home}/.local/bin"
_share="${_home}/.local/share/bun"

if command -v bun >/dev/null 2>&1; then
    log_info "Bun already installed: $(bun --version 2>/dev/null)"
    exit 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install Bun (sudo-free) to ${_share}"
    exit 0
fi

# Platform (bun's naming: linux/darwin, x64/aarch64).
case "$(uname -s)" in
    Linux)  _os="linux" ;;
    Darwin) _os="darwin" ;;
    *) log_error "Bun: unsupported OS $(uname -s)"; exit 1 ;;
esac
case "$(uname -m)" in
    x86_64|amd64)  _arch="x64" ;;
    arm64|aarch64) _arch="aarch64" ;;
    *) log_error "Bun: unsupported architecture $(uname -m)"; exit 1 ;;
esac
_target="${_os}-${_arch}"
# The default x64 build needs AVX2; fall back to the baseline build without it.
if [[ "$_arch" == "x64" ]] && ! grep -qw avx2 /proc/cpuinfo 2>/dev/null; then
    _target="${_target}-baseline"
fi
_asset="bun-${_target}.zip"

# Resolve the release tag (default: latest).
if [[ -n "${DOTFILES_BUN_VERSION:-}" ]]; then
    _tag="bun-v${DOTFILES_BUN_VERSION#v}"
else
    _tag="$(curl -fsSL https://api.github.com/repos/oven-sh/bun/releases/latest 2>/dev/null \
        | grep -oE '"tag_name":[[:space:]]*"bun-v[0-9.]+"' | grep -oE 'bun-v[0-9.]+' | head -1 || true)"
fi
if [[ -z "$_tag" ]]; then
    log_error "Bun: could not resolve a release tag from GitHub"
    exit 1
fi

_rel="https://github.com/oven-sh/bun/releases/download/${_tag}"
_work="$(mktemp -d)"
trap 'rm -rf "$_work"' EXIT

log_info "Installing Bun ${_tag#bun-v} (${_target}, sudo-free)..."
if ! curl -fsSL -o "${_work}/bun.zip" "${_rel}/${_asset}"; then
    log_error "Bun: download failed (${_asset} @ ${_tag})"
    exit 1
fi

# Verify the checksum when the release publishes one (defense-in-depth).
_sums="$(curl -fsSL "${_rel}/SHASUMS256.txt" 2>/dev/null || true)"
if [[ -n "$_sums" ]]; then
    _sum="$(printf '%s\n' "$_sums" | awk -v f="$_asset" '{g=$2; sub(/^\*/,"",g)} g==f{print $1}' | head -1 || true)"
    if [[ "$_sum" =~ ^[a-f0-9]{64}$ ]]; then
        _actual="$(sha256sum "${_work}/bun.zip" | cut -d' ' -f1)"
        if [[ "$_actual" != "$_sum" ]]; then
            log_error "Bun: checksum mismatch (expected ${_sum}, got ${_actual})"
            exit 1
        fi
        log_info "Bun: checksum verified"
    else
        log_warn "Bun: no checksum for ${_asset} in SHASUMS256.txt — installing unverified"
    fi
else
    log_warn "Bun: no SHASUMS256.txt for ${_tag} — installing unverified"
fi

# Bun ships as a zip; extract with unzip, else python3 (no sudo either way).
if command -v unzip >/dev/null 2>&1; then
    unzip -q -o "${_work}/bun.zip" -d "${_work}/x"
elif command -v python3 >/dev/null 2>&1; then
    python3 -m zipfile -e "${_work}/bun.zip" "${_work}/x"
else
    log_error "Bun: need 'unzip' or 'python3' to extract the release zip (neither found)"
    exit 1
fi

_bunbin="$(find "${_work}/x" -type f -name bun -print -quit 2>/dev/null || true)"
if [[ -z "$_bunbin" ]]; then
    log_error "Bun: binary not found inside ${_asset}"
    exit 1
fi

mkdir -p "$_share" "$_bin"
cp -f "$_bunbin" "${_share}/bun"
chmod +x "${_share}/bun"
ln -sf "${_share}/bun" "${_bin}/bun"
# bunx is bun invoked under a different name; a symlink is the supported setup.
ln -sf "${_share}/bun" "${_bin}/bunx"

log_success "Bun installed: $("${_bin}/bun" --version 2>/dev/null)"
