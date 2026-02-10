#!/usr/bin/env bash
# git/os/arch.sh - Arch Linux-specific git setup

# Ensure git is installed
if ! command -v git &>/dev/null; then
    pkg_install git
else
    log_info "Git is already installed on Arch"
fi

# Configure credential cache for Arch (15-minute timeout)
if is_dry_run; then
    log_info "[dry-run] Would configure Arch git credential cache"
else
    git config --global credential.helper "cache --timeout=900"
    log_info "Configured Arch credential cache (15-minute timeout)"
fi
