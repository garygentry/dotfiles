#!/usr/bin/env bash
# herdr/install.sh - Install the Herdr terminal workspace manager (binary only).

_herdr_home="${DOTFILES_HOME:-$HOME}"

if command -v herdr &>/dev/null || [ -x "${_herdr_home}/.local/bin/herdr" ]; then
    log_info "herdr is already installed (updates are owned by 'herdr update')"
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install herdr"
    return 0
fi

log_info "Installing herdr..."
if is_macos && command -v brew &>/dev/null; then
    pkg_install herdr
else
    # Upstream installer: downloads the release binary for this platform, verifies
    # its SHA-256 against the release manifest, and places it in ~/.local/bin.
    curl -fsSL https://herdr.dev/install.sh | HERDR_INSTALL_DIR="${_herdr_home}/.local/bin" sh || {
        log_error "Failed to install herdr"
        return 1
    }
fi

# The generic engine ships no config.toml. Deploy your own from a content overlay.
log_success "herdr installed"
