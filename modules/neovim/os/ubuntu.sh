#!/usr/bin/env bash
# neovim/os/ubuntu.sh - Install Neovim on Ubuntu

if command -v nvim &>/dev/null; then
    log_info "Neovim is already installed on Ubuntu"
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would add Neovim PPA and install via apt"
    return 0
fi

log_info "Adding Neovim PPA for latest version..."
sudo_cmd add-apt-repository -y ppa:neovim-ppa/unstable 2>/dev/null || \
    log_warn "Failed to add Neovim PPA, falling back to default apt package"
sudo_cmd apt-get update -qq
sudo_cmd apt-get install -y neovim
log_success "Neovim installed on Ubuntu"
