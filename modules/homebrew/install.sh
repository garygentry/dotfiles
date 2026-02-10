#!/usr/bin/env bash
# homebrew/install.sh - Install and configure Homebrew

set -euo pipefail

log_info "Configuring Homebrew..."

# Verify Homebrew is installed (done by OS-specific scripts)
if ! command -v brew >/dev/null 2>&1; then
    log_error "Homebrew not found. This should have been installed by the OS-specific script."
    exit 1
fi

# Update Homebrew
log_info "Updating Homebrew..."
brew update

# Configure shell integration
BREW_PREFIX="$(brew --prefix)"

# Add Homebrew to PATH in shell rc files
if [ "$DOTFILES_OS" = "macos" ]; then
    # On macOS, Homebrew is typically at /usr/local or /opt/homebrew
    log_info "Homebrew prefix: $BREW_PREFIX"
fi

log_success "Homebrew configured successfully"
