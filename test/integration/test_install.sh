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

assert_file_exists() {
    local desc="$1"
    local filepath="$2"
    if [[ -f "$filepath" ]]; then
        pass "$desc"
    else
        fail "$desc (file not found: $filepath)"
    fi
}

assert_dir_exists() {
    local desc="$1"
    local dirpath="$2"
    if [[ -d "$dirpath" ]]; then
        pass "$desc"
    else
        fail "$desc (directory not found: $dirpath)"
    fi
}

assert_symlink() {
    local desc="$1"
    local filepath="$2"
    if [[ -L "$filepath" ]]; then
        pass "$desc"
    else
        fail "$desc (not a symlink: $filepath)"
    fi
}

assert_command_exists() {
    local desc="$1"
    local cmd="$2"
    if command -v "$cmd" &>/dev/null; then
        pass "$desc"
    else
        fail "$desc (command not found: $cmd)"
    fi
}

assert_dir_perms() {
    local desc="$1"
    local dirpath="$2"
    local expected="$3"
    local actual
    actual="$(stat -c '%a' "$dirpath" 2>/dev/null || stat -f '%Lp' "$dirpath" 2>/dev/null)"
    if [[ "$actual" == "$expected" ]]; then
        pass "$desc"
    else
        fail "$desc (expected perms $expected, got $actual for $dirpath)"
    fi
}

assert_git_config() {
    local desc="$1"
    local key="$2"
    local expected="$3"
    local actual
    actual="$(git config --global --get "$key" 2>/dev/null || true)"
    if [[ "$actual" == "$expected" ]]; then
        pass "$desc"
    else
        fail "$desc (expected '$expected', got '$actual' for git config $key)"
    fi
}

# ==============================================================================
echo "=== Integration Tests ==="
echo ""

# --- Test 1: dotfiles list ---
echo "--- Test: dotfiles list ---"
LIST_OUTPUT=$("$DOTFILES_BIN" list 2>&1) || true

LIST_MODULES=("ssh" "git" "zsh" "neovim")
for mod in "${LIST_MODULES[@]}"; do
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

INSTALL_MODULES=("ssh" "git" "zsh" "neovim")
for mod in "${INSTALL_MODULES[@]}"; do
    assert_output_contains "install output contains module '$mod'" "$mod" "$INSTALL_OUTPUT"
done

# Verify dependency order: ssh before git, git before zsh, git before neovim
assert_order "ssh appears before git" "ssh" "git" "$INSTALL_OUTPUT"
assert_order "git appears before zsh" "git" "zsh" "$INSTALL_OUTPUT"
assert_order "git appears before neovim" "git" "neovim" "$INSTALL_OUTPUT"

# v2.0.0: --prompt-dependencies flag is accepted
PROMPT_DEPS_OUTPUT=$("$DOTFILES_BIN" install --dry-run --unattended --prompt-dependencies 2>&1)
PROMPT_DEPS_EXIT=$?
if [ "$PROMPT_DEPS_EXIT" -eq 0 ]; then
    pass "install --dry-run --unattended --prompt-dependencies exits with code 0"
else
    fail "install --dry-run --unattended --prompt-dependencies exits with code 0 (got: $PROMPT_DEPS_EXIT)"
fi

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

# v2.0.0: install --help shows --prompt-dependencies flag
INSTALL_HELP=$("$DOTFILES_BIN" install --help 2>&1)
assert_output_contains "install help shows --prompt-dependencies" "prompt-dependencies" "$INSTALL_HELP"

# --- Test 4: Full installation ---
echo ""
echo "--- Test: dotfiles install --unattended (full install) ---"
FULL_OUTPUT=$("$DOTFILES_BIN" install --unattended -v 2>&1) || true
FULL_EXIT=$?

if [ "$FULL_EXIT" -eq 0 ]; then
    pass "install --unattended exits with code 0"
else
    fail "install --unattended exits with code 0 (got: $FULL_EXIT)"
    echo ""
    echo "--- Full install output (on failure) ---"
    echo "$FULL_OUTPUT"
    echo "--- End output ---"
fi

# --- Test 5: SSH module verification ---
echo ""
echo "--- Test: SSH module verification ---"
assert_dir_exists "~/.ssh directory exists" "$HOME/.ssh"
assert_dir_perms "~/.ssh has permissions 700" "$HOME/.ssh" "700"
assert_file_exists "~/.ssh/id_ed25519 exists" "$HOME/.ssh/id_ed25519"
assert_file_exists "~/.ssh/id_ed25519.pub exists" "$HOME/.ssh/id_ed25519.pub"
assert_file_exists "~/.ssh/config exists" "$HOME/.ssh/config"
# ssh/config is deployed as a template (regular file, not symlink)
if [[ -f "$HOME/.ssh/config" && ! -L "$HOME/.ssh/config" ]]; then
    pass "~/.ssh/config is a regular file (template, not symlink)"
else
    fail "~/.ssh/config is a regular file (template, not symlink)"
fi

# --- Test 6: Git module verification ---
echo ""
echo "--- Test: Git module verification ---"
assert_git_config "git init.defaultBranch = main" "init.defaultBranch" "main"
assert_git_config "git push.autoSetupRemote = true" "push.autoSetupRemote" "true"
assert_git_config "git pull.rebase = true" "pull.rebase" "true"
assert_symlink "~/.gitignore_global is symlink" "$HOME/.gitignore_global"
assert_symlink "~/.gitmessage is symlink" "$HOME/.gitmessage"

# --- Test 7: Zsh module verification ---
echo ""
echo "--- Test: Zsh module verification ---"
assert_command_exists "zsh is installed" "zsh"
assert_file_exists "~/.zshrc exists (rendered template)" "$HOME/.zshrc"
# zshrc is deployed as a template (regular file, not symlink)
if [[ -f "$HOME/.zshrc" && ! -L "$HOME/.zshrc" ]]; then
    pass "~/.zshrc is a regular file (template, not symlink)"
else
    fail "~/.zshrc is a regular file (template, not symlink)"
fi
assert_dir_exists "zinit directory exists" "$HOME/.local/share/zinit/zinit.git"
assert_symlink "~/.config/zsh/aliases.zsh is symlink" "$HOME/.config/zsh/aliases.zsh"
assert_symlink "~/.config/zsh/functions.zsh is symlink" "$HOME/.config/zsh/functions.zsh"

# --- Test 8: Neovim module verification ---
echo ""
echo "--- Test: Neovim module verification ---"
assert_command_exists "nvim is installed" "nvim"
assert_dir_exists "~/.config/nvim directory exists" "$HOME/.config/nvim"
assert_symlink "~/.config/nvim/init.lua is symlink" "$HOME/.config/nvim/init.lua"

# ==============================================================================
echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi

exit 0
