#!/usr/bin/env bash
# git/install.sh - Configure git with SSH signing and useful defaults

# Read user settings from environment (set by the Go runner from config.yml)
_git_user_name="${DOTFILES_USER_NAME:-}"
_git_user_email="${DOTFILES_USER_EMAIL:-}"
_git_ssh_key_type="${DOTFILES_PROMPT_SSH_KEY_TYPE:-ed25519}"
_git_ssh_key_file="${DOTFILES_HOME}/.ssh/id_${_git_ssh_key_type}"

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

# Configure SSH signing
if [[ -f "$_git_ssh_key_file" ]]; then
    if is_dry_run; then
        log_info "[dry-run] Would configure git SSH signing with ${_git_ssh_key_file}.pub"
    else
        git config --global gpg.format ssh
        git config --global user.signingkey "${_git_ssh_key_file}.pub"
        git config --global commit.gpgsign true
        git config --global tag.gpgsign true
        log_success "Configured git SSH signing with ${_git_ssh_key_file}.pub"
    fi
else
    log_warn "SSH key not found at ${_git_ssh_key_file}, skipping signing configuration"
fi

# Configure useful defaults
if is_dry_run; then
    log_info "[dry-run] Would set git defaults (defaultBranch, push, pull, aliases)"
else
    # Branch defaults
    git config --global init.defaultBranch main

    # Push/pull behavior
    git config --global push.autoSetupRemote true
    git config --global pull.rebase true

    # Commit template
    git config --global commit.template "${DOTFILES_HOME}/.gitmessage"

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
        if ! has_sudo; then
            log_warn "git-delta requires sudo to install via dpkg on Ubuntu"
            return 1
        fi
        local delta_version="0.18.2"
        local deb_arch="${DOTFILES_ARCH}"
        local url="https://github.com/dandavison/delta/releases/download/${delta_version}/git-delta_${delta_version}_${deb_arch}.deb"
        local tmp
        tmp="$(mktemp -d)"
        if curl -sfL -o "${tmp}/git-delta.deb" "$url"; then
            sudo dpkg -i "${tmp}/git-delta.deb"
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
    git config --global delta.side-by-side true
    git config --global merge.conflictstyle diff3
    git config --global diff.colorMoved default
    log_success "Installed and configured git-delta"
else
    log_warn "git-delta installation failed (optional), continuing without it"
fi
