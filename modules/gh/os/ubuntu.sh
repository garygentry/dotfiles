#!/usr/bin/env bash
# gh/os/ubuntu.sh - Set up GitHub CLI apt repository on Ubuntu

if apt-cache show gh &>/dev/null 2>&1; then
    log_info "GitHub CLI apt repo already configured"
    return 0
fi

log_info "Adding GitHub CLI apt repository..."

# Add GitHub CLI keyring
sudo mkdir -p -m 755 /etc/apt/keyrings
curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
    | sudo tee /etc/apt/keyrings/githubcli-archive-keyring.gpg > /dev/null
sudo chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg

# Add apt source
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
    | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null

sudo apt-get update -qq
