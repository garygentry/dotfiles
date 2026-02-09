# zsh/os/macos.sh - macOS-specific Zsh setup

# macOS ships with zsh as the default shell since Catalina
if command -v zsh &>/dev/null; then
    log_info "Zsh is already available on macOS (built-in)"
else
    log_warn "Zsh not found on macOS, installing via Homebrew..."
    pkg_install zsh
fi
