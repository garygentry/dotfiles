#!/usr/bin/env bash
# scripts/testuser.sh - Throwaway test users for exercising the dotfiles
# installer against the LOCAL working copy (no GitHub round-trip).
#
# Must be run as root (use sudo). Test users get passwordless sudo so installs
# run fully unattended.
#
# Usage:
#   sudo scripts/testuser.sh create  <name>
#   sudo scripts/testuser.sh run     <name> [--profile P] [--force] [-- ...extra]
#   sudo scripts/testuser.sh state   <name>
#   sudo scripts/testuser.sh shell   <name>
#   sudo scripts/testuser.sh destroy <name>
#   sudo scripts/testuser.sh reset   <name>   # destroy + create
#   sudo scripts/testuser.sh list
#
# Typical loop (from the repo root, after editing a fix):
#   make build
#   sudo scripts/testuser.sh reset t1
#   sudo scripts/testuser.sh run   t1
#   sudo scripts/testuser.sh state t1

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEFAULT_PROFILE="minimal"
SUDOERS_PREFIX="/etc/sudoers.d/90-dotfiles-testuser"
MARKER=".dotfiles-testuser"   # file dropped in the home dir to tag our users

# ---------------------------------------------------------------------------
# Output
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
# Guards
# ---------------------------------------------------------------------------
require_root() {
    [ "$(id -u)" -eq 0 ] || fatal "This command must be run as root (use sudo)."
}

validate_name() {
    local name="$1"
    [ -n "$name" ] || fatal "Username required."
    [[ "$name" =~ ^[a-z_][a-z0-9_-]*$ ]] \
        || fatal "Invalid username '$name' (lowercase letters, digits, hyphens, underscores)."
}

home_of()   { echo "/home/$1"; }
is_testuser() { [ -f "$(home_of "$1")/$MARKER" ]; }

# ---------------------------------------------------------------------------
# create
# ---------------------------------------------------------------------------
cmd_create() {
    local name="$1"; validate_name "$name"
    id "$name" &>/dev/null && fatal "User '$name' already exists (use 'reset' or 'destroy')."

    info "Creating user '$name'..."
    useradd -m -s /bin/bash "$name"

    # Passwordless sudo, validated before it goes live.
    local sudoers="${SUDOERS_PREFIX}-${name}"
    echo "$name ALL=(ALL) NOPASSWD:ALL" > "$sudoers"
    chmod 440 "$sudoers"
    visudo -cf "$sudoers" >/dev/null || { rm -f "$sudoers"; fatal "Generated sudoers file failed validation."; }

    touch "$(home_of "$name")/$MARKER"
    chown "$name:$name" "$(home_of "$name")/$MARKER"

    ok "User '$name' ready (passwordless sudo). Sync + install with: sudo $0 run $name"
}

# ---------------------------------------------------------------------------
# run  - sync local repo into ~name/.dotfiles and run the installer as name
# ---------------------------------------------------------------------------
cmd_run() {
    local name="$1"; shift || true
    validate_name "$name"
    id "$name" &>/dev/null || fatal "User '$name' does not exist (run 'create' first)."

    local profile="$DEFAULT_PROFILE"
    local force="" extra=()
    while [ $# -gt 0 ]; do
        case "$1" in
            --profile) profile="$2"; shift 2 ;;
            --force)   force="--force"; shift ;;
            --)        shift; extra=("$@"); break ;;
            *)         fatal "Unknown option to 'run': $1" ;;
        esac
    done

    [ -x "$REPO_ROOT/bin/dotfiles" ] \
        || fatal "No built binary at bin/dotfiles. Run 'make build' first."

    local dest; dest="$(home_of "$name")/.dotfiles"
    info "Syncing $REPO_ROOT -> $dest ..."
    if command -v rsync &>/dev/null; then
        rsync -a --delete \
            --exclude '.git' --exclude '.state' \
            "$REPO_ROOT/" "$dest/"
    else
        warn "rsync not found; using cp (no --delete, stale files may linger)."
        mkdir -p "$dest"
        cp -a "$REPO_ROOT/." "$dest/"
        rm -rf "$dest/.git"
    fi
    chown -R "$name:$name" "$dest"

    info "Running installer as '$name' (profile: $profile, unattended)..."
    # Reset to a truly clean slate each run: drop prior state.
    su - "$name" -c "cd ~/.dotfiles && ./bin/dotfiles install --profile '$profile' --unattended $force ${extra[*]:-}" \
        && ok "Install finished for '$name'." \
        || warn "Installer exited non-zero for '$name' (expected while bugs remain — inspect with: sudo $0 state $name)."
}

# ---------------------------------------------------------------------------
# state - summarize module state files
# ---------------------------------------------------------------------------
cmd_state() {
    local name="$1"; validate_name "$name"
    local dir; dir="$(home_of "$name")/.dotfiles/.state"
    [ -d "$dir" ] || fatal "No state dir at $dir (has 'run' been executed?)."

    printf "\n${BOLD}State for '%s' (%s)${RESET}\n" "$name" "$dir"
    python3 - "$dir" <<'PY'
import json, os, sys, glob
d = sys.argv[1]
rows = []
for f in sorted(glob.glob(os.path.join(d, "*.json"))):
    try:
        j = json.load(open(f))
    except Exception as e:
        rows.append((os.path.basename(f), "PARSE-ERR", str(e))); continue
    rows.append((j.get("name","?"), j.get("status","?"), j.get("error","")))
w = max((len(r[0]) for r in rows), default=4)
for name, status, err in rows:
    mark = {"installed":"\033[0;32m✓\033[0m","failed":"\033[0;31m✗\033[0m"}.get(status,"?")
    line = f"  {mark} {name:<{w}}  {status}"
    if err:
        line += f"   \033[0;33m{err}\033[0m"
    print(line)
print()
PY
}

# ---------------------------------------------------------------------------
# shell / destroy / reset / list
# ---------------------------------------------------------------------------
cmd_shell() {
    local name="$1"; validate_name "$name"
    id "$name" &>/dev/null || fatal "User '$name' does not exist."
    info "Opening a shell as '$name' (exit to return)..."
    su - "$name"
}

cmd_destroy() {
    local name="$1"; validate_name "$name"
    if ! id "$name" &>/dev/null; then warn "User '$name' does not exist; nothing to do."; return 0; fi
    if ! is_testuser "$name"; then
        fatal "'$name' is missing the $MARKER tag — refusing to delete a user this tool did not create."
    fi
    info "Killing processes owned by '$name'..."
    pkill -u "$name" 2>/dev/null || true; sleep 1; pkill -9 -u "$name" 2>/dev/null || true
    info "Removing user '$name' and home directory..."
    userdel -r "$name" 2>/dev/null || { userdel "$name" 2>/dev/null || true; rm -rf "$(home_of "$name")"; }
    rm -f "${SUDOERS_PREFIX}-${name}"
    rm -rf "/var/mail/$name" 2>/dev/null || true
    ok "User '$name' destroyed."
}

cmd_reset() {
    local name="$1"; validate_name "$name"
    cmd_destroy "$name"
    cmd_create "$name"
}

cmd_list() {
    local found=0
    for home in /home/*; do
        [ -f "$home/$MARKER" ] || continue
        local u; u="$(basename "$home")"
        printf "  %s  (home: %s)\n" "$u" "$home"
        found=1
    done
    [ "$found" -eq 1 ] || info "No test users found."
}

# ---------------------------------------------------------------------------
# Dispatch
# ---------------------------------------------------------------------------
usage() {
    sed -n '2,30p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
}

main() {
    local sub="${1:-}"; shift || true
    case "$sub" in
        create)  require_root; cmd_create "$@" ;;
        run)     require_root; cmd_run "$@" ;;
        state)   require_root; cmd_state "$@" ;;
        shell)   require_root; cmd_shell "$@" ;;
        destroy) require_root; cmd_destroy "$@" ;;
        reset)   require_root; cmd_reset "$@" ;;
        list)    require_root; cmd_list "$@" ;;
        ""|-h|--help|help) usage ;;
        *) fatal "Unknown subcommand '$sub' (try: create run state shell destroy reset list)" ;;
    esac
}

main "$@"
