#!/usr/bin/env bash
# azure-cli/install.sh - Install Azure CLI

if command -v az &>/dev/null; then
    log_info "Azure CLI is already installed"
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install Azure CLI"
    return 0
fi

log_info "Installing Azure CLI..."

case "${DOTFILES_PKG_MGR}" in
    brew)
        pkg_install azure-cli
        ;;
    apt)
        # Official Microsoft install script
        curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
        ;;
    pacman)
        pkg_install azure-cli
        ;;
    *)
        log_error "Unsupported package manager for Azure CLI: ${DOTFILES_PKG_MGR}"
        return 1
        ;;
esac

log_success "Azure CLI installed"
