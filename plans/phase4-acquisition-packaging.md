# Plan: Content Overlay — Phase 4 (Acquisition, Packaging & Cleanups)

**Status:** 4A ✅ merged (#8); 4B ✅ merged (#9); 4C ✅ merged (#10); 4D ✅ merged (#11); 4E IN REVIEW (branch `track4/packaging-docs`) · **Branch:** `track4/*` per task group — **final task group**
**Tracks:** Content-overlay model, phase 4 of 4 (final). Multi-session plan — update
checkboxes and the Status line as you go; commit the updated plan with the work.

---

## 1. Context (self-contained — read first)

This repo is a Go + shell **dotfiles engine** gaining an optional **content overlay**:
a user directory (`$DOTFILES_CONTENT_DIR`) laid out like the repo (`config.yml`,
`profiles/`, `modules/`) whose contents overlay the generic repo, so personal
customization lives outside it. See `plans/README.md` for the whole initiative and
`plans/phase3-content-modules.md` for the phase that precedes this one.

By the time this phase starts, the engine **consumes** a local content directory
fully (config + profiles + modules, content-wins). Phase 4 is everything around
that contract: how the directory gets **acquired**, the **packaging** story for a
personal "my-dotfiles" content repo, and a few **cleanups** that were deferred.

### Guiding principles (non-negotiable)
- **Backward compatible.** No content dir / no new flags → behavior identical to
  today. Everything here is additive and opt-in.
- **Simplest, most robust.** The engine stays *consumption-only* (a local content
  dir). Acquisition is a thin, separate layer — prefer shell/bootstrap over adding
  network/git/auth logic into the Go core.
- **Support multiple patterns.** Local dir, public git repo, private git repo
  (auth first), network/FS path — all resolve to the same local content dir.

### Key architectural separation (do not violate)
**Acquiring** the content is separate from **consuming** it. The engine already
consumes a local dir; acquisition just *materializes* that dir from a source, then
points the engine at it. Keep them decoupled so new source patterns never touch the
core.

---

## 2. Task groups (largely independent; ship as separate PRs)

### 4A — Acquisition & bootstrap
Make `bootstrap.sh` able to fetch a content repo/path into the content dir, then
run the engine against it. The engine core does NOT gain git/network logic.

- [x] Add bootstrap flags: `--content-repo <url>` (+ optional `--content-ref`),
      `--content-path <local-or-network-path>`, and `--content-dir <dest>`
      (default `~/.config/dotfiles`). Resolve → materialize → `export
      DOTFILES_CONTENT_DIR`.
- [x] Source kinds: existing local dir (use in place), git repo (clone/pull),
      network/FS path (copy or use in place). Idempotent re-runs (pull if a clone
      already exists).
- [x] **Auth ordering** for private sources: run an optional pre-fetch auth step
      BEFORE the clone. Default is ambient (e.g. a running SSH/1Password agent);
      allow a configurable hook command and/or env credential. Document the
      chicken-and-egg honestly: it only bites when the auth material itself lives
      in the private content — recommend a **public, secret-free** content repo
      (real secrets stay in 1Password at runtime).
      (Implemented as `--content-auth-cmd`/`DOTFILES_CONTENT_AUTH_CMD`, run before
      the clone; ambient agent is the no-hook default.)
- [x] Persist `DOTFILES_CONTENT_DIR` for future shells (write to `~/.zshenv` or
      print the exact line to add). Don't bake it into a repo-managed file.
      (Appends to `~/.zshenv` idempotently, prints the line, opt out with
      `--no-persist-content-dir`.)
- *Acceptance:* clean-machine flow works for (a) public repo, (b) local path;
      private-repo path documented + works when an agent is present. No-flags
      bootstrap is unchanged.
      *Status:* acquisition paths proven hermetically (local dir + a local bare
      "remote" git repo) in `test/bootstrap/acquire_test.sh` (11 cases, no root/
      network); full clean-machine `scripts/testuser.sh` run needs root and is
      deferred to a machine that has passwordless sudo.

### 4B — Parameterize the SSH secret reference
The ssh module hardcodes `op://Personal/SSH Key/<type>` (`modules/ssh/install.sh`,
~line 93). Make it configurable so it isn't personal-specific.

- [x] Read the 1Password item reference from a module setting (e.g.
      `modules.ssh.key_item: "op://Personal/SSH Key"`), exposed to the script as
      `DOTFILES_SETTING_KEY_ITEM` (the `DOTFILES_SETTING_*` plumbing already
      exists). Fall back to the current default if unset.
- *Acceptance:* `key_source: 1password` uses the configured ref; default preserved;
      unit/e2e as feasible.
      *Status:* done. `modules/ssh/install.sh` reads `DOTFILES_SETTING_KEY_ITEM`
      (default `op://Personal/SSH Key`, trailing slash tolerated) and appends the key
      type as the field. Documented in `config.yml` + `config.overlay.example.yml`.
      Proven hermetically in `test/bootstrap/ssh_key_item_test.sh` (default, custom,
      trailing-slash cases); no root/network needed. `key_source: generate` (the
      committed default) never touches 1Password, so the default path is unchanged.

### 4C — Unify the two template contexts
The shell-invoked `render-template` subcommand (`cmd/dotfiles/render_template.go`)
builds a DIFFERENT context than the in-process path: `.User`, `.Module` (config
settings), and `.Secrets` are absent, so a template rendered from a shell script
silently gets empty values for those. Make the two paths equivalent.

- [x] Populate `.User`, `.Module` (from `config.yml modules.<name>.*`, incl. the
      overlay), and `.XDGConfigHome` in the subcommand — sourced from the same
      config the runner uses (load config + content overlay in the subcommand, or
      pass them through env). Keep `.Secrets` explicitly empty but consistent (or
      document that secrets reach scripts only via `get_secret`).
- [x] Test parity: the same template renders identically in-process and via the
      subcommand.
- *Acceptance:* a template using `{{ .User.email }}` / `{{ .Module.key_source }}`
      renders correctly from a module's `render_template` call.
      *Status:* done. Both paths now build the context through one shared
      `module.NewTemplateContext` (runner.go); the `render-template` subcommand
      (`cmd/dotfiles/render_template.go`) reuses `config.Load` — the SAME layered
      base+overlay load — so `.User`, `.Module` (type-preserving, incl. overlay),
      and `.XDGConfigHome` are populated identically. `.Secrets` is a consistent
      empty map in both; secrets reach scripts only via `get_secret`. Parity is
      covered by a table-driven `t.TempDir()` test in
      `cmd/dotfiles/render_template_test.go` (no-overlay + content-overlay cases,
      including a bool setting to prove type preservation). Acceptance verified
      end-to-end against the built binary.

### 4D — Dead-config cleanup
Some `config.yml` settings look wired but aren't consumed.

- [x] `modules.git.default_branch` — made live. `modules/git/install.sh` now
      reads `DOTFILES_SETTING_DEFAULT_BRANCH` (default `main`) for
      `git config --global init.defaultBranch`, and `modules/git/verify.sh`
      checks against the same configured value. Default preserves today's
      behavior exactly.
- [x] `modules.zsh.theme` — removed. It was silently ignored, its committed
      default (`powerlevel10k`) isn't a supported/installed option, and it
      overlapped the existing `zsh_prompt` prompt that already drives
      `ZSH_THEME`. Making it live at that default would have changed behavior at
      defaults (breaking backward compat), so removal was the honest, plan-
      sanctioned choice. Dropped from `config.yml`, `README.md`, and
      `docs/quick-start.md`.
- *Acceptance:* no committed config key is silently ignored; changing it changes
      behavior (or it's gone).
      *Status:* done. `default_branch` proven live by a positive Go unit test
      (`TestBuildEnvVarsGitDefaultBranch`, runner_test.go) and a hermetic shell
      test (`test/bootstrap/git_default_branch_test.sh`, 3 cases: default `main`,
      `trunk`, `develop`; no root/network). `zsh.theme` gone. Note:
      `modules.neovim.colorscheme` (init.lua hardcodes `catppuccin`) is a
      separate dead key outside 4D's scope — flagged for a follow-up.

### 4E — Packaging docs
- [x] Documented the two-repo model and the clean-machine first-install flow in a
      new [`docs/content-overlay.md`](../docs/content-overlay.md) page (engine +
      `my-dotfiles` content repo; public/secret-free recommended; the
      1Password-agent private path + `--content-auth-cmd`; the chicken-and-egg
      called out honestly). Registered as a docs-site page (manifest + symlinker
      + sidebar) and cross-linked from `docs/README.md`, `README.md`, and
      `docs/creating-modules.md`.
- [x] Added a minimal, verified example content repo at
      [`docs/examples/content-repo/`](../docs/examples/content-repo/): overlay
      `config.yml` + `profiles/mine.yml` + a **custom** `hello` module + an
      **override** of the built-in `git` module, with its own README.
      Referenced from `config.overlay.example.yml` (and the new doc).
- *Acceptance:* a new user can follow the doc end to end.
      *Status:* done. The example is proven against the built engine:
      `DOTFILES_CONTENT_DIR=docs/examples/content-repo ./bin/dotfiles install
      --profile mine --dry-run` plans ssh→hello→git→zsh, with `dotfiles list`
      tagging `hello` **custom** and `git` **override**. Docs drift guard
      (`docs-site/check-docs.mjs`) passes locally; the full Astro build runs in
      CI on Node 22 (the docs workflow triggers on push to `main`, not on PRs).

---

## 3. Definition of done
- Bootstrap can acquire a content dir from a public repo and a local path, sets
  `DOTFILES_CONTENT_DIR`, and installs; private-repo path documented and works with
  an agent. No-flags bootstrap unchanged.
- SSH secret ref is configurable; the two template contexts render identically; no
  dead config keys remain.
- Docs describe the clean-machine two-repo flow.
- `go test -race ./...` + `go vet` + integration (Ubuntu + Arch) green; CHANGELOG
  updated; plan checkboxes updated.

## 4. Risks / watch-outs
- **Don't put git/auth in the Go core.** Keep acquisition in bootstrap/shell.
- **Private-repo chicken-and-egg.** Lead users to public+secret-free; treat the
  private path as advanced. Never write secrets into the content repo.
- **4C config loading in a subprocess.** The subcommand must load the SAME layered
  config (base + content overlay) or templates diverge again — reuse `config.Load`.

## 5. Workflow / conventions
Same as the rest of the initiative — see `plans/README.md` §"Workflow" (branch from
`main`, SSH-signed commits via 1Password, PR + CI + squash-merge, CHANGELOG under
`[Unreleased]`, update this plan's checkboxes). Ship each task group as its own PR.

## 6. How to resume
1. Ensure phase 3 is merged. `git checkout main && git pull`.
2. Pick a task group (4A–4E), branch `track4/<group>` from `main`.
3. Read this file; do the group's tasks; test; commit (updating checkboxes); PR.
