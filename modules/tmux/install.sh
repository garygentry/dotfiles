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
_tmux_config="${DOTFILES_PROMPT_TMUX_CONFIG:-skip}"

if [[ "$_tmux_config" == "opinionated" ]]; then
    if is_dry_run; then
        log_info "[dry-run] Would deploy opinionated tmux config to ~/.config/tmux/tmux.conf"
    else
        # Back up and remove legacy ~/.tmux.conf (shadows XDG path on tmux 3.1+)
        _backup_file "${DOTFILES_HOME}/.tmux.conf"

        # Back up any existing XDG config before overwriting
        _backup_file "${DOTFILES_HOME}/.config/tmux/tmux.conf"

        # Deploy to XDG path
        mkdir -p "${DOTFILES_HOME}/.config/tmux"
        cp "${DOTFILES_MODULE_DIR}/tmux.conf" "${DOTFILES_HOME}/.config/tmux/tmux.conf"
        log_success "Deployed opinionated tmux config to ~/.config/tmux/tmux.conf"

        # Deploy cheatsheet
        cp "${DOTFILES_MODULE_DIR}/tmux-cheatsheet.md" "${DOTFILES_HOME}/.config/tmux/tmux-cheatsheet.md"
        log_success "Deployed tmux cheatsheet to ~/.config/tmux/tmux-cheatsheet.md"
    fi
else
    log_info "Skipping tmux config (preset: skip)"
fi

log_success "tmux installed"
