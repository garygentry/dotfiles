#!/usr/bin/env bash
# Hermetic test: the python module asks each package manager for names it has.
# python3-pip is Debian's name; Homebrew and pacman have no such package, and
# asking for it failed the whole module.
#
#   test/bootstrap/python_packages_test.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
# shellcheck source=test/bootstrap/lib.sh
source "${SCRIPT_DIR}/lib.sh"

PASS=0
FAIL=0
TMPROOT="$(mktemp -d)"
trap 'rm -rf "${TMPROOT}"' EXIT

run_install() {
    log_info() { :; }
    log_success() { :; }
    pkg_install() { printf '%s\n' "$*" > "${TMPROOT}/asked"; }
    # shellcheck source=modules/python/install.sh
    source "${REPO_ROOT}/modules/python/install.sh"
}

# check <package manager> <expected packages>
check() {
    local mgr="$1" expected="$2" asked
    rm -f "${TMPROOT}/asked"
    DOTFILES_PKG_MGR="$mgr"
    export DOTFILES_PKG_MGR
    run_errexit run_install
    asked="$(cat "${TMPROOT}/asked" 2>/dev/null || true)"
    if [[ "$ERREXIT_RC" -eq 0 && "$asked" == "$expected" ]]; then
        printf 'PASS: %s installs: %s\n' "$mgr" "$expected"
        PASS=$((PASS + 1))
    else
        printf 'FAIL: %s: rc=%s, asked for "%s", expected "%s"\n' "$mgr" "$ERREXIT_RC" "$asked" "$expected" >&2
        FAIL=$((FAIL + 1))
    fi
}

check brew "python3"
check apt "python3 python3-pip"
check pacman "python python-pip"

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
[[ $FAIL -eq 0 ]]
