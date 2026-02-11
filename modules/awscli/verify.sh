#!/usr/bin/env bash
# awscli/verify.sh - Verify AWS CLI installation

if command -v aws &>/dev/null; then
    _aws_version="$(aws --version 2>/dev/null)"
    log_success "AWS CLI is installed: ${_aws_version}"
else
    log_error "AWS CLI is not installed"
    return 1
fi
