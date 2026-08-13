#!/usr/bin/env bash
# ansible/install.sh - Install Ansible.

if command -v ansible &>/dev/null; then
    log_info "Ansible already installed ($(ansible --version | head -1))"
    exit 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install Ansible"
    exit 0
fi

log_info "Installing Ansible..."

# Distribution packages are preferred: they carry a Python interpreter that matches the
# system's, which is what an agentless tool driving the local machine's SSH stack wants.
# Package names differ — Homebrew and Arch ship "ansible", Debian/Ubuntu split the core
# engine into "ansible-core" and the bundled collections into "ansible".
if is_macos || is_arch; then
    pkg_install ansible
else
    pkg_install ansible || pkg_install ansible-core
fi

if command -v ansible &>/dev/null; then
    log_success "Ansible installed ($(ansible --version | head -1))"
else
    log_error "Ansible installation completed but no ansible binary was found"
    exit 1
fi
