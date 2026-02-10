#!/usr/bin/env bash
#
# helpers.sh - Shell helpers library for the dotfiles management system.
#
# This file is sourced by module install.sh scripts. It provides logging,
# OS detection, package management, file operations, template rendering,
# secret retrieval, and interactive prompts.
#
# All behaviour is driven by environment variables injected by the Go runner:
#   DOTFILES_OS, DOTFILES_ARCH, DOTFILES_PKG_MGR, DOTFILES_HAS_SUDO,
#   DOTFILES_HOME, DOTFILES_DIR, DOTFILES_BIN, DOTFILES_MODULE_DIR,
#   DOTFILES_MODULE_NAME, DOTFILES_INTERACTIVE, DOTFILES_DRY_RUN,
#   DOTFILES_VERBOSE, DOTFILES_USER_NAME, DOTFILES_USER_EMAIL,
#   DOTFILES_USER_GITHUB_USER
# ---------------------------------------------------------------------------

set -euo pipefail

# ===========================================================================
# Logging
# ===========================================================================

# _color ANSI_CODE TEXT
#   Wraps TEXT in the given ANSI colour code when DOTFILES_INTERACTIVE=true.
#   Otherwise prints TEXT undecorated.
_color() {
    local code="$1"; shift
    if [[ "${DOTFILES_INTERACTIVE:-false}" == "true" ]]; then
        printf '\033[%sm%s\033[0m' "$code" "$*"
    else
        printf '%s' "$*"
    fi
}

# log_info MSG  - informational message (blue bullet)
log_info() {
    printf '%s %s\n' "$(_color '0;34' '•')" "$*"
}

# log_warn MSG  - warning (yellow triangle)
log_warn() {
    printf '%s %s\n' "$(_color '0;33' '⚠')" "$*" >&2
}

# log_error MSG - error (red cross)
log_error() {
    printf '%s %s\n' "$(_color '0;31' '✗')" "$*" >&2
}

# log_success MSG - success (green tick)
log_success() {
    printf '%s %s\n' "$(_color '0;32' '✓')" "$*"
}

# ===========================================================================
# OS / environment checks
# ===========================================================================

is_macos()       { [[ "${DOTFILES_OS:-}" == "darwin" ]]; }
is_ubuntu()      { [[ "${DOTFILES_OS:-}" == "ubuntu" ]]; }
is_arch()        { [[ "${DOTFILES_OS:-}" == "arch" ]]; }
has_sudo()       { [[ "${DOTFILES_HAS_SUDO:-false}" == "true" ]]; }
is_interactive() { [[ "${DOTFILES_INTERACTIVE:-false}" == "true" ]]; }
is_dry_run()     { [[ "${DOTFILES_DRY_RUN:-false}" == "true" ]]; }

# ===========================================================================
# Package management
# ===========================================================================

# pkg_installed PKG
#   Return 0 when PKG is already installed, 1 otherwise.
#   Dispatches to the appropriate check based on DOTFILES_PKG_MGR.
pkg_installed() {
    local pkg="$1"
    case "${DOTFILES_PKG_MGR:-}" in
        brew)   brew list "$pkg" &>/dev/null ;;
        apt)    dpkg -s "$pkg" &>/dev/null ;;
        pacman) pacman -Qi "$pkg" &>/dev/null ;;
        *)
            log_error "Unknown package manager: ${DOTFILES_PKG_MGR:-<unset>}"
            return 1
            ;;
    esac
}

# pkg_install PKG1 [PKG2 ...]
#   Install one or more packages that are not already present.
#   Respects dry-run mode (logs what would happen without acting).
pkg_install() {
    local to_install=()
    for pkg in "$@"; do
        if pkg_installed "$pkg"; then
            log_info "Package already installed: $pkg"
        else
            to_install+=("$pkg")
        fi
    done

    if [[ ${#to_install[@]} -eq 0 ]]; then
        return 0
    fi

    local cmd
    case "${DOTFILES_PKG_MGR:-}" in
        brew)   cmd=(brew install) ;;
        apt)
            if has_sudo; then
                cmd=(sudo apt-get install -y)
            else
                log_error "apt requires sudo but DOTFILES_HAS_SUDO is not true"
                return 1
            fi
            ;;
        pacman)
            if has_sudo; then
                cmd=(sudo pacman -S --noconfirm)
            else
                log_error "pacman requires sudo but DOTFILES_HAS_SUDO is not true"
                return 1
            fi
            ;;
        *)
            log_error "Unknown package manager: ${DOTFILES_PKG_MGR:-<unset>}"
            return 1
            ;;
    esac

    if is_dry_run; then
        log_info "[dry-run] Would run: ${cmd[*]} ${to_install[*]}"
        return 0
    fi

    log_info "Installing packages: ${to_install[*]}"
    "${cmd[@]}" "${to_install[@]}"
    log_success "Installed: ${to_install[*]}"
}

# ===========================================================================
# File operations (respect dry-run)
# ===========================================================================

# _backup_file PATH
#   Move PATH to PATH.backup.TIMESTAMP if it exists.
_backup_file() {
    local path="$1"
    if [[ -e "$path" || -L "$path" ]]; then
        local ts
        ts="$(date +%Y%m%d%H%M%S)"
        local backup="${path}.backup.${ts}"
        if is_dry_run; then
            log_info "[dry-run] Would backup: $path -> $backup"
        else
            mv "$path" "$backup"
            log_warn "Backed up existing file: $path -> $backup"
        fi
    fi
}

# link_file SOURCE DEST
#   Create a symlink DEST -> SOURCE.  If DEST already exists and is the
#   correct symlink, do nothing.  Otherwise back up the existing file first.
link_file() {
    local src="$1" dest="$2"

    # Already the correct symlink -- nothing to do.
    if [[ -L "$dest" ]] && [[ "$(readlink "$dest")" == "$src" ]]; then
        log_info "Symlink already correct: $dest -> $src"
        return 0
    fi

    # Back up whatever is currently at dest.
    _backup_file "$dest"

    if is_dry_run; then
        log_info "[dry-run] Would symlink: $dest -> $src"
        return 0
    fi

    mkdir -p "$(dirname "$dest")"
    ln -sf "$src" "$dest"
    log_success "Linked: $dest -> $src"
}

# copy_file SOURCE DEST
#   Copy SOURCE to DEST, backing up any existing file at DEST first.
copy_file() {
    local src="$1" dest="$2"

    _backup_file "$dest"

    if is_dry_run; then
        log_info "[dry-run] Would copy: $src -> $dest"
        return 0
    fi

    mkdir -p "$(dirname "$dest")"
    cp -f "$src" "$dest"
    log_success "Copied: $src -> $dest"
}

# ===========================================================================
# Templates and secrets (delegate to the Go binary)
# ===========================================================================

# render_template SRC DEST
#   Ask the Go runner to render the template at SRC into DEST, passing the
#   current module directory for context.
render_template() {
    local src="$1" dest="$2"

    if is_dry_run; then
        log_info "[dry-run] Would render template: $src -> $dest"
        return 0
    fi

    "${DOTFILES_BIN}" render-template \
        --src "$src" \
        --dest "$dest" \
        --module "${DOTFILES_MODULE_DIR}"
}

# get_secret REF
#   Retrieve a secret via the Go runner and print it to stdout.
get_secret() {
    local ref="$1"
    "${DOTFILES_BIN}" get-secret --ref "$ref"
}

# ===========================================================================
# Interactive prompts
# ===========================================================================

# prompt_input MESSAGE DEFAULT
#   If running interactively, prompt the user.  Otherwise return DEFAULT.
prompt_input() {
    local message="$1" default="${2:-}"

    if ! is_interactive; then
        printf '%s' "$default"
        return 0
    fi

    local reply
    printf '%s [%s]: ' "$message" "$default" >&2
    read -r reply
    if [[ -z "$reply" ]]; then
        printf '%s' "$default"
    else
        printf '%s' "$reply"
    fi
}

# prompt_confirm MESSAGE DEFAULT_BOOL
#   Return 0 for yes, 1 for no.  DEFAULT_BOOL should be "true" or "false".
#   In non-interactive mode the default is used silently.
prompt_confirm() {
    local message="$1" default_bool="${2:-true}"

    if ! is_interactive; then
        [[ "$default_bool" == "true" ]]
        return $?
    fi

    local hint
    if [[ "$default_bool" == "true" ]]; then
        hint="Y/n"
    else
        hint="y/N"
    fi

    local reply
    printf '%s [%s]: ' "$message" "$hint" >&2
    read -r reply

    case "${reply,,}" in
        y|yes) return 0 ;;
        n|no)  return 1 ;;
        "")
            [[ "$default_bool" == "true" ]]
            return $?
            ;;
        *)
            log_warn "Invalid response '$reply', using default ($default_bool)"
            [[ "$default_bool" == "true" ]]
            return $?
            ;;
    esac
}

# prompt_choice MESSAGE OPT1 [OPT2 ...]
#   Present a numbered list and return the chosen option string on stdout.
#   In non-interactive mode the first option is selected automatically.
prompt_choice() {
    local message="$1"; shift
    local options=("$@")

    if [[ ${#options[@]} -eq 0 ]]; then
        log_error "prompt_choice called with no options"
        return 1
    fi

    if ! is_interactive; then
        printf '%s' "${options[0]}"
        return 0
    fi

    printf '%s\n' "$message" >&2
    local i
    for i in "${!options[@]}"; do
        printf '  %d) %s\n' "$((i + 1))" "${options[$i]}" >&2
    done

    local reply
    while true; do
        printf 'Choice [1-%d]: ' "${#options[@]}" >&2
        read -r reply
        if [[ "$reply" =~ ^[0-9]+$ ]] && (( reply >= 1 && reply <= ${#options[@]} )); then
            printf '%s' "${options[$((reply - 1))]}"
            return 0
        fi
        log_warn "Invalid choice '$reply', please enter a number between 1 and ${#options[@]}"
    done
}
