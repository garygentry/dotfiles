# neovim/os/arch.sh - Install Neovim on Arch Linux

if command -v nvim &>/dev/null; then
    log_info "Neovim is already installed on Arch"
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install Neovim via: pacman -S neovim"
    return 0
fi

log_info "Installing Neovim on Arch Linux..."
if has_sudo; then
    sudo pacman -S --noconfirm neovim
    log_success "Neovim installed on Arch"
else
    log_error "sudo is required to install Neovim on Arch"
    return 1
fi
