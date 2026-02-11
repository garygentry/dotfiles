#!/usr/bin/env bash
# rust/install.sh - Install Rust via rustup

if command -v rustc &>/dev/null; then
    log_info "Rust is already installed, updating..."
    if is_dry_run; then
        log_info "[dry-run] Would run rustup update"
        return 0
    fi
    rustup update 2>/dev/null || log_warn "rustup update failed (non-fatal)"
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install Rust via rustup"
    return 0
fi

log_info "Installing Rust via rustup..."
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y

# Source cargo env for this session
# shellcheck disable=SC1091
[[ -f "${HOME}/.cargo/env" ]] && source "${HOME}/.cargo/env"

log_success "Rust installed"
