#!/usr/bin/env bash
set -euo pipefail

log_info "Installing Node.js..."

# On WSL the Windows PATH bleeds in, so `node`/`npm` may resolve to a Windows
# install under /mnt/c. Treat those as absent — we want a real Linux node.
node_path="$(command -v node 2>/dev/null || true)"
if [[ -n "$node_path" && "$node_path" != /mnt/* ]]; then
    log_info "Node.js already installed: $(node --version)"
    exit 0
fi

# Install Node.js
pkg_install nodejs npm

log_success "Node.js installed"
