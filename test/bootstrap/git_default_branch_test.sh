#!/usr/bin/env bash
# Hermetic test for the git module's configurable init.defaultBranch (phase 4D).
#
# No network, no root, no real git writes: we source lib/helpers.sh (as the
# runner does), stub `git` so it records the value passed to
# `git config --global init.defaultBranch <value>`, put a fake `delta` on PATH
# so install_delta short-circuits (no pkg install / curl), and source
# modules/git/install.sh. We then assert how DOTFILES_SETTING_DEFAULT_BRANCH
# resolves (default 'main' when unset).
#
#   test/bootstrap/git_default_branch_test.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

PASS=0
FAIL=0
fail() { printf 'FAIL: %s\n' "$*" >&2; FAIL=$((FAIL + 1)); }
pass() { printf 'PASS: %s\n' "$*"; PASS=$((PASS + 1)); }

TMPROOT="$(mktemp -d)"
trap 'rm -rf "${TMPROOT}"' EXIT

BRANCH_FILE="${TMPROOT}/recorded_branch"

# Run install.sh in an isolated subshell and capture the value that
# `git config --global init.defaultBranch` was called with.
#   run_case [default_branch]
run_case() {
    local default_branch="${1-__UNSET__}"
    rm -f "${BRANCH_FILE}"
    (
        set -euo pipefail
        # shellcheck source=/dev/null
        source "${REPO_ROOT}/lib/helpers.sh"

        # Stub git: record the default branch, swallow every other config call.
        git() {
            if [[ "${1:-}" == "config" && "${3:-}" == "init.defaultBranch" ]]; then
                printf '%s' "${4:-}" > "${BRANCH_FILE}"
            fi
            return 0
        }

        # Minimal runner environment. HOME/DOTFILES_HOME point at the sandbox so
        # nothing touches the real system even if a real command slipped through.
        export DOTFILES_HOME="${TMPROOT}/home-${RANDOM}"
        export DOTFILES_MODULE_DIR="${REPO_ROOT}/modules/git"
        export DOTFILES_OS="ubuntu"
        export DOTFILES_ARCH="amd64"
        unset DOTFILES_DRY_RUN 2>/dev/null || true
        if [[ "${default_branch}" != "__UNSET__" ]]; then
            export DOTFILES_SETTING_DEFAULT_BRANCH="${default_branch}"
        else
            unset DOTFILES_SETTING_DEFAULT_BRANCH 2>/dev/null || true
        fi

        # A fake `delta` makes install_delta report "already installed" and
        # return early, so it never runs pkg_install / curl.
        mkdir -p "${TMPROOT}/bin"
        printf '#!/usr/bin/env bash\nexit 0\n' > "${TMPROOT}/bin/delta"
        chmod +x "${TMPROOT}/bin/delta"
        export PATH="${TMPROOT}/bin:${PATH}"

        # shellcheck source=/dev/null
        source "${REPO_ROOT}/modules/git/install.sh"
    ) >/dev/null 2>&1 || true
    cat "${BRANCH_FILE}" 2>/dev/null || true
}

# --- Case 1: unset default_branch -> historical default 'main' -------------
got="$(run_case)"
want="main"
[ "${got}" = "${want}" ] && pass "default branch (unset)" || fail "default: got '${got}' want '${want}'"

# --- Case 2: custom default_branch -----------------------------------------
got="$(run_case trunk)"
want="trunk"
[ "${got}" = "${want}" ] && pass "custom branch (trunk)" || fail "custom: got '${got}' want '${want}'"

# --- Case 3: another custom value ------------------------------------------
got="$(run_case develop)"
want="develop"
[ "${got}" = "${want}" ] && pass "custom branch (develop)" || fail "develop: got '${got}' want '${want}'"

printf '\n%d passed, %d failed\n' "${PASS}" "${FAIL}"
[ "${FAIL}" -eq 0 ]
