# starship/install.sh - Install Starship cross-shell prompt

# Ensure ~/.config exists for the config symlink
_starship_config_dir="${DOTFILES_HOME}/.config"
if [[ ! -d "$_starship_config_dir" ]]; then
    if is_dry_run; then
        log_info "[dry-run] Would create directory: ${_starship_config_dir}"
    else
        mkdir -p "$_starship_config_dir"
        log_success "Created ${_starship_config_dir}"
    fi
fi

# Install Starship via official installer
if command -v starship &>/dev/null; then
    log_info "Starship is already installed ($(starship --version))"
else
    if is_dry_run; then
        log_info "[dry-run] Would install Starship via official installer"
    else
        log_info "Installing Starship..."
        curl -sS https://starship.rs/install.sh | sh -s -- -y
        log_success "Starship installed"
    fi
fi
