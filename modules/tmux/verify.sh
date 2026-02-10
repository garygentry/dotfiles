#!/usr/bin/env bash
set -euo pipefail
command -v tmux >/dev/null 2>&1 || { log_error "tmux not found"; exit 1; }
[ -d ~/.tmux/plugins/tpm ] || log_warn "TPM not found"
log_success "tmux verification passed"
