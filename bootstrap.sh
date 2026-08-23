#!/usr/bin/env bash
# bootstrap.sh - curl-pipeable bootstrap for garygentry/dotfiles
#
# Usage:
#   curl -sfL https://raw.githubusercontent.com/garygentry/dotfiles/main/bootstrap.sh | bash
#   curl -sfL ... | bash -s -- --unattended --dry-run
#
# Optional content overlay (personal config/profiles/modules kept OUTSIDE this
# engine repo). Any of these flags materializes a content directory and points
# the engine at it via DOTFILES_CONTENT_DIR before running the install:
#   --content-repo <git-url>   Clone (or pull) a content repo into the content dir.
#   --content-ref  <ref>       Branch/tag/commit to check out for --content-repo.
#   --content-path <path>      Use an existing local/network directory in place.
#   --content-dir  <dest>      Where --content-repo materializes (default
#                              ~/.config/dotfiles, or an already-set
#                              DOTFILES_CONTENT_DIR).
#   --content-auth-cmd <cmd>   Command run BEFORE a private clone (e.g. start an
#                              ssh-agent / unlock a credential). Ambient agents
#                              (SSH / 1Password) need no hook.
#   --no-persist-content-dir   Don't append DOTFILES_CONTENT_DIR to ~/.zshenv.
#   --no-content-update        Don't fast-forward an already-materialized content
#                              overlay; use whatever is checked out.
# An already-present content overlay (a git clone, whether from a prior
# --content-repo or a pre-set DOTFILES_CONTENT_DIR) is fast-forwarded to its
# latest by default, so re-running this script alone brings a machine fully up to
# date — engine AND personal content. The refresh is `pull --ff-only`: it never
# overwrites, so local edits or a diverged branch fail cleanly and keep the
# existing checkout. Non-git overlays and --content-path dirs are used as-is.
# Every other argument is forwarded verbatim to `dotfiles install`. With no
# content flags and no pre-set DOTFILES_CONTENT_DIR, behavior is unchanged.
#
# Note on private content repos (chicken-and-egg): the clone runs before the
# engine, so any auth material must already be present (ambient agent or
# --content-auth-cmd). Prefer a PUBLIC, secret-free content repo — real secrets
# stay in 1Password and are fetched at module-install time, never committed.
set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration (override via environment)
# ---------------------------------------------------------------------------
DOTFILES_REPO="${DOTFILES_REPO:-https://github.com/garygentry/dotfiles.git}"
DOTFILES_DIR="${DOTFILES_DIR:-$HOME/.dotfiles}"
DOTFILES_REF="${DOTFILES_REF:-main}"
GO_VERSION="${GO_VERSION:-1.23.6}"

# Content overlay acquisition (all opt-in; env vars provide non-flag defaults).
CONTENT_REPO="${DOTFILES_CONTENT_REPO:-}"
CONTENT_REF="${DOTFILES_CONTENT_REF:-}"
CONTENT_PATH="${DOTFILES_CONTENT_PATH:-}"
CONTENT_DIR_DEST="${DOTFILES_CONTENT_DIR:-$HOME/.config/dotfiles}"
CONTENT_AUTH_CMD="${DOTFILES_CONTENT_AUTH_CMD:-}"
PERSIST_CONTENT_DIR=1
# Fast-forward an already-materialized overlay clone to latest by default so a
# bare re-bootstrap refreshes personal content too. Opt out via --no-content-update.
CONTENT_UPDATE="${DOTFILES_CONTENT_UPDATE:-1}"
# Populated by acquire_content: the resolved local content dir (if any) and
# whether acquisition (clone/copy/in-place) actually ran this invocation.
RESOLVED_CONTENT_DIR=""
CONTENT_ACQUIRED=0
# Arguments forwarded to `dotfiles install` (everything not consumed above).
INSTALL_ARGS=()

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
# Run a command with sudo if needed and available
# ---------------------------------------------------------------------------
as_root() {
    if [ "$(id -u)" -eq 0 ]; then
        "$@"
    elif command -v sudo &>/dev/null; then
        sudo "$@"
    else
        fatal "Root privileges required but sudo is not installed. Please run as root or install sudo."
    fi
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
        ubuntu) as_root apt-get update -qq && as_root apt-get install -y -qq git ;;
        arch)   as_root pacman -Sy --noconfirm git ;;
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

    local go_install_dir
    if [ "$(id -u)" -eq 0 ] || (command -v sudo &>/dev/null && sudo -n true 2>/dev/null); then
        go_install_dir="/usr/local/go"
        info "Installing Go to ${go_install_dir} (with root)..."
        as_root rm -rf "${go_install_dir}"
        as_root tar -C /usr/local -xzf "${tmp}/${tarball}"
    else
        go_install_dir="${HOME}/.local/go"
        info "Installing Go to ${go_install_dir} (no sudo)..."
        rm -rf "${go_install_dir}"
        mkdir -p "${HOME}/.local"
        tar -C "${HOME}/.local" -xzf "${tmp}/${tarball}"
    fi
    rm -rf "$tmp"

    export PATH="${go_install_dir}/bin:${PATH}"
    command -v go &>/dev/null || fatal "Go installation failed"
    ok "Go installed ($(go version))"
}

# ---------------------------------------------------------------------------
# Clone or update the dotfiles repository
# ---------------------------------------------------------------------------
ensure_repo() {
    if [ -d "${DOTFILES_DIR}/.git" ]; then
        info "Dotfiles repo already exists at ${DOTFILES_DIR}, updating to origin/${DOTFILES_REF}..."
        update_repo
    else
        info "Cloning dotfiles repo to ${DOTFILES_DIR}..."
        git clone --branch "$DOTFILES_REF" "$DOTFILES_REPO" "$DOTFILES_DIR" \
            || git clone "$DOTFILES_REPO" "$DOTFILES_DIR"
        ok "Dotfiles repo cloned to ${DOTFILES_DIR}"
    fi
}

# Force the engine checkout to match origin, and FAIL LOUDLY if it can't.
#
# The repo at $DOTFILES_DIR is a managed artifact: personalization lives in the
# content overlay, never here, so nothing of value is ever lost by resetting it.
# A merge/rebase would fail on a dirty tree (legacy deployments wrote back into
# tracked files) and the old code silently continued on stale source — the exact
# bug that surfaced a wrong, truncated module list. We reset instead, but never
# silently: local drift is first captured to a recovery patch, and a fetch/reset
# that genuinely cannot complete aborts rather than building from stale source.
update_repo() {
    local ref="$DOTFILES_REF"

    if ! git -C "$DOTFILES_DIR" fetch --quiet origin "$ref"; then
        fatal "Could not fetch origin/${ref} for ${DOTFILES_DIR}. Check the network/remote and re-run — refusing to build from stale source."
    fi

    # Preserve any local drift as a recovery patch before we discard it.
    if ! git -C "$DOTFILES_DIR" diff --quiet HEAD 2>/dev/null; then
        local backup_dir patch sha
        backup_dir="${DOTFILES_DIR}/.backups"
        sha="$(git -C "$DOTFILES_DIR" rev-parse --short HEAD 2>/dev/null || echo unknown)"
        patch="${backup_dir}/pre-update-${sha}.patch"
        mkdir -p "$backup_dir"
        if git -C "$DOTFILES_DIR" diff HEAD > "$patch" 2>/dev/null && [ -s "$patch" ]; then
            warn "Local changes in ${DOTFILES_DIR} saved to ${patch} before reset"
        else
            rm -f "$patch"
        fi
    fi

    # reset --hard aligns tracked files to origin; clean -fd removes stray
    # untracked files. Neither touches gitignored paths, so runtime state
    # (.state/, .backups/) is preserved across the update.
    if ! git -C "$DOTFILES_DIR" reset --hard --quiet "origin/${ref}"; then
        fatal "Could not reset ${DOTFILES_DIR} to origin/${ref} — refusing to build from stale source."
    fi
    git -C "$DOTFILES_DIR" clean -fd --quiet -e '.backups' -e '.state' || true

    local head_sha head_subject
    head_sha="$(git -C "$DOTFILES_DIR" rev-parse --short HEAD)"
    head_subject="$(git -C "$DOTFILES_DIR" log -1 --format=%s 2>/dev/null || echo '')"
    ok "Dotfiles repo at origin/${ref} (${head_sha}: ${head_subject})"
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
    # Fall back to inherited stdin in headless environments (Docker, CI, cloud-init).
    # Note: /dev/tty can exist as a device node but fail to open when there is
    # no controlling terminal, so we test with an actual read attempt.
    if (exec < /dev/tty) 2>/dev/null; then
        exec ./bin/dotfiles install "$@" < /dev/tty
    else
        exec ./bin/dotfiles install "$@"
    fi
}

# ---------------------------------------------------------------------------
# Expand a leading ~ to $HOME (bootstrap runs before the engine, so paths from
# flags/env are not shell-expanded for us).
# ---------------------------------------------------------------------------
expand_tilde() {
    case "$1" in
        "~")    printf '%s' "$HOME" ;;
        "~/"*)  printf '%s' "$HOME/${1#\~/}" ;;
        *)      printf '%s' "$1" ;;
    esac
}

# ---------------------------------------------------------------------------
# Parse bootstrap-owned flags; forward everything else to `dotfiles install`.
# Unknown flags (e.g. --unattended, --dry-run, --profile) pass through in order,
# so existing invocations behave exactly as before.
# ---------------------------------------------------------------------------
parse_args() {
    # $# includes the flag itself, so a value in $2 requires at least 2 args.
    # Guarding before touching $2 turns a fat-fingered flag into a clear error
    # instead of a raw "$2: unbound variable" from set -u.
    need_val() { [ "$1" -ge 2 ] || fatal "${2} requires a value"; }
    while [ $# -gt 0 ]; do
        case "$1" in
            --content-repo)       need_val $# "$1"; CONTENT_REPO="$2";     shift 2 ;;
            --content-repo=*)     CONTENT_REPO="${1#*=}";      shift ;;
            --content-ref)        need_val $# "$1"; CONTENT_REF="$2";      shift 2 ;;
            --content-ref=*)      CONTENT_REF="${1#*=}";       shift ;;
            --content-path)       need_val $# "$1"; CONTENT_PATH="$2";     shift 2 ;;
            --content-path=*)     CONTENT_PATH="${1#*=}";      shift ;;
            --content-dir)        need_val $# "$1"; CONTENT_DIR_DEST="$2"; shift 2 ;;
            --content-dir=*)      CONTENT_DIR_DEST="${1#*=}";  shift ;;
            --content-auth-cmd)   need_val $# "$1"; CONTENT_AUTH_CMD="$2"; shift 2 ;;
            --content-auth-cmd=*) CONTENT_AUTH_CMD="${1#*=}";  shift ;;
            --no-persist-content-dir) PERSIST_CONTENT_DIR=0;   shift ;;
            --no-content-update)      CONTENT_UPDATE=0;        shift ;;
            --)                   shift; INSTALL_ARGS+=("$@"); break ;;
            *)                    INSTALL_ARGS+=("$1");        shift ;;
        esac
    done
}

# ---------------------------------------------------------------------------
# Optional pre-fetch auth hook, run BEFORE any private clone. Ambient agents
# (SSH / 1Password) need nothing here; this exists for setups that must start or
# unlock a credential first.
# ---------------------------------------------------------------------------
run_content_auth() {
    [ -n "$CONTENT_AUTH_CMD" ] || return 0
    info "Running content auth hook before fetch..."
    bash -c "$CONTENT_AUTH_CMD" || fatal "content auth hook failed: ${CONTENT_AUTH_CMD}"
    ok "Content auth hook complete"
}

# ---------------------------------------------------------------------------
# Materialize the content overlay directory from the requested source and set
# RESOLVED_CONTENT_DIR. Three source kinds, all resolving to one local dir:
#   * --content-path : an existing local/network directory, used in place.
#   * --content-repo : a git repo, cloned (or pulled, idempotently) into the dest.
#   * neither        : honor a pre-set DOTFILES_CONTENT_DIR, else do nothing.
# ---------------------------------------------------------------------------
acquire_content() {
    if [ -n "$CONTENT_REPO" ] && [ -n "$CONTENT_PATH" ]; then
        fatal "--content-repo and --content-path are mutually exclusive"
    fi

    # Source kind: existing local/network path used in place.
    if [ -n "$CONTENT_PATH" ]; then
        local src
        src="$(expand_tilde "$CONTENT_PATH")"
        [ -d "$src" ] || fatal "--content-path ${src} is not a directory"
        RESOLVED_CONTENT_DIR="$(cd "$src" && pwd)"
        CONTENT_ACQUIRED=1
        ok "Using content directory in place: ${RESOLVED_CONTENT_DIR}"
        return
    fi

    # Source kind: git repo cloned/pulled into the destination content dir.
    if [ -n "$CONTENT_REPO" ]; then
        local dest
        dest="$(expand_tilde "$CONTENT_DIR_DEST")"
        run_content_auth
        if [ -d "${dest}/.git" ]; then
            info "Content repo already present at ${dest}, updating..."
            git -C "$dest" fetch --all --prune || warn "content fetch failed; using existing checkout"
            if [ -n "$CONTENT_REF" ]; then
                git -C "$dest" checkout "$CONTENT_REF" || fatal "content checkout ${CONTENT_REF} failed"
                git -C "$dest" merge --ff-only "@{u}" 2>/dev/null || true
            else
                git -C "$dest" pull --ff-only || warn "content pull failed; using existing checkout"
            fi
        else
            [ -e "$dest" ] && [ ! -d "$dest" ] && fatal "content dir ${dest} exists and is not a directory"
            info "Cloning content repo ${CONTENT_REPO} -> ${dest}"
            mkdir -p "$(dirname "$dest")"
            git clone "$CONTENT_REPO" "$dest" || fatal "content clone failed: ${CONTENT_REPO}"
            if [ -n "$CONTENT_REF" ]; then
                git -C "$dest" checkout "$CONTENT_REF" || fatal "content checkout ${CONTENT_REF} failed"
            fi
        fi
        RESOLVED_CONTENT_DIR="$(cd "$dest" && pwd)"
        CONTENT_ACQUIRED=1
        ok "Content repo ready at ${RESOLVED_CONTENT_DIR}"
        return
    fi

    # No acquisition requested: honor a pre-set DOTFILES_CONTENT_DIR, and (by
    # default) fast-forward it if it's a git clone so a bare re-bootstrap also
    # brings the overlay to latest.
    if [ -n "${DOTFILES_CONTENT_DIR:-}" ]; then
        RESOLVED_CONTENT_DIR="$(expand_tilde "$DOTFILES_CONTENT_DIR")"
        refresh_overlay "$RESOLVED_CONTENT_DIR"
    fi
}

# refresh_overlay DIR
#   Fast-forward an existing overlay CLONE to its upstream so re-running the
#   bootstrap alone refreshes personal content. Deliberately non-destructive:
#   `pull --ff-only` advances only on a clean fast-forward and otherwise warns,
#   so local edits or a diverged branch are never clobbered. No-ops for a
#   non-git dir, a detached/upstream-less checkout, --no-content-update, or when
#   offline/unauthenticated (falls back to the existing checkout).
refresh_overlay() {
    local dir="$1"
    [ "$CONTENT_UPDATE" -eq 1 ] || return 0
    [ -d "${dir}/.git" ] || return 0
    git -C "$dir" rev-parse --abbrev-ref '@{u}' >/dev/null 2>&1 || return 0

    info "Refreshing content overlay at ${dir} (ff-only)..."
    if git -C "$dir" fetch --quiet --all --prune 2>/dev/null; then
        if git -C "$dir" -c advice.diverging=false pull --ff-only --quiet 2>/dev/null; then
            ok "Content overlay updated to latest"
        else
            warn "content overlay is not fast-forwardable (local edits or diverged branch); using existing checkout"
        fi
    else
        warn "content overlay fetch failed (offline or auth?); using existing checkout"
    fi
}

# ---------------------------------------------------------------------------
# Make DOTFILES_CONTENT_DIR persist for future shells. Only acts when we
# actually acquired content this run; otherwise the user's existing env already
# carries it. Idempotent: never appends a second export line.
# ---------------------------------------------------------------------------
persist_content_dir() {
    [ "$CONTENT_ACQUIRED" -eq 1 ] || return 0
    [ -n "$RESOLVED_CONTENT_DIR" ] || return 0

    local line="export DOTFILES_CONTENT_DIR=\"${RESOLVED_CONTENT_DIR}\""
    printf '\n'
    info "For future shells, set the content directory with:"
    printf '  %s\n' "$line"

    [ "$PERSIST_CONTENT_DIR" -eq 1 ] || return 0
    local target="${HOME}/.zshenv"
    if [ -f "$target" ] && grep -qF 'DOTFILES_CONTENT_DIR=' "$target"; then
        info "${target} already sets DOTFILES_CONTENT_DIR; leaving it unchanged."
        return
    fi
    if printf '\n# Added by dotfiles bootstrap\n%s\n' "$line" >> "$target"; then
        ok "Persisted DOTFILES_CONTENT_DIR to ${target}"
    else
        warn "could not write ${target}; add the export line above to your shell env manually"
    fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    printf "${BOLD}garygentry/dotfiles bootstrap${RESET}\n"
    printf "%s\n\n" "-------------------------------"

    parse_args "$@"

    detect_platform
    ensure_git
    ensure_go
    ensure_repo

    # Materialize the optional content overlay (git repo / local path) and point
    # the engine at it. No content source + no pre-set var => unchanged behavior.
    acquire_content
    if [ -n "$RESOLVED_CONTENT_DIR" ]; then
        export DOTFILES_CONTENT_DIR="$RESOLVED_CONTENT_DIR"
        info "Content overlay: ${DOTFILES_CONTENT_DIR}"
    fi
    persist_content_dir

    build_and_run ${INSTALL_ARGS[@]+"${INSTALL_ARGS[@]}"}
}

# Run main only when executed/piped, not when sourced for testing.
if [ -z "${DOTFILES_BOOTSTRAP_LIB:-}" ]; then
    main "$@"
fi
