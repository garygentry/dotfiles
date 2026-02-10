#!/usr/bin/env bash
set -euo pipefail
command -v node >/dev/null 2>&1 || { log_error "node not found"; exit 1; }
command -v npm >/dev/null 2>&1 || { log_error "npm not found"; exit 1; }
log_success "Node.js verification passed"
