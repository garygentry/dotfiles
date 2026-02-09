# zsh/os/arch.sh - Arch Linux-specific Zsh setup

if ! command -v zsh &>/dev/null; then
    log_info "Installing Zsh on Arch Linux..."
    pkg_install zsh
else
    log_info "Zsh is already installed on Arch"
fi
