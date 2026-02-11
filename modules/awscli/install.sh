#!/usr/bin/env bash
# awscli/install.sh - Install AWS CLI v2

if command -v aws &>/dev/null; then
    log_info "AWS CLI is already installed, updating..."
    if is_dry_run; then
        log_info "[dry-run] Would update AWS CLI"
        return 0
    fi
    case "${DOTFILES_PKG_MGR}" in
        brew)
            brew upgrade awscli 2>/dev/null || true
            ;;
        *)
            # Official installer with --update flag
            _aws_tmp="$(mktemp -d)"
            curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "${_aws_tmp}/awscliv2.zip"
            unzip -qo "${_aws_tmp}/awscliv2.zip" -d "${_aws_tmp}"
            sudo "${_aws_tmp}/aws/install" --update 2>/dev/null || true
            rm -rf "${_aws_tmp}"
            ;;
    esac
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install AWS CLI"
    return 0
fi

log_info "Installing AWS CLI..."

case "${DOTFILES_PKG_MGR}" in
    brew)
        pkg_install awscli
        ;;
    *)
        # Official installer for Linux
        _aws_tmp="$(mktemp -d)"
        curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "${_aws_tmp}/awscliv2.zip"
        unzip -qo "${_aws_tmp}/awscliv2.zip" -d "${_aws_tmp}"
        sudo "${_aws_tmp}/aws/install"
        rm -rf "${_aws_tmp}"
        ;;
esac

log_success "AWS CLI installed"
