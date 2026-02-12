# Design Rationale

This document explains the key technology choices behind the dotfiles management system — why Go was chosen for the CLI, why shell scripts handle the actual work, and why the system builds from source rather than distributing a pre-built binary.

## Why a Hybrid Go + Shell Architecture?

Dotfiles managers exist on a spectrum. At one end, tools like [GNU Stow](https://www.gnu.org/software/stow/) are pure symlink managers. At the other, frameworks like [Ansible](https://www.ansible.com/) bring full configuration management. This system sits in between: it needs real orchestration (dependency graphs, state tracking, rollback) but the actual work — installing packages, configuring tools — is fundamentally shell scripting.

The hybrid approach plays to each language's strengths:

**Go handles orchestration** — the parts that are hard to get right in shell:

| Capability | Why Go? |
|---|---|
| Dependency resolution | Kahn's algorithm with cycle detection, deterministic priority ordering, transitive expansion — ~150 lines of type-safe code vs. fragile array manipulation in bash |
| State tracking | JSON state files with per-file SHA256 checksums for idempotence. Structured read/write with proper types vs. requiring `jq` and brittle parsing |
| Configuration | `gopkg.in/yaml.v3` parses `config.yml` and `module.yml` into typed structs with compile-time validation. The shell equivalent requires `yq` or `python -c` with no type guarantees |
| Rollback | Structured operation log with LIFO reversal, distinguishing reversible operations (file deploys) from informational ones (script runs). Difficult to track reliably in shell |
| Template rendering | Full Go `text/template` with custom functions (`env`, `default`, `upper`, `lower`, `join`). The shell alternatives — `envsubst` (limited) or `sed`/`awk` (fragile with special characters) — don't scale |
| Interactive UI | Thread-safe spinners, multi-select prompts, TTY detection with graceful fallback. Bash spinners conflict with subprocess output and `read -p` is limited |
| Secrets abstraction | Interface-based provider system (1Password today, extensible to Vault/AWS). Clean abstraction isn't natural in shell |
| Testing | Standard `go test` with race detector, integration tests in Docker. Shell testing frameworks exist but are limited |

**Shell handles system operations** — what it does best:

- Running package managers (`apt install`, `brew install`, `pacman -S`)
- Git configuration, SSH key generation, tool setup
- OS-specific logic that varies by platform
- Anything that benefits from being easily readable and modifiable without recompilation

### The Alternative: What Pure Shell Would Require

A shell-only version of the orchestration layer would need:

- **External dependencies**: `jq` for JSON state files, `yq` for YAML config parsing, possibly `tsort` for dependency ordering — all of which may not be present on a fresh system
- **500+ lines** of state management replacing the `state` package, with no compile-time guarantees
- **Platform inconsistencies**: macOS still ships Bash 3.2 (2007), which lacks associative arrays, `mapfile`, and other features used in modern bash. The system would need to either target Bash 3.2 or require Bash 4+ as a dependency
- **No structured rollback**: tracking operations for undo in shell requires writing to temp files and parsing them back, with no type safety on the metadata

The Go binary is ~2,000 lines across well-separated packages. An equivalent shell implementation would likely be larger, harder to test, and more fragile.

### Why Not Python, Ruby, or Another Scripting Language?

Go was chosen over other scripting languages for several practical reasons:

- **Single binary, zero runtime**: `go build` produces one static binary with no runtime dependencies. Python needs a Python installation (and often a virtualenv to avoid version conflicts). Ruby needs a Ruby installation. Both add complexity to bootstrapping a fresh system
- **Cross-compilation**: Go trivially cross-compiles for any OS/architecture combination, which matters for a tool targeting macOS (Intel and Apple Silicon), Ubuntu, and Arch
- **Startup time**: The Go binary starts in milliseconds. Python and Ruby have noticeable interpreter startup overhead, which adds up when the binary is called back from shell scripts (`get-secret`, `render-template`)
- **Dependency management**: `go.mod` locks dependencies with checksums. No `pip install` issues, no gem version conflicts, no virtualenv to manage on a machine that's still being set up

## Why Build from Source?

The bootstrap script ([bootstrap.sh](../bootstrap.sh)) downloads Go, clones the repo, and runs `go build`. This is deliberate, not a shortcoming.

### The Binary Needs the Repository

The Go binary doesn't embed any assets. At runtime, it reads from the filesystem:

- `config.yml` and `profiles/*.yml` for configuration
- `modules/*/module.yml` for module definitions
- `modules/*/install.sh`, `verify.sh`, and `os/*.sh` for execution
- `lib/helpers.sh`, sourced into every shell script

A pre-built binary without the cloned repository can't do anything. Since `git clone` is always the first step, the binary alone doesn't save a meaningful step.

### The Cost of Pre-Built Distribution

Distributing pre-built binaries would require:

- **Release infrastructure**: goreleaser configuration, GitHub Actions release workflow, artifact signing
- **Platform matrix**: at minimum linux/amd64, linux/arm64, darwin/amd64, darwin/arm64 — four binaries per release
- **Version synchronization**: the binary version must match the repository version. A module schema change or new environment variable requires both a new binary and updated repo content. Building from source eliminates this entire class of problems
- **Download logic**: the bootstrap script would need platform detection, binary download, checksum verification — essentially the same complexity as downloading Go, but repeated for every release

### What Building from Source Provides

- **Guaranteed consistency**: the binary always matches the repository it was built from
- **Zero release overhead**: no CI release pipeline, no artifact hosting, no version tags to manage
- **Hackability**: modify Go code, run `go build`, done. No waiting for a release cycle
- **Simplicity**: one build step (`go build -o bin/dotfiles .`) with no configuration

### Go Is a Lightweight Bootstrap Dependency

Go installs as a self-contained tarball (~70MB), extracts to a single directory, requires no system libraries, and works identically on all supported platforms. The bootstrap script handles this automatically. Compare this to bootstrapping Python (version management, pip, virtualenv) or Node.js (nvm, npm ecosystem).

The build itself takes 3-5 seconds on modern hardware. For a tool that runs once during system setup, this is negligible.

## Design Tradeoffs

Every architecture involves tradeoffs. Here are the ones this system makes consciously:

| Decision | Benefit | Cost |
|---|---|---|
| Go for orchestration | Type safety, testability, single binary | Requires Go toolchain at bootstrap |
| Shell for execution | Readable, modifiable, no recompilation needed | Platform inconsistencies (Bash versions, coreutils differences) |
| Build from source | Always in sync, zero release infra | ~70MB Go download + ~5s build time on first run |
| Runtime file loading | Modules can be added/modified without rebuilding | Binary is useless without the repository |
| YAML configuration | Human-readable, well-supported | Requires a parsing library (Go) or external tool (shell) |

These tradeoffs optimize for the primary use case: a developer setting up a new machine, where reliability and correctness matter more than shaving seconds off the initial bootstrap.

## References

- [Architecture](architecture.md) — System design and component overview
- [Installation](installation.md) — Bootstrap and installation process
- [Creating Modules](creating-modules.md) — How the Go + shell boundary works in practice
