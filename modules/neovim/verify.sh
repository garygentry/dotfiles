#!/usr/bin/env bash
# neovim/verify.sh - Verify Neovim installation

_nvim_errors=0

# Check nvim is installed, runs, and meets minimum version
_nvim_min_version="0.8.0"
if command -v nvim &>/dev/null; then
    _nvim_version="$(nvim --version 2>/dev/null | head -n1)"
    if [[ -n "$_nvim_version" ]]; then
        _nvim_semver="$(echo "$_nvim_version" | grep -oP '\d+\.\d+\.\d+')"
        if [[ -n "$_nvim_semver" ]] && [[ "$(printf '%s\n' "$_nvim_min_version" "$_nvim_semver" | sort -V | head -n1)" == "$_nvim_min_version" ]]; then
            log_success "Neovim is installed: ${_nvim_version}"
        else
            log_error "Neovim ${_nvim_semver:-unknown} is too old (need >= ${_nvim_min_version})"
            _nvim_errors=$((_nvim_errors + 1))
        fi
    else
        log_warn "Neovim binary found but --version returned empty output"
        _nvim_errors=$((_nvim_errors + 1))
    fi
else
    log_error "Neovim (nvim) is not installed"
    _nvim_errors=$((_nvim_errors + 1))
fi

# The generic engine installs the neovim binary only and ships no config, so a
# missing init.lua is expected. A config (from a content overlay) is informational.
_nvim_init_lua="${DOTFILES_XDG_CONFIG_HOME}/nvim/init.lua"
if [[ -L "$_nvim_init_lua" || -f "$_nvim_init_lua" ]]; then
    log_info "Neovim config present: ${_nvim_init_lua}"
fi

if [[ $_nvim_errors -gt 0 ]]; then
    log_error "Neovim verification failed with ${_nvim_errors} error(s)"
    exit 1
fi

log_success "Neovim verification passed"
