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

# --- Migration + repo-cleanliness fixtures (asserted after the full install) ---
# 1. Simulate an OLD-MODEL deployment: ~/.zshrc as a symlink straight into the
#    repo. The new model manages ~/.zshrc as a rendered template; a correct
#    install must migrate the link to a real file WITHOUT writing back through it
#    into the repo source. (On this first, stateless install existingFile==nil,
#    so the deploy — and thus the migration — runs.)
LEGACY_ZSHRC_TARGET="${DOTFILES_DIR}/modules/zsh/zshrc.tmpl"
rm -f "$HOME/.zshrc"
ln -s "$LEGACY_ZSHRC_TARGET" "$HOME/.zshrc"
# 2. Snapshot the repo's modules/ tree so we can prove the install never writes
#    back into it (the failure that dirties the checkout and breaks git pull).
#    Captured as a shell string (no temp file / diffutils dependency).
REPO_MANIFEST_BEFORE="$(cd "$DOTFILES_DIR" && find modules -type f -exec sha256sum {} + | sort)"

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
assert_file_exists "~/.ssh/config exists" "$HOME/.ssh/config"
# ssh/config is deployed as a template (regular file, not symlink)
if [[ -f "$HOME/.ssh/config" && ! -L "$HOME/.ssh/config" ]]; then
    pass "~/.ssh/config is a regular file (template, not symlink)"
else
    fail "~/.ssh/config is a regular file (template, not symlink)"
fi

# Key expectations depend on the configured key_source. Only ssh sets it, so a
# whole-file grep is sufficient (defaults to generate if absent).
SSH_KEY_SOURCE="$(grep -E '^[[:space:]]*key_source:' "${DOTFILES_DIR}/config.yml" | head -1 | awk '{print $2}' || true)"
SSH_KEY_SOURCE="${SSH_KEY_SOURCE:-generate}"
echo "  (ssh key_source=${SSH_KEY_SOURCE})"
case "$SSH_KEY_SOURCE" in
    generate|1password)
        assert_file_exists "~/.ssh/id_ed25519 exists" "$HOME/.ssh/id_ed25519"
        assert_file_exists "~/.ssh/id_ed25519.pub exists" "$HOME/.ssh/id_ed25519.pub"
        # Anchor to a real directive line so a comment mentioning the word
        # "IdentitiesOnly" is not mistaken for the directive being set.
        if grep -qE '^[[:space:]]*IdentitiesOnly[[:space:]]+yes' "$HOME/.ssh/config"; then
            pass "github config pins IdentitiesOnly (managed key)"
        else
            fail "github config pins IdentitiesOnly (managed key)"
        fi
        ;;
    agent)
        if [[ ! -f "$HOME/.ssh/id_ed25519" ]]; then
            pass "no local key generated (agent mode)"
        else
            fail "no local key generated (agent mode)"
        fi
        if grep -qE '^[[:space:]]*IdentitiesOnly[[:space:]]+yes' "$HOME/.ssh/config"; then
            fail "github config omits IdentitiesOnly (agent mode)"
        else
            pass "github config omits IdentitiesOnly (agent mode)"
        fi
        ;;
    none)
        pass "key_source=none: no key/config assertions"
        ;;
esac

# --- Test 6: Git module verification ---
echo ""
echo "--- Test: Git module verification ---"
assert_git_config "git init.defaultBranch = main" "init.defaultBranch" "main"
assert_git_config "git push.autoSetupRemote = true" "push.autoSetupRemote" "true"
# Generic engine imposes no pull strategy (opt into rebase from a content overlay).
assert_git_config "git pull.rebase not forced (generic default)" "pull.rebase" ""
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
# ~/.zshenv carries the managed user-PATH block, so NON-interactive zsh (ssh host
# cmd, zsh -c, agent tool shells) also sees ~/.local/bin.
assert_file_exists "~/.zshenv exists" "$HOME/.zshenv"
if grep -qxF "# >>> dotfiles: path >>>" "$HOME/.zshenv" 2>/dev/null; then
    pass "~/.zshenv has the managed PATH block"
else
    fail "~/.zshenv has the managed PATH block"
fi
if [[ "$(zsh -c 'print -r -- $path[1]' 2>/dev/null)" == "$HOME/.local/bin" ]]; then
    pass "non-interactive zsh puts ~/.local/bin first on PATH"
else
    fail "non-interactive zsh puts ~/.local/bin first on PATH"
fi
# Re-running the zsh install must not duplicate the block (idempotent upsert).
if [[ "$(grep -cxF "# >>> dotfiles: path >>>" "$HOME/.zshenv")" == "1" ]]; then
    pass "~/.zshenv PATH block appears exactly once"
else
    fail "~/.zshenv PATH block appears exactly once"
fi
# Interactive start is clean: no errors on stderr, and the completion dump is
# cached under XDG_CACHE_HOME (not rebuilt into $HOME on every start).
_zsh_rc=0
_zsh_err="$(zsh -i -c exit 2>&1 >/dev/null)" || _zsh_rc=$?
if [[ -z "$_zsh_err" && "$_zsh_rc" -eq 0 ]]; then
    pass "zsh -i starts with empty stderr and exit 0"
else
    fail "zsh -i starts with empty stderr and exit 0 (rc=${_zsh_rc}, stderr: ${_zsh_err})"
fi
if ls "$HOME/.cache/zsh/zcompdump-"* >/dev/null 2>&1; then
    pass "completion dump cached under ~/.cache/zsh"
else
    fail "completion dump cached under ~/.cache/zsh"
fi

# --- Test 8: Neovim module verification ---
echo ""
echo "--- Test: Neovim module verification ---"
assert_command_exists "nvim is installed" "nvim"
# The generic engine installs the neovim binary only and ships no config; a
# personal init.lua is expected to come from a content overlay, not the engine.
if [[ ! -e "$HOME/.config/nvim/init.lua" ]]; then
    pass "engine ships no init.lua (config comes from an overlay)"
else
    fail "engine unexpectedly shipped ~/.config/nvim/init.lua"
fi

# --- Test 9: Legacy symlink migration + repo cleanliness ---
echo ""
echo "--- Test: legacy symlink migration and repo cleanliness ---"

# The pre-seeded legacy ~/.zshrc symlink must have become a real file.
if [[ -f "$HOME/.zshrc" && ! -L "$HOME/.zshrc" ]]; then
    pass "legacy ~/.zshrc symlink was migrated to a regular file"
else
    fail "legacy ~/.zshrc symlink was migrated to a regular file"
fi

# The repo source the old link pointed at must not have been written through:
# it must still hold Go-template syntax, not the rendered shell output.
if grep -q '{{' "$LEGACY_ZSHRC_TARGET"; then
    pass "repo source zshrc.tmpl not clobbered via write-through"
else
    fail "repo source zshrc.tmpl not clobbered via write-through"
fi

# The install must not have modified ANY file under modules/ (no write-back).
# Compared as shell strings so no diffutils dependency (absent on the arch image).
REPO_MANIFEST_AFTER="$(cd "$DOTFILES_DIR" && find modules -type f -exec sha256sum {} + | sort)"
if [ "$REPO_MANIFEST_BEFORE" = "$REPO_MANIFEST_AFTER" ]; then
    pass "install left the repo modules/ tree byte-for-byte unchanged"
else
    fail "install modified the repo modules/ tree (write-back detected)"
    echo "--- before ---"; printf '%s\n' "$REPO_MANIFEST_BEFORE"
    echo "--- after ----"; printf '%s\n' "$REPO_MANIFEST_AFTER"
fi

# A second install is idempotent AND still leaves the repo clean.
RERUN_OUTPUT=$("$DOTFILES_BIN" install --unattended 2>&1) || true
RERUN_EXIT=$?
if [ "$RERUN_EXIT" -eq 0 ]; then
    pass "second install --unattended exits 0 (idempotent)"
else
    fail "second install --unattended exits 0 (idempotent) (got: $RERUN_EXIT)"
fi
REPO_MANIFEST_RERUN="$(cd "$DOTFILES_DIR" && find modules -type f -exec sha256sum {} + | sort)"
if [ "$REPO_MANIFEST_BEFORE" = "$REPO_MANIFEST_RERUN" ]; then
    pass "re-install still left the repo modules/ tree unchanged"
else
    fail "re-install modified the repo modules/ tree"
fi

# --- Test: mise module (opt-in; driven by a fixture content overlay) ---
# Runs LAST: it installs node through mise and changes PATH blocks. No GitHub token
# is set here, so a passing --locked install proves no API calls were needed.
echo ""
echo "--- Test: mise module + nodejs provider=mise (fixture overlay) ---"
MISE_FIXTURE="${DOTFILES_DIR}/test/integration/fixtures/mise-overlay"
MISE_OUT="$(env -u GITHUB_TOKEN -u MISE_GITHUB_TOKEN DOTFILES_CONTENT_DIR="$MISE_FIXTURE" \
    ./bin/dotfiles install --unattended mise nodejs extra-tools 2>&1)" || true
assert_output_contains "mise + nodejs + overlay tool module install succeeded" "0 failed" "$MISE_OUT"
assert_file_exists "mise binary at ~/.local/bin/mise" "$HOME/.local/bin/mise"
if [[ "$("$HOME/.local/bin/mise" --version 2>/dev/null | awk '{print $1}')" == "2026.9.14" ]]; then
    pass "mise is the pinned version"
else
    fail "mise is the pinned version"
fi
for _frag in mise nodejs extra-tools; do
    assert_file_exists "module ${_frag} declared its tools in conf.d/${_frag}.toml" "$HOME/.config/mise/conf.d/${_frag}.toml"
done
assert_file_exists "settings rendered to conf.d/dotfiles-settings.toml" "$HOME/.config/mise/conf.d/dotfiles-settings.toml"
if grep -qxF 'trusted_config_paths = ["~/workspace"]' "$HOME/.config/mise/conf.d/dotfiles-settings.toml"; then
    pass "list-valued mise setting rendered as a TOML array"
else
    fail "list-valued mise setting rendered as a TOML array"
fi
if cmp -s "$MISE_FIXTURE/mise.lock" "$HOME/.config/mise/mise.lock"; then
    pass "managed mise.lock installed from the overlay"
else
    fail "managed mise.lock installed from the overlay"
fi
_shims="$HOME/.local/share/mise/shims"
if [[ "$(zsh -c 'print -r -- $path[1]' 2>/dev/null)" == "$_shims" ]]; then
    pass "non-interactive zsh puts the mise shims first"
else
    fail "non-interactive zsh puts the mise shims first"
fi
for _t in "node:v22.19.0" "fzf:0.65.2" "rg:ripgrep 14.1.1"; do
    _bin="${_t%%:*}"; _want="${_t#*:}"
    if zsh -c "$_bin --version" 2>/dev/null | head -1 | grep -qF "$_want"; then
        pass "zsh -c resolves $_bin ($_want)"
    else
        fail "zsh -c resolves $_bin ($_want)"
    fi
done
if [[ "$(sh -lc 'command -v node' 2>/dev/null)" == "$_shims/node" ]]; then
    pass "login sh (sh -lc) resolves node through the shims"
else
    fail "login sh (sh -lc) resolves node through the shims"
fi
if [[ "$(zsh -c 'npm config get prefix' 2>/dev/null)" == "$HOME/.local" ]]; then
    pass "npm global prefix stays ~/.local under the mise provider"
else
    fail "npm global prefix stays ~/.local under the mise provider"
fi
MISE_OUT2="$(env -u GITHUB_TOKEN -u MISE_GITHUB_TOKEN DOTFILES_CONTENT_DIR="$MISE_FIXTURE" \
    ./bin/dotfiles install --unattended mise nodejs extra-tools 2>&1)" || true
assert_output_contains "mise re-install is a no-op" "0 succeeded, 0 failed" "$MISE_OUT2"
if [[ "$(grep -cxF '# >>> dotfiles: mise >>>' "$HOME/.zshenv")" == "1" ]]; then
    pass "~/.zshenv mise block appears exactly once"
else
    fail "~/.zshenv mise block appears exactly once"
fi
# A module that leaves the host stops declaring its tools: uninstall the overlay
# module, re-run mise, and its fragment must be gone (the others stay).
DOTFILES_CONTENT_DIR="$MISE_FIXTURE" ./bin/dotfiles uninstall --unattended extra-tools >/dev/null 2>&1 || true
env -u GITHUB_TOKEN -u MISE_GITHUB_TOKEN DOTFILES_CONTENT_DIR="$MISE_FIXTURE" \
    ./bin/dotfiles install --unattended --force mise >/dev/null 2>&1 || true
if [[ ! -f "$HOME/.config/mise/conf.d/extra-tools.toml" && -f "$HOME/.config/mise/conf.d/nodejs.toml" ]]; then
    pass "fragment of an uninstalled module is removed; others kept"
else
    fail "fragment of an uninstalled module is removed; others kept"
fi

# ==============================================================================
echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi

exit 0
