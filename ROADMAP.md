# Roadmap

This document outlines planned enhancements and features for the dotfiles management system, organized by priority and impact.

Items are grouped into four tiers. Each tier represents roughly a phase of work, moving from completing the core feature set to expanding the ecosystem.

---

## Tier 1 — Core Completeness

*These close functional gaps that exist today. High impact, well-understood scope.*

### 1.1 Fully Implement Module Uninstall

The `dotfiles uninstall` command exists in the CLI and the state system already records operations, but the actual reversal logic is unimplemented. This is the most impactful missing feature.

**Scope:**
- Execute recorded operations in reverse order (remove files, restore backups, remove empty dirs)
- Display a rollback plan with `--dry-run` before committing
- Interactive confirmation prompt (skippable with `--unattended` or `--force`)
- Clear reporting: what was removed, what requires manual cleanup (packages, scripts)
- Handle edge cases: files modified after install, missing backups, non-empty directories

**Related:** `internal/state/state.go` already defines the `Operation` and `RollbackInstructions()` structures. The runner wires to `module.RunnerUI` but the execution path is incomplete.

---

### 1.2 Module Schema Validation

`module.yml` files are parsed with loose YAML unmarshaling. Typos in field names (e.g. `dependancies` instead of `dependencies`) silently pass through, producing hard-to-debug runtime failures.

**Scope:**
- Validate required fields (`name`, `description`, `version`) on module load
- Warn on unknown top-level keys (catches typos)
- Validate `type` values in `files:` entries (`symlink`, `copy`, `template` only)
- Validate `type` values in `prompts:` entries (`input`, `confirm`, `choice` only)
- Validate `depends_on` references point to existing prompt keys
- Surface validation errors at `dotfiles list` / `dotfiles install` time with clear messages
- Add a `dotfiles validate` subcommand for linting all modules without installing

---

### 1.3 State Schema Migration

When the `ModuleState` or `FileState` structs gain new fields, existing `.state/*.json` files silently lack those fields. Currently this is managed by zero-value defaults in Go, but there is no formal migration strategy and no version tracking in state files.

**Scope:**
- Add a `schema_version` field to state files
- Implement a migration function that upgrades older state files on first read
- Log a one-time notice when a migration is performed
- Document the migration approach so future schema changes follow the same pattern

---

### 1.4 Binary Download Safety

Several modules download pre-compiled binaries (cloud CLIs, compiled tools) using `curl` without verifying checksums. This is a security and reliability gap.

**Scope:**
- Add a `download_file` helper to `lib/helpers.sh` that:
  - Downloads a file to a temp path
  - Verifies SHA256 checksum if provided
  - Handles `amd64` / `arm64` variants automatically using `$DOTFILES_ARCH`
  - Is dry-run safe
- Update affected modules to use it
- Document the pattern in `docs/creating-modules.md`

---

### 1.5 ARM64 Module Coverage

Several modules hard-code `amd64` download URLs or assume x86 package availability. As ARM64 (Apple Silicon, ARM servers) becomes more common, this causes silent failures on those platforms.

**Scope:**
- Audit every module that downloads a binary or uses architecture-specific packages
- Ensure all use `$DOTFILES_ARCH` to select the correct artifact
- Add ARM64 variants where missing
- Add CI test matrix entry for ARM64 (if feasible)

---

## Tier 2 — Ecosystem Expansion

*These meaningfully extend what the system can do. Well-motivated, moderate scope.*

### 2.1 Additional Secrets Providers

Only 1Password is supported. Many users rely on other tools.

**Providers to add (in priority order):**
1. **Environment variables** — `DOTFILES_SECRET_<KEY>=value`. Zero dependency, useful for CI/CD and users who don't use a password manager.
2. **Bitwarden** — via the `bw` CLI. Open-source alternative to 1Password.
3. **`pass`** — Unix Password Manager. Common on Linux systems.
4. **macOS Keychain** — via `security find-generic-password`. Native on macOS, zero extra install.

**Scope:**
- Each provider implements the existing `secrets.Provider` interface — no Go changes needed beyond adding files
- Update `config.yml` to accept `provider: env | bitwarden | pass | keychain | 1password`
- Document each provider in `docs/` with its reference format

---

### 2.2 Template Function Library Expansion

The template system currently has 7 helper functions. Common patterns in config file generation need more.

**Functions to add:**
- `toJSON` / `fromJSON` — serialize/deserialize structured data
- `base64` / `base64d` — encode/decode (needed for kubeconfig, SSH keys in templates)
- `indent N` — indent a block by N spaces (YAML embedding)
- `quote` / `squote` — shell-safe quoting
- `hasPrefix` / `hasSuffix` — string matching
- `replace OLD NEW` — string substitution
- `toPascalCase` / `toCamelCase` — naming convention conversion
- `readFile PATH` — embed a file's content into a template

---

### 2.3 Expanded Integration Test Coverage

Integration tests currently only test successful first-run installation on Ubuntu and Arch. Critical paths are untested.

**Scenarios to add:**
- **Idempotence**: Run `dotfiles install` twice, assert second run produces no changes
- **Partial failure recovery**: Simulate a module failing mid-install, assert rollback works
- **Selective reinstall**: Change a module's install.sh, assert only that module re-runs
- **Uninstall**: Install then uninstall, assert files are removed and state is clean
- **Dry-run**: Assert `--dry-run` produces zero filesystem changes
- **macOS**: Add macOS runner to CI matrix (GitHub Actions has macOS runners)
- **Profile filtering**: Assert `--profile minimal` installs only the declared modules

---

### 2.4 Profile Stacking and Inheritance

Currently a profile is a flat list of modules. Users with multiple machine types (workstation vs server, personal vs work) need composable profiles.

**Scope:**
- Add `extends:` field to profile YAML: `extends: [minimal, cloud-tools]`
- Resolve extends recursively (with cycle detection)
- Allow per-profile config overrides (e.g., override `git.default_branch` per profile)
- Document profile inheritance in `docs/quick-start.md`

**Example:**
```yaml
# profiles/work.yml
extends: [developer]
modules:
  - aws
  - kubernetes
modules_config:
  git:
    default_email: me@work.com
```

---

### 2.5 External / Private Module Repositories

Power users want to keep private or machine-specific modules outside the main repo (personal tools, work-specific configs, secret-laden modules).

**Scope:**
- Add `module_paths:` to `config.yml` — a list of additional directories to scan for modules
- Module discovery scans all paths; later paths take priority over earlier ones (allows override)
- Paths can be absolute or relative to `$DOTFILES_DIR`
- Paths can reference git repos that are cloned automatically using `github_clone`

**Example config.yml:**
```yaml
module_paths:
  - modules/                        # built-in (default)
  - ~/.dotfiles-private/modules/    # local private modules
  - ~/.dotfiles-work/modules/       # work-specific modules
```

---

### 2.6 `dotfiles validate` Subcommand

Provide a way to lint all modules without installing them. Useful before committing changes.

**Scope:**
- Parse and validate every `module.yml` in all configured paths (see 2.5)
- Check for:
  - Required field presence
  - Unknown fields (typo detection)
  - Dependency references to non-existent modules
  - `files:` source paths that don't exist within the module directory
  - Circular dependency detection (already in resolver, expose it here)
- Exit code 0 = all valid, 1 = validation failures found
- Machine-readable output with `--json` flag
- Run this in CI as a pre-merge check

---

## Tier 3 — Developer Experience

*Improvements that make the day-to-day workflow smoother for module authors and contributors.*

### 3.1 Per-Module Documentation Standard

Modules lack individual documentation. Users and contributors must read `install.sh` to understand what a module configures and what prompts it offers.

**Scope:**
- Define a standard `modules/<name>/README.md` template covering:
  - What the module installs/configures
  - Prompt options and their effects
  - Config keys accepted in `config.yml`
  - Dependencies and why they're needed
  - Manual steps required after installation (if any)
- Update `dotfiles new` to generate a populated `README.md` stub
- Retroactively add READMEs to all existing modules

---

### 3.2 Retry Logic in Shell Helpers

Network and package manager operations fail transiently (mirrors down, rate limits, DNS hiccups). Scripts currently have no retry mechanism.

**Scope:**
- Add a `retry N CMD [ARGS...]` helper to `lib/helpers.sh`:
  - Retries `CMD` up to `N` times with exponential backoff (2s, 4s, 8s)
  - Logs each failure with attempt number
  - Propagates the final exit code if all attempts fail
  - Is dry-run aware
- Update `pkg_install` to use `retry 3` internally
- Document in `docs/creating-modules.md`

---

### 3.3 Module Health Checks

There is no ongoing verification that installed modules are still in the expected state (e.g., a user manually removed a symlink, or a system update clobbered a config).

**Scope:**
- `dotfiles status --check` runs each installed module's `verify.sh` (if present)
- Reports per-module pass/fail
- Does not modify system state
- Optional: `dotfiles install --fix` automatically re-deploys files that fail verification

---

### 3.4 Search and Filter in Module Selector

The interactive grid selector has no search capability. With 20+ modules, finding the right one requires scanning visually.

**Scope:**
- Press `/` to enter filter mode in the module grid
- Filter by module name or description (substring match)
- Filter by tag (`/tag:shell`, `/tag:cloud`)
- Filtered view shows only matching modules; Esc clears filter
- Filter state is not preserved after selection (grid resets)

---

### 3.5 Structured Error Context

Error messages from failed module installations are sometimes opaque. The actual failure reason is buried in captured script output.

**Scope:**
- Attach structured context to all user-facing errors (file path, module name, phase, exit code)
- Display a short "what failed" summary above the script output in error boxes
- Include the command that triggered the failure when available
- Provide a direct hint to the relevant troubleshooting doc section where applicable

---

## Tier 4 — Advanced Features

*Longer-horizon items. Well-motivated but require more design work.*

### 4.1 Multi-Machine Dotfiles Sync

Users with multiple machines need a way to keep their dotfile state consistent without manually re-running the installer on each machine.

**Approach options (requires design):**
- **Pull model**: A scheduled job runs `git pull && dotfiles install` on each machine
- **State sync**: Share `.state/` directory via git/cloud storage so machines know what's installed
- **Remote apply**: `dotfiles install --remote user@host` (SSH-based remote execution)

---

### 4.2 Declarative Package Lists in `module.yml`

Currently, packages are installed procedurally in `install.sh`. Declaring them in `module.yml` would enable cross-platform automation and uninstall support.

**Example:**
```yaml
packages:
  brew: [neovim, ripgrep]
  apt: [neovim, ripgrep]
  pacman: [neovim, ripgrep]
```

- Go runner handles the package installs (no shell script needed for simple modules)
- Enables automatic package removal during `dotfiles uninstall`
- Makes modules readable at a glance

---

### 4.3 Module Marketplace / Registry

A way to discover and share modules beyond the built-in set.

**Concept:**
- A community registry of modules at a known URL
- `dotfiles search <tool>` queries the registry
- `dotfiles add garygentry/my-module` adds an external module to `module_paths`
- Registry format: a simple git repo with an index YAML

---

### 4.4 Secrets Provider: System Keyring

For users who don't use a dedicated password manager, store secrets in the OS keyring (macOS Keychain, GNOME Keyring, KWallet).

- macOS: `security` CLI
- Linux: `secret-tool` (libsecret)
- Zero additional software on either platform

---

## Technical Debt

*These are not features but code quality improvements that reduce future maintenance cost.*

| Item | Location | Notes |
|------|----------|-------|
| Remove or integrate unused logging package | `internal/logging/` | Defined but not used; UI package duplicates its function |
| Eliminate `any` type in RunnerUI interface | `internal/module/runner.go`, `internal/ui/` | Workaround for import cycle; could be resolved with a shared types package |
| Add benchmarks for dependency resolver | `internal/module/resolver_test.go` | Kahn's algorithm, confirm O(V+E) in practice |
| Expand error wrapping with `%w` throughout | All packages | Many `fmt.Errorf` calls don't wrap the underlying error |
| Consolidate color constants | `internal/ui/`, `lib/helpers.sh` | ANSI codes defined separately in Go and shell; should share a single source |
| Add `go vet` and `staticcheck` to CI | `.github/workflows/` | Currently only `go test` and `golangci-lint` |

---

## Recently Completed

For context, these items were delivered in the most recent development cycle:

- **v2.1.0**: Full UX overhaul — compact grid module selector, collapsible script output with auto-expanding errors, real-time progress bar with time estimates, smart output pattern recognition
- **Idempotence system**: SHA256 change detection for modules and files, user modification protection, `--force` / `--skip-failed` / `--update-only` flags
- **`github_clone` helper**: Standardized pattern for cloning GitHub repos from module scripts
- **`depends_on` in prompts**: Conditional prompt display based on other prompt answers
- **Starship preset selection**: Starship prompt theme picker in the zsh module
- **Bootstrap improvements**: Auto-detects non-interactive stdin, unattended mode for CI/CD
