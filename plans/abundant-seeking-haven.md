# Investigation: Go Usage Rationale & Build-from-Source Design

## Context

This document examines two questions about the dotfiles project architecture:
1. Why use Go instead of pure shell scripts?
2. Why does bootstrap download/build Go rather than distributing a pre-built binary?

---

## Part 1: Why Go Over Pure Shell?

### What Go Actually Does (vs. What Shell Does)

The division is deliberate: **Go orchestrates, shell executes**.

**Go handles** (the hard-to-do-in-shell parts):
- Dependency resolution via Kahn's algorithm ([resolver.go](internal/module/resolver.go)) — topological sort with cycle detection, priority ordering, OS filtering, transitive expansion
- Per-module/per-file state tracking with SHA256 checksums ([state/](internal/state/)) — knows when to skip, when to update, detects user modifications
- Structured rollback on failure — LIFO operation reversal distinguishing reversible (file ops) from non-reversible (scripts)
- YAML config parsing into typed structs ([config/](internal/config/), [module/schema.go](internal/module/schema.go)) — compile-time type safety, validation at parse time
- Go `text/template` rendering with custom functions ([template/](internal/template/)) — env, default, upper, lower, join, etc.
- Thread-safe UI with spinners, interactive multi-select prompts, TTY detection ([ui/](internal/ui/))
- Secrets provider abstraction ([secrets/](internal/secrets/)) — 1Password today, interface-ready for Vault/AWS/etc.
- System detection ([sysinfo/](internal/sysinfo/)) — OS distro from `/etc/os-release`, arch, package manager inference, sudo availability

**Shell handles** (what it's good at):
- Running `apt install`, `brew install`, `pacman -S`
- Git configuration, SSH key generation
- File manipulation on the target system
- OS-specific installation logic (`modules/*/os/{linux,macos}.sh`)

### What Would Be Hard or Fragile in Pure Shell

| Capability | Go | Pure Shell |
|---|---|---|
| Dependency graph resolution | Type-safe Kahn's algorithm, cycle detection with readable error paths | Would need external tools or ~200 lines of fragile array manipulation |
| State/idempotence tracking | JSON state files with per-file checksums, structured read/write | Requires `jq` dependency, brittle parsing, no type guarantees |
| YAML config parsing | `gopkg.in/yaml.v3` into typed structs | Requires `yq` or `python -c`, no validation |
| Rollback on failure | Structured operation log, LIFO reversal | Procedural, error-prone, hard to track what to undo |
| Template rendering | Full `text/template` with custom funcs | `envsubst` (limited) or `sed`/`awk` (fragile with special chars) |
| Interactive UI | `charmbracelet/huh` library, mutex-safe spinners | Basic `read -p`, spinners conflict with subprocess output |
| Secrets abstraction | Interface-based, swappable providers | Hardcoded provider logic, no clean abstraction |
| Testability | Standard `go test`, integration tests with race detector | Manual bash testing, limited assertion frameworks |

### The Alternative: What Pure Shell Would Look Like

A pure shell version would need:
- **External dependencies**: `jq` (JSON), `yq` (YAML), possibly `tsort` (dependency ordering)
- **~500+ lines** of state management code replacing the `state` package
- **No compile-time safety** — config typos become runtime errors
- **No structured rollback** — would need to manually track operations in temp files
- **Platform-inconsistent behavior** — bash version differences, macOS ships bash 3.2

The Go binary is ~2,000 LOC across well-separated packages. The equivalent shell code would likely be larger, harder to test, and require more external dependencies.

---

## Part 2: Why Build-from-Source Instead of a Pre-Built Binary?

### How Bootstrap Currently Works

[bootstrap.sh](bootstrap.sh) performs these steps:
1. Detect OS and architecture
2. Ensure `git` is installed
3. **Download and install Go 1.23.6** from `go.dev/dl/`
4. Clone the dotfiles repo to `~/.dotfiles`
5. Run `go build -o bin/dotfiles .`
6. Execute `./bin/dotfiles install`

### Why Not Distribute a Pre-Built Binary?

There are several reasons, ranging from architectural to practical:

#### 1. The Binary Is Inseparable from the Repository

The Go binary doesn't embed anything. At runtime it:
- Reads `config.yml` and `profiles/*.yml` from the repo directory
- Scans `modules/*/module.yml` for module definitions
- Sources `lib/helpers.sh` into every shell script execution
- Reads `modules/*/install.sh`, `verify.sh`, OS-specific scripts from disk

A pre-built binary without the cloned repo is useless. You always need `git clone` first, so the binary alone doesn't save much.

#### 2. The Repo Must Be Cloned Anyway

Every module's install scripts, config templates, and dotfiles live in the repository. The Go binary orchestrates these files — it doesn't contain them. Since `git clone` is unavoidable, the incremental cost of `go build` (~3-5 seconds) is minor.

#### 3. Distribution Complexity

Providing pre-built binaries would require:
- **Release infrastructure**: goreleaser or equivalent, GitHub releases workflow
- **4+ binary variants**: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
- **Version synchronization**: binary version must match repo version (module schema changes, new env vars, etc.)
- **Download logic in bootstrap**: detect platform, fetch correct binary from GitHub releases, verify checksums

The bootstrap script would still need to clone the repo, so you'd be adding infrastructure to save ~5 seconds of build time.

#### 4. Development Workflow Benefits

Building from source means:
- Any local changes to Go code take effect on next `go build`
- No version mismatch between binary and repo
- Contributors can iterate without a release cycle
- `go build` is deterministic — same source always produces same behavior

#### 5. Go Is a Reasonable Bootstrap Dependency

Go downloads as a single tarball (~70MB), extracts to a self-contained directory, requires no system library dependencies, and works on all target platforms. The bootstrap script handles this automatically. Compare to alternatives like requiring Python (version conflicts, venv management) or Node (npm ecosystem complexity).

### Could We Distribute a Pre-Built Binary?

Yes, technically. The tradeoffs would be:

| Aspect | Build from source | Pre-built binary |
|---|---|---|
| Bootstrap time | +3-5s build, +70MB Go download | Faster first run |
| Infrastructure | None needed | goreleaser, GitHub releases, CI workflow |
| Version sync | Always in sync | Must ensure binary matches repo |
| Hackability | Edit Go, rebuild, done | Must rebuild or download new release |
| Dependencies | Go (auto-installed) | None beyond git |
| Complexity | Simple | More moving parts |

The current approach trades slightly slower first-run for zero release infrastructure and guaranteed consistency.

---

## Summary

**Go over shell**: Go provides type-safe dependency resolution, structured state management, rollback, templating, and testability that would be fragile or require external dependencies in pure shell. The architecture wisely uses Go for orchestration and shell for system operations.

**Build from source over pre-built binary**: The binary is useless without the cloned repository, so `git clone` is always required. Building from source adds ~5 seconds but eliminates release infrastructure, version synchronization concerns, and platform-specific binary distribution. Go's self-contained toolchain makes it a lightweight bootstrap dependency.
