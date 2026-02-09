# ssh/os/macos.sh - macOS-specific SSH setup

# macOS has SSH built-in, just ensure the agent is running
log_info "macOS: SSH is built-in, verifying ssh-agent..."

if ! pgrep -x ssh-agent &>/dev/null; then
    if is_dry_run; then
        log_info "[dry-run] Would start ssh-agent on macOS"
    else
        eval "$(ssh-agent -s)" &>/dev/null
        log_success "Started ssh-agent on macOS"
    fi
else
    log_info "ssh-agent is already running on macOS"
fi
