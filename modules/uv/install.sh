#!/usr/bin/env bash
# uv/install.sh - Install uv, the Python package and project manager from Astral.

_uv_bin_dir="${DOTFILES_HOME}/.local/bin"

# uv may already be present but not yet on PATH (a fresh install in this same run, or a
# shell that has not been restarted), so look in its install directory too.
_uv_find() {
    command -v uv 2>/dev/null && return 0
    [[ -x "${_uv_bin_dir}/uv" ]] && echo "${_uv_bin_dir}/uv" && return 0
    return 1
}

if _uv_path="$(_uv_find)"; then
    log_info "uv is already installed ($("$_uv_path" --version))"
    exit 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install uv"
    exit 0
fi

log_info "Installing uv..."

if is_macos && command -v brew &>/dev/null; then
    brew install uv
elif is_arch; then
    pkg_install uv
else
    # Astral's installer is the supported path on Linux, and keeps `uv self update`
    # working. Debian/Ubuntu have no uv package worth using.
    mkdir -p "$_uv_bin_dir"
    curl -LsSf https://astral.sh/uv/install.sh | env UV_INSTALL_DIR="$_uv_bin_dir" sh
fi

if _uv_path="$(_uv_find)"; then
    log_success "uv installed ($("$_uv_path" --version))"
else
    log_error "uv installation completed but no uv binary was found"
    exit 1
fi
