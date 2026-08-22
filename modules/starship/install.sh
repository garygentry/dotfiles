#!/usr/bin/env bash
# starship/install.sh - Install Starship cross-shell prompt

# Ensure ~/.config exists for the config symlink
_starship_config_dir="${DOTFILES_XDG_CONFIG_HOME}"
if [[ ! -d "$_starship_config_dir" ]]; then
    if is_dry_run; then
        log_info "[dry-run] Would create directory: ${_starship_config_dir}"
    else
        mkdir -p "$_starship_config_dir"
        log_success "Created ${_starship_config_dir}"
    fi
fi

# Install Starship via official installer
_starship_bin_dir="${DOTFILES_HOME}/.local/bin"
if command -v starship &>/dev/null; then
    log_info "Starship is already installed ($(starship --version))"
else
    if is_dry_run; then
        log_info "[dry-run] Would install Starship via official installer"
    else
        log_info "Installing Starship..."
        mkdir -p "$_starship_bin_dir"
        curl -sS https://starship.rs/install.sh | sh -s -- -y -b "$_starship_bin_dir"
        log_success "Starship installed"
    fi
fi

# Resolve the starship binary — it may not be on PATH yet after a fresh install
_starship_bin="$(command -v starship 2>/dev/null || echo "${_starship_bin_dir}/starship")"

# Apply an official starship preset. Personal/curated configs belong in a content
# overlay (deploy your own starship.toml), not in the generic engine.
_preset="${DOTFILES_PROMPT_STARSHIP_PRESET:-nerd-font-symbols}"
_starship_cfg="${DOTFILES_XDG_CONFIG_HOME}/starship.toml"

if is_dry_run; then
    log_info "[dry-run] Would apply starship preset: ${_preset}"
else
    "$_starship_bin" preset "$_preset" > "$_starship_cfg"
    log_success "Applied starship preset: ${_preset}"
fi
