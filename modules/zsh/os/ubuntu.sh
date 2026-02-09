# zsh/os/ubuntu.sh - Ubuntu-specific Zsh setup

if ! command -v zsh &>/dev/null; then
    log_info "Installing Zsh on Ubuntu..."
    pkg_install zsh
else
    log_info "Zsh is already installed on Ubuntu"
fi
