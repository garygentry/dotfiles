#!/usr/bin/env bash
# zsh/install.sh - Install and configure Zsh with Zinit plugin manager

# Install zsh if not already present (OS-specific scripts handle this too)
if ! command -v zsh &>/dev/null; then
    log_info "Zsh not found, installing..."
    pkg_install zsh
else
    log_info "Zsh is already installed"
fi

# Create zsh config directory
_zsh_config_dir="${DOTFILES_HOME}/.config/zsh"
if [[ ! -d "$_zsh_config_dir" ]]; then
    if is_dry_run; then
        log_info "[dry-run] Would create directory: ${_zsh_config_dir}"
    else
        mkdir -p "$_zsh_config_dir"
        log_success "Created ${_zsh_config_dir}"
    fi
fi

# Install Zinit plugin manager
_zsh_zinit_home="${DOTFILES_HOME}/.local/share/zinit/zinit.git"
if [[ ! -d "$_zsh_zinit_home" ]]; then
    if is_dry_run; then
        log_info "[dry-run] Would install Zinit plugin manager to ${_zsh_zinit_home}"
    else
        log_info "Installing Zinit plugin manager..."
        mkdir -p "$(dirname "$_zsh_zinit_home")"
        git clone https://github.com/zdharma-continuum/zinit.git "$_zsh_zinit_home"
        log_success "Zinit installed to ${_zsh_zinit_home}"
    fi
else
    log_info "Zinit is already installed at ${_zsh_zinit_home}"
fi

# Set default shell to zsh if not already
_zsh_current_shell="$(basename "${SHELL:-}")"
if [[ "$_zsh_current_shell" != "zsh" ]]; then
    _zsh_path="$(command -v zsh)"
    if [[ -n "$_zsh_path" ]]; then
        if is_dry_run; then
            log_info "[dry-run] Would change default shell to ${_zsh_path}"
        else
            # Ensure zsh is in /etc/shells
            if ! grep -qF "$_zsh_path" /etc/shells 2>/dev/null; then
                if has_sudo; then
                    echo "$_zsh_path" | sudo tee -a /etc/shells >/dev/null
                    log_info "Added ${_zsh_path} to /etc/shells"
                else
                    log_warn "Cannot add ${_zsh_path} to /etc/shells without sudo"
                fi
            fi
            if chsh -s "$_zsh_path"; then
                log_success "Default shell changed to zsh"
            else
                log_warn "Failed to change default shell to zsh (chsh exited non-zero)"
                log_warn "You can change it manually with: chsh -s $_zsh_path"
            fi
        fi
    fi
else
    log_info "Default shell is already zsh"
fi
