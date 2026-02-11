#!/usr/bin/env bash
# gemini-cli/install.sh - Install Gemini CLI

if command -v gemini &>/dev/null; then
    log_info "Gemini CLI is already installed, updating..."
    if is_dry_run; then
        log_info "[dry-run] Would update Gemini CLI"
        return 0
    fi
    npm update -g @google/gemini-cli 2>/dev/null || log_warn "Gemini CLI update failed (non-fatal)"
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install Gemini CLI"
    return 0
fi

log_info "Installing Gemini CLI..."
npm install -g @google/gemini-cli

log_success "Gemini CLI installed"
