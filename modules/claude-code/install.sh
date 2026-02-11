#!/usr/bin/env bash
# claude-code/install.sh - Install Claude Code CLI

if command -v claude &>/dev/null; then
    log_info "Claude Code is already installed, updating..."
    if is_dry_run; then
        log_info "[dry-run] Would update Claude Code"
        return 0
    fi
    claude update 2>/dev/null || log_info "Claude Code self-updates automatically"
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install Claude Code"
    return 0
fi

log_info "Installing Claude Code..."
curl -fsSL https://claude.ai/install.sh | bash

log_success "Claude Code installed"
