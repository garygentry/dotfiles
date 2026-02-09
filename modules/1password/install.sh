# 1password/install.sh - Install 1Password CLI

if command -v op &>/dev/null; then
    log_success "1Password CLI (op) is already installed"
else
    log_info "1Password CLI not found, installing via OS-specific script..."
    # The runner executes os/<os>.sh before install.sh, so if we reach here
    # and op is still not available, the OS script may have just installed it.
    # Re-check after a moment (the OS script runs first in the runner pipeline).
    if ! command -v op &>/dev/null; then
        log_warn "1Password CLI was not installed by the OS-specific script"
        log_info "Please install 1Password CLI manually: https://developer.1password.com/docs/cli/get-started/"
    fi
fi
