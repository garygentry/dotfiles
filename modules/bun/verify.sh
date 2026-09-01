#!/usr/bin/env bash
# bun/verify.sh - Verify Bun installation
set -euo pipefail

_bin_dir="${DOTFILES_HOME:-$HOME}/.local/bin"
_bun="$(command -v bun 2>/dev/null || { [[ -x "${_bin_dir}/bun" ]] && echo "${_bin_dir}/bun"; } || true)"

if [[ -z "$_bun" ]]; then
    log_error "bun not found"
    exit 1
fi

log_success "Bun verification passed: $("$_bun" --version 2>/dev/null)"
