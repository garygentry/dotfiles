#!/usr/bin/env bash
# lazygit/install.sh - Install lazygit, sudo-free.
#
# modules.lazygit.tools set: declare and install it through mise.
# Otherwise: Homebrew on macOS, else the upstream release tarball into ~/.local/bin.
set -euo pipefail

_home="${DOTFILES_HOME:-$HOME}"
_bin="${_home}/.local/bin"

if [[ -n "${DOTFILES_SETTING_TOOLS:-}" ]]; then
    mise_sync_tools "$DOTFILES_MODULE_NAME"
    exit 0
fi
# No tools declared: drop a fragment from an earlier mise-managed run (one owner).
mise_sync_tools "$DOTFILES_MODULE_NAME"

if command -v lazygit >/dev/null 2>&1 || [[ -x "${_bin}/lazygit" ]]; then
    _have="$(command -v lazygit 2>/dev/null || echo "${_bin}/lazygit")"
    log_info "lazygit already installed: $("$_have" --version 2>/dev/null | grep -oE 'version=[^,]+' || true)"
    exit 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install lazygit (sudo-free) to ${_bin}"
    exit 0
fi

if is_macos && command -v brew >/dev/null 2>&1; then
    pkg_install lazygit
    log_success "lazygit installed"
    exit 0
fi

case "$(uname -s)" in
    Linux)  _os="linux" ;;
    Darwin) _os="darwin" ;;
    *) log_error "lazygit: unsupported OS $(uname -s)"; exit 1 ;;
esac
case "$(uname -m)" in
    x86_64|amd64)  _arch="x86_64" ;;
    arm64|aarch64) _arch="arm64" ;;
    *) log_error "lazygit: unsupported architecture $(uname -m)"; exit 1 ;;
esac

# Resolve the tag WITHOUT the GitHub API: follow the /releases/latest redirect.
if [[ -n "${DOTFILES_LAZYGIT_VERSION:-}" ]]; then
    _ver="${DOTFILES_LAZYGIT_VERSION#v}"
else
    _eff="$(curl -fsSL -o /dev/null -w '%{url_effective}' https://github.com/jesseduffield/lazygit/releases/latest 2>/dev/null || true)"
    _ver="$(printf '%s' "$_eff" | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | tail -1 || true)"
    _ver="${_ver#v}"
fi
if [[ -z "$_ver" ]]; then
    log_error "lazygit: could not resolve a release tag"
    exit 1
fi

_asset="lazygit_${_ver}_${_os}_${_arch}.tar.gz"
_rel="https://github.com/jesseduffield/lazygit/releases/download/v${_ver}"
_work="$(mktemp -d)"
trap 'rm -rf "$_work"' EXIT

_sum="$(curl -fsSL "${_rel}/checksums.txt" 2>/dev/null | awk -v f="$_asset" '$2==f{print $1}' | head -1 || true)"
if [[ ! "$_sum" =~ ^[a-f0-9]{64}$ ]]; then
    log_warn "lazygit: no checksum for ${_asset} in checksums.txt — installing unverified"
    _sum=""
fi

log_info "Installing lazygit ${_ver} (${_os}/${_arch}, sudo-free)..."
if ! download_file "${_rel}/${_asset}" "${_work}/lazygit.tar.gz" "$_sum"; then
    log_error "lazygit: download/checksum verification failed (${_asset})"
    exit 1
fi
tar -xzf "${_work}/lazygit.tar.gz" -C "$_work" lazygit
mkdir -p "$_bin"
install -m 0755 "${_work}/lazygit" "${_bin}/lazygit"

if ! "${_bin}/lazygit" --version >/dev/null 2>&1; then
    log_error "lazygit: installed but not runnable (${_asset} may be wrong for this host)"
    exit 1
fi
log_success "lazygit installed: ${_ver}"
