#!/usr/bin/env bash
# ssh/verify.sh - Verify SSH configuration
#
# Exits non-zero when an essential artifact is missing, so the runner records a
# genuine failure (and re-runs the module on later invocations). What counts as
# essential depends on key_source:
#
#   generate/1password  a local key, the config, and a resolvable GitHub key
#   agent               the config only (no local key is expected)
#   none                nothing (the module manages nothing)

set -euo pipefail

_ssh_dir="${DOTFILES_HOME}/.ssh"
_ssh_key_source="${DOTFILES_SETTING_KEY_SOURCE:-generate}"
_ssh_errors=0

# The managed block of ~/.ssh/config is what install.sh last rendered. Content
# outside it belongs to the user, so nothing below reads from there.
_ssh_managed_block=""
if [[ -f "${_ssh_dir}/config" ]]; then
    _ssh_managed_block="$(awk '
        { sub(/\r$/, "") }
        $0 == "# <<< dotfiles managed <<<" { inb = 0 }
        inb { print }
        $0 == "# >>> dotfiles managed >>>" { inb = 1 }
    ' "${_ssh_dir}/config")"
fi

# auto resolves at install time; verify what it resolved to, as recorded in the
# managed block ("# key_source: <mode>"). Without a record only the config is checked.
if [[ "$_ssh_key_source" == "auto" ]]; then
    _ssh_recorded="$(awk '/^# key_source: / { print $3; exit }' <<<"$_ssh_managed_block")"
    _ssh_key_source="${_ssh_recorded:-auto}"
fi

# The local key is the one ssh will use: the Host * IdentityFile install.sh rendered.
# It can differ from the prompted type, because install.sh reuses an existing key.
# A block with none (not rendered yet, or from agent mode) falls back to that type.
_ssh_identity="$(awk '
    /^[[:space:]]*Host[[:space:]]/ { inhost = ($2 == "*") }
    inhost && /^[[:space:]]*IdentityFile[[:space:]]/ { print $2; exit }
' <<<"$_ssh_managed_block")"
_ssh_key_file="${_ssh_dir}/id_${DOTFILES_PROMPT_SSH_KEY_TYPE:-ed25519}"
[[ -z "$_ssh_identity" ]] || _ssh_key_file="${_ssh_identity/#\~/$DOTFILES_HOME}"

if [[ "$_ssh_key_source" == "none" ]]; then
    log_success "SSH key_source=none: nothing to verify"
    exit 0
fi

# .ssh directory with 700 permissions.
if [[ -d "$_ssh_dir" ]]; then
    _ssh_perms="$(stat -c '%a' "$_ssh_dir" 2>/dev/null || stat -f '%Lp' "$_ssh_dir" 2>/dev/null)"
    if [[ "$_ssh_perms" == "700" ]]; then
        log_success "SSH directory exists with correct permissions (700)"
    else
        log_error "SSH directory permissions are ${_ssh_perms}, expected 700"
        _ssh_errors=$((_ssh_errors + 1))
    fi
else
    log_error "SSH directory does not exist: ${_ssh_dir}"
    _ssh_errors=$((_ssh_errors + 1))
fi

# SSH config is rendered in every managed mode.
if [[ -f "${_ssh_dir}/config" ]]; then
    log_success "SSH config exists: ${_ssh_dir}/config"
else
    log_error "SSH config not found: ${_ssh_dir}/config"
    _ssh_errors=$((_ssh_errors + 1))
fi

# A local key is required only when we manage one.
if [[ "$_ssh_key_source" == "generate" || "$_ssh_key_source" == "1password" ]]; then
    if [[ -f "$_ssh_key_file" ]]; then
        log_success "SSH key exists: ${_ssh_key_file}"
    else
        log_error "SSH key not found: ${_ssh_key_file}"
        _ssh_errors=$((_ssh_errors + 1))
    fi

    # The GitHub IdentityFile referenced in config must exist — unless the host
    # declares no GitHub ssh at all (modules.ssh.github_host: false).
    if [[ "${DOTFILES_SETTING_GITHUB_HOST:-true}" == "false" ]]; then
        log_info "github_host=false: no GitHub ssh configured (git uses https); skipping IdentityFile check"
    elif [[ -f "${_ssh_dir}/config" ]]; then
        _ssh_github_identity="$(awk '
            /^[[:space:]]*Host[[:space:]]+github\.com[[:space:]]*$/ { found=1; next }
            found && /^[[:space:]]*Host[[:space:]]/ { found=0 }
            found && /^[[:space:]]*IdentityFile[[:space:]]/ { print $2; exit }
        ' "${_ssh_dir}/config")"

        if [[ -n "$_ssh_github_identity" ]]; then
            _ssh_github_identity_expanded="${_ssh_github_identity/#\~/$DOTFILES_HOME}"
            if [[ -f "$_ssh_github_identity_expanded" ]]; then
                log_success "GitHub IdentityFile exists: ${_ssh_github_identity}"
            else
                log_error "GitHub IdentityFile not found: ${_ssh_github_identity}"
                _ssh_errors=$((_ssh_errors + 1))
            fi
        fi
    fi
elif [[ "$_ssh_key_source" == "agent" ]]; then
    log_info "key_source=agent: not checking for a local key (agent-provided)"
fi

if [[ $_ssh_errors -gt 0 ]]; then
    log_error "SSH verification failed with ${_ssh_errors} error(s)"
    exit 1
fi

log_success "SSH verification passed"
