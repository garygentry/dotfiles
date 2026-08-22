# Roadmap

This document outlines planned enhancements and features for the dotfiles management system, organized by priority and impact.

Items are grouped into four tiers. Each tier represents roughly a phase of work, moving from completing the core feature set to expanding the ecosystem.

---

## Tier 1 — Core Completeness

*These close functional gaps that exist today. High impact, well-understood scope.*

### 1.1 Fully Implement Module Uninstall — ✅ Shipped

`dotfiles uninstall` executes recorded operations in reverse order (removing files,
restoring backups, removing empty dirs), displays a rollback plan with `--dry-run`, prompts
for confirmation (skippable with `--unattended`/`--force`), and reports what was removed vs.
what needs manual cleanup (packages, executed scripts). See the
[Rollback and Uninstall Guide](docs/rollback-guide.md). Remaining polish (e.g. package
removal) is tracked under 4.2.

---

### 1.2 Module Schema Validation — ✅ Largely shipped

The `dotfiles validate` subcommand lints every `module.yml` for schema errors — invalid
field values, missing required fields, and broken references — with `--json` output and a
`--strict` mode that rejects unknown YAML keys (catching typos like `dependancies`). Exit
code is 0 when all modules pass, 1 otherwise. (This also delivers what was tracked
separately as 2.6.)

**Remaining:** surfacing validation inline at `dotfiles list` / `dotfiles install` time (the
dedicated `validate` command exists; the always-on inline path is the open piece).

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

### 2.5 External / Private Module Repositories — ✅ Largely shipped (via content overlay)

Delivered by the **content-overlay** initiative: a `$DOTFILES_CONTENT_DIR` directory
(laid out like the repo, with its own `modules/`) is scanned as an additional root and
deep-merged over the engine, with content-wins override precedence keyed on module `name`.
The bootstrap script can clone the overlay from a public or private repo
(`--content-repo` / `--content-auth-cmd`). See [Content Overlay](docs/content-overlay.md).

**Remaining:** a `module_paths:` list in `config.yml` for **multiple** external roots (the
overlay currently supports one content root, not an arbitrary list of `~/.dotfiles-work/`,
`~/.dotfiles-private/`, … paths).

---

### 2.6 `dotfiles validate` Subcommand — ✅ Shipped (see 1.2)

Delivered: `dotfiles validate [modules...]` lints every `module.yml` (required fields,
invalid values, broken references) with `--json` and `--strict`, exit 0/1 for CI use. Any
further checks (e.g. `files:` source-path existence) fold into 1.2's remaining work.

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
- `dotfiles add you/my-module` adds an external module to `module_paths`
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

- **Content overlay (phases 1–4)**: optional `$DOTFILES_CONTENT_DIR` overlay of
  `config.yml`/`profiles/`/`modules/`, deep-merged over the generic engine with
  content-wins override precedence and `built-in`/`override`/`custom` tags; bootstrap
  acquisition (`--content-repo`), depersonalized defaults, and ssh `key_source`/`key_item`.
  See [Content Overlay](docs/content-overlay.md).
- **Module uninstall & schema validation**: `dotfiles uninstall` reversal (1.1) and the
  `dotfiles validate` subcommand (1.2 / 2.6) shipped.
- **v2.1.0**: Full UX overhaul — compact grid module selector, collapsible script output with auto-expanding errors, real-time progress bar with time estimates, smart output pattern recognition
- **Idempotence system**: SHA256 change detection for modules and files, user modification protection, `--force` / `--skip-failed` / `--update-only` flags
- **`github_clone` helper**: Standardized pattern for cloning GitHub repos from module scripts
- **`depends_on` in prompts**: Conditional prompt display based on other prompt answers
- **Starship preset selection**: Starship prompt theme picker in the zsh module
- **Bootstrap improvements**: Auto-detects non-interactive stdin, unattended mode for CI/CD
