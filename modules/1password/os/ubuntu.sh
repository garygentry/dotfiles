#!/usr/bin/env bash
# 1password/os/ubuntu.sh - Install 1Password CLI on Ubuntu via apt

if command -v op &>/dev/null; then
    log_info "1Password CLI already installed on Ubuntu"
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install 1Password CLI via apt on Ubuntu"
    return 0
fi

# Every step below needs root (apt repo + package install). Degrade cleanly on a
# host without usable sudo instead of hard-failing the whole profile, matching
# gh/neovim. install.sh then warns that op is missing and points at manual setup.
require_sudo "1Password CLI (apt install)" || return 0

log_info "Adding 1Password apt repository..."

# Add the 1Password GPG key
sudo_cmd mkdir -p /usr/share/keyrings
curl -sS https://downloads.1password.com/linux/keys/1password.asc \
    | sudo_cmd gpg --dearmor --output /usr/share/keyrings/1password-archive-keyring.gpg 2>/dev/null

# Add the 1Password apt repository
echo "deb [arch=amd64 signed-by=/usr/share/keyrings/1password-archive-keyring.gpg] https://downloads.1password.com/linux/debian/amd64 stable main" \
    | sudo_cmd tee /etc/apt/sources.list.d/1password.list >/dev/null

# Set up debsig-verify policy
sudo_cmd mkdir -p /etc/debsig/policies/AC2D62742012EA22/
curl -sS https://downloads.1password.com/linux/debian/debsig/1password.pol \
    | sudo_cmd tee /etc/debsig/policies/AC2D62742012EA22/1password.pol >/dev/null
sudo_cmd mkdir -p /usr/share/debsig/keyrings/AC2D62742012EA22
curl -sS https://downloads.1password.com/linux/keys/1password.asc \
    | sudo_cmd gpg --dearmor --output /usr/share/debsig/keyrings/AC2D62742012EA22/debsig.gpg 2>/dev/null

# Install 1Password CLI
sudo_cmd apt-get update -qq
sudo_cmd apt-get install -y 1password-cli
log_success "1Password CLI installed on Ubuntu"
