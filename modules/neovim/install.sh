#!/usr/bin/env bash
# neovim/install.sh - Install Neovim

# Install neovim if not already present (OS-specific scripts handle the
# platform-specific installation method; this is a fallback using pkg_install)
if ! command -v nvim &>/dev/null; then
    log_info "Neovim not found, installing..."
    pkg_install neovim
else
    log_info "Neovim is already installed"
fi

# Install ripgrep (required by Telescope live_grep)
if ! command -v rg &>/dev/null; then
    log_info "ripgrep not found, installing..."
    pkg_install ripgrep
else
    log_info "ripgrep is already installed"
fi

# Ensure the nvim config directory exists
_nvim_config="${DOTFILES_XDG_CONFIG_HOME}/nvim"
if [[ ! -d "$_nvim_config" ]]; then
    if is_dry_run; then
        log_info "[dry-run] Would create directory: ${_nvim_config}"
    else
        mkdir -p "$_nvim_config"
        log_success "Created ${_nvim_config}"
    fi
fi
