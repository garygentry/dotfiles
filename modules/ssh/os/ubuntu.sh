#!/usr/bin/env bash
# ssh/os/ubuntu.sh - Ubuntu-specific SSH setup

# Ensure openssh-client is installed
if ! command -v ssh &>/dev/null; then
    pkg_install openssh-client
else
    log_info "openssh-client is already installed"
fi

# Ensure ssh-agent is running
if ! pgrep -x ssh-agent &>/dev/null; then
    if is_dry_run; then
        log_info "[dry-run] Would start ssh-agent on Ubuntu"
    else
        eval "$(ssh-agent -s)" &>/dev/null
        log_success "Started ssh-agent on Ubuntu"
    fi
else
    log_info "ssh-agent is already running on Ubuntu"
fi
