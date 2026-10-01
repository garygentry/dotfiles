#!/usr/bin/env bash
# lazygit/install.sh - Install lazygit, sudo-free.
#
# modules.lazygit.tools set: declare and install it through mise. Otherwise:
# Homebrew on macOS, else the upstream release tarball into
# ~/.local/share/lazygit (module-owned), linked from ~/.local/bin.
set -euo pipefail

_home="${DOTFILES_HOME:-$HOME}"
_bin="${_home}/.local/bin"
_share="${_home}/.local/share/lazygit"
_shims="${_home}/.local/share/mise/shims"

# _drop_release: remove this module's release install, so mise is the only owner.
_drop_release() {
    [[ -d "$_share" ]] || return 0
    if [[ -L "${_bin}/lazygit" && "$(readlink "${_bin}/lazygit")" == "${_share}/"* ]]; then
        rm -f "${_bin}/lazygit"
    fi
    rm -rf "$_share"
    log_info "lazygit: removed the release install in ${_share} (mise owns lazygit now)"
}

# _find: a RUNNABLE lazygit on PATH or in ~/.local/bin, skipping mise shims
# (a shim left behind by a removed mise declaration fails with "No version is set").
_find() {
    local p
    while IFS= read -r p; do
        [[ "$p" == "${_shims}/"* ]] && continue
        "$p" --version >/dev/null 2>&1 && { printf '%s' "$p"; return 0; }
    done < <(type -ap lazygit 2>/dev/null || true; [[ -x "${_bin}/lazygit" ]] && printf '%s\n' "${_bin}/lazygit")
    return 1
}
_ver_of() { "$1" --version 2>/dev/null | grep -oE 'version=[0-9]+\.[0-9]+\.[0-9]+' | head -1 | cut -d= -f2 || true; }

if [[ -n "${DOTFILES_SETTING_TOOLS:-}" ]]; then
    mise_sync_tools "$DOTFILES_MODULE_NAME"
    is_dry_run && exit 0
    # With a lockfile, mise skips a tool that has no entry for this platform. Then
    # fall through to Homebrew / the release tarball.
    if "${_shims}/lazygit" --version >/dev/null 2>&1; then
        _drop_release
        exit 0
    fi
    log_warn "lazygit: mise provides no lazygit on this platform; falling back to Homebrew / the release binary"
else
    # No tools declared: drop a fragment from an earlier mise-managed run (one owner).
    mise_sync_tools "$DOTFILES_MODULE_NAME"
fi
if [[ -x "${_shims}/lazygit" ]] && ! "${_shims}/lazygit" --version >/dev/null 2>&1; then
    log_warn "lazygit: a stale mise shim (${_shims}/lazygit) shadows this install in shells that put the shims first; remove it with \`mise uninstall lazygit\`"
fi

_want="${DOTFILES_LAZYGIT_VERSION:-}"; _want="${_want#v}"
if _have="$(_find)"; then
    _hv="$(_ver_of "$_have")"
    if [[ -z "$_want" || "$_hv" == "$_want" ]]; then
        log_info "lazygit already installed: ${_hv:-unknown version} (${_have})"
        exit 0
    fi
    if [[ "$(readlink -f "$_have")" != "${_share}/"* ]]; then
        log_warn "lazygit ${_hv} at ${_have} is not this module's install; leaving it (DOTFILES_LAZYGIT_VERSION=${_want} not applied)"
        exit 0
    fi
    log_info "lazygit ${_hv} installed, DOTFILES_LAZYGIT_VERSION wants ${_want}: reinstalling"
fi

if is_dry_run; then
    log_info "[dry-run] Would install lazygit (sudo-free) to ${_share}"
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
if [[ -n "$_want" ]]; then
    _ver="$_want"
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

# Fail closed without a checksum unless DOTFILES_ALLOW_UNVERIFIED=1.
_sum="$(curl -fsSL "${_rel}/checksums.txt" 2>/dev/null | awk -v f="$_asset" '$2==f{print $1}' | head -1 || true)"
if [[ ! "$_sum" =~ ^[a-f0-9]{64}$ ]]; then
    _sum=""
    if [[ "${DOTFILES_ALLOW_UNVERIFIED:-}" == 1 ]]; then
        log_warn "lazygit: no checksum for ${_asset} in checksums.txt — installing unverified (DOTFILES_ALLOW_UNVERIFIED=1)"
    else
        log_error "lazygit: no checksum for ${_asset} in the release's checksums.txt (set DOTFILES_ALLOW_UNVERIFIED=1 to install anyway)"
        exit 1
    fi
fi

log_info "Installing lazygit ${_ver} (${_os}/${_arch}, sudo-free)..."
if ! download_file "${_rel}/${_asset}" "${_work}/lazygit.tar.gz" "$_sum"; then
    log_error "lazygit: download/checksum verification failed (${_asset})"
    exit 1
fi
tar -xzf "${_work}/lazygit.tar.gz" -C "$_work" lazygit
mkdir -p "$_share" "$_bin"
install -m 0755 "${_work}/lazygit" "${_share}/lazygit"
ln -sf "${_share}/lazygit" "${_bin}/lazygit"

if ! "${_bin}/lazygit" --version >/dev/null 2>&1; then
    log_error "lazygit: installed but not runnable (${_asset} may be wrong for this host)"
    exit 1
fi
log_success "lazygit installed: ${_ver}"
