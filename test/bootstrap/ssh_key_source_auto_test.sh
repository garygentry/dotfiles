#!/usr/bin/env bash
# Hermetic test for the ssh module's key_source=auto resolution (#57).
#
# No network, no root, no real agent: we source lib/helpers.sh (as the runner
# does), put a fake ssh-add on PATH whose exit code we choose, render the real
# config template with a freshly built engine binary, and source
# modules/ssh/install.sh against a temp HOME.
#
#   test/bootstrap/ssh_key_source_auto_test.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

PASS=0
FAIL=0
fail() { printf 'FAIL: %s\n' "$*" >&2; FAIL=$((FAIL + 1)); }
pass() { printf 'PASS: %s\n' "$*"; PASS=$((PASS + 1)); }

TMPROOT="$(mktemp -d)"
trap 'rm -rf "${TMPROOT}"' EXIT

(cd "$REPO_ROOT" && go build -o "${TMPROOT}/dotfiles" .)

# run_case <name> <ssh-add exit code> [existing ~/.ssh/config content]
#   Runs install.sh with key_source=auto; leaves HOME at ${TMPROOT}/<name> and
#   the output in ${TMPROOT}/<name>.log.
run_case() {
    local name="$1" agent_rc="$2" existing="${3-}"
    local home="${TMPROOT}/${name}"
    mkdir -p "${home}/.ssh" "${TMPROOT}/bin-${name}"
    chmod 700 "${home}/.ssh"
    [[ -z "$existing" ]] || printf '%s\n' "$existing" > "${home}/.ssh/config"
    printf '#!/usr/bin/env bash\n[ %s -eq 0 ] && echo "256 SHA256:fake agent-key (ED25519)"\nexit %s\n' \
        "$agent_rc" "$agent_rc" > "${TMPROOT}/bin-${name}/ssh-add"
    chmod +x "${TMPROOT}/bin-${name}/ssh-add"
    (
        set -euo pipefail
        export PATH="${TMPROOT}/bin-${name}:${PATH}"
        export HOME="$home" DOTFILES_HOME="$home"
        export DOTFILES_DIR="$REPO_ROOT" DOTFILES_BIN="${TMPROOT}/dotfiles"
        export DOTFILES_MODULE_DIR="${REPO_ROOT}/modules/ssh" DOTFILES_MODULE_NAME="ssh"
        export DOTFILES_SETTING_KEY_SOURCE="auto" DOTFILES_DRY_RUN="false"
        export DOTFILES_USER_EMAIL="test@example.invalid"
        # shellcheck source=/dev/null
        source "${REPO_ROOT}/lib/helpers.sh"
        # shellcheck source=/dev/null
        source "${REPO_ROOT}/modules/ssh/install.sh"
    ) > "${TMPROOT}/${name}.log" 2>&1 || { fail "${name}: install.sh failed"; sed 's/^/    /' "${TMPROOT}/${name}.log" >&2; }
}

expect_mode() {
    local name="$1" want="$2"
    local got
    got="$(awk '/^# key_source: / { print $3; exit }' "${TMPROOT}/${name}/.ssh/config" 2>/dev/null || true)"
    if [[ "$got" == "$want" ]]; then pass "${name}: recorded key_source ${want}"; else fail "${name}: recorded key_source '${got}', want ${want}"; fi
}
expect_no_key() {
    if [[ ! -e "${TMPROOT}/$1/.ssh/id_ed25519" ]]; then pass "$1: no local key generated"; else fail "$1: a local key was generated"; fi
}
expect_log() {
    if grep -qF -- "$2" "${TMPROOT}/$1.log"; then pass "$1: log mentions '$2'"; else fail "$1: log lacks '$2'"; fi
}

agent_block='# >>> dotfiles managed >>>
# key_source: agent
Host *
# <<< dotfiles managed <<<'
legacy_agent_block='# >>> dotfiles managed >>>
Host github.com
    # key_source=agent: keys are provided by the SSH agent (e.g. 1Password).
# <<< dotfiles managed <<<'

# 1. Live agent with a key -> agent; config has no AddKeysToAgent/IdentityFile.
run_case live 0
expect_mode live agent
expect_no_key live
expect_log live "ssh-add -l found 1 key(s)"
if grep -qE '^[[:space:]]*(AddKeysToAgent|IdentityFile)[[:space:]]' "${TMPROOT}/live/.ssh/config"; then
    fail "live: agent-mode config sets AddKeysToAgent or IdentityFile"
else
    pass "live: agent-mode config omits AddKeysToAgent and IdentityFile"
fi

# 2. Agent unreachable on a host that used agent mode -> keep agent, warn.
run_case was_agent 2 "$agent_block"
expect_mode was_agent agent
expect_no_key was_agent
expect_log was_agent "NOT generating a local key"

# 3. Same, with a block written before the key_source line existed.
run_case was_agent_legacy 2 "$legacy_agent_block"
expect_mode was_agent_legacy agent
expect_no_key was_agent_legacy

# 4. First install, no agent -> generate, with a loud warning naming the key.
run_case first 2
expect_mode first generate
if [[ -f "${TMPROOT}/first/.ssh/id_ed25519" ]]; then pass "first: key generated"; else fail "first: no key generated"; fi
expect_log first "generated a NEW passphrase-less key: ${TMPROOT}/first/.ssh/id_ed25519 (SHA256:"
if grep -q 'AddKeysToAgent yes' "${TMPROOT}/first/.ssh/config"; then pass "first: generate-mode config keeps AddKeysToAgent"; else fail "first: AddKeysToAgent missing in generate mode"; fi

# run_verify <name> <key_source> — runs verify.sh against the case's HOME; returns its status.
run_verify() {
    local name="$1" source="$2" home="${TMPROOT}/$1"
    (
        set -euo pipefail
        export HOME="$home" DOTFILES_HOME="$home"
        export DOTFILES_DIR="$REPO_ROOT" DOTFILES_BIN="${TMPROOT}/dotfiles"
        export DOTFILES_MODULE_DIR="${REPO_ROOT}/modules/ssh" DOTFILES_MODULE_NAME="ssh"
        export DOTFILES_SETTING_KEY_SOURCE="$source" DOTFILES_DRY_RUN="false"
        # shellcheck source=/dev/null
        source "${REPO_ROOT}/lib/helpers.sh"
        # shellcheck source=/dev/null
        source "${REPO_ROOT}/modules/ssh/verify.sh"
    ) > "${TMPROOT}/${name}.verify.log" 2>&1
}

# 5. Only an RSA key, no agent -> install reuses it and verify accepts it, for auto and generate.
mkdir -p "${TMPROOT}/rsa_only/.ssh"
ssh-keygen -q -t rsa -b 2048 -N "" -f "${TMPROOT}/rsa_only/.ssh/id_rsa"
run_case rsa_only 2
expect_mode rsa_only generate
expect_no_key rsa_only
for source in auto generate; do
    if run_verify rsa_only "$source"; then
        pass "rsa_only: verify passes with key_source=${source}"
    else
        fail "rsa_only: verify failed with key_source=${source}"
        sed 's/^/    /' "${TMPROOT}/rsa_only.verify.log" >&2
    fi
done

# 6. The key is gone -> verify still fails.
rm -f "${TMPROOT}/rsa_only/.ssh/id_rsa"
if run_verify rsa_only auto; then
    fail "no_key: verify passed with no local key"
else
    pass "no_key: verify fails with no local key"
fi

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
[[ $FAIL -eq 0 ]]
