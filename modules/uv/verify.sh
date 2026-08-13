#!/usr/bin/env bash
set -euo pipefail

_uv_bin_dir="${DOTFILES_HOME}/.local/bin"

_uv_path="$(command -v uv 2>/dev/null || true)"
if [[ -z "$_uv_path" && -x "${_uv_bin_dir}/uv" ]]; then
    _uv_path="${_uv_bin_dir}/uv"
fi

if [[ -z "$_uv_path" ]]; then
    log_error "uv not found on PATH or in ${_uv_bin_dir}"
    exit 1
fi

# uvx ships alongside uv; its absence means a partial install rather than a working one.
if [[ ! -x "$(dirname "$_uv_path")/uvx" ]] && ! command -v uvx &>/dev/null; then
    log_error "uv found at ${_uv_path} but uvx is missing"
    exit 1
fi

log_success "uv verification passed ($("$_uv_path" --version))"
