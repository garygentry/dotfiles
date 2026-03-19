#!/usr/bin/env bash
set -euo pipefail

log_info "Installing tmux..."

pkg_install tmux

# Install TPM (Tmux Plugin Manager)
if [ ! -d ~/.tmux/plugins/tpm ]; then
    git clone https://github.com/tmux-plugins/tpm ~/.tmux/plugins/tpm
    log_info "Installed TPM (Tmux Plugin Manager)"
else
    log_info "TPM already installed, skipping"
fi

# Install catppuccin/tmux manually (used via 'run' in tmux.conf, not TPM)
_catppuccin_dir="${DOTFILES_XDG_CONFIG_HOME}/tmux/plugins/catppuccin/tmux"
if [ ! -d "$_catppuccin_dir" ]; then
    mkdir -p "${DOTFILES_XDG_CONFIG_HOME}/tmux/plugins/catppuccin"
    git clone -b v2.1.3 --depth 1 \
        https://github.com/catppuccin/tmux.git \
        "$_catppuccin_dir"
    log_info "Installed catppuccin/tmux (v2.1.3)"
else
    log_info "catppuccin/tmux already installed, skipping"
fi

# Deploy tmux configuration
_tmux_config="${DOTFILES_PROMPT_TMUX_CONFIG:-skip}"

if [[ "$_tmux_config" == "opinionated" ]]; then
    if is_dry_run; then
        log_info "[dry-run] Would deploy opinionated tmux config to ~/.config/tmux/tmux.conf"
        log_info "[dry-run] Would install TPM plugins headlessly"
    else
        # Back up any existing XDG config before overwriting
        _backup_file "${DOTFILES_XDG_CONFIG_HOME}/tmux/tmux.conf"

        # Deploy to XDG path
        mkdir -p "${DOTFILES_XDG_CONFIG_HOME}/tmux"
        cp "${DOTFILES_MODULE_DIR}/tmux.conf" "${DOTFILES_XDG_CONFIG_HOME}/tmux/tmux.conf"
        log_success "Deployed opinionated tmux config to ${DOTFILES_XDG_CONFIG_HOME}/tmux/tmux.conf"

        # Deploy cheatsheet
        cp "${DOTFILES_MODULE_DIR}/tmux-cheatsheet.md" "${DOTFILES_XDG_CONFIG_HOME}/tmux/tmux-cheatsheet.md"
        log_success "Deployed tmux cheatsheet to ${DOTFILES_XDG_CONFIG_HOME}/tmux/tmux-cheatsheet.md"

        # Install TPM plugins headlessly (runs tpm's install script without an active session)
        log_info "Installing TPM plugins headlessly..."
        if "${HOME}/.tmux/plugins/tpm/bin/install_plugins" 2>&1 | grep -qiE "(error|failed)"; then
            log_warn "TPM plugin install may have encountered issues — run 'Prefix + I' inside tmux to retry"
        else
            log_success "TPM plugins installed"
        fi
    fi
else
    log_info "Skipping tmux config (preset: skip)"
fi

log_success "tmux installed"