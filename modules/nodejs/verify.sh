#!/usr/bin/env bash
# nodejs/verify.sh - Verify Node.js installation
set -euo pipefail

_bin_dir="${DOTFILES_HOME:-$HOME}/.local/bin"
_node="$(command -v node 2>/dev/null || { [[ -x "${_bin_dir}/node" ]] && echo "${_bin_dir}/node"; } || true)"
_npm="$(command -v npm 2>/dev/null || { [[ -x "${_bin_dir}/npm" ]] && echo "${_bin_dir}/npm"; } || true)"

if [[ -z "$_node" ]]; then
    log_error "node not found"
    exit 1
fi
if [[ -z "$_npm" ]]; then
    log_error "npm not found"
    exit 1
fi

log_success "Node.js verification passed: $("$_node" --version 2>/dev/null) (npm $("$_npm" --version 2>/dev/null))"
