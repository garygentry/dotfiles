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
# shellcheck source=test/bootstrap/lib.sh
source "${SCRIPT_DIR}/lib.sh"

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
    run_errexit install_case > "${TMPROOT}/${name}.log" 2>&1
    [ "$ERREXIT_RC" -eq 0 ] || { fail "${name}: install.sh failed"; sed 's/^/    /' "${TMPROOT}/${name}.log" >&2; }
}

# Runs in run_errexit, and sees run_case's locals.
install_case() {
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

# 3b. Same, with github_host: false, where the old block has no GitHub comment to go by.
run_case was_agent_no_github 2 '# >>> dotfiles managed >>>
Host *
    ServerAliveInterval 60
# <<< dotfiles managed <<<'
expect_mode was_agent_no_github agent
expect_no_key was_agent_no_github

# 3c. An old block from generate mode names a key, so it stays generate.
run_case was_generate_legacy 2 '# >>> dotfiles managed >>>
Host *
    IdentityFile ~/.ssh/id_ed25519
# <<< dotfiles managed <<<'
expect_mode was_generate_legacy generate

# 3d. An old block with the agent comment stays agent, even with a hand-added key elsewhere.
run_case was_agent_extra 2 '# >>> dotfiles managed >>>
Host work
    IdentityFile ~/.ssh/deploy
Host github.com
    # key_source=agent: keys are provided by the SSH agent (e.g. 1Password).
# <<< dotfiles managed <<<'
expect_mode was_agent_extra agent
expect_no_key was_agent_extra

# 3e. An empty block says nothing about agent use, so it isn't treated as agent mode.
run_case was_empty 2 '# >>> dotfiles managed >>>
# <<< dotfiles managed <<<'
expect_mode was_empty generate

# 4. First install, no agent -> generate, with a loud warning naming the key.
run_case first 2
expect_mode first generate
if [[ -f "${TMPROOT}/first/.ssh/id_ed25519" ]]; then pass "first: key generated"; else fail "first: no key generated"; fi
expect_log first "generated a NEW passphrase-less key: ${TMPROOT}/first/.ssh/id_ed25519 (SHA256:"
if grep -q 'AddKeysToAgent yes' "${TMPROOT}/first/.ssh/config"; then pass "first: generate-mode config keeps AddKeysToAgent"; else fail "first: AddKeysToAgent missing in generate mode"; fi

# run_verify <name> <key_source> — runs verify.sh against the case's HOME; returns its
# status. A child bash keeps errexit on even when the caller tests the result with `if`.
run_verify() {
    local name="$1" source="$2" home="${TMPROOT}/$1"
    (
        export HOME="$home" DOTFILES_HOME="$home"
        export DOTFILES_DIR="$REPO_ROOT" DOTFILES_BIN="${TMPROOT}/dotfiles"
        export DOTFILES_MODULE_DIR="${REPO_ROOT}/modules/ssh" DOTFILES_MODULE_NAME="ssh"
        export DOTFILES_SETTING_KEY_SOURCE="$source" DOTFILES_DRY_RUN="false"
        bash -euo pipefail -c 'source "$1"; source "$2"' _ \
            "${REPO_ROOT}/lib/helpers.sh" "${REPO_ROOT}/modules/ssh/verify.sh"
    ) > "${TMPROOT}/${name}.verify.log" 2>&1
}
expect_verify_ok() {
    if run_verify "$1" "$2"; then
        pass "$1: verify passes with key_source=$2"
    else
        fail "$1: verify failed with key_source=$2"
        sed 's/^/    /' "${TMPROOT}/$1.verify.log" >&2
    fi
}
expect_verify_fails() {
    if run_verify "$1" "$2"; then fail "$1: verify passed with key_source=$2"; else pass "$1: verify fails with key_source=$2"; fi
}

# 5. Only an RSA or ECDSA key, no agent -> install reuses it and verify accepts it.
for type in rsa ecdsa; do
    name="${type}_only"
    mkdir -p "${TMPROOT}/${name}/.ssh"
    ssh-keygen -q -t "$type" -N "" -f "${TMPROOT}/${name}/.ssh/id_${type}"
    run_case "$name" 2
    expect_mode "$name" generate
    expect_no_key "$name"
    expect_verify_ok "$name" auto
    expect_verify_ok "$name" generate
done

# 6. The key is gone -> verify fails. No GitHub block, so only the local key check can.
export DOTFILES_SETTING_GITHUB_HOST=false
run_case gone 2
rm -f "${TMPROOT}/gone/.ssh/id_ed25519"
expect_verify_fails gone auto

# 7. The key the config names is gone but another exists -> verify fails, so the runner
# re-runs the module, and that run points the config at the other key.
run_case dangling 2
rm -f "${TMPROOT}/dangling/.ssh/id_ed25519"
ssh-keygen -q -t rsa -b 2048 -N "" -f "${TMPROOT}/dangling/.ssh/id_rsa"
expect_verify_fails dangling auto
run_case dangling 2
expect_verify_ok dangling auto

# 8. A "# key_source:" comment outside the managed block is ignored.
run_case outside 2 '# key_source: none'
rm -f "${TMPROOT}/outside/.ssh/id_ed25519"
expect_verify_fails outside auto
unset DOTFILES_SETTING_GITHUB_HOST

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
[[ $FAIL -eq 0 ]]
