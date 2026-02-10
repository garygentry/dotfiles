#!/usr/bin/env bash
# homebrew/verify.sh - Verify Homebrew installation

set -euo pipefail

log_info "Verifying Homebrew installation..."

# Check if brew command exists
if ! command -v brew >/dev/null 2>&1; then
    log_error "Homebrew not found in PATH"
    exit 1
fi

# Check brew version
BREW_VERSION=$(brew --version | head -n1)
log_info "Found: $BREW_VERSION"

# Check if brew doctor reports any issues
log_info "Running brew doctor..."
if brew doctor >/dev/null 2>&1; then
    log_success "Homebrew is healthy"
else
    log_warn "Homebrew doctor found some issues (non-critical)"
fi

log_success "Homebrew verification passed"
