#!/usr/bin/env bash
# ssh/os/arch.sh - Arch Linux-specific SSH setup

# Ensure openssh is installed
if ! command -v ssh &>/dev/null; then
    pkg_install openssh
else
    log_info "openssh is already installed"
fi

# Ensure ssh-agent is running
if ! pgrep -x ssh-agent &>/dev/null; then
    if is_dry_run; then
        log_info "[dry-run] Would start ssh-agent on Arch"
    else
        eval "$(ssh-agent -s)" &>/dev/null
        log_success "Started ssh-agent on Arch"
    fi
else
    log_info "ssh-agent is already running on Arch"
fi
