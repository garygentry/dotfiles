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
            _aws_arch="x86_64"
            [[ "${DOTFILES_ARCH:-}" == "arm64" ]] && _aws_arch="aarch64"
            download_file \
                "https://awscli.amazonaws.com/awscli-exe-linux-${_aws_arch}.zip" \
                "${_aws_tmp}/awscliv2.zip"
            unzip -qo "${_aws_tmp}/awscliv2.zip" -d "${_aws_tmp}"
            sudo_cmd "${_aws_tmp}/aws/install" --update 2>/dev/null || true
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
        _aws_arch="x86_64"
        [[ "${DOTFILES_ARCH:-}" == "arm64" ]] && _aws_arch="aarch64"
        download_file \
            "https://awscli.amazonaws.com/awscli-exe-linux-${_aws_arch}.zip" \
            "${_aws_tmp}/awscliv2.zip"
        unzip -qo "${_aws_tmp}/awscliv2.zip" -d "${_aws_tmp}"
        sudo_cmd "${_aws_tmp}/aws/install"
        rm -rf "${_aws_tmp}"
        ;;
esac

log_success "AWS CLI installed"
