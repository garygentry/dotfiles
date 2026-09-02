# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **`skip_if_file` on module prompts** — a prompt is suppressed (its default used) when any listed
  path already exists (`~` and `~/.config` expanded; a dangling symlink counts). The `ssh` module
  uses it for the `ssh_key_type` prompt, which is pure noise on a host that already has a key —
  `install.sh` detects and reuses an existing key regardless of the answer.
- **`bun` module** (in the `developer` profile) — installs the official Bun release binary into
  `~/.local` **sudo-free**, checksum-verified against the release's `SHASUMS256.txt`. Resolves the
  latest release **without the GitHub API** (following the `/releases/latest` redirect, so a fleet
  behind one NAT can't exhaust the 60-req/hr API limit). Handles linux/darwin, x64/aarch64, **musl**,
  and the `-baseline` builds — detects a missing **AVX2** CPU (common on VMs) and installs
  `-baseline` so bun doesn't die with "Illegal instruction"; extracts the zip with `unzip` or,
  failing that, `python3` (no sudo either way). Symlinks `bun` + `bunx`, then runs the binary to
  confirm it's not a wrong-arch dud. Pin with `DOTFILES_BUN_VERSION`.
- **ssh `key_source: auto`** — a conservative default that resolves at runtime: **agent** when a
  live SSH agent already holds a key (`ssh-add -l`), otherwise **generate** (reusing an existing
  local key if present). Lets one setting work across a fleet where some hosts run the 1Password
  agent and others (e.g. freshly provisioned guests) carry only a baked key.
- **graceful no-sudo degradation for package modules** — new `sudo_usable`/`require_sudo` helpers.
  `btop`, `nodejs`, `gh`, and `neovim` now **skip cleanly with a clear warning** when no usable
  sudo is available (not root, no passwordless sudo, or an unattended run) instead of hard-failing
  the whole profile; their `verify.sh` treats the skipped state as non-fatal.
- **bootstrap now fast-forwards an already-materialized content overlay by default**, so
  re-running the script alone brings a machine fully up to date — engine *and* personal
  content. The refresh is `git pull --ff-only` (advances only on a clean fast-forward;
  otherwise warns and keeps the existing checkout — never clobbers local edits), applied when
  the overlay is a git clone with an upstream. No-ops for `--content-path` dirs, non-git
  overlays, detached/upstream-less checkouts, and offline/unauthenticated runs. Opt out with
  `--no-content-update` (or `DOTFILES_CONTENT_UPDATE=0`). This mirrors the engine's own
  auto-update but stays `--ff-only`, never `reset --hard`, since the overlay can hold real
  authoring work.

### Fixed

- **`1password` no longer hard-fails on a no-sudo host.** The Ubuntu installer
  (`modules/1password/os/ubuntu.sh`) ran its apt-repo + package-install steps through `sudo_cmd`
  with no guard, so a host without usable sudo aborted the whole profile instead of degrading —
  the pattern `gh`/`neovim` already avoid. It now leads with `require_sudo … || return 0`, so the
  step skips cleanly with a warning and `install.sh` points at manual setup. (The macOS/Arch
  installers already degraded — Arch falls back to a `~/.local/bin` download.)
- **a working 1Password service-account token no longer triggers a spurious sign-in prompt.**
  `IsAuthenticated()` gated on `op account list`, which is empty for a service account, so a host
  driven by `OP_SERVICE_ACCOUNT_TOKEN` reported unauthenticated and the interactive
  `dotfiles install --profile gwg` asked "Set up 1Password?" even though secret resolution worked.
  It now validates a present token directly via `op whoami` (no `--account`, which a token rejects)
  before the account-list path, so a working service account counts as authenticated.
- **`nodejs` installs sudo-free instead of degrading to nothing.** The module used
  `require_sudo → pkg_install nodejs npm`, so on a host without usable sudo it skipped and left no
  node at all (breaking anything downstream that needs `node`/`npm`). It now installs the official
  Node.js prebuilt **tarball** into `~/.local/share/node` and symlinks `node`/`npm`/`npx` into
  `~/.local/bin` — the same deterministic, sudo-free pattern as Go and claude-code — checksum-
  verified against `nodejs.org`'s `SHASUMS256.txt` (which also supplies the newest patch of the
  line, so no version is hardcoded). Pin the LTS line with `DOTFILES_NODE_MAJOR` (default 22).
  `verify.sh` no longer needs its no-sudo escape hatch.
- **claude-code installs non-interactively instead of hanging the profile.** The module no longer
  pipes `curl https://claude.ai/install.sh | bash`, whose final step (`claude install`) is an
  interactive TUI that never completes under automation — with a terminal it waits for keypresses,
  and fully detached (setsid, closed stdin) it hangs before printing a line (occasionally exiting 0
  having installed nothing), even with telemetry/autoupdate/nonessential-traffic disabled. On a
  clean guest this stalled the whole run until the 5-minute per-module timeout and cascaded into
  `claude-code-config` ("claude CLI not found"). The module now follows Anthropic's documented
  direct-download path (Setup -> *Binary integrity and code signing*): resolve the release from
  `…/claude-code-releases/{stable,latest}`, **GPG-verify the signed `manifest.json`** against the
  pinned Anthropic release key (fingerprint `31DDDE24…`, imported into an ephemeral keyring — a
  failed verification is fatal; a missing signature or absent gpg degrades to sha256-only with a
  warning), sha256-check the native binary (linux/darwin x64/arm64, musl-aware), and install into
  the **native installer's own on-disk layout**: the versioned binary under
  `~/.local/share/claude/versions/<version>` with `~/.local/bin/claude` symlinked to it (a
  hand-placed launcher is supported from v2.1.207+; auto-update leaves it in place). No TUI is ever
  invoked; no sudo and no Node are required. Idempotent (fast-paths when `claude` is already
  present) and pinnable via `DOTFILES_CLAUDE_CODE_VERSION` (default channel: `stable`).
- **ssh module no longer clobbers `~/.ssh/config`** — it now owns only a delimited
  `# >>> dotfiles managed >>>` … `# <<< dotfiles managed <<<` block, splicing the rendered config
  in (replacing the block in place on re-runs, appending after existing content on first adoption)
  and preserving everything outside it. This protects a host-provisioned deploy-key
  `Host github.com` entry, which an earlier full re-render would drop in `agent` mode (breaking
  `git@github.com`). A symlinked config is demoted first; a one-time backup is taken on adoption.
- **`pkg_install` now returns the installer's real exit status.** `log_success` previously ran
  unconditionally as the last statement, so the function returned 0 even when the install failed —
  masking failures (e.g. apt without sudo) and leaving `if pkg_install …` / `pkg_install … || …`
  fallbacks (fzf, zoxide, zellij, golang, lazygit, ansible) dead. Fixes fzf's "not found after
  install" verify mismatch at the root.
- **file deployment survives a symlinked destination directory.** A content module deploying files
  under a directory that is a dangling symlink (e.g. `~/.config/nvim` left pointing at a prior
  engine's files after an upgrade) no longer fails `MkdirAll` with `EEXIST`; the link is reconciled
  (backed up if it holds out-of-managed-root content) and a real directory is created.
- **starship**: the module now installs **or upgrades** to the latest release instead of
  skipping when a starship binary is already present. A stale binary left by an older install
  couldn't parse newer config keys (e.g. the `cpp` module or the `ALTLinux` OS symbol),
  warning `Unknown key`/`unknown variant` on every shell. The official installer is run with
  `-f`; the module version bump makes idempotence re-run it on existing machines.

## [3.0.0] - 2026-08-23

### Changed

- **Depersonalization pass: the engine now ships generic, unopinionated defaults**, with
  personal/curated content moved to a content overlay. Backward-compatibility note: several
  of these flip out-of-box behavior, so review if you install the engine bare (no overlay).
  - **Default `profile` is now `minimal`** (`git`, `zsh`) instead of `developer`, so a
    bare `dotfiles install` no longer pulls in docker + an AI CLI. `developer` remains an
    opt-in profile. (`config.yml`, `internal/config/config.go` fallbacks + tests.)
  - **git**: `pull.rebase=true` and `delta.side-by-side=true` are no longer imposed;
    commit/tag **SSH auto-signing** is now opt-in via `modules.git.sign_commits` (default
    off — signing *capability* is still configured), and the conventional-commits
    **commit template** is opt-in via `modules.git.commit_template` (default off).
  - **neovim**: ships the binary only (no bundled `init.lua`); bring your own config via an
    overlay. **tmux**: ships the binary + TPM only (the opinionated Catppuccin config and
    cheatsheet moved out; the `opinionated` preset/prompt is gone). **starship**: keeps
    official presets only (the bundled `custom` starship.toml and `custom` preset option
    removed).
  - **zsh**: `aliases.zsh`/`functions.zsh` trimmed to universal entries; personal git/docker
    shortcuts and `myip`/`weather` moved out. **fonts**: the Nerd Font is now configurable
    via `modules.fonts.font` (+ optional `modules.fonts.font_url`), default `FiraCode`.
  - Removed personal artifacts from the engine: `profiles/gwg-unattended.yml`,
    `profiles/debug-failing.yml` (and `scripts/testuser.sh` now defaults to `minimal`), the
    internal `plans/` dev-notes, and the README author/contact block.
- **Bootstrap now force-syncs the engine checkout to `origin` instead of `git pull` (#30).**
  `$DOTFILES_DIR` is a managed artifact (personalization lives in the content overlay), so
  bootstrap `fetch`es and `reset --hard`s to `origin/$DOTFILES_REF` (plus `clean -fd`, which
  preserves gitignored `.state/`/`.backups/`), saving any local drift to
  `.backups/pre-update-<sha>.patch` first. It now **fails loudly** if it cannot update rather
  than silently building from stale source — the failure mode that surfaced a wrong,
  truncated module list on a previously-installed machine. New `DOTFILES_REF` env var
  (default `main`).

### Added

- **Automatic migration of legacy symlink deployments to real files (#30).** When a config
  that an older release symlinked straight out of the repo is now managed as a
  `template`/`copy`, install replaces the link with a real file instead of following it
  (following it wrote back into and dirtied the repo). Deterministic and conservative: a link
  resolving into a managed repo is removed as engine-owned; a link elsewhere has its content
  backed up to `.backups/` first; nothing is destructively overwritten. Old machines
  self-heal on the next install/bootstrap. Adds the `demote_symlink` helper (`lib/helpers.sh`)
  for shell-generated configs, and exports `DOTFILES_CONTENT_DIR` to module scripts.
- **`dotfiles validate` flags tool-writable symlink destinations (#30).** A `symlink` aimed
  at a config the owning tool rewrites at runtime (`~/.zshrc`, `~/.gitconfig`,
  `~/.config/starship.toml`, …) is now reported — such a destination must be `template`/`copy`
  or every runtime write flows back into the repo and dirties the checkout.

### Fixed

- **`copy`/`template` deploys no longer write through a pre-existing symlink at the
  destination (#30)** — `os.WriteFile` followed the link and recreated/clobbered the repo
  source. A changed deployment type (e.g. `symlink`→`template`) now also forces a redeploy.

## [2.3.0] - 2026-08-22

### Changed

- **Removed dead `modules.neovim.colorscheme` config (#13).** The committed
  `neovim.colorscheme: catppuccin` key was silently ignored — `modules/neovim/init.lua`
  hardcodes catppuccin (plugin install, `setup(...)`, `colorscheme("catppuccin")`, and the
  lualine statusline theme), and nothing read the setting. Rather than expose a knob that
  only accepts one value (catppuccin is the only installed scheme, so any other value would
  break nvim startup), the key is dropped — the same call made for `zsh.theme` in 4D. No
  behavior change: neovim still uses catppuccin. Switchable colorschemes remain a possible
  future feature (would need plugin management), tracked separately from this cleanup.

- **BREAKING (default): ssh `key_item` default changed** from `op://Personal/SSH Key` to
  the neutral generic `op://Private/SSH Key` (1Password's current default personal-vault
  name), so no personal vault name is baked into the engine. This only affects users who
  set `modules.ssh.key_source: 1password` **and** relied on the old default vault name; if
  the SSH key lives in a `Personal` vault, set `modules.ssh.key_item: op://Personal/SSH Key`
  explicitly. Anyone already setting `key_item`, or not using `key_source: 1password`, is
  unaffected. Changed in `modules/ssh/install.sh`, `config.yml`,
  `config.overlay.example.yml`, and the hermetic ssh test.

- **Dead-config cleanup (phase 4D).** Two committed `config.yml` keys that looked wired
  but were silently ignored are now either live or gone. `modules.git.default_branch` is
  now live: `modules/git/install.sh` applies it via
  `git config --global init.defaultBranch` (and `verify.sh` checks against the same
  value), defaulting to `main` so unset config behaves exactly as before.
  `modules.zsh.theme` is removed: its committed default (`powerlevel10k`) was never a
  supported or installed option and overlapped the existing `zsh_prompt` prompt that
  already drives `ZSH_THEME`, so it was dropped from `config.yml`, `README.md`, and
  `docs/quick-start.md` rather than made live (which would have changed default
  behavior). No behavior change at defaults.

### Fixed

- **`render-template --module` now works (#21).** The `--module` flag was registered but
  its value was never read, so the module whose `config.yml` settings populate `.Module`
  could only be chosen via the `DOTFILES_MODULE_NAME` env var. The flag now selects the
  module (overriding `DOTFILES_MODULE_NAME` when given); its help text is corrected from the
  misleading "Module directory" to "Module name …". No change for the shell `render_template`
  helper, which still relies on the env var the runner exports.

- **Uninstall no longer proposes removing `$HOME` (#22).** When a module deployed a file
  directly into an already-existing directory (e.g. `~/.gitignore_global`, whose parent is
  `$HOME`), the installer recorded a bogus `dir_create` operation for that pre-existing
  directory, so `dotfiles uninstall <module> --dry-run` listed an alarming
  "Remove directory: /home/<user>". The deploy path now records a `dir_create` op only when
  it actually creates the directory. (The rollback executor already rmdir'd empty
  directories only, so nothing was ever deleted — but the plan was wrong.) State files
  written before this fix keep the stale op until the module is reinstalled.

- **Unified template contexts (phase 4C).** Templates rendered from a shell script via
  the `render_template` helper now see the same context as those rendered in-process by
  the runner. Previously the hidden `render-template` subcommand left `.User`, the
  module's config settings (`.Module`), and `.XDGConfigHome` empty — so a template
  rendered from an `install.sh` silently got blank values for `{{ .User.email }}`,
  `{{ .Module.key_source }}`, etc. Both paths now build the context through a single
  shared `module.NewTemplateContext`, and the subcommand reloads the SAME layered config
  (base `config.yml` + optional content overlay) via `config.Load`, so overlay values and
  YAML types (e.g. bool settings) are preserved identically. `.Secrets` is a consistent
  empty map on both paths; secrets reach scripts only through `get_secret`. No behavior
  change for the in-process path or when no content overlay is set.

### Added

- **Content-overlay packaging docs (phase 4E).** New [`docs/content-overlay.md`](docs/content-overlay.md)
  documents the two-repo model (generic engine + your `my-dotfiles` content repo) and
  the clean-machine first-install flow end to end: bootstrapping from a public
  (recommended, secret-free) content repo, a local/network path, and the private-repo
  path via an ambient SSH/1Password agent or `--content-auth-cmd` — with the
  chicken-and-egg called out honestly. A minimal, verified example content repo lives at
  [`docs/examples/content-repo/`](docs/examples/content-repo/) (overlay `config.yml`, a
  profile, a custom `hello` module, and an override of the built-in `git` module),
  referenced from `config.overlay.example.yml`. Docs-only; no engine behavior change.
- **Configurable SSH 1Password item (phase 4B).** The `ssh` module no longer hardcodes
  `op://Personal/SSH Key` for `key_source: 1password`. Set `modules.ssh.key_item` in
  `config.yml` (or a content overlay) to point at your own item; the key type
  (`ed25519`/`rsa`) is still appended as the field, so the full reference is
  `<key_item>/<key_type>`. When unset it defaults to `op://Personal/SSH Key`, so existing
  configs behave exactly as before.
- **Content overlay (phase 4A: acquisition & bootstrap).** `bootstrap.sh` can now
  materialize a content overlay directory and point the engine at it before installing,
  via new flags: `--content-repo <git-url>` (with optional `--content-ref <ref>`) clones
  or idempotently pulls a content repo, `--content-path <dir>` uses an existing
  local/network directory in place, and `--content-dir <dest>` chooses where a repo lands
  (default `~/.config/dotfiles`). The resolved directory is exported as
  `DOTFILES_CONTENT_DIR` and appended to `~/.zshenv` for future shells (opt out with
  `--no-persist-content-dir`). Private sources are supported through an ambient SSH/1Password
  agent or an optional `--content-auth-cmd` hook run before the clone; a public, secret-free
  content repo is recommended. All acquisition lives in the bootstrap/shell layer — the Go
  core gains no git/network/auth logic — and with no content flags (and no pre-set
  `DOTFILES_CONTENT_DIR`) bootstrap behaves exactly as before. Every non-content argument is
  still forwarded verbatim to `dotfiles install`.
- **Content overlay (phase 3: modules).** Modules are now discovered from your content
  directory's `modules/` in addition to the engine's built-in `modules/`, with **same-name
  content-wins precedence**: drop a `modules/<name>/` in your content dir to add a custom
  module, or a same-named directory to override a built-in one wholesale (whole-module
  replacement, not merging). `install`, `list`, `status`, and `validate` all discover
  across both roots, and a content profile can reference custom or overridden modules by
  name. `list` and `status` add a `Source` column (`built-in` / `override` / `custom`) when
  a content overlay is active so overrides are never silent. With no `DOTFILES_CONTENT_DIR`
  set, discovery and output are exactly as before.
- **Content overlay (phase 2: profiles).** Bare profile names now resolve from your
  content directory's `profiles/` before the engine's built-in `profiles/`, so a content
  profile of the same name overrides a built-in one and content-only profiles work by
  name (`--profile mine`). Path profiles (`--profile /path/to.yml`) are unchanged. The
  installer also announces the overlay in use and warns when `DOTFILES_CONTENT_DIR`
  points at a directory that does not exist, instead of silently ignoring it.

### Documentation

- **Docs scrub — docs site & new Extending guide (docs-scrub §4.D/§4.G).** Added
  [`docs/extending.md`](docs/extending.md), a hands-on tutorial that builds a `my-dotfiles`
  content repo from scratch — overlay `config.yml`, a content profile, a **custom** module,
  and an **override** module — ending in a working `--dry-run`, referencing the shipped
  [`docs/examples/content-repo/`](docs/examples/content-repo) as the finished result. It is
  registered as a docs-site page (manifest, `setup-docs.sh`, sidebar, `.gitignore`) and
  cross-linked from the docs index, root README, content-overlay, and creating-modules docs.
  Rewrote the two native docs-site pages (`index.mdx` product hero and `guides/setup.mdx`
  quick-setup) from generator placeholders into real content with base-safe root-absolute
  links, and replaced the placeholder "Documentation for dotfiles" site description in all
  three synced spots. `node docs-site/check-docs.mjs` reports no drift (the full Node-22
  Astro build runs on push to `main`).

- **Docs scrub — top-level artifacts & ROADMAP (docs-scrub §4.E).** Deleted the one-off
  `DOCUMENTATION_UPDATES.md` and `UX_ENHANCEMENT_SUMMARY.md` summaries (their content lives
  in this changelog). Refreshed `ROADMAP.md`: genericized the registry example
  (`you/my-module`) and re-verified statuses against what shipped — module **uninstall**
  (1.1) and the **`dotfiles validate`** subcommand (1.2 / 2.6) are marked shipped, and
  external/private modules (2.5) noted as largely delivered by the content overlay, with a
  "Recently Completed" entry for the overlay initiative. Updated `CONTRIBUTING.md` (Go
  ≥1.23, the docs-site four-place sync + `check-docs` guard, and the `plans/` +
  content-overlay workflow).

- **Docs scrub — structural overlay coverage (docs-scrub §4.B).** Extended the architecture
  and authoring docs to cover the content-overlay model: `docs/architecture.md` now
  documents config **deep-merge** layering (`config.Load`), content-wins module
  **discovery** with `built-in`/`override`/`custom` tags and the `list`/`status` **Source**
  column, and flags `noop` as the shipped default provider; `docs/design-rationale.md` adds
  a "generic engine + content overlay" rationale section and reflects `key_source`-gated SSH
  handling; `docs/creating-modules.md` warns that `get_secret` fails (not returns empty)
  under the default `noop` provider and shows a guarded call; `docs/idempotence.md` notes
  that config-hash change detection sees the merged overlay values; `docs/ux-features.md`
  documents the overlay **Source** column; and `docs/README.md` adds the CI/CD guide link.

- **Docs scrub — template context (docs-scrub §4.C).** Corrected the Go template-context
  description everywhere it appears (`docs/architecture.md`, `docs/creating-modules.md`,
  `docs/cli-reference.md`, `docs/troubleshooting.md`) to match
  `internal/template/render.go`: `.User` keys are **lowercase** (`{{ .User.name }}`, not
  `.User.Name`, which renders empty), `.XDGConfigHome` is now listed, `.Module` holds only
  the module's `config.yml` settings (prompt answers arrive via `.Env` as
  `DOTFILES_PROMPT_*`, not in `.Module`), and `.Secrets` is documented as an always-empty
  map during rendering (secrets reach scripts through the `get_secret` helper). The
  troubleshooting "wrong field name" example, which previously recommended the wrong fix,
  now points at `{{ .User.name }}`.

- **Docs scrub — core defaults & narrative (docs-scrub §4.A/§4.F).** Brought the
  operational docs in line with the shipped engine: `secrets.provider: noop` (not
  1Password) shown as the committed default across `README.md`, `docs/quick-start.md`,
  `docs/cli-reference.md`, `docs/troubleshooting.md`, and `docs/ci-cd-guide.md`, with the
  **content overlay** taught as the personalization path and `modules.ssh.key_source`
  documented. Repo-wide sweeps: removed the nonexistent `dotfiles rollback`/`dotfiles
  version` commands, corrected the stale `1password → ssh → git` dependency chain (ssh has
  no deps; `git` depends on `ssh`), replaced the old five-module developer-profile listing
  with the real curated set, standardized Go references on ≥1.23, and fixed `USER` repo-URL
  placeholders to `garygentry`. Also noted the `list`/`status` built-in/override/custom
  **Source** column and that `--content-*` are bootstrap flags, not `dotfiles` flags.

## [2.2.0] - 2026-08-21

### Added

- **`--profile` accepts a path to a profile file.** An argument containing a `/` or
  ending in `.yml`/`.yaml` is read as a literal path (with `~` expanded and relative
  paths resolved from the working directory); anything else is a bare name looked up in
  `profiles/` exactly as before. This lets a project keep its own profile alongside its
  own code instead of adding it to this repo.
- **`uv` module** — the Python package and project manager from Astral, installed via
  Homebrew on macOS, the distro package on Arch, and Astral's installer elsewhere.
- **`ansible` module** — agentless configuration management, from the distribution
  package so its Python interpreter matches the system's.
- **`scripts/testuser.sh`** — a throwaway test-user harness for exercising the installer
  against the local working copy (no GitHub round-trip). Subcommands `create`, `run`,
  `state`, `shell`, `destroy`, `reset`, and `list` create passwordless-sudo users, sync
  the repo into their `~/.dotfiles`, and run the installer unattended. Paired with a new
  `debug-failing` profile scoped to the modules that broke on a fresh WSL install.
- **`ssh` module `key_source` setting** (`generate` | `agent` | `1password` | `none`).
  `agent` skips local key generation and renders a GitHub config **without**
  `IdentityFile`/`IdentitiesOnly`, so keys from an external agent (e.g. the 1Password
  SSH agent) are actually offered — previously the module always pinned a local key and
  set `IdentitiesOnly yes`, which broke agent-based GitHub auth. `1password` retrieves
  the key via `op` (and fails loudly if it can't); `none` leaves `~/.ssh` untouched.
- **Module config settings are now available to shell scripts** as
  `DOTFILES_SETTING_<KEY>` (from `config.yml` → `modules.<name>.*`). Previously scripts
  could only see prompt answers, so a module could not read its own configured settings.
- **Content overlay (phase 1: config).** Point `DOTFILES_CONTENT_DIR` at a directory and
  its `config.yml` is deep-merged over the repo's — scalars override when set, and
  `modules.<name>.*` merges per key, so you can change one setting without redefining a
  module. This lets personal or machine-specific settings live outside the generic repo
  in your own content directory (the same layout the engine uses: `config.yml`,
  `profiles/`, `modules/` — later phases wire up profiles and modules). When
  `DOTFILES_CONTENT_DIR` is unset the engine behaves exactly as before. See
  `config.overlay.example.yml`.

### Changed

- **A missing explicitly-requested profile is now a hard error.** Asking for a profile
  with `--profile` or `DOTFILES_PROFILE` and not getting it exits non-zero instead of
  silently falling back to installing *every* module — the previous behaviour turned a
  typo into a full install. A profile named only in `config.yml` still falls back, since
  that is a default rather than a request.
- **The committed `config.yml` now ships conservative, unopinionated defaults** for the
  widest audience: no secret provider (`noop`) and `ssh.key_source: generate`. Personal
  or machine-specific choices (e.g. `1password` + `key_source: agent`) belong in your
  content overlay rather than the shared repo.

### Fixed

- **Package installs now refresh the index first.** `pkg_install` runs `apt-get update`
  (or `pacman -Sy`) once per script before the first install. On a fresh system — e.g. a
  new WSL Ubuntu — the apt lists start empty, so package installs previously failed with
  exit 100 (*"Unable to locate package"*). This affected every module that installs a
  plain distro package without its own `os/*.sh` refresh (ripgrep, btop, nodejs, fzf).
- **`python` module now installs `pip3`.** `install.sh` short-circuited when `python3`
  was already present (as it is on most distros) and never installed `python3-pip`, so
  `verify.sh` failed on a fresh machine. It now installs both unconditionally
  (idempotently).
- **`nodejs` module ignores a Windows `node` on WSL.** A `node` resolving under `/mnt/`
  (the Windows PATH bleeding into WSL) no longer counts as installed, so the real Linux
  package is installed instead of being silently skipped.
- **`fzf` git fallback hardened.** The fallback no longer swallows the package manager's
  error output, reuses an existing `~/.fzf` checkout, and cleans up a partial one before
  cloning.
- **`ssh` module `verify.sh` now actually fails.** It previously counted problems as
  *warnings* and always exited 0, so ssh was marked `installed` even with no key or
  config, and the runner's post-install re-verify could never detect a missing key. It
  now exits non-zero when an essential artifact is missing (key/config for managed
  modes; config for `agent`; nothing for `none`).
- **`ssh` key generation no longer aborts when `USER` is unset.** The key comment fell
  back to `${USER}@host`, which crashed under `set -u` if `USER` was absent from the
  environment; it now derives the name from `id -un`.
- **A clean re-run no longer erases rollback history.** Files skipped as up-to-date now
  still record a `file_deploy` operation, so a no-op re-run keeps the module
  uninstallable — previously the second run persisted zero operations and a later
  `uninstall` reported nothing to roll back and left the deployed files in place.
- **User-edited deployed files are backed up before being overwritten.** A copy/template
  whose destination diverges from what was last deployed — including when the source
  *also* changed in the same interval, previously a silent data-losing overwrite — is now
  backed up first, and the backup path is recorded so rollback can restore it.
- **Rollback can actually restore a modified file.** `createBackup` now returns its path
  and the deploy records it as `backup_path`; previously the restore branch read a field
  that was never written, so it silently did nothing.
- **A failed re-run no longer blanks the last-good checksums.** The prior module and
  config hashes are carried into the run, so a failure persists them instead of clearing
  the change-detection guards for the next run.
- **`lib/helpers.sh` changes now invalidate module checksums.** Every module sources it,
  so editing shared helper logic re-runs each module once (a one-time cost) instead of
  leaving them all reporting "up-to-date" against changed behavior.
- **The verify-on-skip check gets the same prompt defaults an install would.** It ran
  with an empty prompt environment, so a `verify.sh` reading `DOTFILES_PROMPT_*` could
  fail spuriously and force a needless full reinstall.

## [2.1.0] - 2026-02-16

### Added

- **🎨 Comprehensive UX Overhaul**: Dramatically improved terminal user experience with modern, polished interface

  - **Compact Grid-Based Module Selection**
    - Interactive grid layout (3-5 columns) reduces module selection from 28+ lines to ~10 lines (64% reduction)
    - Keyboard navigation with arrow keys (↑↓←→)
    - Quick selection shortcuts: Space (toggle), A (select all), N (select none)
    - Live preview pane shows full description of highlighted module
    - Catppuccin Mocha theming throughout

  - **Collapsible Script Output**
    - Smart output buffering with spinner animations during execution
    - Automatic sudo detection preserves interactivity when needed
    - Compact one-line summaries on success (90% output reduction)
    - Auto-expanding bordered error boxes showing last 30 lines on failure
    - `--verbose` flag streams all output for debugging
    - Reduces total installation output from 200+ lines to 50-80 lines

  - **Overall Progress Tracking**
    - Real-time progress bar showing current module and completion percentage
    - Elapsed time and estimated time remaining (based on average module duration)
    - Completion summary with success/failed/skipped counts
    - Total installation time tracking

  - **Enhanced Script Execution Feedback**
    - Real-time output parsing to detect current operations
    - Pattern recognition for `log_info`, `log_success`, and `pkg_install` helpers
    - Context-aware progress updates during script execution

### Changed

- **Module Selection**: Interactive grid-based selector replaces vertical list in TTY mode
- **Script Output**: Buffered with spinners by default; streams when sudo detected or `--verbose` enabled
- **Progress Display**: Added progress bar and time estimation to module execution

### Technical Details

- New files:
  - `internal/ui/multiselect.go` - Custom Bubble Tea grid component
  - `internal/ui/progress.go` - Progress bar component with time estimation
- Enhanced `internal/module/runner.go` with output buffering and pattern recognition
- Updated `RunnerUI` interface with progress tracking methods
- All changes maintain full backward compatibility (non-TTY graceful degradation)

### Performance

- Minimal overhead: Added complexity is primarily I/O-bound (already the bottleneck)
- Script output buffering typically < 1MB per script
- Grid rendering and progress updates are O(n) where n is small

## [2.0.0] - 2026-02-11

### ⚠️ Breaking Changes

- **starship module**: Removed `zsh` dependency. Starship is correctly modeled as a cross-shell prompt that works with any shell (fish, bash, zsh, etc.). If you were relying on starship to auto-install zsh, explicitly add zsh to your profile or installation command.

### Added

- **Smart Prompt Behavior**: Only explicitly selected modules show interactive prompts. Auto-included dependencies use sensible defaults.
  - When you run `dotfiles install gh`, only `gh` shows prompts
  - Dependencies (`git`, `ssh`) use their default values automatically
  - Reduces friction and confusion during installation

- **`--prompt-dependencies` flag**: Force interactive prompts for all modules, including auto-included dependencies
  - Use when you want to configure dependency modules during installation
  - Example: `dotfiles install gh --prompt-dependencies` will prompt for gh, git, and ssh configurations

- **`show_when` prompt field**: Module authors can now control when prompts are shown
  - `explicit_install`: Only show when module is explicitly selected (default)
  - `always`: Always show, even for auto-included dependencies
  - `interactive`: Always show in interactive mode
  - Added to zsh module prompts (framework, plugins, theme)

### Changed

- **Prompt behavior**: Modules installed as dependencies (not explicitly requested) now use default values for all prompts instead of showing interactive prompts
- **Dependency tracking**: ExecutionPlan now tracks which modules were explicitly requested vs auto-included
- **Verbose logging**: With `-v` flag, you can see which defaults are being used for auto-included modules

### Technical Details

- Added `ExplicitlyRequested map[string]bool` to `ExecutionPlan` in resolver
- Added `ExplicitModules` and `PromptDependencies` fields to `RunConfig`
- Implemented `shouldShowPrompt()` logic in runner to filter prompts based on context
- Updated `handlePrompts()` to respect explicit vs auto-included distinction

### Migration Guide

**If you use starship with fish/bash:**
- No action needed! Starship will install without pulling in zsh

**If you relied on starship to install zsh:**
- Add `zsh` explicitly to your profile or install command
- Before: `dotfiles install starship` (installed both starship and zsh)
- After: `dotfiles install starship zsh` (explicit selection)

**If you want to configure auto-included dependencies:**
- Use the new `--prompt-dependencies` flag
- Example: `dotfiles install neovim --prompt-dependencies` will prompt for neovim, git, and ssh

## [1.1.0] - 2026-02-10

### Added
- Comprehensive idempotence system with update detection
- Post-install notes system for modules
- 12 new modules: docker, rust, awscli, azure-cli, gcloud, claude-code, gemini-cli, ghostty, tmux, zellij, zoxide, btop
- Oh My Zsh as alternative plugin framework for zsh module

### Changed
- Improved git-delta module to handle sudo prompts
- Enhanced state tracking with operation history

### Fixed
- Fixed git-delta sudo prompt issues

## [1.0.0] - 2026-02-09

### Added
- Initial release with core dotfiles management system
- Go-based CLI with shell module execution
- Dependency resolution using Kahn's algorithm
- State tracking and idempotent operations
- Template rendering with Go templates
- 1Password secrets integration
- Core modules: 1password, ssh, git, zsh, neovim, fonts, fzf, ripgrep, lazygit, gh, fish, starship, python, golang, nodejs
- Profile system for module sets
- Interactive and unattended modes
- Comprehensive test suite

[Unreleased]: https://github.com/garygentry/dotfiles/compare/v3.0.0...HEAD
[3.0.0]: https://github.com/garygentry/dotfiles/compare/v2.3.0...v3.0.0
[2.3.0]: https://github.com/garygentry/dotfiles/compare/v2.2.0...v2.3.0
[2.2.0]: https://github.com/garygentry/dotfiles/compare/v2.1.0...v2.2.0
[2.1.0]: https://github.com/garygentry/dotfiles/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/garygentry/dotfiles/compare/v1.1.0...v2.0.0
[1.1.0]: https://github.com/garygentry/dotfiles/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/garygentry/dotfiles/releases/tag/v1.0.0
