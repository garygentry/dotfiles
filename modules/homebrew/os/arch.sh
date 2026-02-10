#!/usr/bin/env bash
# homebrew/os/arch.sh - Install Homebrew on Arch Linux

set -euo pipefail

log_info "Installing Homebrew on Arch Linux..."

# Check if Homebrew is already installed
if command -v brew >/dev/null 2>&1; then
    log_info "Homebrew is already installed"
    brew --version
    exit 0
fi

# Install required dependencies
log_info "Installing Homebrew dependencies..."
sudo pacman -Sy --needed --noconfirm base-devel procps-ng curl file git

# Install Homebrew
log_info "Installing Homebrew (this may take a few minutes)..."
NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Add Homebrew to PATH for the current session
if [ -x "/home/linuxbrew/.linuxbrew/bin/brew" ]; then
    eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
fi

log_success "Homebrew installed on Arch Linux"
