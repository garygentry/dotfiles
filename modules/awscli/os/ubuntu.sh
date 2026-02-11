#!/usr/bin/env bash
# awscli/os/ubuntu.sh - Ensure unzip is available for AWS CLI installer

if ! command -v unzip &>/dev/null; then
    log_info "Installing unzip (required for AWS CLI)..."
    pkg_install unzip
fi
