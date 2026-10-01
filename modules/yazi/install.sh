#!/usr/bin/env bash
# yazi/install.sh - Install the yazi file manager (yazi + ya), sudo-free.
#
# modules.yazi.tools set: declare and install those tools through mise.
# Otherwise: Homebrew on macOS, else the upstream release zip into ~/.local.
set -euo pipefail

_home="${DOTFILES_HOME:-$HOME}"
_bin="${_home}/.local/bin"
_share="${_home}/.local/share/yazi"

if [[ -n "${DOTFILES_SETTING_TOOLS:-}" ]]; then
    mise_sync_tools "$DOTFILES_MODULE_NAME"
    exit 0
fi
# No tools declared: drop a fragment from an earlier mise-managed run, so mise
# stops owning yazi alongside the install below (two owners).
mise_sync_tools "$DOTFILES_MODULE_NAME"

if command -v yazi >/dev/null 2>&1 || [[ -x "${_bin}/yazi" ]]; then
    _have="$(command -v yazi 2>/dev/null || echo "${_bin}/yazi")"
    log_info "yazi already installed: $("$_have" --version 2>/dev/null | grep -m1 -oE '[0-9]+\.[0-9]+\.[0-9]+' || true)"
    exit 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install yazi (sudo-free) to ${_share}"
    exit 0
fi

if is_macos && command -v brew >/dev/null 2>&1; then
    pkg_install yazi
    log_success "yazi installed"
    exit 0
fi

case "$(uname -s)" in
    Linux)  _os="unknown-linux-musl" ;;   # static: runs on any glibc or musl
    Darwin) _os="apple-darwin" ;;
    *) log_error "yazi: unsupported OS $(uname -s)"; exit 1 ;;
esac
case "$(uname -m)" in
    x86_64|amd64)  _arch="x86_64" ;;
    arm64|aarch64) _arch="aarch64" ;;
    *) log_error "yazi: unsupported architecture $(uname -m)"; exit 1 ;;
esac
_asset="yazi-${_arch}-${_os}.zip"

# Resolve the tag WITHOUT the GitHub API: follow the /releases/latest redirect.
if [[ -n "${DOTFILES_YAZI_VERSION:-}" ]]; then
    _tag="v${DOTFILES_YAZI_VERSION#v}"
else
    _eff="$(curl -fsSL -o /dev/null -w '%{url_effective}' https://github.com/sxyazi/yazi/releases/latest 2>/dev/null || true)"
    _tag="$(printf '%s' "$_eff" | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | tail -1 || true)"
fi
if [[ -z "$_tag" ]]; then
    log_error "yazi: could not resolve a release tag"
    exit 1
fi

_work="$(mktemp -d)"
trap 'rm -rf "$_work"' EXIT

# yazi publishes no checksums file; the API reports each asset's sha256 digest.
# One best-effort call: on a rate limit, install unverified with a warning.
_sum="$(curl -fsSL "https://api.github.com/repos/sxyazi/yazi/releases/tags/${_tag}" 2>/dev/null \
    | awk -v f="$_asset" '/"name":/ {n = ($0 ~ "\"" f "\"")} n && /"digest":/ {print; exit}' \
    | grep -oE '[a-f0-9]{64}' || true)"
[[ -n "$_sum" ]] || log_warn "yazi: no published digest for ${_asset} (GitHub API unavailable?) — installing unverified"

log_info "Installing yazi ${_tag#v} (${_arch}, sudo-free)..."
if ! download_file "https://github.com/sxyazi/yazi/releases/download/${_tag}/${_asset}" "${_work}/yazi.zip" "$_sum"; then
    log_error "yazi: download/checksum verification failed (${_asset} @ ${_tag})"
    exit 1
fi

if command -v unzip >/dev/null 2>&1; then
    unzip -q -o "${_work}/yazi.zip" -d "${_work}/x"
elif command -v python3 >/dev/null 2>&1; then
    python3 -m zipfile -e "${_work}/yazi.zip" "${_work}/x"
else
    log_error "yazi: need 'unzip' or 'python3' to extract the release zip (neither found)"
    exit 1
fi

mkdir -p "$_share" "$_bin"
for _b in yazi ya; do
    _src="$(find "${_work}/x" -type f -name "$_b" -print -quit 2>/dev/null || true)"
    if [[ -z "$_src" ]]; then
        log_error "yazi: ${_b} not found inside ${_asset}"
        exit 1
    fi
    cp -f "$_src" "${_share}/${_b}"
    chmod +x "${_share}/${_b}"
    ln -sf "${_share}/${_b}" "${_bin}/${_b}"
done

_yv="$("${_bin}/yazi" --version 2>/dev/null | grep -m1 -oE '[0-9]+\.[0-9]+\.[0-9]+' || true)"
if [[ -z "$_yv" ]]; then
    log_error "yazi: installed but not runnable (${_asset} may be wrong for this host)"
    exit 1
fi
log_success "yazi installed: ${_yv}"
