#!/usr/bin/env bash
# zsh/verify.sh - Verify Zsh installation and configuration

_zsh_errors=0

# Check zsh is installed
if command -v zsh &>/dev/null; then
    _zsh_version="$(zsh --version 2>/dev/null | head -n1)"
    log_success "Zsh is installed: ${_zsh_version}"
else
    log_error "Zsh is not installed"
    _zsh_errors=$((_zsh_errors + 1))
fi

# Check .zshrc is linked
_zsh_rc="${DOTFILES_HOME}/.zshrc"
if [[ -L "$_zsh_rc" ]]; then
    log_success ".zshrc is symlinked: ${_zsh_rc}"
elif [[ -f "$_zsh_rc" ]]; then
    log_info ".zshrc exists but is not a symlink: ${_zsh_rc}"
else
    log_warn ".zshrc not found: ${_zsh_rc}"
    _zsh_errors=$((_zsh_errors + 1))
fi

# Check Zinit is installed
_zsh_zinit_home="${DOTFILES_HOME}/.local/share/zinit/zinit.git"
if [[ -d "$_zsh_zinit_home" ]]; then
    log_success "Zinit is installed at ${_zsh_zinit_home}"
else
    log_warn "Zinit is not installed at ${_zsh_zinit_home}"
    _zsh_errors=$((_zsh_errors + 1))
fi

# Check aliases file is linked
_zsh_aliases="${DOTFILES_HOME}/.config/zsh/aliases.zsh"
if [[ -L "$_zsh_aliases" ]]; then
    log_success "aliases.zsh is symlinked"
elif [[ -f "$_zsh_aliases" ]]; then
    log_info "aliases.zsh exists but is not a symlink"
else
    log_warn "aliases.zsh not found: ${_zsh_aliases}"
    _zsh_errors=$((_zsh_errors + 1))
fi

# Check functions file is linked
_zsh_functions="${DOTFILES_HOME}/.config/zsh/functions.zsh"
if [[ -L "$_zsh_functions" ]]; then
    log_success "functions.zsh is symlinked"
elif [[ -f "$_zsh_functions" ]]; then
    log_info "functions.zsh exists but is not a symlink"
else
    log_warn "functions.zsh not found: ${_zsh_functions}"
    _zsh_errors=$((_zsh_errors + 1))
fi

if [[ $_zsh_errors -gt 0 ]]; then
    log_warn "Zsh verification completed with ${_zsh_errors} warning(s)"
else
    log_success "Zsh verification passed"
fi
