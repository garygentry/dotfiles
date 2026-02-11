#!/usr/bin/env bash
# gcloud/verify.sh - Verify Google Cloud CLI installation

if command -v gcloud &>/dev/null; then
    _gcloud_version="$(gcloud version 2>/dev/null | head -1)"
    log_success "Google Cloud CLI is installed: ${_gcloud_version}"
else
    log_error "Google Cloud CLI is not installed"
    return 1
fi
