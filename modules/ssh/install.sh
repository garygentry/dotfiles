#!/usr/bin/env bash
# ssh/install.sh - Configure SSH keys and settings
#
# Key management is driven by the `key_source` setting (config.yml ->
# modules.ssh.key_source), exposed to this script as DOTFILES_SETTING_KEY_SOURCE:
#
#   generate   (default) generate a local key if one does not already exist
#   1password  retrieve the key from 1Password via the `op` CLI
#   agent      do NOT manage a local key; rely on an external SSH agent
#              (e.g. the 1Password agent). The rendered config avoids pinning
#              an IdentityFile / IdentitiesOnly so agent keys are offered.
#   none       leave ~/.ssh keys and config entirely untouched
#
# For key_source=1password the item is configurable via modules.ssh.key_item
# (DOTFILES_SETTING_KEY_ITEM), defaulting to "op://Private/SSH Key".

_ssh_dir="${DOTFILES_HOME}/.ssh"
_ssh_key_source="${DOTFILES_SETTING_KEY_SOURCE:-generate}"
_ssh_key_type="${DOTFILES_PROMPT_SSH_KEY_TYPE:-ed25519}"

# 1Password item holding the SSH key (key_source=1password). Configurable via
# config.yml -> modules.ssh.key_item, exposed as DOTFILES_SETTING_KEY_ITEM. The
# key type is appended as the field, so the full secret reference is
# "<key_item>/<key_type>" (e.g. op://Private/SSH Key/ed25519). Defaults to a
# neutral generic item ("op://Private/SSH Key" — 1Password's default personal
# vault) so no personal vault name is baked in. A single trailing slash is
# tolerated so "op://Vault/Item/" doesn't double up.
_ssh_key_item="${DOTFILES_SETTING_KEY_ITEM:-op://Private/SSH Key}"
_ssh_key_item="${_ssh_key_item%/}"

case "$_ssh_key_source" in
    generate|1password|agent|none) ;;
    *)
        log_error "Invalid ssh key_source '${_ssh_key_source}' (expected: generate, 1password, agent, none)"
        exit 1
        ;;
esac
log_info "SSH key source: ${_ssh_key_source}"

# 'none' — hands off: do not touch keys or config.
if [[ "$_ssh_key_source" == "none" ]]; then
    log_info "key_source=none: leaving ~/.ssh keys and config untouched"
    exit 0
fi

# ---------------------------------------------------------------------------
# Detect existing SSH keys (an existing key wins over the requested type)
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

_ssh_key_file="${_ssh_dir}/id_${_ssh_key_type}"

# Export the resolved values for the config template.
export DOTFILES_SSH_KEY_SOURCE="$_ssh_key_source"
export DOTFILES_SSH_KEY_TYPE="$_ssh_key_type"

# ---------------------------------------------------------------------------
# Create .ssh directory with correct permissions
# ---------------------------------------------------------------------------
if [[ ! -d "$_ssh_dir" ]]; then
    if is_dry_run; then
        log_info "[dry-run] Would create directory: ${_ssh_dir} (mode 700)"
    else
        mkdir -p "$_ssh_dir"
        chmod 700 "$_ssh_dir"
        log_success "Created ${_ssh_dir} with permissions 700"
    fi
else
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

# ---------------------------------------------------------------------------
# Acquire the key according to key_source
# ---------------------------------------------------------------------------
case "$_ssh_key_source" in
    1password)
        if [[ -f "$_ssh_key_file" ]]; then
            log_info "SSH key already exists at ${_ssh_key_file}"
        elif ! command -v op &>/dev/null; then
            log_error "key_source=1password but the 1Password CLI (op) is not available"
            exit 1
        else
            log_info "Retrieving SSH key from 1Password (${_ssh_key_item}/${_ssh_key_type})..."
            _ssh_op_key="$(get_secret "${_ssh_key_item}/${_ssh_key_type}" 2>/dev/null || true)"
            if [[ -z "$_ssh_op_key" ]]; then
                log_error "key_source=1password but no key found at ${_ssh_key_item}/${_ssh_key_type}"
                exit 1
            fi
            if is_dry_run; then
                log_info "[dry-run] Would write SSH key from 1Password to ${_ssh_key_file}"
            else
                printf '%s\n' "$_ssh_op_key" > "$_ssh_key_file"
                chmod 600 "$_ssh_key_file"
                log_success "SSH key retrieved from 1Password and saved to ${_ssh_key_file}"
            fi
        fi
        ;;
    generate)
        if [[ -f "$_ssh_key_file" ]]; then
            log_info "SSH key already exists at ${_ssh_key_file}"
        else
            _ssh_email="${DOTFILES_USER_EMAIL:-${DOTFILES_PROMPT_EMAIL:-${USER:-$(id -un 2>/dev/null || echo user)}@$(hostname)}}"
            if is_dry_run; then
                log_info "[dry-run] Would generate ${_ssh_key_type} SSH key at ${_ssh_key_file}"
            else
                log_info "Generating ${_ssh_key_type} SSH key..."
                if [[ "$_ssh_key_type" == "rsa" ]]; then
                    ssh-keygen -t rsa -b 4096 -C "$_ssh_email" -f "$_ssh_key_file" -N ""
                else
                    ssh-keygen -t "$_ssh_key_type" -C "$_ssh_email" -f "$_ssh_key_file" -N ""
                fi
                chmod 600 "$_ssh_key_file"
                chmod 644 "${_ssh_key_file}.pub"
                log_success "SSH key generated at ${_ssh_key_file}"
            fi
        fi
        ;;
    agent)
        log_info "key_source=agent: not generating a local key; relying on the SSH agent"
        ;;
esac

# For managed-key modes, ensure the public key exists (e.g. if only the private
# key was restored from 1Password).
if [[ "$_ssh_key_source" != "agent" ]] && [[ -f "$_ssh_key_file" ]] && [[ ! -f "${_ssh_key_file}.pub" ]]; then
    if is_dry_run; then
        log_info "[dry-run] Would regenerate public key from ${_ssh_key_file}"
    else
        ssh-keygen -y -f "$_ssh_key_file" > "${_ssh_key_file}.pub"
        chmod 644 "${_ssh_key_file}.pub"
        log_success "Regenerated public key: ${_ssh_key_file}.pub"
    fi
fi

# ---------------------------------------------------------------------------
# Resolve the GitHub IdentityFile (managed-key modes only). In agent mode the
# template omits IdentityFile/IdentitiesOnly so the agent's keys are offered.
# ---------------------------------------------------------------------------
if [[ "$_ssh_key_source" != "agent" ]]; then
    _ssh_github_key="~/.ssh/id_${_ssh_key_type}"
    _ssh_existing_github_key=""

    if [[ -f "${_ssh_dir}/config" ]]; then
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
fi

# ---------------------------------------------------------------------------
# Back up existing config and render the new one
# ---------------------------------------------------------------------------
if [[ -f "${_ssh_dir}/config" ]]; then
    _backup_file "${_ssh_dir}/config"
fi

render_template "${DOTFILES_MODULE_DIR}/config" "${_ssh_dir}/config"
