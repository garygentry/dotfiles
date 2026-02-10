# Plan: Repeatable Docker Integration Tests

## Context

The existing integration tests only verify CLI output (`list`, `--dry-run`, `--help`) but never run an actual `dotfiles install`. We need end-to-end tests that spin up clean Ubuntu 22.04 and Arch Linux containers, run a real installation, and verify each module's effects (packages installed, files symlinked, configs applied). We also want GitHub Actions CI to run these automatically.

## Important: 1Password Transitive Dependency

The `ssh` module declares `dependencies: [1password]` in `modules/ssh/module.yml`. The resolver (`internal/module/resolver.go:85-117`) performs transitive dependency expansion via BFS -- requesting `[ssh, git, zsh, neovim]` in a test profile will **automatically pull in `1password`** as a transitive dependency.

**This is fine.** The 1password OS scripts install only the `op` CLI binary (no account needed). The ssh module then tries `get_secret` from 1Password, which fails (not authenticated), and falls back to SSH key generation (ssh/install.sh:33-49). All 5 modules will run and pass.

No Go code changes needed. If we later want to skip 1password entirely (to save ~30s), that would require adding optional/soft dependency support to the resolver -- a separate task.

## Files to Create/Modify

### 1. `profiles/test.yml` (NEW)

Test profile excluding 1password (resolver will add it back as transitive dep, but this communicates intent):

```yaml
modules:
  - ssh
  - git
  - zsh
  - neovim
```

### 2. `.dockerignore` (NEW)

Reduce build context and prevent cache invalidation from irrelevant changes:

```
.git
bin/
plans/
reference/
*.log
.DS_Store
.vscode/
.idea/
.terraform/
*.tfstate*
packer_cache/
*.secret
*_secret*
secrets.yml
secrets.yaml
```

### 3. `test/integration/Dockerfile.ubuntu` (MODIFY)

Changes needed:
- Add `software-properties-common` -- required by neovim's `add-apt-repository` (neovim/os/ubuntu.sh:15)
- Add `gpg` -- required by 1password's `gpg --dearmor` (1password/os/ubuntu.sh:19)
- Add `ENV DOTFILES_PROFILE=test`
- Remove redundant `COPY test/integration/test_install.sh` (already inside the repo COPY)

### 4. `test/integration/Dockerfile.arch` (MODIFY)

Changes needed:
- Add `pacman-key --init && pacman-key --populate archlinux` -- prevents GPG signature errors
- Add `openssh` and `unzip` packages -- ssh module needs `ssh-keygen`, 1password fallback needs `unzip`
- Add `ENV DOTFILES_PROFILE=test`
- Remove redundant COPY
- Keep `archlinux:latest` (pinning Arch is impractical -- rolling release tags rot quickly)

### 5. `test/integration/test_install.sh` (REWRITE ~250 lines)

Keep all existing smoke tests, then add:

**New test section: Full installation**
- Run `dotfiles install --unattended -v` (verbose for debugging on failure)
- Capture output and exit code
- Print full output only on failure

**Per-module verification assertions:**

| Module | Assertions |
|--------|-----------|
| **SSH** | `~/.ssh` dir exists with perms 700, `id_ed25519` + `.pub` exist, `~/.ssh/config` exists (template, not symlink) |
| **git** | `git config --global init.defaultBranch` = main, `push.autoSetupRemote` = true, `pull.rebase` = true, `~/.gitignore_global` is symlink, `~/.gitmessage` is symlink |
| **zsh** | `zsh` command exists, `~/.zshrc` is symlink, `~/.local/share/zinit/zinit.git` dir exists, `~/.config/zsh/aliases.zsh` + `functions.zsh` are symlinks |
| **neovim** | `nvim` command exists, `~/.config/nvim` dir exists, `~/.config/nvim/init.lua` is symlink |

**New helper functions:**
- `assert_file_exists`, `assert_dir_exists`, `assert_symlink`, `assert_command_exists`, `assert_dir_perms`, `assert_git_config`

### 6. `Makefile` (MODIFY)

- Add `DOCKER_BUILDKIT=1` for better caching
- Add `test-integration` target (runs both OS targets)
- Update `clean` to also remove Docker test images
- Keep existing target names for backward compatibility

### 7. `.github/workflows/ci.yml` (NEW)

```
Trigger: push to main, PRs to main
Concurrency: cancel in-progress runs for same branch

Jobs:
  unit-tests:
    - checkout, setup-go (from go.mod), go test -v -race ./..., go vet

  integration-tests (needs: unit-tests):
    - matrix: [ubuntu, arch]
    - fail-fast: false (OS failures are independent)
    - Docker Buildx + GHA layer caching (cache-from/to type=gha)
    - Build image, run tests
```

Docker layer caching via `docker/build-push-action` with `type=gha` cache stores layers 1-4 (base image through Go install) across runs. Only the repo COPY + build layers rebuild on code changes (~30s vs ~4min cold).

## Verification

After implementation:
1. `make test` -- unit tests still pass
2. `make test-integration-ubuntu` -- builds container, runs full install, all assertions pass
3. `make test-integration-arch` -- same for Arch
4. `make test-all` -- runs everything
5. Push branch, open PR -- GitHub Actions runs both jobs, both green

Expected test output: ~32 assertions, covering smoke tests + full install verification across both OS targets. Full run ~5-6min cold, ~2min cached.

## Manual Intervention Required

1. **Docker** -- already running on this machine (confirmed)
2. **GitHub Actions** -- must be enabled on the repo (Settings > Actions > General). Enabled by default for GitHub repos.
3. **No secrets/tokens needed** -- workflow uses only public images and repos
4. **Periodic maintenance** -- the hardcoded 1Password CLI version `v2.24.0` in `modules/1password/os/arch.sh:30` will need updating when that URL goes stale

## Flakiness Risks

| Risk | Mitigation |
|------|-----------|
| Arch mirror sync issues | `pacman-key --init` + `--populate` in Dockerfile |
| Neovim PPA unavailable | Script already falls back to base apt package (neovim/os/ubuntu.sh:15-16) |
| 1Password download URL stale (Arch) | Hardcoded v2.24.0 -- will need periodic bumps |
| Zinit GitHub clone rate-limited | Low risk; could pre-clone in Dockerfile if it becomes flaky |
| `chsh` fails in container | zsh install.sh already has `\|\| log_warn` fallback -- test does NOT assert default shell |
