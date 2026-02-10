#!/usr/bin/env bash
set -euo pipefail

log_info "Installing Nerd Fonts..."

# Install FiraCode Nerd Font
if [ "$DOTFILES_OS" = "macos" ] && command -v brew >/dev/null 2>&1; then
    brew tap homebrew/cask-fonts
    brew install --cask font-fira-code-nerd-font
elif [ "$DOTFILES_PKG_MGR" = "apt" ]; then
    mkdir -p ~/.local/share/fonts
    cd ~/.local/share/fonts
    curl -fLo "FiraCode Nerd Font.ttf" https://github.com/ryanoasis/nerd-fonts/raw/HEAD/patched-fonts/FiraCode/Regular/FiraCodeNerdFont-Regular.ttf
    fc-cache -fv
fi

log_success "Nerd Fonts installed"
