#!/usr/bin/env bash
# hello/install.sh — a custom module that lives in your content overlay, not in
# the engine repo. It is authored exactly like a built-in module: it inherits
# `set -euo pipefail` and the lib/helpers.sh functions (log_*, is_dry_run,
# pkg_install, …) and the DOTFILES_* environment from the Go runner.

if is_dry_run; then
    log_info "[dry-run] Would greet from the hello content module"
    return 0
fi

log_success "Hello from your content overlay, ${DOTFILES_USER_NAME:-friend}!"
