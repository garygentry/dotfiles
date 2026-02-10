#!/usr/bin/env bash
set -euo pipefail
command -v go >/dev/null 2>&1 || { log_error "go not found"; exit 1; }
[ -d ~/go ] || log_warn "GOPATH directory not found"
log_success "Go verification passed"
