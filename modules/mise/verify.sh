#!/usr/bin/env bash
# mise/verify.sh - Verify mise, the declared tools, and shim reachability
set -euo pipefail

_home="${DOTFILES_HOME:-$HOME}"
_cfg_dir="${DOTFILES_XDG_CONFIG_HOME:-${_home}/.config}/mise"
_conf="${_cfg_dir}/conf.d/dotfiles.toml"
_want="${DOTFILES_SETTING_VERSION:-}"
_want="${_want#v}"
_errors=0

_mise="$(command -v mise 2>/dev/null || true)"
[[ -z "$_mise" && -x "${_home}/.local/bin/mise" ]] && _mise="${_home}/.local/bin/mise"
if [[ -z "$_mise" ]]; then
    log_error "mise not found"
    exit 1
fi
_ver="$("$_mise" --version 2>/dev/null | awk '{print $1}' || true)"
if [[ -z "$_ver" ]]; then
    log_error "mise is present but not runnable (${_mise})"
    exit 1
fi
if [[ -n "$_want" && "$_mise" == "${_home}/.local/bin/mise" && "$_ver" != "$_want" ]]; then
    log_error "mise ${_ver} at ${_mise} is not the pinned ${_want}"
    _errors=$((_errors + 1))
else
    log_success "mise ${_ver} (${_mise})"
fi

# Every declared tool is installed at exactly the declared version.
if [[ -f "$_conf" ]]; then
    _n=0
    while IFS= read -r _t; do
        [[ -z "$_t" ]] && continue
        _n=$((_n + 1))
        if "$_mise" where "$_t" >/dev/null 2>&1; then
            :
        else
            log_error "declared tool not installed: ${_t} (run: mise install ${_t})"
            _errors=$((_errors + 1))
        fi
    done < <(awk '
        /^\[/ { in_tools = ($0 == "[tools]"); next }
        in_tools && /^"[^"]+" = "[^"]*"$/ {
            split($0, kv, /" = "/); name = substr(kv[1], 2); ver = kv[2]; sub(/"$/, "", ver)
            print name "@" ver
        }' "$_conf")
    [[ $_n -gt 0 ]] && log_success "${_n} declared tool(s) checked"
fi

# The managed lockfile matches its source.
if [[ -n "${DOTFILES_SETTING_LOCKFILE:-}" ]]; then
    _lock_src="$DOTFILES_SETTING_LOCKFILE"
    [[ "$_lock_src" != /* ]] && _lock_src="${DOTFILES_CONTENT_DIR:-${DOTFILES_DIR}}/${_lock_src}"
    if cmp -s "$_lock_src" "${_cfg_dir}/mise.lock"; then
        log_success "mise.lock matches ${_lock_src}"
    else
        log_error "${_cfg_dir}/mise.lock differs from ${_lock_src} (re-run the install, or re-lock)"
        _errors=$((_errors + 1))
    fi
fi

# Shims reachable where rc-file activation never runs.
_shims="${XDG_DATA_HOME:-${_home}/.local/share}/mise/shims"
if command -v zsh >/dev/null 2>&1; then
    if HOME="$_home" zsh -c 'print -rl -- $path' 2>/dev/null | grep -qxF "$_shims"; then
        log_success "non-interactive zsh has the mise shims on PATH"
    else
        log_error "non-interactive zsh lacks ${_shims} on PATH (check the managed block in ~/.zshenv)"
        _errors=$((_errors + 1))
    fi
fi
if HOME="$_home" sh -lc 'printf "%s\n" "$PATH"' 2>/dev/null | tr ':' '\n' | grep -qxF "$_shims"; then
    log_success "login sh (sh -lc) has the mise shims on PATH"
else
    log_warn "login sh (sh -lc) lacks ${_shims} on PATH (check the managed block in the login profile)"
fi

if [[ $_errors -gt 0 ]]; then
    log_error "mise verification failed with ${_errors} error(s)"
    exit 1
fi
log_success "mise verification passed"
