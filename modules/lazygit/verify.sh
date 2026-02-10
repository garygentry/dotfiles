#!/usr/bin/env bash
set -euo pipefail
command -v lazygit >/dev/null 2>&1 || { log_error "lazygit not found"; exit 1; }
log_success "lazygit verification passed"
