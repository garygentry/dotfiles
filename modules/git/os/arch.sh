#!/usr/bin/env bash
# git/os/arch.sh - Arch Linux-specific git setup

# Ensure git is installed
if ! command -v git &>/dev/null; then
    pkg_install git
else
    log_info "Git is already installed on Arch"
fi
