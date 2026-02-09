# neovim/verify.sh - Verify Neovim installation

_nvim_errors=0

# Check nvim is installed and runs
if command -v nvim &>/dev/null; then
    _nvim_version="$(nvim --version 2>/dev/null | head -n1)"
    if [[ -n "$_nvim_version" ]]; then
        log_success "Neovim is installed: ${_nvim_version}"
    else
        log_warn "Neovim binary found but --version returned empty output"
        _nvim_errors=$((_nvim_errors + 1))
    fi
else
    log_error "Neovim (nvim) is not installed"
    _nvim_errors=$((_nvim_errors + 1))
fi

# Check init.lua is linked
_nvim_init_lua="${DOTFILES_HOME}/.config/nvim/init.lua"
if [[ -L "$_nvim_init_lua" ]]; then
    log_success "init.lua is symlinked: ${_nvim_init_lua}"
elif [[ -f "$_nvim_init_lua" ]]; then
    log_info "init.lua exists but is not a symlink: ${_nvim_init_lua}"
else
    log_warn "init.lua not found: ${_nvim_init_lua}"
    _nvim_errors=$((_nvim_errors + 1))
fi

# Check nvim config directory exists
_nvim_config="${DOTFILES_HOME}/.config/nvim"
if [[ -d "$_nvim_config" ]]; then
    log_success "Neovim config directory exists: ${_nvim_config}"
else
    log_warn "Neovim config directory not found: ${_nvim_config}"
    _nvim_errors=$((_nvim_errors + 1))
fi

if [[ $_nvim_errors -gt 0 ]]; then
    log_warn "Neovim verification completed with ${_nvim_errors} warning(s)"
else
    log_success "Neovim verification passed"
fi
