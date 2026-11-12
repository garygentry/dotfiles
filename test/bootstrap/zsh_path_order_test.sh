#!/usr/bin/env bash
# Hermetic test: the PATH block in the zshrc template keeps the inherited order.
# It used to list /usr/bin and the other system dirs ahead of the inherited
# $path, and typeset -U keeps the first copy, so on Apple Silicon /usr/bin moved
# ahead of /opt/homebrew/bin and `python3` ran the Xcode copy, not Homebrew's.
#
#   test/bootstrap/zsh_path_order_test.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

if ! command -v zsh >/dev/null 2>&1; then
    printf 'SKIP: zsh not found\n'
    exit 0
fi

PASS=0
FAIL=0
TMPROOT="$(mktemp -d)"
trap 'rm -rf "${TMPROOT}"' EXIT

# The block runs from `typeset -U path` to the `)` that closes path=( ... ).
awk '/^typeset -U path/ {on=1} on {print} on && /^\)$/ {exit}' \
    "${REPO_ROOT}/modules/zsh/zshrc.tmpl" > "${TMPROOT}/block.zsh"
if ! grep -q '^path=(' "${TMPROOT}/block.zsh" || ! grep -q '^)$' "${TMPROOT}/block.zsh"; then
    printf 'FAIL: could not find the PATH block in modules/zsh/zshrc.tmpl\n' >&2
    exit 1
fi

# resolved <inherited PATH>: the PATH the block produces, with HOME and
# DOTFILES_DIR fixed so the expected strings are stable.
resolved() {
    HOME=/h DOTFILES_DIR=/d PATH="$1" \
        /usr/bin/env zsh -f -c "source ${TMPROOT}/block.zsh; print -r -- \$PATH"
}

# check <name> <inherited PATH> <expected PATH>
check() {
    local name="$1" got
    got="$(resolved "$2")"
    if [[ "$got" == "$3" ]]; then
        printf 'PASS: %s\n' "$name"
        PASS=$((PASS + 1))
    else
        printf 'FAIL: %s\n  got:      %s\n  expected: %s\n' "$name" "$got" "$3" >&2
        FAIL=$((FAIL + 1))
    fi
}

# Apple Silicon login shell: brew shellenv put Homebrew first, and it stays ahead of /usr/bin.
check "Homebrew stays ahead of /usr/bin" \
    "/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin" \
    "/h/.local/bin:/h/bin:/usr/local/bin:/d/bin:/opt/homebrew/bin:/opt/homebrew/sbin:/usr/bin:/bin:/usr/sbin:/sbin:/usr/local/sbin"

# Root on Proxmox: zsh skips /etc/profile, so the sbin dirs are still added.
check "missing sbin dirs are added" \
    "/usr/bin:/bin" \
    "/h/.local/bin:/h/bin:/usr/local/bin:/d/bin:/usr/bin:/bin:/usr/local/sbin:/usr/sbin:/sbin"

# Linuxbrew after the system dirs keeps its place; nothing moves it either way.
check "inherited order is kept" \
    "/usr/local/bin:/usr/bin:/bin:/home/linuxbrew/.linuxbrew/bin" \
    "/h/.local/bin:/h/bin:/usr/local/bin:/d/bin:/usr/bin:/bin:/home/linuxbrew/.linuxbrew/bin:/usr/local/sbin:/usr/sbin:/sbin"

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
[[ $FAIL -eq 0 ]]
