#!/usr/bin/env bash
# yazi/verify.sh - Verify yazi and ya run (mise shims, PATH or ~/.local/bin).
set -euo pipefail

_home="${DOTFILES_HOME:-$HOME}"
_yazi=""
for _b in yazi ya; do
    _p="$(command -v "$_b" 2>/dev/null || true)"
    [[ -z "$_p" && -x "${_home}/.local/bin/${_b}" ]] && _p="${_home}/.local/bin/${_b}"
    [[ -z "$_p" && -x "${_home}/.local/share/mise/shims/${_b}" ]] && _p="${_home}/.local/share/mise/shims/${_b}"
    if [[ -z "$_p" ]] || ! "$_p" --version >/dev/null 2>&1; then
        log_error "${_b} not found or not runnable"
        exit 1
    fi
    [[ "$_b" == yazi ]] && _yazi="$_p"
done
log_success "yazi is installed: $("$_yazi" --version 2>/dev/null | grep -m1 -oE '[0-9]+\.[0-9]+\.[0-9]+' || true)"
