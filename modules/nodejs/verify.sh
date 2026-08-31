#!/usr/bin/env bash
set -euo pipefail
if ! command -v node >/dev/null 2>&1 || ! command -v npm >/dev/null 2>&1; then
    if ! sudo_usable; then
        log_warn "Node.js not installed — skipped (no usable sudo); not a failure"
        exit 0
    fi
    command -v node >/dev/null 2>&1 || { log_error "node not found"; exit 1; }
    command -v npm >/dev/null 2>&1 || { log_error "npm not found"; exit 1; }
fi
log_success "Node.js verification passed"
