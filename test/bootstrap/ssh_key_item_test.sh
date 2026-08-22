#!/usr/bin/env bash
# Hermetic test for the ssh module's configurable 1Password key item (phase 4B).
#
# No network, no root, no 1Password: we source lib/helpers.sh (as the runner
# does), stub get_secret / render_template, and source modules/ssh/install.sh
# with key_source=1password in dry-run mode. get_secret records the exact secret
# reference it was asked for, so we can assert how DOTFILES_SETTING_KEY_ITEM
# resolves to "<key_item>/<key_type>".
#
#   test/bootstrap/ssh_key_item_test.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

PASS=0
FAIL=0
fail() { printf 'FAIL: %s\n' "$*" >&2; FAIL=$((FAIL + 1)); }
pass() { printf 'PASS: %s\n' "$*"; PASS=$((PASS + 1)); }

TMPROOT="$(mktemp -d)"
trap 'rm -rf "${TMPROOT}"' EXIT

REF_FILE="${TMPROOT}/requested_ref"

# Run install.sh in an isolated subshell and capture the secret reference that
# get_secret was called with. Echoes the resolved ref (or empty on failure).
#   run_case <key_type> [key_item]
run_case() {
    local key_type="$1" key_item="${2-__UNSET__}"
    rm -f "${REF_FILE}"
    (
        set -euo pipefail
        # shellcheck source=/dev/null
        source "${REPO_ROOT}/lib/helpers.sh"

        # Stub the runner-provided seams so the script has no real side effects.
        get_secret() { printf '%s' "$1" > "${REF_FILE}"; printf 'FAKE-KEY\n'; }
        render_template() { :; }

        # Minimal runner environment for the 1password path.
        export DOTFILES_HOME="${TMPROOT}/home-${key_type}-${RANDOM}"
        export DOTFILES_MODULE_DIR="${REPO_ROOT}/modules/ssh"
        export DOTFILES_SETTING_KEY_SOURCE="1password"
        export DOTFILES_PROMPT_SSH_KEY_TYPE="${key_type}"
        export DOTFILES_DRY_RUN="true"
        if [[ "${key_item}" != "__UNSET__" ]]; then
            export DOTFILES_SETTING_KEY_ITEM="${key_item}"
        else
            unset DOTFILES_SETTING_KEY_ITEM 2>/dev/null || true
        fi

        # command -v op must succeed; provide a dummy op on PATH.
        mkdir -p "${TMPROOT}/bin"
        printf '#!/usr/bin/env bash\nexit 0\n' > "${TMPROOT}/bin/op"
        chmod +x "${TMPROOT}/bin/op"
        export PATH="${TMPROOT}/bin:${PATH}"

        # shellcheck source=/dev/null
        source "${REPO_ROOT}/modules/ssh/install.sh"
    ) >/dev/null 2>&1 || true
    cat "${REF_FILE}" 2>/dev/null || true
}

# --- Case 1: unset key_item -> neutral generic default ---------------------
got="$(run_case ed25519)"
want="op://Private/SSH Key/ed25519"
[ "${got}" = "${want}" ] && pass "default item (unset)" || fail "default item: got '${got}' want '${want}'"

# --- Case 2: unset key_item, rsa type --------------------------------------
got="$(run_case rsa)"
want="op://Private/SSH Key/rsa"
[ "${got}" = "${want}" ] && pass "default item (rsa field)" || fail "default rsa: got '${got}' want '${want}'"

# --- Case 3: custom key_item -----------------------------------------------
got="$(run_case ed25519 "op://Private/My SSH Key")"
want="op://Private/My SSH Key/ed25519"
[ "${got}" = "${want}" ] && pass "custom item" || fail "custom item: got '${got}' want '${want}'"

# --- Case 4: custom key_item with a trailing slash (no double slash) --------
got="$(run_case ed25519 "op://Vault/Item/")"
want="op://Vault/Item/ed25519"
[ "${got}" = "${want}" ] && pass "trailing slash tolerated" || fail "trailing slash: got '${got}' want '${want}'"

printf '\n%d passed, %d failed\n' "${PASS}" "${FAIL}"
[ "${FAIL}" -eq 0 ]
