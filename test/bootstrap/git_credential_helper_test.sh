#!/usr/bin/env bash
# Hermetic test for the git module's opt-in credential.helper (#61).
#
# No network, no root: real git against a sandbox HOME (so --global writes the
# sandbox's ~/.gitconfig), a fake `delta` so install_delta returns early, and
# the module's OS script plus install.sh sourced as the runner does.
#
#   test/bootstrap/git_credential_helper_test.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

PASS=0
FAIL=0
fail() { printf 'FAIL: %s\n' "$*" >&2; FAIL=$((FAIL + 1)); }
pass() { printf 'PASS: %s\n' "$*"; PASS=$((PASS + 1)); }

TMPROOT="$(mktemp -d)"
trap 'rm -rf "${TMPROOT}"' EXIT

mkdir -p "${TMPROOT}/bin"
printf '#!/usr/bin/env bash\nexit 0\n' > "${TMPROOT}/bin/delta"
chmod +x "${TMPROOT}/bin/delta"

# run_case <name> <os> <setting or __UNSET__> [existing helper]
#   Echoes the global credential.helper values afterwards, one per line.
run_case() {
    local name="$1" os="$2" setting="$3" existing="${4-}"
    local home="${TMPROOT}/${name}"
    mkdir -p "$home"
    if [[ -n "$existing" ]]; then HOME="$home" git config --global --add credential.helper "$existing"; fi
    (
        set -euo pipefail
        export HOME="$home" DOTFILES_HOME="$home" PATH="${TMPROOT}/bin:${PATH}"
        export DOTFILES_MODULE_DIR="${REPO_ROOT}/modules/git" DOTFILES_OS="$os" DOTFILES_ARCH="amd64"
        unset DOTFILES_DRY_RUN GIT_CONFIG_GLOBAL 2>/dev/null || true
        if [[ "$setting" != "__UNSET__" ]]; then
            export DOTFILES_SETTING_CREDENTIAL_HELPER="$setting"
        else
            unset DOTFILES_SETTING_CREDENTIAL_HELPER 2>/dev/null || true
        fi
        # shellcheck source=/dev/null
        source "${REPO_ROOT}/lib/helpers.sh"
        # shellcheck source=/dev/null
        source "${REPO_ROOT}/modules/git/os/${os}.sh"
        # shellcheck source=/dev/null
        source "${REPO_ROOT}/modules/git/install.sh"
    ) > "${TMPROOT}/${name}.log" 2>&1 || { fail "${name}: install failed"; sed 's/^/    /' "${TMPROOT}/${name}.log" >&2; }
    HOME="$home" git config --global --get-all credential.helper || true
}

expect() {
    local desc="$1" got="$2" want="$3"
    if [[ "$got" == "$want" ]]; then pass "$desc"; else fail "${desc}: got '${got}', want '${want}'"; fi
}

for os in ubuntu arch; do
    expect "${os}: unset sets no helper" "$(run_case "${os}-unset" "$os" __UNSET__)" ""
    expect "${os}: unset keeps the host's helper" \
        "$(run_case "${os}-keep" "$os" __UNSET__ "store")" "store"
done
expect "configured value is set" "$(run_case set ubuntu "cache --timeout=900")" "cache --timeout=900"
mkdir -p "${TMPROOT}/multi"
HOME="${TMPROOT}/multi" git config --global --add credential.helper a
expect "configured value replaces every existing helper" \
    "$(run_case multi ubuntu "osxkeychain" "b")" "osxkeychain"

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
