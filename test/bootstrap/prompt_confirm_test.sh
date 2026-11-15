#!/usr/bin/env bash
# Hermetic test: prompt_confirm reads y/yes/n/no in any case, and falls back to
# the default on an empty or unknown reply. It used ${reply,,}, which macOS's
# /bin/bash 3.2 rejects as a bad substitution, failing any script that asked.
#
#   test/bootstrap/prompt_confirm_test.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

PASS=0
FAIL=0
TMPROOT="$(mktemp -d)"
trap 'rm -rf "${TMPROOT}"' EXIT

pass() { printf 'PASS: %s\n' "$1"; PASS=$((PASS + 1)); }
fail() { printf 'FAIL: %s\n' "$1" >&2; FAIL=$((FAIL + 1)); }

# lib/helpers.sh must stay free of bash 4 case conversion; CI's bash is 5, so
# this static check is what catches it before a Mac does.
case_conv="$(grep -nE '\$\{[A-Za-z_][A-Za-z0-9_]*(\[[^]]*\])?(,,?|\^\^?)[^}]*\}' "${REPO_ROOT}/lib/helpers.sh" \
    | grep -vE '^[0-9]+:[[:space:]]*#' || true)"
if [[ -n "$case_conv" ]]; then
    printf '%s\n' "$case_conv" >&2
    fail "lib/helpers.sh uses \${var,,} or \${var^^} (bash 4 only)"
else
    pass "lib/helpers.sh has no bash 4 case conversion"
fi

# prompt_confirm reads only from a terminal, so each case runs under script(1)
# for a pty, with the reply typed on its stdin.
if ! command -v script >/dev/null 2>&1; then
    printf 'SKIP: script not found\n'
    printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
    [[ $FAIL -eq 0 ]]
    exit
fi

cat > "${TMPROOT}/ask.sh" <<EOF
source "${REPO_ROOT}/lib/helpers.sh"
DOTFILES_INTERACTIVE=true
if prompt_confirm "Continue?" "\$1"; then echo "RESULT=yes"; else echo "RESULT=no"; fi
EOF

# check <reply> <default> <expected yes|no>
check() {
    local reply="$1" default="$2" expected="$3" out got
    out="$(printf '%s\n' "$reply" | script -qec "bash ${TMPROOT}/ask.sh ${default}" /dev/null 2>&1 || true)"
    got="$(printf '%s' "$out" | tr -d '\r' | sed -n 's/^.*RESULT=//p' | tail -1)"
    if [[ "$got" == "$expected" ]]; then
        pass "reply '${reply}' with default ${default} -> ${expected}"
    else
        fail "reply '${reply}' with default ${default}: expected ${expected}, got '${got}'. Output: ${out}"
    fi
}

check y     false yes
check YES   false yes
check Yes   false yes
check n     true  no
check NO    true  no
check ""    true  yes
check ""    false no
check maybe true  yes
check maybe false no

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
[[ $FAIL -eq 0 ]]
