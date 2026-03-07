#!/usr/bin/env bash
# ssh/install.sh - Configure SSH keys and settings

_ssh_dir="${DOTFILES_HOME}/.ssh"
_ssh_key_type="${DOTFILES_PROMPT_SSH_KEY_TYPE:-ed25519}"

# ---------------------------------------------------------------------------
# Detect existing SSH keys
# ---------------------------------------------------------------------------
_ssh_detected_type=""
if [[ -f "${_ssh_dir}/id_ed25519" ]]; then
    _ssh_detected_type="ed25519"
elif [[ -f "${_ssh_dir}/id_rsa" ]]; then
    _ssh_detected_type="rsa"
elif [[ -f "${_ssh_dir}/id_ecdsa" ]]; then
    _ssh_detected_type="ecdsa"
fi

if [[ -n "$_ssh_detected_type" ]]; then
    log_info "Detected existing ${_ssh_detected_type} SSH key at ${_ssh_dir}/id_${_ssh_detected_type}"
    _ssh_key_type="$_ssh_detected_type"
fi

# Determine key file path based on type
_ssh_key_file="${_ssh_dir}/id_${_ssh_key_type}"

# Create .ssh directory with correct permissions
if [[ ! -d "$_ssh_dir" ]]; then
    if is_dry_run; then
        log_info "[dry-run] Would create directory: ${_ssh_dir} (mode 700)"
    else
        mkdir -p "$_ssh_dir"
        chmod 700 "$_ssh_dir"
        log_success "Created ${_ssh_dir} with permissions 700"
    fi
else
    # Ensure permissions are correct even if directory exists
    _ssh_current_perms="$(stat -c '%a' "$_ssh_dir" 2>/dev/null || stat -f '%Lp' "$_ssh_dir" 2>/dev/null)"
    if [[ "$_ssh_current_perms" != "700" ]]; then
        if is_dry_run; then
            log_info "[dry-run] Would fix permissions on ${_ssh_dir} to 700"
        else
            chmod 700 "$_ssh_dir"
            log_info "Fixed permissions on ${_ssh_dir} to 700"
        fi
    else
        log_info "${_ssh_dir} already exists with correct permissions"
    fi
fi

# Try to retrieve SSH key from 1Password if available
if [[ ! -f "$_ssh_key_file" ]]; then
    if command -v op &>/dev/null; then
        log_info "Attempting to retrieve SSH key from 1Password..."
        _ssh_op_key="$(get_secret "op://Personal/SSH Key/${_ssh_key_type}" 2>/dev/null || true)"
        if [[ -n "$_ssh_op_key" ]]; then
            if is_dry_run; then
                log_info "[dry-run] Would write SSH key from 1Password to ${_ssh_key_file}"
            else
                printf '%s\n' "$_ssh_op_key" > "$_ssh_key_file"
                chmod 600 "$_ssh_key_file"
                log_success "SSH key retrieved from 1Password and saved to ${_ssh_key_file}"
            fi
        else
            log_info "No SSH key found in 1Password, will generate a new one"
        fi
    fi
fi

# Generate SSH key if it still doesn't exist
if [[ ! -f "$_ssh_key_file" ]]; then
    _ssh_email="${DOTFILES_PROMPT_EMAIL:-${USER}@$(hostname)}"
    if is_dry_run; then
        log_info "[dry-run] Would generate ${_ssh_key_type} SSH key at ${_ssh_key_file}"
    else
        log_info "Generating ${_ssh_key_type} SSH key..."
        if [[ "$_ssh_key_type" == "rsa" ]]; then
            ssh-keygen -t rsa -b 4096 -C "$_ssh_email" -f "$_ssh_key_file" -N ""
        else
            ssh-keygen -t ed25519 -C "$_ssh_email" -f "$_ssh_key_file" -N ""
        fi
        chmod 600 "$_ssh_key_file"
        chmod 644 "${_ssh_key_file}.pub"
        log_success "SSH key generated at ${_ssh_key_file}"
    fi
else
    log_info "SSH key already exists at ${_ssh_key_file}"
fi

# ---------------------------------------------------------------------------
# Handle GitHub config conflict
# ---------------------------------------------------------------------------
_ssh_github_key="~/.ssh/id_${_ssh_key_type}"
_ssh_existing_github_key=""

if [[ -f "${_ssh_dir}/config" ]]; then
    # Extract IdentityFile from existing Host github.com block
    _ssh_existing_github_key="$(awk '
        /^[[:space:]]*Host[[:space:]]+github\.com[[:space:]]*$/ { found=1; next }
        found && /^[[:space:]]*Host[[:space:]]/ { found=0 }
        found && /^[[:space:]]*IdentityFile[[:space:]]/ { print $2; exit }
    ' "${_ssh_dir}/config")"
fi

if [[ -n "$_ssh_existing_github_key" ]] && [[ "$_ssh_existing_github_key" != "$_ssh_github_key" ]]; then
    log_info "Existing GitHub SSH config uses: ${_ssh_existing_github_key}"
    log_info "Detected/chosen key is: ${_ssh_github_key}"

    _ssh_github_choice="$(prompt_choice \
        "GitHub SSH key conflict - which key should github.com use?" \
        "Keep existing (${_ssh_existing_github_key})" \
        "Use detected key (${_ssh_github_key})")"

    if [[ "$_ssh_github_choice" == "Keep existing"* ]]; then
        _ssh_github_key="$_ssh_existing_github_key"
        log_info "Keeping existing GitHub key: ${_ssh_github_key}"
    else
        log_info "Using detected key for GitHub: ${_ssh_github_key}"
    fi
fi

export DOTFILES_SSH_GITHUB_KEY="$_ssh_github_key"

# ---------------------------------------------------------------------------
# Back up existing config and render new one
# ---------------------------------------------------------------------------
if [[ -f "${_ssh_dir}/config" ]]; then
    _backup_file "${_ssh_dir}/config"
fi

render_template "${DOTFILES_MODULE_DIR}/config" "${_ssh_dir}/config"
