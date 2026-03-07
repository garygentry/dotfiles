#!/usr/bin/env bash
set -euo pipefail
command -v tmux >/dev/null 2>&1 || { log_error "tmux not found"; exit 1; }
[ -d ~/.tmux/plugins/tpm ] || log_warn "TPM not found"
if [[ "${DOTFILES_PROMPT_TMUX_CONFIG:-skip}" == "opinionated" ]]; then
    [ -f "${DOTFILES_HOME}/.config/tmux/tmux.conf" ] || log_warn "Opinionated tmux config not found at ~/.config/tmux/tmux.conf"
fi
if [ -f "${DOTFILES_HOME}/.tmux.conf" ]; then
    log_warn "Legacy ~/.tmux.conf exists and may shadow ~/.config/tmux/tmux.conf on tmux < 3.1"
fi
log_success "tmux verification passed"
