#!/usr/bin/env bash
# ghostty/os/ubuntu.sh - Pre-install for Ghostty on Ubuntu (snap)

# Ensure snapd is installed
if ! command -v snap &>/dev/null; then
    log_info "Installing snapd..."
    pkg_install snapd
fi
