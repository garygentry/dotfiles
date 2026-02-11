#!/usr/bin/env bash
# azure-cli/os/ubuntu.sh - Pre-install for Azure CLI on Ubuntu

# The official install script handles repo setup, but we need curl
if ! command -v curl &>/dev/null; then
    log_info "Installing curl (required for Azure CLI)..."
    pkg_install curl
fi
