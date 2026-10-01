#!/usr/bin/env bash
# yazi/install.sh - Install the yazi file manager (yazi + ya), sudo-free.
#
# modules.yazi.tools set: declare and install those tools through mise (it must
# include yazi itself). Otherwise: Homebrew on macOS, else the upstream release
# zip into ~/.local/share/yazi (module-owned) with yazi and ya linked from
# ~/.local/bin.
set -euo pipefail

_home="${DOTFILES_HOME:-$HOME}"
_bin="${_home}/.local/bin"
_share="${_home}/.local/share/yazi"
_shims="${_home}/.local/share/mise/shims"

# _drop_release: remove this module's release install (the share dir and the
# ~/.local/bin links into it), so mise is the only owner.
_drop_release() {
    [[ -d "$_share" ]] || return 0
    local b
    for b in yazi ya; do
        if [[ -L "${_bin}/${b}" && "$(readlink "${_bin}/${b}")" == "${_share}/"* ]]; then
            rm -f "${_bin}/${b}"
        fi
    done
    rm -rf "$_share"
    log_info "yazi: removed the release install in ${_share} (mise owns yazi now)"
}

# _find BIN: a RUNNABLE BIN on PATH or in ~/.local/bin, skipping mise shims
# (a shim left behind by a removed mise declaration fails with "No version is set").
_find() {
    local p
    while IFS= read -r p; do
        [[ "$p" == "${_shims}/"* ]] && continue
        "$p" --version >/dev/null 2>&1 && { printf '%s' "$p"; return 0; }
    done < <(type -ap "$1" 2>/dev/null || true; [[ -x "${_bin}/$1" ]] && printf '%s\n' "${_bin}/$1")
    return 1
}
_ver_of() { "$1" --version 2>/dev/null | grep -m1 -oE '[0-9]+\.[0-9]+\.[0-9]+' || true; }

if [[ -n "${DOTFILES_SETTING_TOOLS:-}" ]]; then
    # The map may carry preview helpers (glow, 7zip), but yazi itself must be in it:
    # `yazi`, or a backend spec such as github:sxyazi/yazi[matching=musl].
    # (here-string, not a pipe: under pipefail `printf | grep -q` can fail on SIGPIPE)
    if ! grep -qE '^([^=]*[:/])?yazi(\[[^]]*\])?=' <<< "$DOTFILES_SETTING_TOOLS"; then
        log_error "yazi: modules.yazi.tools must declare yazi itself (e.g. yazi: \"26.9.1\")"
        exit 1
    fi
    mise_sync_tools "$DOTFILES_MODULE_NAME"
    is_dry_run && exit 0
    # With a lockfile, mise skips a tool that has no entry for this platform (e.g.
    # a musl-only lock on macOS). Then fall through to Homebrew / the release zip.
    if "${_shims}/yazi" --version >/dev/null 2>&1 && "${_shims}/ya" --version >/dev/null 2>&1; then
        _drop_release
        exit 0
    fi
    log_warn "yazi: mise provides no yazi on this platform; falling back to Homebrew / the release binary"
else
    # No tools declared: drop a fragment from an earlier mise-managed run, so mise
    # stops owning yazi alongside the install below (two owners).
    mise_sync_tools "$DOTFILES_MODULE_NAME"
fi
if [[ -x "${_shims}/yazi" ]] && ! "${_shims}/yazi" --version >/dev/null 2>&1; then
    log_warn "yazi: a stale mise shim (${_shims}/yazi) shadows this install in shells that put the shims first; remove it with \`mise uninstall\` (the yazi entry \`mise ls\` shows)"
fi

_want="${DOTFILES_YAZI_VERSION:-}"; _want="${_want#v}"
if _have="$(_find yazi)" && _find ya >/dev/null; then
    _hv="$(_ver_of "$_have")"
    if [[ -z "$_want" || "$_hv" == "$_want" ]]; then
        log_info "yazi already installed: ${_hv} (${_have})"
        exit 0
    fi
    if [[ "$(readlink -f "$_have")" != "${_share}/"* ]]; then
        log_warn "yazi ${_hv} at ${_have} is not this module's install; leaving it (DOTFILES_YAZI_VERSION=${_want} not applied)"
        exit 0
    fi
    log_info "yazi ${_hv} installed, DOTFILES_YAZI_VERSION wants ${_want}: reinstalling"
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
if [[ -n "$_want" ]]; then
    _tag="v${_want}"
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

# yazi publishes no checksums file; the API reports each asset's sha256 digest
# (GITHUB_TOKEN, when set, lifts the 60 req/h anonymous limit). Fail closed
# without one unless DOTFILES_ALLOW_UNVERIFIED=1.
_auth=()
[[ -n "${GITHUB_TOKEN:-}" ]] && _auth=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
_sum="$(curl -fsSL ${_auth[@]+"${_auth[@]}"} "https://api.github.com/repos/sxyazi/yazi/releases/tags/${_tag}" 2>/dev/null \
    | awk -v f="$_asset" '/"name":/ {n = ($0 ~ "\"" f "\"")} n && /"digest":/ {print; exit}' \
    | grep -oE '[a-f0-9]{64}' || true)"
if [[ -z "$_sum" ]]; then
    if [[ "${DOTFILES_ALLOW_UNVERIFIED:-}" == 1 ]]; then
        log_warn "yazi: no published digest for ${_asset} @ ${_tag} — installing unverified (DOTFILES_ALLOW_UNVERIFIED=1)"
    else
        log_error "yazi: no sha256 digest for ${_asset} @ ${_tag} (GitHub API rate-limited or unavailable). Set GITHUB_TOKEN, declare modules.yazi.tools for a locked mise install, or set DOTFILES_ALLOW_UNVERIFIED=1"
        exit 1
    fi
fi

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

_yv="$(_ver_of "${_bin}/yazi")"
if [[ -z "$_yv" ]]; then
    log_error "yazi: installed but not runnable (${_asset} may be wrong for this host)"
    exit 1
fi
log_success "yazi installed: ${_yv}"
