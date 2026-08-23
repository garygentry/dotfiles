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

# Install OR upgrade Starship to the latest release. We deliberately do NOT skip
# when a starship is already present: a stale binary can't parse config keys from
# newer releases (it warns "Unknown key"/"unknown variant" on every shell). The
# official installer fetches the latest; -f overwrites the existing binary.
_starship_bin_dir="${DOTFILES_HOME}/.local/bin"
if is_dry_run; then
    if command -v starship &>/dev/null; then
        log_info "[dry-run] Would upgrade Starship to the latest release (have: $(starship --version))"
    else
        log_info "[dry-run] Would install the latest Starship to ${_starship_bin_dir}"
    fi
else
    mkdir -p "$_starship_bin_dir"
    log_info "Installing/upgrading Starship to the latest release..."
    curl -sS https://starship.rs/install.sh | sh -s -- -y -f -b "$_starship_bin_dir"
    log_success "Starship up to date ($("${_starship_bin_dir}/starship" --version 2>/dev/null | head -1 || echo installed))"
fi

# Resolve the starship binary — it may not be on PATH yet after a fresh install
_starship_bin="$(command -v starship 2>/dev/null || echo "${_starship_bin_dir}/starship")"

# Apply an official starship preset. Personal/curated configs belong in a content
# overlay (deploy your own starship.toml), not in the generic engine.
_preset="${DOTFILES_PROMPT_STARSHIP_PRESET:-nerd-font-symbols}"
_starship_cfg="${DOTFILES_XDG_CONFIG_HOME}/starship.toml"

# Older releases symlinked this path out of the repo; redirecting into a symlink
# would follow it and clobber the repo source. Migrate any legacy link first.
demote_symlink "$_starship_cfg"

if is_dry_run; then
    log_info "[dry-run] Would apply starship preset: ${_preset}"
else
    "$_starship_bin" preset "$_preset" > "$_starship_cfg"
    log_success "Applied starship preset: ${_preset}"
fi
