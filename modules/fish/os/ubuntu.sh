#!/usr/bin/env bash
# fish/os/ubuntu.sh - Add Fish shell PPA on Ubuntu

if apt-cache policy fish 2>/dev/null | grep -q "fish-shell"; then
    log_info "Fish shell PPA already configured"
    return 0
fi

log_info "Adding Fish shell PPA..."
sudo_cmd apt-add-repository -y ppa:fish-shell/release-4
sudo_cmd apt-get update -qq
