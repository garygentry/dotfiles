#!/usr/bin/env bash
set -euo pipefail

# Which Nerd Font to install is configurable (default FiraCode) so the engine
# imposes no particular font. Override from your content overlay:
#   modules.fonts.font      -> DOTFILES_SETTING_FONT      (family name, e.g. JetBrainsMono)
#   modules.fonts.font_url  -> DOTFILES_SETTING_FONT_URL  (direct .ttf URL, apt/Linux path)
_font="${DOTFILES_SETTING_FONT:-FiraCode}"

# Derive the Homebrew cask name from the family: FiraCode -> font-fira-code-nerd-font
_font_cask="font-$(printf '%s' "$_font" | sed -E 's/([a-z0-9])([A-Z])/\1-\2/g' | tr '[:upper:]' '[:lower:]')-nerd-font"

# Derive the nerd-fonts release .ttf path (used for the apt/Linux install), with
# an escape hatch for fonts whose patched-font path doesn't follow the pattern.
_font_url="${DOTFILES_SETTING_FONT_URL:-https://github.com/ryanoasis/nerd-fonts/raw/HEAD/patched-fonts/${_font}/Regular/${_font}NerdFont-Regular.ttf}"

log_info "Installing Nerd Font: ${_font}..."

if [ "$DOTFILES_OS" = "macos" ] && command -v brew >/dev/null 2>&1; then
    brew tap homebrew/cask-fonts 2>/dev/null || true
    brew install --cask "$_font_cask"
elif [ "$DOTFILES_PKG_MGR" = "apt" ]; then
    mkdir -p ~/.local/share/fonts
    cd ~/.local/share/fonts
    curl -fLo "${_font} Nerd Font.ttf" "$_font_url"
    fc-cache -fv
fi

log_success "Nerd Font installed: ${_font}"
