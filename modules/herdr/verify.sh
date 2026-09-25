#!/usr/bin/env bash
# herdr/verify.sh - Verify the herdr installation.

_herdr_home="${DOTFILES_HOME:-$HOME}"
_herdr_bin="$(command -v herdr 2>/dev/null || true)"
[[ -z "$_herdr_bin" && -x "${_herdr_home}/.local/bin/herdr" ]] && _herdr_bin="${_herdr_home}/.local/bin/herdr"

if [[ -z "$_herdr_bin" ]]; then
    log_error "herdr not found"
    return 1 2>/dev/null || exit 1
fi

log_success "herdr is installed: $("$_herdr_bin" --version 2>/dev/null)"
