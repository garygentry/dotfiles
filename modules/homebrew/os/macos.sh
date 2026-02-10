#!/usr/bin/env bash
# homebrew/os/macos.sh - Install Homebrew on macOS

set -euo pipefail

log_info "Installing Homebrew on macOS..."

# Check if Homebrew is already installed
if command -v brew >/dev/null 2>&1; then
    log_info "Homebrew is already installed"
    brew --version
    exit 0
fi

# Install Homebrew
log_info "Installing Homebrew (this may take a few minutes)..."

# Run the official Homebrew install script
NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Add Homebrew to PATH for the current session
if [ -x "/opt/homebrew/bin/brew" ]; then
    # Apple Silicon
    eval "$(/opt/homebrew/bin/brew shellenv)"
elif [ -x "/usr/local/bin/brew" ]; then
    # Intel Mac
    eval "$(/usr/local/bin/brew shellenv)"
fi

log_success "Homebrew installed successfully"
