#!/usr/bin/env bash
# zsh/install.sh - Install and configure Zsh with Zinit or Oh My Zsh

_zsh_framework="${DOTFILES_PROMPT_ZSH_FRAMEWORK:-zinit}"

# Install zsh if not already present (OS-specific scripts handle this too)
if ! command -v zsh &>/dev/null; then
    log_info "Zsh not found, installing..."
    pkg_install zsh
else
    log_info "Zsh is already installed"
fi

# Create zsh config directory
_zsh_config_dir="${DOTFILES_XDG_CONFIG_HOME}/zsh"
if [[ ! -d "$_zsh_config_dir" ]]; then
    if is_dry_run; then
        log_info "[dry-run] Would create directory: ${_zsh_config_dir}"
    else
        mkdir -p "$_zsh_config_dir"
        log_success "Created ${_zsh_config_dir}"
    fi
fi

# Install plugin framework
if [[ "$_zsh_framework" == "ohmyzsh" ]]; then
    # Install Oh My Zsh
    _zsh_omz_dir="${DOTFILES_HOME}/.oh-my-zsh"
    if [[ ! -d "$_zsh_omz_dir" ]]; then
        if is_dry_run; then
            log_info "[dry-run] Would install Oh My Zsh to ${_zsh_omz_dir}"
        else
            log_info "Installing Oh My Zsh..."
            git clone https://github.com/ohmyzsh/ohmyzsh.git "$_zsh_omz_dir"
            log_success "Oh My Zsh installed to ${_zsh_omz_dir}"
        fi
    else
        log_info "Oh My Zsh is already installed at ${_zsh_omz_dir}"
    fi
else
    # Install Zinit plugin manager
    _zsh_zinit_home="${DOTFILES_HOME}/.local/share/zinit/zinit.git"
    if [[ ! -d "$_zsh_zinit_home" ]]; then
        if is_dry_run; then
            log_info "[dry-run] Would install Zinit plugin manager to ${_zsh_zinit_home}"
        else
            log_info "Installing Zinit plugin manager..."
            mkdir -p "$(dirname "$_zsh_zinit_home")"
            git clone https://github.com/zdharma-continuum/zinit.git "$_zsh_zinit_home"
            log_success "Zinit installed to ${_zsh_zinit_home}"
        fi
    else
        log_info "Zinit is already installed at ${_zsh_zinit_home}"
    fi
fi

# Set default shell to zsh if not already.
# Read from /etc/passwd rather than $SHELL: when the user has already run
# 'exec zsh', $SHELL is set to zsh by the running process, which would
# fool the check even though the login shell in /etc/passwd is still bash.
if is_macos; then
    _zsh_login_shell="$(basename "$(dscl . -read /Users/"${USER:-$(whoami)}" UserShell | awk '{print $2}')")"
else
    _zsh_login_shell="$(basename "$(getent passwd "${USER:-$(whoami)}" | cut -d: -f7)")"
fi
if [[ "$_zsh_login_shell" != "zsh" ]]; then
    _zsh_path="$(command -v zsh)"
    if [[ -n "$_zsh_path" ]]; then
        if is_dry_run; then
            log_info "[dry-run] Would change default shell to ${_zsh_path}"
        else
            # Ensure zsh is in /etc/shells
            if ! grep -qF "$_zsh_path" /etc/shells 2>/dev/null; then
                if is_root || has_sudo; then
                    echo "$_zsh_path" | sudo_cmd tee -a /etc/shells >/dev/null
                    log_info "Added ${_zsh_path} to /etc/shells"
                else
                    log_warn "Cannot add ${_zsh_path} to /etc/shells without sudo"
                fi
            fi
            if chsh -s "$_zsh_path"; then
                log_success "Default shell changed to zsh"
            else
                log_warn "Failed to change default shell to zsh (chsh exited non-zero)"
                log_warn "You can change it manually with: chsh -s $_zsh_path"
            fi
        fi
    fi
else
    log_info "Default shell is already zsh"
fi

# Add zsh auto-exec to the login profile as a reliable fallback.
# This starts zsh on login even when chsh hasn't taken effect yet (e.g.,
# requires a password, container environment, or system restart). Only fires
# in interactive sessions so batch/script logins are unaffected.
_zsh_autoexec_marker="# dotfiles: auto-exec zsh on login"
_zsh_autoexec_block="${_zsh_autoexec_marker}
if [[ -z \"\${ZSH_VERSION}\" ]] && [[ \$- == *i* ]] && command -v zsh &>/dev/null; then
    exec zsh -l
fi"

# macOS uses ~/.bash_profile; Linux uses ~/.profile
if is_macos; then
    _login_profile="${DOTFILES_HOME}/.bash_profile"
else
    _login_profile="${DOTFILES_HOME}/.profile"
fi

if is_dry_run; then
    log_info "[dry-run] Would add zsh auto-exec to ${_login_profile}"
elif grep -qF "$_zsh_autoexec_marker" "${_login_profile}" 2>/dev/null; then
    log_info "Zsh auto-exec already configured in ${_login_profile}"
else
    printf '\n%s\n' "${_zsh_autoexec_block}" >> "${_login_profile}"
    log_success "Added zsh auto-exec to ${_login_profile}"
fi
