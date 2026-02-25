#!/usr/bin/env bash
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
        _starship_bin_dir="${DOTFILES_HOME}/.local/bin"
        mkdir -p "$_starship_bin_dir"
        curl -sS https://starship.rs/install.sh | sh -s -- -y -b "$_starship_bin_dir"
        log_success "Starship installed"
    fi
fi

# Apply starship preset or custom config
_preset="${DOTFILES_PROMPT_STARSHIP_PRESET:-custom}"
_starship_cfg="${DOTFILES_HOME}/.config/starship.toml"

if is_dry_run; then
    log_info "[dry-run] Would apply starship preset: ${_preset}"
elif [[ "$_preset" == "custom" ]]; then
    cp "${DOTFILES_MODULE_DIR}/starship.toml" "$_starship_cfg"
    log_success "Applied custom starship config"
else
    starship preset "$_preset" > "$_starship_cfg"
    log_success "Applied starship preset: ${_preset}"
fi
