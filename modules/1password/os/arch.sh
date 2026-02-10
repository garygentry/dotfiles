#!/usr/bin/env bash
# 1password/os/arch.sh - Install 1Password CLI on Arch Linux

if command -v op &>/dev/null; then
    log_info "1Password CLI already installed on Arch"
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install 1Password CLI on Arch Linux"
    return 0
fi

log_info "Installing 1Password CLI on Arch Linux..."

# Try AUR helper (yay or paru) first
if command -v yay &>/dev/null; then
    yay -S --noconfirm 1password-cli
    log_success "1Password CLI installed via yay"
elif command -v paru &>/dev/null; then
    paru -S --noconfirm 1password-cli
    log_success "1Password CLI installed via paru"
else
    # Direct download fallback
    log_info "No AUR helper found, downloading 1Password CLI directly..."
    _op_tmpdir="$(mktemp -d)"
    _op_arch="amd64"
    if [[ "${DOTFILES_ARCH:-}" == "arm64" ]]; then
        _op_arch="arm64"
    fi
    curl -sSfL "https://cache.agilebits.com/dist/1P/op2/pkg/v2.24.0/op_linux_${_op_arch}_v2.24.0.zip" \
        -o "${_op_tmpdir}/op.zip"
    cd "${_op_tmpdir}" && unzip -o op.zip
    if has_sudo; then
        sudo install -m 0755 op /usr/local/bin/op
    else
        install -m 0755 op "${DOTFILES_HOME}/.local/bin/op"
    fi
    rm -rf "${_op_tmpdir}"
    log_success "1Password CLI installed via direct download"
fi
