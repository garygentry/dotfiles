#!/usr/bin/env bash
# Hermetic tests for bootstrap.sh content-overlay acquisition.
#
# No network and no root: a bare git repo on the local filesystem stands in for
# a "remote", and a plain directory stands in for a local/network content path.
# The script is sourced as a library (DOTFILES_BOOTSTRAP_LIB=1) so we can call
# acquire_content / parse_args directly without running the full installer.
#
#   test/bootstrap/acquire_test.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
# shellcheck source=test/bootstrap/lib.sh
source "${SCRIPT_DIR}/lib.sh"

export DOTFILES_BOOTSTRAP_LIB=1
# shellcheck source=/dev/null
source "${REPO_ROOT}/bootstrap.sh"

PASS=0
FAIL=0
fail() { printf 'FAIL: %s\n' "$*" >&2; FAIL=$((FAIL + 1)); }
pass() { printf 'PASS: %s\n' "$*"; PASS=$((PASS + 1)); }

TMPROOT="$(mktemp -d)"
trap 'rm -rf "${TMPROOT}"' EXIT

# Case output goes to case.log, printed only on failure. Never call these from a
# condition: see lib.sh.
expect_ok() {
    run_errexit "$2" > "${TMPROOT}/case.log" 2>&1
    if [ "${ERREXIT_RC}" -eq 0 ]; then
        pass "$1"
    else
        fail "$1 (rc=${ERREXIT_RC})"
        cat "${TMPROOT}/case.log" >&2
    fi
}
# expect_fails <desc> <function> <message>: the failure must be the expected one, since
# under errexit any failing command would also make the case exit non-zero.
expect_fails() {
    run_errexit "$2" > "${TMPROOT}/case.log" 2>&1
    if [ "${ERREXIT_RC}" -ne 0 ] && grep -qF -- "$3" "${TMPROOT}/case.log"; then
        pass "$1"
    else
        fail "$1 (rc=${ERREXIT_RC})"
        cat "${TMPROOT}/case.log" >&2
    fi
}

# Isolate HOME so persistence writes to a throwaway ~/.zshenv.
export HOME="${TMPROOT}/home"
mkdir -p "${HOME}"

# Reset all acquisition globals between cases (they are set once at source time).
reset_content_vars() {
    CONTENT_REPO=""
    CONTENT_REF=""
    CONTENT_PATH=""
    CONTENT_DIR_DEST="${HOME}/.config/dotfiles"
    CONTENT_AUTH_CMD=""
    PERSIST_CONTENT_DIR=1
    RESOLVED_CONTENT_DIR=""
    CONTENT_ACQUIRED=0
    unset DOTFILES_CONTENT_DIR 2>/dev/null || true
}

git_quiet() { git -c user.email=t@example.com -c user.name=tester "$@"; }

# Build a bare repo (the "remote") seeded on main with a config.yml.
make_remote() {
    local remote="$1" seed="${TMPROOT}/seed"
    git -c init.defaultBranch=main init -q --bare "${remote}"
    git_quiet clone -q "${remote}" "${seed}"
    echo "user: {name: seeded}" > "${seed}/config.yml"
    git_quiet -C "${seed}" add -A
    git_quiet -C "${seed}" commit -qm "init"
    git_quiet -C "${seed}" branch -M main
    git_quiet -C "${seed}" push -q -u origin main
    rm -rf "${seed}"
}

# --- Case 1: local path used in place --------------------------------------
case_1() {
    reset_content_vars
    src="${TMPROOT}/local-content"
    mkdir -p "${src}"
    echo "user: {}" > "${src}/config.yml"
    CONTENT_PATH="${src}"
    acquire_content
    [ "${RESOLVED_CONTENT_DIR}" = "${src}" ] || { echo "resolved=${RESOLVED_CONTENT_DIR}"; exit 1; }
    [ "${CONTENT_ACQUIRED}" -eq 1 ] || exit 1
}
expect_ok "local path used in place" case_1

# --- Case 2: nonexistent local path is rejected ----------------------------
case_2() {
    reset_content_vars
    CONTENT_PATH="${TMPROOT}/does-not-exist"
    acquire_content
}
expect_fails "nonexistent path rejected" case_2 "--content-path ${TMPROOT}/does-not-exist is not a directory"

# --- Case 3: git repo cloned into the destination --------------------------
REMOTE="${TMPROOT}/remote.git"
make_remote "${REMOTE}"
DEST="${TMPROOT}/dest"
case_3() {
    reset_content_vars
    CONTENT_REPO="${REMOTE}"
    CONTENT_DIR_DEST="${DEST}"
    acquire_content
    [ -d "${DEST}/.git" ] || exit 1
    [ -f "${DEST}/config.yml" ] || exit 1
    [ "${RESOLVED_CONTENT_DIR}" = "${DEST}" ] || exit 1
}
expect_ok "git repo cloned into dest" case_3

# --- Case 4: idempotent re-run pulls instead of re-cloning -----------------
case_4() {
    reset_content_vars
    CONTENT_REPO="${REMOTE}"
    CONTENT_DIR_DEST="${DEST}"
    acquire_content
    [ -d "${DEST}/.git" ] || exit 1
    [ -f "${DEST}/config.yml" ] || exit 1
}
expect_ok "idempotent re-run updates existing clone" case_4

# --- Case 5: auth hook runs before the clone -------------------------------
AUTH_DEST="${TMPROOT}/auth-dest"
AUTH_MARKER="${TMPROOT}/auth-ran"
case_5() {
    reset_content_vars
    CONTENT_REPO="${REMOTE}"
    CONTENT_DIR_DEST="${AUTH_DEST}"
    # Marker must not exist yet, and must exist after the clone completes.
    CONTENT_AUTH_CMD="test ! -d '${AUTH_DEST}/.git' && touch '${AUTH_MARKER}'"
    acquire_content
    [ -f "${AUTH_MARKER}" ] || exit 1
    [ -d "${AUTH_DEST}/.git" ] || exit 1
}
expect_ok "auth hook runs before clone" case_5

# --- Case 6: --content-repo and --content-path are mutually exclusive -------
case_6() {
    reset_content_vars
    CONTENT_REPO="${REMOTE}"
    CONTENT_PATH="${TMPROOT}/local-content"
    acquire_content
}
expect_fails "repo+path mutually exclusive" case_6 "mutually exclusive"

# --- Case 7: no source + no env var => no content dir (unchanged behavior) --
case_7() {
    reset_content_vars
    acquire_content
    [ -z "${RESOLVED_CONTENT_DIR}" ] || exit 1
    [ "${CONTENT_ACQUIRED}" -eq 0 ] || exit 1
}
expect_ok "no source leaves content dir unset" case_7

# --- Case 8: pre-set DOTFILES_CONTENT_DIR is honored, no acquisition --------
case_8() {
    reset_content_vars
    preset="${TMPROOT}/preset"
    mkdir -p "${preset}"
    export DOTFILES_CONTENT_DIR="${preset}"
    acquire_content
    [ "${RESOLVED_CONTENT_DIR}" = "${preset}" ] || exit 1
    [ "${CONTENT_ACQUIRED}" -eq 0 ] || exit 1
}
expect_ok "pre-set DOTFILES_CONTENT_DIR honored" case_8

# --- Case 9: parse_args splits content flags from install args -------------
case_9() {
    reset_content_vars
    INSTALL_ARGS=()
    parse_args --content-repo "${REMOTE}" --unattended --profile developer --content-ref main
    [ "${CONTENT_REPO}" = "${REMOTE}" ] || exit 1
    [ "${CONTENT_REF}" = "main" ] || exit 1
    [ "${#INSTALL_ARGS[@]}" -eq 3 ] || { echo "install args: ${INSTALL_ARGS[*]}"; exit 1; }
    [ "${INSTALL_ARGS[0]}" = "--unattended" ] || exit 1
    [ "${INSTALL_ARGS[1]}" = "--profile" ] || exit 1
    [ "${INSTALL_ARGS[2]}" = "developer" ] || exit 1
}
expect_ok "parse_args separates content and install args" case_9

# --- Case 10: persistence writes ~/.zshenv once (idempotent) ---------------
case_10() {
    reset_content_vars
    rm -f "${HOME}/.zshenv"
    RESOLVED_CONTENT_DIR="${TMPROOT}/local-content"
    CONTENT_ACQUIRED=1
    persist_content_dir >/dev/null
    persist_content_dir >/dev/null
    n="$(grep -cF 'DOTFILES_CONTENT_DIR=' "${HOME}/.zshenv" || true)"
    [ "${n}" -eq 1 ] || { echo "count=${n}"; exit 1; }
}
expect_ok "persistence appends ~/.zshenv exactly once" case_10

# --- Case 11: --no-persist-content-dir skips writing ~/.zshenv --------------
case_11() {
    reset_content_vars
    rm -f "${HOME}/.zshenv"
    PERSIST_CONTENT_DIR=0
    RESOLVED_CONTENT_DIR="${TMPROOT}/local-content"
    CONTENT_ACQUIRED=1
    persist_content_dir >/dev/null
    [ ! -f "${HOME}/.zshenv" ] || exit 1
}
expect_ok "--no-persist-content-dir skips ~/.zshenv" case_11

# --- Case 12: value-taking flag with no value fails cleanly (not set -u crash) -
case_12() {
    reset_content_vars
    INSTALL_ARGS=()
    parse_args --content-repo
}
expect_fails "trailing value flag fails with clear error" case_12 "--content-repo requires a value"

printf '\n%d passed, %d failed\n' "${PASS}" "${FAIL}"
[ "${FAIL}" -eq 0 ]
