#!/usr/bin/env bash
# rust/verify.sh - Verify Rust installation

# Ensure cargo/rustc are on PATH for this check
# shellcheck disable=SC1091
[[ -f "${HOME}/.cargo/env" ]] && source "${HOME}/.cargo/env"

if command -v rustc &>/dev/null && command -v cargo &>/dev/null; then
    _rustc_version="$(rustc --version 2>/dev/null)"
    log_success "Rust is installed: ${_rustc_version}"
else
    log_error "Rust is not installed (rustc or cargo not found)"
    return 1
fi
