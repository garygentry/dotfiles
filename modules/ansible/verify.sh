#!/usr/bin/env bash
set -euo pipefail

command -v ansible >/dev/null 2>&1 || { log_error "ansible not found"; exit 1; }
command -v ansible-playbook >/dev/null 2>&1 || { log_error "ansible-playbook not found"; exit 1; }

log_success "Ansible verification passed ($(ansible --version | head -1))"
