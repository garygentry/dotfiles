# Dotfiles Management System - Phase 1 Implementation Plan

## Context

Build a highly flexible, configurable dotfiles management system from scratch. The system uses a Go CLI binary for orchestration (config parsing, dependency resolution, template rendering, secret management, pretty output) with shell-based modules for actual install logic (package installs, symlinks, OS-specific commands). This approach gives robust orchestration with easy-to-author modules.

**Target**: Working end-to-end system on macOS, Ubuntu, and Arch Linux with 5 core modules: 1password, ssh, git, zsh, neovim.

## Repository Structure

```
/home/gary/workspace/dotfiles/
  go.mod
  main.go                              # CLI entry point
  bootstrap.sh                         # Curl-pipeable bootstrap
  config.yml                           # Default user configuration
  profiles/
    minimal.yml
    developer.yml
  lib/
    helpers.sh                         # Shell helper library for modules
  cmd/
    dotfiles/
      root.go                          # Root cobra command + persistent flags
      install.go                       # install subcommand (5-phase flow)
      list.go                          # list subcommand
      render_template.go               # Hidden: called by shell helpers
      get_secret.go                    # Hidden: called by shell helpers
  internal/
    config/
      config.go                        # YAML config + profile parsing
      config_test.go
    sysinfo/
      sysinfo.go                       # OS/arch/sudo/pkg-mgr detection
      sysinfo_test.go
    module/
      schema.go                        # module.yml types + parsing
      discovery.go                     # Scan modules/ directory
      resolver.go                      # Topological sort dependency resolution
      runner.go                        # Execute modules (env vars, shell out)
      schema_test.go
      resolver_test.go
      runner_test.go
    secrets/
      provider.go                      # SecretProvider interface + factory
      onepassword.go                   # 1Password CLI implementation
      noop.go                          # No-op fallback
      provider_test.go
    template/
      render.go                        # Go text/template rendering
      render_test.go
    state/
      state.go                         # Per-module JSON state tracking
      state_test.go
    ui/
      ui.go                            # Spinners, colors, prompts, summaries
      ui_test.go
  modules/
    1password/
      module.yml, install.sh, os/{macos,ubuntu,arch}.sh, verify.sh, uninstall.sh
    ssh/
      module.yml, install.sh, os/{macos,ubuntu,arch}.sh, config/config.tpl, verify.sh, uninstall.sh
    git/
      module.yml, install.sh, os/{macos,ubuntu,arch}.sh, config/{gitignore_global,commit_template}, verify.sh, uninstall.sh
    zsh/
      module.yml, install.sh, os/{macos,ubuntu,arch}.sh, config/{.zshrc,zsh/}, verify.sh, uninstall.sh
    neovim/
      module.yml, install.sh, os/{macos,ubuntu,arch}.sh, config/{init.lua,...}, verify.sh, uninstall.sh
  test/
    integration/
      Dockerfile.ubuntu
      Dockerfile.arch
      test_install.sh
```

## Implementation Steps

### Step 1: Go Project Scaffolding + CLI Skeleton

**Files**: `go.mod`, `main.go`, `cmd/dotfiles/{root,install,list}.go`

- Initialize Go module as `github.com/garygentry/dotfiles` (Go 1.22+, cobra for CLI, gopkg.in/yaml.v3)
- Root command with persistent flags: `--verbose`, `--dry-run`
- Stub `install` command with flags: `--unattended`, `--fail-fast`, positional module args
- Stub `list` command

**Verify**: `go build -o bin/dotfiles . && ./bin/dotfiles --help`

### Step 2: System Detection (`internal/sysinfo/`)

**Files**: `internal/sysinfo/sysinfo.go`, `internal/sysinfo/sysinfo_test.go`

- `SystemInfo` struct: OS, Arch, PkgMgr, HasSudo, User, HomeDir, DotfilesDir, IsInteractive
- OS detection: `runtime.GOOS` for macOS, parse `/etc/os-release` ID for Linux
- Arch: `runtime.GOARCH`
- PkgMgr: brew (macOS), apt (Ubuntu), pacman (Arch)
- Sudo: test `sudo -n true` with timeout
- `Detect() (*SystemInfo, error)` called once at startup

**Verify**: `go test ./internal/sysinfo/ -v`

### Step 3: Configuration Parsing (`internal/config/`)

**Files**: `internal/config/config.go`, `internal/config/config_test.go`, `config.yml`, `profiles/{minimal,developer}.yml`

- `Config` struct with Profile, DotfilesDir, Secrets, User, Modules (map[string]any)
- `Load(dotfilesDir)` parses config.yml
- `LoadProfile(dotfilesDir, name)` parses profile YAML (list of module names)
- Environment variable overrides: `DOTFILES_PROFILE`, `DOTFILES_SECRETS_PROVIDER`
- `GetModuleSetting(moduleName, key)` helper for module-specific config

**Verify**: `go test ./internal/config/ -v`

### Step 4: Module Schema + Discovery (`internal/module/`)

**Files**: `internal/module/schema.go`, `internal/module/discovery.go`, `internal/module/schema_test.go`

- `Module` struct: Name, Description, Version, Priority, Dependencies, OS, Requires, Files, Prompts, Tags, Dir (runtime)
- `ParseModuleYAML(path)` unmarshals module.yml
- `Discover(modulesDir)` scans directories, returns sorted by priority
- `SupportsOS(os)` filter method

**Verify**: `go test ./internal/module/ -v`

### Step 5: Dependency Resolution (`internal/module/resolver.go`)

**Files**: `internal/module/resolver.go`, `internal/module/resolver_test.go`

- `ExecutionPlan` struct: Modules (ordered), Skipped (OS-incompatible)
- `Resolve(modules, requested, osName)` implements Kahn's algorithm (BFS topological sort)
- Auto-includes transitive dependencies for requested modules
- Within same topological level, sort by Priority then name
- Cycle detection with descriptive error (full cycle path)
- Missing dependency detection

**Verify**: `go test ./internal/module/ -v -run TestResolve`

### Step 6: UI Output (`internal/ui/`)

**Files**: `internal/ui/ui.go`, `internal/ui/ui_test.go`

- Catppuccin Mocha color palette (ANSI RGB escape codes)
- Braille spinner (`Spinner` struct with goroutine + done channel)
- Log methods: Info, Warn, Error, Success, Debug (verbose only)
- Spinner methods: StartSpinner, StopSpinnerSuccess/Fail/Skip
- Prompt methods: PromptInput, PromptConfirm, PromptChoice
- `PrintExecutionPlan` and `PrintModuleSummary` for structured output
- Non-TTY mode: strip colors, disable spinners, prefix with `[INFO]`/`[WARN]`

**Verify**: `go test ./internal/ui/ -v`

### Step 7: Secret Provider (`internal/secrets/`)

**Files**: `internal/secrets/{provider,onepassword,noop}.go`, `internal/secrets/provider_test.go`

- `Provider` interface: Name, Available, Authenticate, IsAuthenticated, GetSecret(ref)
- `NewProvider(name, cfg)` factory
- `OnePasswordProvider`: calls `op` CLI via exec.Command with 30s timeout
  - `Available()`: `exec.LookPath("op")`
  - `Authenticate()`: `op --account X vault list`
  - `GetSecret(ref)`: `op --account X read 'ref'`, capture stdout
- `NoopProvider`: returns errors for GetSecret (used when no provider configured)

**Verify**: `go test ./internal/secrets/ -v`

### Step 8: Template Rendering (`internal/template/`)

**Files**: `internal/template/render.go`, `internal/template/render_test.go`

- `Context` struct: User, OS, Arch, Home, DotfilesDir, Module (map), Secrets (map), Env (map)
- `Render(templatePath, ctx)` and `RenderString(templateStr, ctx)`
- Custom template functions: `env`, `default`, `upper`, `lower`, `contains`, `join`
- Uses Go `text/template` (not html/template)
- Templates use `.tpl` extension

**Verify**: `go test ./internal/template/ -v`

### Step 9: State Tracking (`internal/state/`)

**Files**: `internal/state/state.go`, `internal/state/state_test.go`

- `Store` with dir at `~/.dotfiles/.state/`
- `ModuleState`: Name, Version, Status, InstalledAt, UpdatedAt, OS, Error, Checksum
- One JSON file per module: `.state/<name>.json`
- Methods: Get, Set, GetAll, Remove

**Verify**: `go test ./internal/state/ -v`

### Step 10: Shell Helpers Library (`lib/helpers.sh`)

**Files**: `lib/helpers.sh`

~200 lines of bash providing the module API:
- **Logging**: log_info, log_warn, log_error, log_success (color-aware via DOTFILES_INTERACTIVE)
- **OS checks**: is_macos, is_ubuntu, is_arch, has_sudo, is_interactive, is_dry_run
- **Packages**: pkg_install, pkg_installed (dispatches to brew/apt/pacman based on DOTFILES_PKG_MGR)
- **Files**: link_file (symlink + backup), copy_file (copy + backup)
- **Templates**: render_template (calls `dotfiles render-template` subprocess)
- **Secrets**: get_secret (calls `dotfiles get-secret` subprocess)
- **Prompts**: prompt_input, prompt_confirm, prompt_choice (no-op in unattended mode)
- All file operations respect dry-run mode

**Verify**: `shellcheck lib/helpers.sh`

### Step 11: Module Runner (`internal/module/runner.go`)

**Files**: `internal/module/runner.go`, `internal/module/runner_test.go`

The core orchestration engine. For each module in the execution plan:
1. Handle prompts (interactive) or use defaults (unattended)
2. Build env vars map (all `DOTFILES_*` vars + prompt answers as `DOTFILES_PROMPT_<KEY>`)
3. Run `os/<os>.sh` if exists (via `bash -c` wrapper that sources helpers.sh first)
4. Run `install.sh` (same wrapper)
5. Deploy `files` entries (symlink/copy/template)
6. Run `verify.sh` if exists
7. Record result in state store

Script execution: wraps each script with `set -euo pipefail` + `source helpers.sh`, runs via `exec.Command("bash", "-c", wrapperScript)`, streams stdout/stderr through UI.

On failure: records as failed, continues to next module (unless `--fail-fast`).

**Verify**: `go test ./internal/module/ -v -run TestRunner`

### Step 12: CLI Internal Subcommands

**Files**: `cmd/dotfiles/render_template.go`, `cmd/dotfiles/get_secret.go`

Hidden cobra commands (not shown in help) that shell helpers call back into:
- `dotfiles render-template --src <path> --module <dir>` — reads context from env vars, renders template to stdout
- `dotfiles get-secret --ref <ref>` — loads config, initializes provider, writes secret to stdout

**Verify**: Manual test with env vars set

### Step 13: Wire Up install + list Commands

**Files**: Update `cmd/dotfiles/{install,list,root}.go`

Replace stubs with full implementations:
- `install` RunE: 5-phase flow (config -> secrets -> resolve -> execute -> summary)
- `list` RunE: discover modules, read state, print table (Name, Description, OS, Status)
- `root` init: register all subcommands

**Verify**: `go build -o bin/dotfiles . && ./bin/dotfiles install --dry-run && ./bin/dotfiles list`

### Step 14: Five Core Modules

**Files**: All files under `modules/{1password,ssh,git,zsh,neovim}/`

Dependency chain: **1password -> ssh -> git -> zsh, neovim**

| Module | Priority | Key Actions |
|--------|----------|-------------|
| 1password | 10 | Install op CLI, authenticate, deploy agent.toml |
| ssh | 20 | Retrieve keys from 1Password (or generate), deploy ~/.ssh/config template |
| git | 30 | Configure git (user, SSH signing, delta, aliases), deploy gitignore + commit template |
| zsh | 40 | Install zsh, zinit, set default shell, symlink .zshrc + modular configs |
| neovim | 50 | Install neovim, symlink config dir (lazy.nvim self-installs on first launch) |

Each module has: module.yml, install.sh, os/{macos,ubuntu,arch}.sh, verify.sh, uninstall.sh, config files.

**Verify**: `./bin/dotfiles install --dry-run`

### Step 15: Bootstrap Script

**Files**: `bootstrap.sh`

~80 lines. Curl-pipeable (`curl -sfL ... | bash`):
1. Detect OS + arch
2. Install git if missing
3. Install Go if missing (download from go.dev)
4. Clone repo to ~/.dotfiles (or git pull if exists)
5. `go build -o bin/dotfiles .`
6. `exec dotfiles install "$@"`

Supports `--unattended` passthrough for IaC.

**Verify**: `shellcheck bootstrap.sh && bash bootstrap.sh --dry-run`

### Step 16: Integration Testing

**Files**: `test/integration/{Dockerfile.ubuntu,Dockerfile.arch,test_install.sh}`, `Makefile`

- Dockerfiles for clean Ubuntu 22.04 and Arch environments
- Test scripts run `dotfiles install --unattended --dry-run` and `dotfiles list`
- Makefile targets: `test` (unit), `test-integration-ubuntu`, `test-integration-arch`

**Verify**: `make test && make test-integration-ubuntu`

## Key Architecture Patterns

**Go <-> Shell Communication**:
- Go -> Shell: environment variables (`DOTFILES_*`)
- Shell -> Go: subprocess calls (`dotfiles render-template`, `dotfiles get-secret`) via stdout/exit code
- Shell -> Go: exit codes for success/failure

**Idempotency**: `pkg_install` checks before installing, `link_file` checks symlink target, `git config` is naturally idempotent, state tracking records last-run checksums.

**Error Handling**: Scripts run with `set -euo pipefail`. Non-zero exit = module failure. Runner catches and continues (unless `--fail-fast`). Summary shows all results.

**Execution Modes**: Interactive (prompts for optional inputs), Unattended (uses defaults, no prompts, for IaC), Dry-run (shows plan without changes).

## Verification Plan

1. After each step, run `go test ./...` to verify no regressions
2. After Step 13, `dotfiles install --dry-run` should show the full execution plan
3. After Step 14, `dotfiles install --dry-run` should list all 5 modules in dependency order
4. After Step 15, `bootstrap.sh --dry-run` should work on a clean system
5. After Step 16, Docker integration tests pass on Ubuntu and Arch
