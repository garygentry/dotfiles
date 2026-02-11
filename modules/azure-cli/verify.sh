#!/usr/bin/env bash
# azure-cli/verify.sh - Verify Azure CLI installation

if command -v az &>/dev/null; then
    _az_version="$(az version --output tsv 2>/dev/null | head -1)"
    log_success "Azure CLI is installed: ${_az_version}"
else
    log_error "Azure CLI is not installed"
    return 1
fi
