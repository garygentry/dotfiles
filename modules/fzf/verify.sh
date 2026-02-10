#!/usr/bin/env bash
set -euo pipefail
command -v fzf >/dev/null 2>&1 || { log_error "fzf not found"; exit 1; }
log_success "fzf verification passed"
