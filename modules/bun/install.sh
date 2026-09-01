#!/usr/bin/env bash
# bun/install.sh - Install Bun (sudo-free, official release binary)
#
# Installs the official Bun release binary into ~/.local (sudo-free), verified
# against the release's SHASUMS256.txt when published. Handles linux/darwin,
# x64/aarch64, musl, and the -baseline builds (the x64 default needs AVX2, so on
# a CPU without it — common on VMs — we pick -baseline). Pin a version with
# DOTFILES_BUN_VERSION=X.Y.Z (default: latest).
set -euo pipefail

_home="${DOTFILES_HOME:-$HOME}"
_bin="${_home}/.local/bin"
_share="${_home}/.local/share/bun"

# Already installed? Match PATH OR our own ~/.local/bin/bun (may be off PATH on
# a fresh host).
if command -v bun >/dev/null 2>&1 || [[ -x "${_bin}/bun" ]]; then
    _have="$(command -v bun 2>/dev/null || echo "${_bin}/bun")"
    log_info "Bun already installed: $("$_have" --version 2>/dev/null)"
    exit 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install Bun (sudo-free) to ${_share}"
    exit 0
fi

# Platform (bun's naming: linux/darwin, x64/aarch64; -musl on musl libc;
# -baseline for x64 CPUs without AVX2).
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
# musl build on musl libc (Linux only).
if [[ "$_os" == "linux" ]] && { [[ -e /lib/libc.musl-x86_64.so.1 ]] || [[ -e /lib/libc.musl-aarch64.so.1 ]] || ldd /bin/ls 2>&1 | grep -q musl; }; then
    _target="${_target}-musl"
fi
# baseline build for x64 without AVX2 (Linux only; /proc/cpuinfo is Linux). The
# asset order is <os>-<arch>[-musl]-baseline, so append after -musl.
if [[ "$_os" == "linux" && "$_arch" == "x64" ]] && ! grep -qw avx2 /proc/cpuinfo 2>/dev/null; then
    _target="${_target}-baseline"
fi
_asset="bun-${_target}.zip"

# Resolve the release tag WITHOUT the GitHub API (60 req/hr/IP would throttle a
# fleet): follow the /releases/latest redirect and read the tag from the URL.
if [[ -n "${DOTFILES_BUN_VERSION:-}" ]]; then
    _tag="bun-v${DOTFILES_BUN_VERSION#v}"
else
    _eff="$(curl -fsSL -o /dev/null -w '%{url_effective}' https://github.com/oven-sh/bun/releases/latest 2>/dev/null || true)"
    _tag="$(printf '%s' "$_eff" | grep -oE 'bun-v[0-9.]+' | head -1 || true)"
fi
if [[ -z "$_tag" ]]; then
    log_error "Bun: could not resolve a release tag"
    exit 1
fi

_rel="https://github.com/oven-sh/bun/releases/download/${_tag}"
_work="$(mktemp -d)"
trap 'rm -rf "$_work"' EXIT

# Checksum when the release publishes one (defense-in-depth; best-effort).
_sum=""
_sums="$(curl -fsSL "${_rel}/SHASUMS256.txt" 2>/dev/null || true)"
if [[ -n "$_sums" ]]; then
    _sum="$(printf '%s\n' "$_sums" | awk -v f="$_asset" '{g=$2; sub(/^\*/,"",g)} g==f{print $1}' | head -1 || true)"
    [[ "$_sum" =~ ^[a-f0-9]{64}$ ]] || { log_warn "Bun: no checksum for ${_asset} in SHASUMS256.txt — installing unverified"; _sum=""; }
else
    log_warn "Bun: no SHASUMS256.txt for ${_tag} — installing unverified"
fi

log_info "Installing Bun ${_tag#bun-v} (${_target}, sudo-free)..."
# download_file does portable checksum (sha256sum/shasum) + curl/wget + abort;
# an empty _sum means download-without-verify.
if ! download_file "${_rel}/${_asset}" "${_work}/bun.zip" "$_sum"; then
    log_error "Bun: download/checksum verification failed (${_asset} @ ${_tag})"
    exit 1
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

# Execute it — a wrong-arch/baseline binary must fail here, not report empty.
_bv="$("${_bin}/bun" --version 2>/dev/null || true)"
if [[ -z "$_bv" ]]; then
    log_error "Bun: installed but not runnable (${_target} may be wrong for this CPU/libc)"
    exit 1
fi
log_success "Bun installed: ${_bv}"
