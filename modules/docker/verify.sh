#!/usr/bin/env bash
set -euo pipefail
command -v docker >/dev/null 2>&1 || { log_error "docker not found"; exit 1; }
groups | grep -q docker || log_warn "User not in docker group"
log_success "Docker verification passed"
