#!/usr/bin/env bash
# bootstrap.sh - curl-pipeable bootstrap for garygentry/dotfiles
# Usage: curl -sfL https://raw.githubusercontent.com/garygentry/dotfiles/main/bootstrap.sh | bash
#        curl -sfL ... | bash -s -- --unattended --dry-run
set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration (override via environment)
# ---------------------------------------------------------------------------
DOTFILES_REPO="${DOTFILES_REPO:-https://github.com/garygentry/dotfiles.git}"
DOTFILES_DIR="${DOTFILES_DIR:-$HOME/.dotfiles}"
GO_VERSION="${GO_VERSION:-1.23.6}"

# ---------------------------------------------------------------------------
# Color helpers (disabled when not connected to a terminal)
# ---------------------------------------------------------------------------
if [ -t 1 ]; then
    RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'
    BLUE='\033[0;34m'; BOLD='\033[1m'; RESET='\033[0m'
else
    RED=''; GREEN=''; YELLOW=''; BLUE=''; BOLD=''; RESET=''
fi

info()  { printf "${BLUE}${BOLD}[info]${RESET}  %s\n" "$*"; }
ok()    { printf "${GREEN}${BOLD}[ok]${RESET}    %s\n" "$*"; }
warn()  { printf "${YELLOW}${BOLD}[warn]${RESET}  %s\n" "$*" >&2; }
fatal() { printf "${RED}${BOLD}[error]${RESET} %s\n" "$*" >&2; exit 1; }

# ---------------------------------------------------------------------------
# OS / architecture detection
# ---------------------------------------------------------------------------
detect_platform() {
    case "$(uname -s)" in
        Darwin) OS="darwin" ;;
        Linux)
            if [ -f /etc/os-release ]; then
                . /etc/os-release
                case "${ID:-}" in
                    ubuntu|debian) OS="ubuntu" ;;
                    arch|manjaro)  OS="arch"   ;;
                    *)             OS="linux"  ;;
                esac
            else
                OS="linux"
            fi
            ;;
        *) fatal "Unsupported operating system: $(uname -s)" ;;
    esac

    case "$(uname -m)" in
        x86_64)       ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *) fatal "Unsupported architecture: $(uname -m)" ;;
    esac

    # Go download uses "linux" regardless of distro
    if [ "$OS" != "darwin" ]; then
        GO_OS="linux"
    else
        GO_OS="darwin"
    fi

    info "Detected platform: ${OS}/${ARCH}"
}

# ---------------------------------------------------------------------------
# Install git if missing
# ---------------------------------------------------------------------------
ensure_git() {
    if command -v git &>/dev/null; then
        ok "git is already installed ($(git --version))"
        return
    fi
    info "Installing git..."
    case "$OS" in
        darwin) xcode-select --install 2>/dev/null || true ;;
        ubuntu) sudo apt-get update -qq && sudo apt-get install -y -qq git ;;
        arch)   sudo pacman -Sy --noconfirm git ;;
        *)      fatal "Cannot install git automatically on ${OS}. Please install git and retry." ;;
    esac
    command -v git &>/dev/null || fatal "git installation failed"
    ok "git installed ($(git --version))"
}

# ---------------------------------------------------------------------------
# Install Go if missing
# ---------------------------------------------------------------------------
ensure_go() {
    if command -v go &>/dev/null; then
        ok "go is already installed ($(go version))"
        return
    fi

    local tarball="go${GO_VERSION}.${GO_OS}-${ARCH}.tar.gz"
    local url="https://go.dev/dl/${tarball}"
    local tmp
    tmp="$(mktemp -d)"

    info "Downloading Go ${GO_VERSION} for ${GO_OS}/${ARCH}..."
    if command -v curl &>/dev/null; then
        curl -sfL -o "${tmp}/${tarball}" "$url"
    elif command -v wget &>/dev/null; then
        wget -qO "${tmp}/${tarball}" "$url"
    else
        fatal "Neither curl nor wget found. Cannot download Go."
    fi

    info "Installing Go to /usr/local/go..."
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf "${tmp}/${tarball}"
    rm -rf "$tmp"

    export PATH="/usr/local/go/bin:${PATH}"
    command -v go &>/dev/null || fatal "Go installation failed"
    ok "Go installed ($(go version))"
}

# ---------------------------------------------------------------------------
# Clone or update the dotfiles repository
# ---------------------------------------------------------------------------
ensure_repo() {
    if [ -d "${DOTFILES_DIR}/.git" ]; then
        info "Dotfiles repo already exists at ${DOTFILES_DIR}, pulling latest..."
        git -C "$DOTFILES_DIR" pull --ff-only || warn "git pull failed; continuing with existing checkout"
        ok "Dotfiles repo updated"
    else
        info "Cloning dotfiles repo to ${DOTFILES_DIR}..."
        git clone "$DOTFILES_REPO" "$DOTFILES_DIR"
        ok "Dotfiles repo cloned to ${DOTFILES_DIR}"
    fi
}

# ---------------------------------------------------------------------------
# Build and execute the dotfiles binary
# ---------------------------------------------------------------------------
build_and_run() {
    info "Building dotfiles binary..."
    cd "$DOTFILES_DIR"
    go build -o bin/dotfiles .
    ok "Binary built at ${DOTFILES_DIR}/bin/dotfiles"

    info "Running: dotfiles install $*"
    # Redirect stdin from /dev/tty so the Go binary gets an interactive
    # terminal even when this script is piped from curl (curl | bash).
    # Fall back to inherited stdin in headless environments (Docker, CI).
    if [ -e /dev/tty ]; then
        exec ./bin/dotfiles install "$@" < /dev/tty
    else
        exec ./bin/dotfiles install "$@"
    fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    printf "${BOLD}garygentry/dotfiles bootstrap${RESET}\n"
    printf "%s\n\n" "-------------------------------"

    detect_platform
    ensure_git
    ensure_go
    ensure_repo
    build_and_run "$@"
}

main "$@"
