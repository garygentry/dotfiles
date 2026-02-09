# 1password/os/ubuntu.sh - Install 1Password CLI on Ubuntu via apt

if command -v op &>/dev/null; then
    log_info "1Password CLI already installed on Ubuntu"
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install 1Password CLI via apt on Ubuntu"
    return 0
fi

log_info "Adding 1Password apt repository..."

# Add the 1Password GPG key
if has_sudo; then
    sudo mkdir -p /usr/share/keyrings
    curl -sS https://downloads.1password.com/linux/keys/1password.asc \
        | sudo gpg --dearmor --output /usr/share/keyrings/1password-archive-keyring.gpg 2>/dev/null

    # Add the 1Password apt repository
    echo "deb [arch=amd64 signed-by=/usr/share/keyrings/1password-archive-keyring.gpg] https://downloads.1password.com/linux/debian/amd64 stable main" \
        | sudo tee /etc/apt/sources.list.d/1password.list >/dev/null

    # Set up debsig-verify policy
    sudo mkdir -p /etc/debsig/policies/AC2D62742012EA22/
    curl -sS https://downloads.1password.com/linux/debian/debsig/1password.pol \
        | sudo tee /etc/debsig/policies/AC2D62742012EA22/1password.pol >/dev/null
    sudo mkdir -p /usr/share/debsig/keyrings/AC2D62742012EA22
    curl -sS https://downloads.1password.com/linux/keys/1password.asc \
        | sudo gpg --dearmor --output /usr/share/debsig/keyrings/AC2D62742012EA22/debsig.gpg 2>/dev/null

    # Install 1Password CLI
    sudo apt-get update -qq
    sudo apt-get install -y 1password-cli
    log_success "1Password CLI installed on Ubuntu"
else
    log_error "sudo is required to install 1Password CLI on Ubuntu"
    return 1
fi
