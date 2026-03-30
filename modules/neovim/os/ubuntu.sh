#!/usr/bin/env bash
# neovim/os/ubuntu.sh - Install Neovim on Ubuntu

_nvim_min_version="0.11.0"

# Check if an adequate version is already installed
if command -v nvim &>/dev/null; then
    _nvim_current="$(nvim --version 2>/dev/null | head -n1 | grep -oP '\d+\.\d+\.\d+')"
    if [[ "$(printf '%s\n' "$_nvim_min_version" "$_nvim_current" | sort -V | head -n1)" == "$_nvim_min_version" ]]; then
        log_info "Neovim ${_nvim_current} is already installed on Ubuntu"
        return 0
    fi
    log_warn "Neovim ${_nvim_current} is too old (need >= ${_nvim_min_version}), upgrading..."
fi

if is_dry_run; then
    log_info "[dry-run] Would add Neovim PPA and install via apt"
    return 0
fi

log_info "Adding Neovim PPA for latest version..."
sudo_cmd add-apt-repository -y ppa:neovim-ppa/unstable 2>/dev/null || \
    log_warn "Failed to add Neovim PPA, falling back to default apt package"
sudo_cmd apt-get update -qq
sudo_cmd apt-get install -y neovim

# Verify installed version meets minimum requirement
_nvim_current="$(nvim --version 2>/dev/null | head -n1 | grep -oP '\d+\.\d+\.\d+')"
if [[ -z "$_nvim_current" ]] || [[ "$(printf '%s\n' "$_nvim_min_version" "$_nvim_current" | sort -V | head -n1)" != "$_nvim_min_version" ]]; then
    log_warn "apt installed Neovim ${_nvim_current:-unknown} which is too old, installing from GitHub release..."
    sudo_cmd apt-get remove -y neovim 2>/dev/null || true

    _nvim_gh_version="$(curl -s https://api.github.com/repos/neovim/neovim/releases/latest | grep -Po '"tag_name": "v\K[^"]*')"
    _nvim_gh_url="https://github.com/neovim/neovim/releases/download/v${_nvim_gh_version}/nvim-linux-x86_64.tar.gz"
    [[ "${DOTFILES_ARCH:-}" == "arm64" ]] && _nvim_gh_url="https://github.com/neovim/neovim/releases/download/v${_nvim_gh_version}/nvim-linux-arm64.tar.gz"

    download_file "$_nvim_gh_url" "/tmp/nvim-linux.tar.gz"
    sudo_cmd rm -rf /opt/nvim-linux
    sudo_cmd tar xzf /tmp/nvim-linux.tar.gz -C /opt
    # The tarball extracts to nvim-linux-x86_64 or nvim-linux-arm64
    _nvim_extracted="$(ls -d /opt/nvim-linux-* 2>/dev/null | head -n1)"
    if [[ -n "$_nvim_extracted" ]]; then
        sudo_cmd mv "$_nvim_extracted" /opt/nvim-linux
    fi
    sudo_cmd ln -sf /opt/nvim-linux/bin/nvim /usr/local/bin/nvim
    rm -f /tmp/nvim-linux.tar.gz
    log_success "Neovim ${_nvim_gh_version} installed from GitHub release"
else
    log_success "Neovim ${_nvim_current} installed on Ubuntu"
fi
