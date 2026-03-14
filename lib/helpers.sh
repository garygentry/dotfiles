#!/usr/bin/env bash
#
# helpers.sh - Shell helpers library for the dotfiles management system.
#
# This file is sourced by module install.sh scripts. It provides logging,
# OS detection, package management, file operations, template rendering,
# secret retrieval, and interactive prompts.
#
# All behaviour is driven by environment variables injected by the Go runner:
#   DOTFILES_OS, DOTFILES_ARCH, DOTFILES_PKG_MGR, DOTFILES_HAS_SUDO, DOTFILES_IS_ROOT,
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

is_macos()       { [[ "${DOTFILES_OS:-}" == "macos" ]]; }
is_ubuntu()      { [[ "${DOTFILES_OS:-}" == "ubuntu" ]]; }
is_arch()        { [[ "${DOTFILES_OS:-}" == "arch" ]]; }
has_sudo()             { [[ "${DOTFILES_HAS_SUDO:-false}" == "true" ]]; }
has_passwordless_sudo() { [[ "${DOTFILES_HAS_PASSWORDLESS_SUDO:-false}" == "true" ]]; }
is_root()              { [[ "${DOTFILES_IS_ROOT:-false}" == "true" ]]; }
is_interactive()       { [[ "${DOTFILES_INTERACTIVE:-false}" == "true" ]]; }
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
            if is_root; then
                cmd=(apt-get install -y)
            elif has_sudo; then
                cmd=(sudo apt-get install -y)
            else
                log_error "apt requires sudo but sudo is not available"
                return 1
            fi
            ;;
        pacman)
            if is_root; then
                cmd=(pacman -S --noconfirm)
            elif has_sudo; then
                cmd=(sudo pacman -S --noconfirm)
            else
                log_error "pacman requires sudo but sudo is not available"
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
# GitHub / git operations
# ===========================================================================

# github_clone REPO DEST [REF]
#   Clone a GitHub repository into DEST if it does not already exist.
#
#   REPO can be:
#     "user/repo"   - explicit owner (works for any GitHub repo)
#     "repo"        - short form; prepends $DOTFILES_USER_GITHUB_USER
#
#   REF is the branch/tag to clone (default: "main").
#
#   Idempotent: does nothing if DEST already exists.
#   Dry-run safe: logs what would happen without acting.
github_clone() {
    local repo="$1" dest="$2" ref="${3:-main}"

    # Expand short-form repo name using the configured GitHub username.
    if [[ "$repo" != */* ]]; then
        if [[ -z "${DOTFILES_USER_GITHUB_USER:-}" ]]; then
            log_error "github_clone: short repo '$repo' given but DOTFILES_USER_GITHUB_USER is not set"
            return 1
        fi
        repo="${DOTFILES_USER_GITHUB_USER}/${repo}"
    fi

    if [[ -d "$dest" ]]; then
        log_info "Already cloned: $repo"
        return 0
    fi

    if is_dry_run; then
        log_info "[dry-run] Would clone github.com/${repo}@${ref} -> $dest"
        return 0
    fi

    log_info "Cloning github.com/${repo}@${ref} -> $dest"
    git clone --depth 1 --branch "$ref" "https://github.com/${repo}.git" "$dest"
    log_success "Cloned: $repo -> $dest"
}

# ===========================================================================
# File downloads
# ===========================================================================

# download_file URL DEST [SHA256]
#   Downloads URL to DEST using curl (with wget fallback).
#   If SHA256 is provided, verifies the checksum after download and aborts on
#   mismatch.  Idempotent: skips download if DEST already exists and the
#   checksum matches (or no checksum is provided).
#   Dry-run safe: logs what would happen without acting.
download_file() {
    local url="$1" dest="$2" expected_sha="${3:-}"

    # Helper: compute sha256 of a file
    _sha256() {
        if command -v sha256sum >/dev/null 2>&1; then
            sha256sum "$1" | awk '{print $1}'
        elif command -v shasum >/dev/null 2>&1; then
            shasum -a 256 "$1" | awk '{print $1}'
        else
            log_error "No sha256 tool found (tried sha256sum, shasum)"
            return 1
        fi
    }

    # If DEST already exists, check whether we can skip the download.
    if [[ -f "$dest" ]]; then
        if [[ -n "$expected_sha" ]]; then
            local actual_sha
            actual_sha="$(_sha256 "$dest")"
            if [[ "$actual_sha" == "$expected_sha" ]]; then
                log_info "Already downloaded (checksum ok): $dest"
                return 0
            else
                log_info "Checksum mismatch on cached file, re-downloading: $dest"
            fi
        else
            log_info "Already downloaded (no checksum): $dest"
            return 0
        fi
    fi

    if is_dry_run; then
        log_info "[dry-run] Would download: $url -> $dest"
        return 0
    fi

    mkdir -p "$(dirname "$dest")"

    log_info "Downloading: $url"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" -o "$dest"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$dest" "$url"
    else
        log_error "No download tool found (tried curl, wget)"
        return 1
    fi

    if [[ -n "$expected_sha" ]]; then
        local actual_sha
        actual_sha="$(_sha256 "$dest")"
        if [[ "$actual_sha" != "$expected_sha" ]]; then
            log_error "Checksum mismatch for $dest"
            log_error "  expected: $expected_sha"
            log_error "  got:      $actual_sha"
            rm -f "$dest"
            return 1
        fi
        log_info "Checksum verified: $dest"
    fi

    log_success "Downloaded: $dest"
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

    if ! is_interactive || [[ ! -t 0 ]]; then
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

    if ! is_interactive || [[ ! -t 0 ]]; then
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

    if ! is_interactive || [[ ! -t 0 ]]; then
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
