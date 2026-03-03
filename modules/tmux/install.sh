#!/usr/bin/env bash
set -euo pipefail

log_info "Installing tmux..."

pkg_install tmux

# Install TPM (Tmux Plugin Manager)
if [ ! -d ~/.tmux/plugins/tpm ]; then
    git clone https://github.com/tmux-plugins/tpm ~/.tmux/plugins/tpm
    log_info "Installed TPM (Tmux Plugin Manager)"
fi

# Deploy tmux configuration
_tmux_config="${DOTFILES_PROMPT_TMUX_CONFIG:-none}"

if [[ "$_tmux_config" == "custom" ]]; then
    if is_dry_run; then
        log_info "[dry-run] Would deploy custom tmux config to ~/.tmux.conf"
    else
        _backup_file "${DOTFILES_HOME}/.tmux.conf"
        cp "${DOTFILES_MODULE_DIR}/tmux.conf" "${DOTFILES_HOME}/.tmux.conf"
        log_success "Deployed custom tmux config"

        # Deploy cheatsheet
        mkdir -p "${DOTFILES_HOME}/.config/tmux"
        cp "${DOTFILES_MODULE_DIR}/tmux-cheatsheet.md" "${DOTFILES_HOME}/.config/tmux/tmux-cheatsheet.md"
        log_success "Deployed tmux cheatsheet to ~/.config/tmux/tmux-cheatsheet.md"
    fi
else
    log_info "Skipping tmux config (preset: none)"
fi

log_success "tmux installed"
