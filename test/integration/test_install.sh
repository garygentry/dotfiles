#!/usr/bin/env bash
set -euo pipefail

DOTFILES_BIN="${DOTFILES_DIR}/bin/dotfiles"
PASS=0
FAIL=0

# --- Helpers ---

pass() {
    echo "  PASS: $1"
    PASS=$((PASS + 1))
}

fail() {
    echo "  FAIL: $1"
    FAIL=$((FAIL + 1))
}

assert_exit_zero() {
    local desc="$1"
    shift
    if "$@" > /dev/null 2>&1; then
        pass "$desc"
    else
        fail "$desc (exit code: $?)"
    fi
}

assert_output_contains() {
    local desc="$1"
    local needle="$2"
    local haystack="$3"
    if echo "$haystack" | grep -qF "$needle"; then
        pass "$desc"
    else
        fail "$desc (expected output to contain '$needle')"
    fi
}

assert_order() {
    local desc="$1"
    local first="$2"
    local second="$3"
    local output="$4"
    local pos_first pos_second
    pos_first=$(echo "$output" | grep -nF "$first" | head -1 | cut -d: -f1)
    pos_second=$(echo "$output" | grep -nF "$second" | head -1 | cut -d: -f1)
    if [ -z "$pos_first" ]; then
        fail "$desc ('$first' not found in output)"
        return
    fi
    if [ -z "$pos_second" ]; then
        fail "$desc ('$second' not found in output)"
        return
    fi
    if [ "$pos_first" -lt "$pos_second" ]; then
        pass "$desc"
    else
        fail "$desc ('$first' at line $pos_first should appear before '$second' at line $pos_second)"
    fi
}

# ==============================================================================
echo "=== Integration Tests ==="
echo ""

# --- Test 1: dotfiles list ---
echo "--- Test: dotfiles list ---"
LIST_OUTPUT=$("$DOTFILES_BIN" list 2>&1) || true

MODULES=("1password" "ssh" "git" "zsh" "neovim")
for mod in "${MODULES[@]}"; do
    assert_output_contains "list shows module '$mod'" "$mod" "$LIST_OUTPUT"
done

# --- Test 2: dotfiles install --dry-run --unattended ---
echo ""
echo "--- Test: dotfiles install --dry-run --unattended ---"
INSTALL_OUTPUT=$("$DOTFILES_BIN" install --dry-run --unattended 2>&1)
INSTALL_EXIT=$?

if [ "$INSTALL_EXIT" -eq 0 ]; then
    pass "install --dry-run --unattended exits with code 0"
else
    fail "install --dry-run --unattended exits with code 0 (got: $INSTALL_EXIT)"
fi

assert_output_contains "output contains 'Execution Plan'" "Execution Plan" "$INSTALL_OUTPUT"

for mod in "${MODULES[@]}"; do
    assert_output_contains "install output contains module '$mod'" "$mod" "$INSTALL_OUTPUT"
done

# Verify dependency order: 1password before ssh, ssh before git, git before zsh, git before neovim
assert_order "1password appears before ssh" "1password" "ssh" "$INSTALL_OUTPUT"
assert_order "ssh appears before git" "ssh" "git" "$INSTALL_OUTPUT"
assert_order "git appears before zsh" "git" "zsh" "$INSTALL_OUTPUT"
assert_order "git appears before neovim" "git" "neovim" "$INSTALL_OUTPUT"

# --- Test 3: dotfiles --help ---
echo ""
echo "--- Test: dotfiles --help ---"
HELP_OUTPUT=$("$DOTFILES_BIN" --help 2>&1)
HELP_EXIT=$?

if [ "$HELP_EXIT" -eq 0 ]; then
    pass "--help exits with code 0"
else
    fail "--help exits with code 0 (got: $HELP_EXIT)"
fi

assert_output_contains "--help shows 'install' command" "install" "$HELP_OUTPUT"
assert_output_contains "--help shows 'list' command" "list" "$HELP_OUTPUT"
assert_output_contains "--help shows 'Available Commands' or command info" "dotfiles" "$HELP_OUTPUT"

# ==============================================================================
echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi

exit 0
