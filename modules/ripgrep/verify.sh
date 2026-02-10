#!/usr/bin/env bash
set -euo pipefail

log_info "Verifying ripgrep installation..."

if ! command -v rg >/dev/null 2>&1; then
    log_error "ripgrep not found in PATH"
    exit 1
fi

log_success "ripgrep verification passed"
