#!/usr/bin/env bash
# gcloud/install.sh - Install Google Cloud CLI

if command -v gcloud &>/dev/null; then
    log_info "Google Cloud CLI is already installed"
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install Google Cloud CLI"
    return 0
fi

log_info "Installing Google Cloud CLI..."

case "${DOTFILES_PKG_MGR}" in
    brew)
        brew install google-cloud-cli
        ;;
    apt)
        pkg_install google-cloud-cli
        ;;
    *)
        log_error "Unsupported package manager for gcloud: ${DOTFILES_PKG_MGR}"
        return 1
        ;;
esac

log_success "Google Cloud CLI installed"
