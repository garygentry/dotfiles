#!/usr/bin/env bash
set -euo pipefail
command -v python3 >/dev/null 2>&1 || { log_error "python3 not found"; exit 1; }
command -v pip3 >/dev/null 2>&1 || { log_error "pip3 not found"; exit 1; }
log_success "Python verification passed"
