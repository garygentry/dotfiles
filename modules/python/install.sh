#!/usr/bin/env bash
set -euo pipefail

log_info "Installing Python..."

# python3 is usually preinstalled on Linux, but pip3 frequently is NOT, and
# verify.sh requires both. pkg_install is idempotent (already-present packages
# are skipped), so this is safe to run unconditionally.
pkg_install python3 python3-pip

if command -v python3 >/dev/null 2>&1; then
    log_info "Python: $(python3 --version 2>&1)"
fi

log_success "Python installed"
