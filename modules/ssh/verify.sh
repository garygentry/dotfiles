# ssh/verify.sh - Verify SSH configuration

_ssh_dir="${DOTFILES_HOME}/.ssh"
_ssh_key_type="${DOTFILES_PROMPT_SSH_KEY_TYPE:-ed25519}"
_ssh_key_file="${_ssh_dir}/id_${_ssh_key_type}"
_ssh_errors=0

# Check .ssh directory exists with correct permissions
if [[ -d "$_ssh_dir" ]]; then
    _ssh_perms="$(stat -c '%a' "$_ssh_dir" 2>/dev/null || stat -f '%Lp' "$_ssh_dir" 2>/dev/null)"
    if [[ "$_ssh_perms" == "700" ]]; then
        log_success "SSH directory exists with correct permissions (700)"
    else
        log_warn "SSH directory permissions are ${_ssh_perms}, expected 700"
        _ssh_errors=$((_ssh_errors + 1))
    fi
else
    log_error "SSH directory does not exist: ${_ssh_dir}"
    _ssh_errors=$((_ssh_errors + 1))
fi

# Check SSH key exists
if [[ -f "$_ssh_key_file" ]]; then
    log_success "SSH key exists: ${_ssh_key_file}"
else
    log_warn "SSH key not found: ${_ssh_key_file}"
    _ssh_errors=$((_ssh_errors + 1))
fi

# Check SSH config exists
if [[ -f "${_ssh_dir}/config" ]]; then
    log_success "SSH config exists: ${_ssh_dir}/config"
else
    log_warn "SSH config not found: ${_ssh_dir}/config"
    _ssh_errors=$((_ssh_errors + 1))
fi

if [[ $_ssh_errors -gt 0 ]]; then
    log_warn "SSH verification completed with ${_ssh_errors} warning(s)"
else
    log_success "SSH verification passed"
fi
