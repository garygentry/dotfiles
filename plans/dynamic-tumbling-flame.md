# Fix Clean-User Installation Issues

## Context
Running `bootstrap.sh` on a clean test user (no sudo, no prior config) exposed five issues: sudo requirement blocks bootstrap, git-delta unavailable via apt on Ubuntu, delta failure kills the entire git module, script error output appears twice, and git user config values never reach install scripts.

## Changes

### 1. Bootstrap Go install without sudo
**File**: `bootstrap.sh` (lines 107-114)

Check for passwordless sudo via `sudo -n true`; if unavailable, install Go to `$HOME/.local/go` instead of `/usr/local/go`.

```bash
if command -v sudo &>/dev/null && sudo -n true 2>/dev/null; then
    # install to /usr/local/go with sudo
else
    # install to $HOME/.local/go without sudo
fi
export PATH="${go_install_dir}/bin:${PATH}"
```

### 2. git-delta: install from GitHub releases on Ubuntu
**File**: `modules/git/install.sh` (lines 74-92)

Replace `pkg_install git-delta` with an `install_delta` function that:
- On Ubuntu: downloads `.deb` from `https://github.com/dandavison/delta/releases/download/<ver>/git-delta_<ver>_<arch>.deb` and installs via `sudo dpkg -i` (requires `has_sudo` check)
- On other platforms: uses existing `pkg_install git-delta`
- Available debs: `git-delta_0.18.2_{amd64,arm64,armhf,i386}.deb`

### 3. Make delta non-fatal
**File**: `modules/git/install.sh` (same section as #2)

Wrap delta install in an `if` guard so `set -e` doesn't propagate:
```bash
if install_delta; then
    # configure delta as pager
else
    log_warn "git-delta installation failed (optional), continuing without it"
fi
```
This works because `set -e` does not trigger on commands used as `if` conditions.

### 4. Fix output duplication
**File**: `internal/module/runner.go` — `runScript()` (lines 296-303)

Currently the error embeds full script output: `fmt.Errorf("script %s failed: %w\n%s", ..., output)`. This gets printed once at failure (runner.go:122) and again in the summary (install.go:140).

Fix: On failure, print output once (via `cfg.UI.Info` if not already shown by verbose mode), then return a clean error without embedded output:
```go
if err != nil {
    if len(output) > 0 && !cfg.Verbose {
        cfg.UI.Info(string(output))
    }
    return fmt.Errorf("script %s failed: %w", filepath.Base(scriptPath), err)
}
```

### 5. Inject user config as env vars for scripts
**Files**: `internal/module/runner.go` — `buildEnvVars()`, `modules/git/install.sh`, `lib/helpers.sh`

**Problem**: `config.yml` user values are available in template context but never exported as env vars. Scripts can't access them.

**Fix**:
- In `buildEnvVars()` (after line 227): add `DOTFILES_USER_NAME`, `DOTFILES_USER_EMAIL`, `DOTFILES_USER_GITHUB_USER` from `cfg.Config.User.*`
- In `modules/git/install.sh`: change `DOTFILES_PROMPT_USER_NAME` → `DOTFILES_USER_NAME` (same for email)
- In `lib/helpers.sh`: document the new env vars in the header comment
- In `internal/module/runner_test.go` `TestBuildEnvVars`: add assertions for the new vars (test config already sets `Name: "Test User"` etc.)

## Verification
1. `go test ./internal/module/...` — confirms env var injection and clean error format
2. Test on clean user without sudo: `sudo useradd -m -s /bin/bash dottest && sudo -iu dottest` then run bootstrap — Go should install to `~/.local/go`
3. Test git module: run `dotfiles install git` — delta should install from .deb on Ubuntu, git config should succeed even if delta fails
4. Verify no output duplication on module failure
