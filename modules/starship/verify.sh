#!/usr/bin/env bash
# starship/verify.sh - Verify Starship installation

_starship_bin="$(command -v starship 2>/dev/null || echo "${DOTFILES_HOME}/.local/bin/starship")"

if [[ ! -x "$_starship_bin" ]]; then
    log_error "Starship is not installed"
    exit 1
fi

log_success "Starship is installed ($("$_starship_bin" --version))"
