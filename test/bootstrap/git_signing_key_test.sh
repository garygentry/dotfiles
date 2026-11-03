#!/usr/bin/env bash
# Hermetic test for the git module's choice of signing key (#80).
#
# No network, no root: real git against a sandbox HOME, a fake `delta` so
# install_delta returns early, and install.sh sourced as the runner does.
#
#   test/bootstrap/git_signing_key_test.sh
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

# run_case <name> <sign_commits> <signing_key or ""> <key files to create...>
run_case() {
    local name="$1" sign="$2" signing_key="$3" home="${TMPROOT}/$1" key
    shift 3
    mkdir -p "${home}/.ssh"
    for key in "$@"; do : > "${home}/.ssh/${key}"; done
    (
        set -euo pipefail
        export HOME="$home" DOTFILES_HOME="$home" PATH="${TMPROOT}/bin:${PATH}"
        export DOTFILES_MODULE_DIR="${REPO_ROOT}/modules/git" DOTFILES_OS="ubuntu" DOTFILES_ARCH="amd64"
        export DOTFILES_SETTING_SIGN_COMMITS="$sign" DOTFILES_SETTING_SIGNING_KEY="$signing_key"
        unset DOTFILES_DRY_RUN GIT_CONFIG_GLOBAL 2>/dev/null || true
        # shellcheck source=/dev/null
        source "${REPO_ROOT}/lib/helpers.sh"
        # shellcheck source=/dev/null
        source "${REPO_ROOT}/modules/git/os/ubuntu.sh"
        # shellcheck source=/dev/null
        source "${REPO_ROOT}/modules/git/install.sh"
    ) > "${TMPROOT}/${name}.log" 2>&1 || { fail "${name}: install failed"; sed 's/^/    /' "${TMPROOT}/${name}.log" >&2; }
}
expect_signing_key() {
    local name="$1" want="$2" got
    got="$(HOME="${TMPROOT}/${name}" git config --global --get user.signingkey || true)"
    if [[ "$got" == "$want" ]]; then pass "${name}: signing key '${want}'"; else fail "${name}: signing key '${got}', want '${want}'"; fi
}

run_case rsa_only true "" id_rsa
expect_signing_key rsa_only "${TMPROOT}/rsa_only/.ssh/id_rsa.pub"
if [[ "$(HOME="${TMPROOT}/rsa_only" git config --global --get commit.gpgsign)" == "true" ]]; then
    pass "rsa_only: sign_commits turns signing on"
else
    fail "rsa_only: commit.gpgsign not set"
fi

run_case ecdsa_only false "" id_ecdsa
expect_signing_key ecdsa_only "${TMPROOT}/ecdsa_only/.ssh/id_ecdsa.pub"

run_case both false "" id_rsa id_ed25519
expect_signing_key both "${TMPROOT}/both/.ssh/id_ed25519.pub"

# shellcheck disable=SC2088  # the literal tilde is the setting under test
run_case dedicated false "~/.ssh/signing" id_rsa signing
expect_signing_key dedicated "${TMPROOT}/dedicated/.ssh/signing.pub"

run_case none_off false ""
expect_signing_key none_off ""
if grep -qF "skipping signing configuration" "${TMPROOT}/none_off.log"; then pass "none_off: warns that signing is skipped"; else fail "none_off: no warning"; fi

run_case none_on true ""
expect_signing_key none_on ""
if grep -qF "commits won't be signed" "${TMPROOT}/none_on.log"; then pass "none_on: warns that commits won't be signed"; else fail "none_on: no warning"; fi

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
[[ $FAIL -eq 0 ]]
