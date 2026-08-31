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
#   auto       resolve conservatively at runtime: agent mode when a live SSH
#              agent already holds a key, otherwise generate mode (reusing an
#              existing local key if present). The safe default for hosts that
#              may or may not carry a baked key/agent (e.g. provisioned guests).
#   none       leave ~/.ssh keys and config entirely untouched
#
# Only a delimited "# >>> dotfiles managed >>>" ... "# <<< dotfiles managed <<<"
# block of ~/.ssh/config is owned by this module; content outside it (notably a
# host-provisioned deploy-key `Host github.com` entry) is preserved across runs.
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
    generate|1password|agent|none|auto) ;;
    *)
        log_error "Invalid ssh key_source '${_ssh_key_source}' (expected: generate, 1password, agent, none, auto)"
        exit 1
        ;;
esac

# 'auto' resolves to a concrete mode before anything else runs, so the rest of
# the module (key acquisition, GitHub key resolution, the config template) sees
# a plain agent/generate value and behaves exactly as if it were set explicitly.
if [[ "$_ssh_key_source" == "auto" ]]; then
    if ssh-add -l >/dev/null 2>&1; then
        log_info "key_source=auto: a live SSH agent holds a key -> agent mode"
        _ssh_key_source="agent"
    else
        log_info "key_source=auto: no agent key available -> generate mode (existing key reused if present)"
        _ssh_key_source="generate"
    fi
fi
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
# Render the module's config and splice it into ~/.ssh/config as a delimited
# managed block, preserving everything the user or host provisioning put outside
# it. The block is appended after existing content, so an earlier (e.g. baked
# deploy-key) `Host github.com` entry wins under ssh's first-match-per-option rule.
# ---------------------------------------------------------------------------
_ssh_begin_marker="# >>> dotfiles managed >>>"
_ssh_end_marker="# <<< dotfiles managed <<<"

# _ssh_write_managed_block TARGET BODYFILE
#   Write BODYFILE into TARGET as a single delimited managed block, touching only
#   what this module owns. The rules — chosen so external content is NEVER lost:
#     - A "clean" block is a begin marker whose next marker line is an end marker
#       (no intervening begin). The FIRST clean block's span is replaced in place;
#       any further clean blocks are removed (coalesced to one).
#     - Everything that is not part of a clean block is preserved verbatim,
#       including a stray/orphan marker (e.g. a begin with no matching end, left by
#       a truncated write or a manual edit) — such lines are never a delete anchor,
#       so they can't swallow a following `Host` entry.
#     - With no clean block, the rendered block is appended after existing content.
#     - A brand-new file gets a short header plus the block.
#   Marker matching tolerates a trailing CR so a CRLF config is handled correctly.
_ssh_write_managed_block() {
    local target="$1" body="$2" tmp
    tmp="$(mktemp)"

    if [[ -f "$target" ]]; then
        awk -v begin="$_ssh_begin_marker" -v end="$_ssh_end_marker" -v bodyfile="$body" '
            function norm(s) { sub(/\r$/, "", s); return s }
            function emit_block(   b) {
                print begin
                while ((getline b < bodyfile) > 0) print b
                close(bodyfile)
                print end
            }
            { ln[NR] = $0; m[NR] = norm($0) }
            END {
                n = NR
                # Mark every clean (non-nested) begin..end span for deletion;
                # remember the first ones position as the in-place insert point.
                insert = 0
                i = 1
                while (i <= n) {
                    if (m[i] == begin) {
                        j = i + 1
                        while (j <= n && m[j] != begin && m[j] != end) j++
                        if (j <= n && m[j] == end) {
                            if (insert == 0) insert = i
                            for (k = i; k <= j; k++) del[k] = 1
                            i = j + 1
                            continue
                        }
                    }
                    i++
                }
                printed = 0
                for (i = 1; i <= n; i++) {
                    if (i == insert) { emit_block(); printed = 1 }
                    if (!del[i]) print ln[i]
                }
                if (!printed) {          # no clean block existed: append one
                    if (n > 0) print ""
                    emit_block()
                }
            }
        ' "$target" > "$tmp"
    else
        {
            printf '# SSH config — the block below is managed by dotfiles.\n'
            printf '# Content outside the managed markers is preserved across runs.\n\n'
            printf '%s\n' "$_ssh_begin_marker"
            cat "$body"
            printf '%s\n' "$_ssh_end_marker"
        } > "$tmp"
    fi

    install -m 600 "$tmp" "$target"
    rm -f "$tmp"
}

if is_dry_run; then
    log_info "[dry-run] Would splice the managed block into ${_ssh_dir}/config (preserving external content)"
else
    # The target must be a real file we can splice into, not a symlink into a repo.
    demote_symlink "${_ssh_dir}/config"

    # Back up once, only when first adopting a pre-existing unmanaged config.
    # Substring match (not -x) so a CRLF marker line still counts as "managed".
    if [[ -f "${_ssh_dir}/config" ]] && ! grep -qF -- "$_ssh_begin_marker" "${_ssh_dir}/config"; then
        cp "${_ssh_dir}/config" "${_ssh_dir}/config.backup.$(date +%Y%m%d%H%M%S)"
        log_warn "Backed up existing SSH config before adopting the managed block"
    fi

    _ssh_rendered="$(mktemp)"
    render_template "${DOTFILES_MODULE_DIR}/config" "$_ssh_rendered"
    _ssh_write_managed_block "${_ssh_dir}/config" "$_ssh_rendered"
    rm -f "$_ssh_rendered"
    log_success "Updated dotfiles-managed block in ${_ssh_dir}/config"
fi
