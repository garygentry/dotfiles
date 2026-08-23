# Dotfiles Manager

Go + Shell hybrid dotfiles manager. Go handles orchestration (CLI, config, dependency resolution, state tracking); shell scripts handle actual installation in `modules/*/install.sh`.

## Architecture context

The engine is intentionally **generic and unopinionated**, with an optional **content overlay** (`$DOTFILES_CONTENT_DIR`: user `config.yml`/`profiles/`/`modules/` that overlay the repo), alongside re-run/rollback safety. Personalization — identity, opinionated module config, curated profiles — lives in a separate content repo, never in this engine. The docs (repo markdown + the Astro docs site) reflect the shipped engine; `docs/content-overlay.md` is the conceptual model and `docs/extending.md` is the build-your-own-overlay tutorial. **Core invariant to preserve: with no content dir set, behavior is identical to a plain repo checkout.** The repo-root `ROADMAP.md` is the product-feature backlog; open follow-ups live as GitHub issues.

## Commands

```bash
make build                    # Build binary to bin/dotfiles
make test                     # Unit tests (go test ./...)
make test-integration         # Docker-based integration tests (Ubuntu + Arch)
make test-all                 # Unit + integration
make lint                     # go vet
make lint-shell               # shellcheck on modules/ and lib/
make lint-all                 # go vet + shellcheck
```

## Project Structure

- `cmd/dotfiles/` - CLI commands (Cobra). Entry point: `main.go` → `cmd/dotfiles/root.go`
- `internal/config/` - YAML config loading, profiles, module settings
- `internal/module/` - Module discovery, dependency resolution (Kahn's algorithm), execution, checksums
- `internal/state/` - JSON state files in `~/.dotfiles/.state/`, rollback tracking
- `internal/secrets/` - Provider interface (1Password, noop fallback)
- `internal/sysinfo/` - OS, arch, package manager, sudo detection
- `internal/template/` - Go text/template with custom funcs (env, default, upper, lower, contains, join, trimSpace)
- `internal/ui/` - Spinners, prompts, progress bars (charmbracelet/huh + bubbletea)
- `internal/logging/` - Structured logging via slog
- `modules/` - 30+ modules, each with `module.yml`, `install.sh`, optional `verify.sh`, `os/*.sh`, `files/`
- `profiles/` - Profile YAML files selecting module sets (developer, minimal, test)
- `lib/helpers.sh` - Shared shell functions sourced by all install scripts
- `test/integration/` - Docker-based integration tests

## Key Patterns

- **Dependency resolution** uses Kahn's algorithm (topological sort) in `internal/module/resolve.go`
- **Idempotence** is multi-level: module checksums, config hashes, file-level source/deployed hashes
- **State files** (`~/.dotfiles/.state/<name>.json`) track version, status, checksums, file states, operations for rollback
- **Module execution order**: OS script → install.sh → file deployment → verify.sh
- **Shell scripts** inherit `set -euo pipefail`, use `lib/helpers.sh` functions, receive `DOTFILES_*` env vars from Go runner
- **File deployment types**: symlink, copy, template (Go text/template). Invariant: only symlink read-only reference files — configs the owning tool rewrites at runtime (`~/.zshrc`, `~/.gitconfig`, `~/.config/starship.toml`) must be `template`/`copy`, else the tool writes back into the repo and dirties the checkout. `dotfiles validate` enforces this; the engine auto-migrates old symlink deployments to real files on the next install.

## Shell Script Conventions

Install scripts use helpers from `lib/helpers.sh`:
- `pkg_install`, `pkg_installed` - idempotent, auto-selects brew/apt/pacman
- `log_info`, `log_warn`, `log_error`, `log_success` - colored output
- `is_macos`, `is_ubuntu`, `is_arch`, `has_sudo`, `is_interactive`, `is_dry_run` - guards
- All env context via `DOTFILES_*` vars (OS, ARCH, PKG_MGR, HOME, DIR, PROMPT_*, USER_*)

## Testing

- Unit tests: table-driven, `t.TempDir()` for isolation, no external frameworks
- Integration tests: Docker containers for Ubuntu and Arch, test full install/uninstall flows
- CI runs: shellcheck → go vet → unit tests (with `-race`) → integration tests

## Gotchas

- Terminal state management is sensitive on macOS — reads from fresh `/dev/tty` instead of `os.Stdin` to avoid corruption after BubbleTea prompts
- Platform-specific ioctl constants for FIONREAD defined per OS (`ioctl_linux.go`, `ioctl_darwin.go`)
- Module prompts have conditional logic (`depends_on`, `show_when: explicit_install`) — test prompt flows carefully
- `--unattended` mode uses prompt defaults; ensure defaults are sensible when adding new prompts
- Debian is mapped to ubuntu for OS detection
