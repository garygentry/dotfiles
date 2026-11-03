#!/usr/bin/env bash
# Hermetic test for the git module's choice of signing key (#80).
#
# No network, no root: real git against a sandbox HOME, a fake `delta` so
# install_delta returns early, and install.sh sourced as the runner does.
#
#   test/bootstrap/git_signing_key_test.sh
set -euo pipefail

# `git config --global` must only ever see the sandbox HOME.
unset XDG_CONFIG_HOME GIT_CONFIG_GLOBAL DOTFILES_SETTING_SIGNING_KEY

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

# keys <name> <file...> — creates empty files in the case's ~/.ssh.
keys() {
    local name="$1" file; shift
    mkdir -p "${TMPROOT}/${name}/.ssh"
    for file in "$@"; do : > "${TMPROOT}/${name}/.ssh/${file}"; done
}
# block <name> [identity] — writes a managed block naming that IdentityFile, or none (agent mode).
block() {
    mkdir -p "${TMPROOT}/$1/.ssh"
    {
        echo '# >>> dotfiles managed >>>'
        echo 'Host *'
        [[ -z "${2-}" ]] || echo "    IdentityFile ~/.ssh/$2"
        echo '    ServerAliveInterval 60'
        echo '# <<< dotfiles managed <<<'
    } > "${TMPROOT}/$1/.ssh/config"
}
# run_case <name> <sign_commits> [signing_key]
run_case() {
    local name="$1" sign="$2" signing_key="${3-}" home="${TMPROOT}/$1"
    mkdir -p "$home"
    (
        set -euo pipefail
        export HOME="$home" DOTFILES_HOME="$home" PATH="${TMPROOT}/bin:${PATH}"
        export DOTFILES_MODULE_DIR="${REPO_ROOT}/modules/git" DOTFILES_OS="ubuntu" DOTFILES_ARCH="amd64"
        export DOTFILES_SETTING_SIGN_COMMITS="$sign" DOTFILES_SETTING_SIGNING_KEY="$signing_key"
        unset DOTFILES_DRY_RUN 2>/dev/null || true
        # shellcheck source=/dev/null
        source "${REPO_ROOT}/lib/helpers.sh"
        # shellcheck source=/dev/null
        source "${REPO_ROOT}/modules/git/os/ubuntu.sh"
        # shellcheck source=/dev/null
        source "${REPO_ROOT}/modules/git/install.sh"
    ) > "${TMPROOT}/${name}.log" 2>&1 || { fail "${name}: install failed"; sed 's/^/    /' "${TMPROOT}/${name}.log" >&2; }
}
git_get() { HOME="${TMPROOT}/$1" git config --global --get "$2" || true; }
expect_config() {
    local got; got="$(git_get "$1" "$2")"
    if [[ "$got" == "$3" ]]; then pass "$1: $2 is '$3'"; else fail "$1: $2 is '$got', want '$3'"; fi
}
expect_log() {
    if grep -qF -- "$2" "${TMPROOT}/$1.log"; then pass "$1: log mentions '$2'"; else fail "$1: log lacks '$2'"; fi
}
pub() { printf '%s' "${TMPROOT}/$1/.ssh/$2.pub"; }

# The ssh module reused id_rsa, or id_ecdsa: git signs with it.
keys rsa id_rsa id_rsa.pub; block rsa id_rsa; run_case rsa true
expect_config rsa user.signingkey "$(pub rsa id_rsa)"
expect_config rsa commit.gpgsign true

keys ecdsa id_ecdsa id_ecdsa.pub; block ecdsa id_ecdsa; run_case ecdsa false
expect_config ecdsa user.signingkey "$(pub ecdsa id_ecdsa)"
expect_config ecdsa commit.gpgsign ""

# The config decides, not which keys happen to exist.
keys named id_ed25519 id_ed25519.pub id_rsa id_rsa.pub; block named id_rsa; run_case named false
expect_config named user.signingkey "$(pub named id_rsa)"

# An agent host doesn't pick up a stale local key the ssh module isn't using.
keys agent_stale id_ecdsa id_ecdsa.pub; block agent_stale; run_case agent_stale true
expect_config agent_stale user.signingkey ""
expect_config agent_stale commit.gpgsign ""
expect_log agent_stale "signing isn't configured"

# With no key named, id_ed25519 is still the default.
keys agent_ed id_ed25519 id_ed25519.pub; block agent_ed; run_case agent_ed false
expect_config agent_ed user.signingkey "$(pub agent_ed id_ed25519)"

# The setting wins over the config.
keys dedicated id_rsa id_rsa.pub signing signing.pub; block dedicated id_rsa
# shellcheck disable=SC2088  # the literal tilde is the setting under test
run_case dedicated false "~/.ssh/signing"
expect_config dedicated user.signingkey "$(pub dedicated signing)"

# A missing .pub would make every commit fail, so signing isn't set up.
keys no_pub id_rsa; block no_pub id_rsa; run_case no_pub true
expect_config no_pub user.signingkey ""
expect_config no_pub commit.gpgsign ""
expect_log no_pub "signing isn't configured"

# With signing not requested, a key the host already has stays. Requesting signing replaces it.
keys keep id_rsa id_rsa.pub; block keep id_rsa
HOME="${TMPROOT}/keep" git config --global user.signingkey ABCD1234
run_case keep false
expect_config keep user.signingkey ABCD1234
expect_config keep gpg.format ""
run_case keep true
expect_config keep user.signingkey "$(pub keep id_rsa)"

# No key, signing off.
run_case none false
expect_config none user.signingkey ""
expect_log none "skipping signing configuration"

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
[[ $FAIL -eq 0 ]]
