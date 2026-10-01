#!/usr/bin/env bash
# yazi/verify.sh - Verify yazi and ya run (PATH, ~/.local/bin or mise shims).
set -euo pipefail

_home="${DOTFILES_HOME:-$HOME}"
# _runnable BIN: the first candidate that actually runs (a stale mise shim doesn't).
_runnable() {
    local p
    while IFS= read -r p; do
        [[ -n "$p" ]] && "$p" --version >/dev/null 2>&1 && { printf '%s' "$p"; return 0; }
    done < <(type -ap "$1" 2>/dev/null || true; printf '%s\n' "${_home}/.local/bin/$1" "${_home}/.local/share/mise/shims/$1")
    return 1
}
_yazi="$(_runnable yazi)" || { log_error "yazi not found or not runnable"; exit 1; }
_runnable ya >/dev/null || { log_error "ya not found or not runnable"; exit 1; }
log_success "yazi is installed: $("$_yazi" --version 2>/dev/null | grep -m1 -oE '[0-9]+\.[0-9]+\.[0-9]+' || true)"
