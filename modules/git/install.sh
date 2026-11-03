#!/usr/bin/env bash
# git/install.sh - Configure git with SSH signing and useful defaults

# Read user settings from environment (set by the Go runner from config.yml)
_git_user_name="${DOTFILES_USER_NAME:-}"
_git_user_email="${DOTFILES_USER_EMAIL:-}"
# The default signing key is the one the ssh module uses: the Host * IdentityFile in its
# managed block of ~/.ssh/config. An agent host, or one the module leaves alone, names none.
_git_ssh_key_file="${DOTFILES_HOME}/.ssh/id_ed25519"
if [[ -f "${DOTFILES_HOME}/.ssh/config" ]]; then
    _git_ssh_identity="$(awk '
        { sub(/\r$/, "") }
        $0 == "# <<< dotfiles managed <<<" { inb = 0 }
        inb && /^[[:space:]]*Host[[:space:]]/ { inhost = ($2 == "*") }
        inb && inhost && /^[[:space:]]*IdentityFile[[:space:]]/ { print $2; exit }
        $0 == "# >>> dotfiles managed >>>" { inb = 1 }
    ' "${DOTFILES_HOME}/.ssh/config")"
    [[ -z "$_git_ssh_identity" ]] || _git_ssh_key_file="${_git_ssh_identity/#\~/$DOTFILES_HOME}"
fi
# modules.git.signing_key (DOTFILES_SETTING_SIGNING_KEY): the PRIVATE key path to sign
# commits with (its .pub becomes user.signingkey). Default: the host's SSH identity key above.
# Set it to keep signing on a dedicated key rather than an identity key that is also used to
# log in to other machines. A leading ~ means $DOTFILES_HOME.
if [[ -n "${DOTFILES_SETTING_SIGNING_KEY:-}" ]]; then
    _git_ssh_key_file="${DOTFILES_SETTING_SIGNING_KEY/#\~/$DOTFILES_HOME}"
fi

# init.defaultBranch is configurable via config.yml -> modules.git.default_branch,
# exposed as DOTFILES_SETTING_DEFAULT_BRANCH. Defaults to 'main' so unset config
# behaves exactly as before.
_git_default_branch="${DOTFILES_SETTING_DEFAULT_BRANCH:-main}"

# modules.git.credential_helper (DOTFILES_SETTING_CREDENTIAL_HELPER): the value for
# the global credential.helper, e.g. "cache --timeout=900" or "osxkeychain". Unset
# leaves the host's helper alone: one set on every install could be shadowed but
# never removed, which got in the way of hosts that wire their own.
_git_credential_helper="${DOTFILES_SETTING_CREDENTIAL_HELPER:-}"

# Opinionated behaviors are opt-in so the generic engine imposes nothing. Enable
# from your content overlay: modules.git.sign_commits / modules.git.commit_template.
_git_sign_commits="${DOTFILES_SETTING_SIGN_COMMITS:-false}"
_git_commit_template="${DOTFILES_SETTING_COMMIT_TEMPLATE:-false}"

# Configure git user identity
if [[ -n "$_git_user_name" ]]; then
    if is_dry_run; then
        log_info "[dry-run] Would set git user.name to '${_git_user_name}'"
    else
        git config --global user.name "$_git_user_name"
        log_success "Set git user.name to '${_git_user_name}'"
    fi
else
    log_info "Skipping git user.name (not configured in config.yml)"
fi

if [[ -n "$_git_user_email" ]]; then
    if is_dry_run; then
        log_info "[dry-run] Would set git user.email to '${_git_user_email}'"
    else
        git config --global user.email "$_git_user_email"
        log_success "Set git user.email to '${_git_user_email}'"
    fi
else
    log_info "Skipping git user.email (not configured in config.yml)"
fi

if [[ -n "$_git_credential_helper" ]]; then
    if is_dry_run; then
        log_info "[dry-run] Would set git credential.helper to '${_git_credential_helper}'"
    else
        git config --global --replace-all credential.helper "$_git_credential_helper"
        log_success "Set git credential.helper to '${_git_credential_helper}'"
    fi
else
    log_info "Leaving git credential.helper as it is (modules.git.credential_helper not set)"
fi

# Configure SSH signing. user.signingkey names the .pub, and git can't sign or commit
# with signing on if that file is missing, so both must exist.
if [[ -f "$_git_ssh_key_file" && -f "${_git_ssh_key_file}.pub" ]]; then
    if is_dry_run; then
        log_info "[dry-run] Would configure git SSH signing with ${_git_ssh_key_file}.pub"
    elif [[ "$_git_sign_commits" != "true" ]] && [[ -n "$(git config --global --get user.signingkey || true)" ]]; then
        # Signing wasn't asked for, so don't replace a key the host already has (it may be GPG).
        log_info "Leaving the existing git user.signingkey as it is (modules.git.sign_commits not set)"
    else
        # Set up SSH-signing capability (format + key) so commits can be signed
        # on demand, but only force-enable signing when explicitly opted in.
        git config --global gpg.format ssh
        git config --global user.signingkey "${_git_ssh_key_file}.pub"
        if [[ "$_git_sign_commits" == "true" ]]; then
            git config --global commit.gpgsign true
            git config --global tag.gpgsign true
            log_success "Configured git SSH signing (auto-sign enabled) with ${_git_ssh_key_file}.pub"
        else
            log_success "Configured git SSH signing capability with ${_git_ssh_key_file}.pub (auto-sign off; enable via modules.git.sign_commits)"
        fi
    fi
elif [[ "$_git_sign_commits" == "true" ]]; then
    log_warn "No SSH key and .pub at ${_git_ssh_key_file}, so signing isn't configured although modules.git.sign_commits is on. If an earlier run set it up, commits will fail until the key is back or modules.git.signing_key names another."
else
    log_warn "No SSH key and .pub at ${_git_ssh_key_file}, skipping signing configuration"
fi

# Configure useful defaults
if is_dry_run; then
    log_info "[dry-run] Would set git init.defaultBranch to '${_git_default_branch}' (and push defaults, aliases)"
else
    # Branch defaults
    git config --global init.defaultBranch "$_git_default_branch"

    # Push/pull behavior (no pull.rebase opinion imposed; opt in via your overlay)
    git config --global push.autoSetupRemote true

    # Commit template (opt-in via modules.git.commit_template)
    if [[ "$_git_commit_template" == "true" ]]; then
        git config --global commit.template "${DOTFILES_HOME}/.gitmessage"
    fi

    # Global gitignore
    git config --global core.excludesfile "${DOTFILES_HOME}/.gitignore_global"

    # Aliases
    git config --global alias.co checkout
    git config --global alias.br branch
    git config --global alias.ci commit
    git config --global alias.st status
    git config --global alias.lg "log --oneline --graph --decorate --all"

    log_success "Configured git defaults and aliases"
fi

# Install delta for better diffs (optional — failure here should not break git module)
install_delta() {
    if command -v delta &>/dev/null; then
        log_info "git-delta is already installed"
        return 0
    fi

    if is_dry_run; then
        log_info "[dry-run] Would install git-delta for better diffs"
        return 0
    fi

    log_info "Installing git-delta for better diffs..."

    if is_ubuntu; then
        # delta is not in Ubuntu's apt repos; install from GitHub releases
        if ! is_interactive && ! is_root && ! has_sudo; then
            log_warn "git-delta requires sudo to install via dpkg on Ubuntu (non-interactive mode)"
            return 1
        fi
        local delta_version="0.18.2"
        local deb_arch="${DOTFILES_ARCH}"
        local url="https://github.com/dandavison/delta/releases/download/${delta_version}/git-delta_${delta_version}_${deb_arch}.deb"
        local tmp
        tmp="$(mktemp -d)"
        if curl -sfL -o "${tmp}/git-delta.deb" "$url"; then
            sudo_cmd dpkg -i "${tmp}/git-delta.deb"
            rm -rf "$tmp"
        else
            rm -rf "$tmp"
            log_warn "Failed to download git-delta from GitHub releases"
            return 1
        fi
    else
        pkg_install git-delta
    fi
}

if install_delta; then
    git config --global core.pager delta
    git config --global interactive.diffFilter "delta --color-only"
    git config --global delta.navigate true
    git config --global merge.conflictstyle diff3
    git config --global diff.colorMoved default
    log_success "Installed and configured git-delta"
else
    log_warn "git-delta installation failed (optional), continuing without it"
fi
