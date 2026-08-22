# Plan: Comprehensive Documentation Scrub

**Status:** 🚧 IN PROGRESS — §4.H (ssh key_item default) merged (#15); §4.A + §4.F
sweeps F1–F7 on branch `docs/scrub-core-defaults`. Remaining: §4.C template-context
(incl. F8), §4.B structural, §4.D/§4.G site, §4.E artifacts. · **Branch:** `docs/scrub-*`
per work group
**Scope:** All human-facing docs (repo markdown + the Astro/Starlight docs site) brought
into line with the merged content-overlay initiative (phases 1–4), a new **Extensibility
guide + tutorial** (§4.G), and one **deliberate default change** (ssh `key_item`, §4.H).
Mostly docs; the single engine change is called out explicitly. Multi-session; update the
checkboxes and Status line as you go and commit them with the work.

> **Execute this in a clean session.** It is self-contained: §2 is the authoritative
> "current truth" to align docs to; §4 is the itemized work with file:line anchors;
> §5 is the docs-site mechanics you must respect; §7 is the definition of done. Read
> `plans/README.md` first for the initiative context and the shared Workflow section.

> **Guiding principle (maintainer directive, 2026-08-21):** choose defaults by what is
> **genuinely best**, NOT by preserving backward compatibility. Where a default is
> suboptimal (personal-flavored, legacy), change it to the better one and accept the
> back-compat break (with a CHANGELOG note). This deliberately loosens the "strictly
> backward-compatible at defaults" constraint that governed phases 1–4 — it is not a
> license for churn; keep changes minimal and in the right layer. (The overlay's opt-in
> property — no content dir ⇒ engine still works — is correctness, not a "default", and
> still holds.)

---

## 1. Why

Phases 1–4 of the content-overlay initiative shipped (config/profiles/modules overlay,
depersonalization, re-run/rollback safety, ssh `key_source`/`key_item`, template-context
unification, dead-config cleanup, bootstrap acquisition, packaging docs). The **engine**
is current; much of the **documentation predates these changes** and now misstates
defaults, the personalization model, the CLI surface, and the template context. A
focused audit (README + all of `docs/`, the `docs-site/`, and a repo-wide grep) found
concrete drift, catalogued in §4. This plan fixes all of it.

Line numbers below are anchors from the audit and **will drift as you edit** — re-grep
the quoted text if a line doesn't match.

---

## 2. Ground truth (align every doc to THIS)

Verified against code on the branch this plan was written from. If a doc contradicts
this section, the doc is wrong.

### 2.1 Shipped defaults (committed `config.yml`)
- `secrets.provider: **noop**` (NOT 1password). 1Password/any provider is **opt-in**,
  via the overlay or `DOTFILES_SECRETS_PROVIDER`.
- `modules.ssh.key_source: **generate**`. Values: `generate | agent | 1password | none`.
- `modules.ssh.key_item` default is being **changed** from the personal-flavored
  `op://Personal/SSH Key` to a neutral generic (recommended `op://Private/SSH Key` —
  1Password's current default personal-vault name). Used only when `key_source: 1password`
  (key type appended as the field). Deliberate back-compat break per the guiding principle
  (§4.H, §6-D1). Docs must describe whatever value ships as the new default.
- `modules.git.default_branch: main` — **live** (git module reads
  `DOTFILES_SETTING_DEFAULT_BRANCH`).
- `modules.zsh.theme` — **REMOVED** (was dead). zsh prompt/theme comes from the
  `zsh_prompt` prompt (`starship|robbyrussell|agnoster`), not a config key.
- `modules.neovim.colorscheme: catppuccin` — still **dead config** (init.lua hardcodes
  it). Tracked in **issue #13**; do NOT fix it here, but docs must not present it as a
  working knob.
- Full committed key set: `profile`; `secrets.{provider,account}`;
  `user.{name,email,github_user}`; `modules.ssh.{key_type,key_source,key_item}`;
  `modules.git.default_branch`; `modules.neovim.colorscheme`. No other `modules.*` keys
  exist — any doc showing one is wrong.

### 2.2 Personalization model (the central narrative)
- The repo is a **generic engine**. Personal config/profiles/modules live in an optional
  **content overlay** dir (`$DOTFILES_CONTENT_DIR`), laid out like the repo, deep-merged
  over it. Canonical guide: `docs/content-overlay.md`; example: `docs/examples/content-repo/`;
  annotated overlay: `config.overlay.example.yml`.
- Docs must stop telling users to put identity/secrets in the committed
  `~/.dotfiles/config.yml`; the intended path is the overlay. (Editing the repo config
  still works, but is not what we teach.)

### 2.3 CLI surface (authoritative — from `cmd/dotfiles/`)
- Commands: `install`, `uninstall`, `list`, `status`, `new`, `validate`, `get-secret`,
  `render-template` (+ cobra's `help`/`completion`). **There is NO `version` command**
  and **no `rollback` command** (uninstall is the reversal path).
- Flags: root/persistent `--verbose/-v`, `--dry-run`, `--log-json`, `--unattended`;
  `install`: `--profile`, `--fail-fast`, `--force`, `--skip-failed`, `--update-only`,
  `--prompt-dependencies`; `uninstall`: `--force`; `validate`: `--json`, `--strict`;
  `new`: `--priority`, `--depends`, `--os`; `get-secret`: `--ref`; `render-template`:
  `--src`, `--dest`, `--module`.
- The `--content-*` flags (`--content-repo/-ref/-path/-dir/-auth-cmd`,
  `--no-persist-content-dir`) are **`bootstrap.sh` flags, NOT `dotfiles` flags**. Never
  show them as `dotfiles install --content-...`.
- `dotfiles list` / `dotfiles status` now show a **built-in / override / custom** tag per
  module (overlay feature).

### 2.4 Template context (single source of truth: `internal/template/render.go:12` + `internal/module/runner.go` NewTemplateContext)
Exact fields — nothing more, nothing less:
- `.User` — `map[string]string` with **lowercase** keys `name`, `email`, `github_user`.
  So `{{ .User.name }}` / `.email` / `.github_user`. `{{ .User.Name }}` renders EMPTY and
  is wrong everywhere it appears.
- `.OS`, `.Arch`, `.Home`, `.DotfilesDir`, **`.XDGConfigHome`** (often omitted in docs — add it).
- `.Module` — `map[string]any` of **config settings only** (`config.yml modules.<name>.*`,
  incl. overlay, type-preserving). It does **NOT** contain prompt answers. Prompt answers
  reach templates/scripts via `.Env`/env as `DOTFILES_PROMPT_*`.
- `.Secrets` — always an **empty (non-nil) map** for script/subcommand rendering; secrets
  reach scripts only via the `get_secret` helper. Do not describe it as "retrieved secrets".
- `.Env` — environment overrides (`index .Env "DOTFILES_PROMPT_X"` is the real pattern;
  see `modules/ssh/config`, `modules/zsh/zshrc.tmpl`).
- Since phase 4C, the shell-invoked `render-template` subcommand builds this **same**
  context (with overlay) as the in-process runner.

### 2.5 Other facts
- `profiles/developer.yml` = `ssh, git, zsh, starship, tmux, docker, fzf, neovim, btop,
  claude-code, nodejs` (11 modules, no `1password`). ~**30** modules exist under `modules/`.
  Developer is a **curated set**, not "all modules".
- Module dependencies: `ssh` deps `[]` (NOT 1password); `git` deps `[ssh]`; `zsh` deps
  `[git]`. The old `1password → ssh → git` chain shown in several docs is stale.
- Go: `go.mod` requires **1.23**; bootstrap installs 1.23.6. Standardize doc references
  on **Go ≥ 1.23** (kill "1.22"/`golang:1.22-alpine`).
- Canonical repo URL is `github.com/garygentry/dotfiles` — keep it (it is the real engine
  repo). Content-repo examples use `you/my-dotfiles` — keep.

---

## 3. Approach & PR slicing

Ship as a few reviewable, docs-only PRs off `main` (branch `docs/scrub-<group>`), not one
mega-diff. Recommended order (each independently mergeable):

1. **`docs/scrub-core-defaults`** — §4.A (central narrative + provider/ssh defaults) and
   §4.F cross-file sweeps (provider representation, `rollback`→`uninstall`, `version`
   removal, ssh-dep chain, developer profile, Go version, repo-URL placeholder). Highest
   user impact.
2. **`docs/scrub-template-context`** — §4.C (fix the template-context block in the 3 docs
   + troubleshooting). Small, surgical, one canonical block.
3. **`docs/scrub-site`** — §4.D (rewrite the two native docs-site pages + site metadata)
   and the new **Extensibility guide/tutorial page** §4.G (also a docs-site page). Respect
   §5. This is the only group touching `docs-site/`.
4. **`docs/scrub-artifacts`** — §4.E (delete the two stale summaries, ROADMAP genericize,
   CONTRIBUTING check).
5. **`fix/ssh-key-item-default`** — §4.H: the deliberate ssh `key_item` default change
   (NOT docs-only — code + comments + docs + test + CHANGELOG "breaking default" note).
   Ship this first or standalone; the doc groups then describe the new default value.

The §6 decisions are now resolved (see §6). Update this plan's checkboxes + Status and add
a CHANGELOG `[Unreleased]` entry per PR.

---

## 4. Work items

### A. Central narrative & shipped defaults (across operational docs)
The recurring fix: (a) stop showing `provider: 1password` as the committed default; (b)
teach the **overlay** as the personalization path; (c) show/describe `ssh.key_source`.

- [x] `README.md:172-174` config example → `provider: noop` (or drop the `secrets:` block
      and point to the overlay). `:18` & `:236` soften "Integrated/Seamless 1Password" to
      "optional 1Password (opt-in) secrets".
- [x] `README.md:155-159` Modules table: fix ssh "Dependencies: 1password" → none;
      zsh "Zinit" → "Zinit or Oh My Zsh"; note the table is a sample of ~30 modules.
- [x] `README.md:181-190` config example: add `key_source` under `ssh:` so readers learn
      generation is selectable.
- [x] `README.md:163-186` "Configuration" section: reframe personalization to the overlay
      (link `docs/content-overlay.md` / `config.overlay.example.yml`) instead of editing
      `~/.dotfiles/config.yml` in place.
- [x] `README.md:194-203` Developer Profile block → real `profiles/developer.yml` contents.
- [x] `docs/quick-start.md:208-210` → `provider: noop`; `:30-33` sample `list` output →
      current modules/phrasing; `:167` "all modules" → curated set; add an overlay pointer
      in the Configuration section (198-222).
- [x] `docs/installation.md`: add a short **content-overlay bootstrap** subsection
      (`--content-repo` / `--content-path` / `DOTFILES_CONTENT_DIR`) linking
      `docs/content-overlay.md`; fix the `USER` vs `garygentry` placeholder (see §4.F);
      align Go version (§2.5).
- [x] `docs/cli-reference.md:568-570` → `provider: noop`; `:207-215` sample `list` →
      current modules + note the built-in/override/custom **tag column**; document
      `render-template --module`; add a pointer that `--content-*` are bootstrap flags.
- [x] `docs/troubleshooting.md`: SSH section (130-157, 335, 574) → describe `key_source`
      (generate is the default, not a fallback); "Skip Secrets" (333-341) → note default is
      already `noop`; add an overlay-not-applied / override-name gotcha entry.
- [x] `docs/ci-cd-guide.md`: replace `provider: ""` (53-54, 348-349, 653-655) with `noop`;
      mark nonexistent profiles (`server`/`ci`/`prod`/…) as "create your own"; add a
      content-overlay-in-CI note (`--content-repo` to materialize config in a container);
      align Go version (`:147`).

### B. Architecture / rationale / creating-modules (structural coverage) ✅ DONE (branch `docs/scrub-structural`)
- [x] `docs/architecture.md`: fix the `Context` struct (201-212) per §2.4; replace the
      stale `1password→ssh→git` dependency example (362-378); note `noop` is the shipped
      default (187-193); **add** coverage of: config **overlay** layering, content-wins
      module **discovery** with built-in/override/custom tags (100-108), and the unified
      in-process vs `render-template` context (4C).
- [x] `docs/design-rationale.md`: add the **engine vs content-overlay** separation as a
      rationale point; `:31` reword "SSH key generation" to reflect `key_source` gating.
- [x] `docs/creating-modules.md`: fix the template-context section (463-474) and the
      gitconfig example (481-483) per §2.4 (lowercase `.User` keys, `.Module` = settings
      not prompts, `.Secrets` empty, add `.XDGConfigHome`); fix the priority/deps example
      (580-584) stale 1password chain; note default provider is `noop` near the
      `get_secret` example (373-378). (The overlay section 750-777 is correct — leave it.)
- [x] `docs/idempotence.md:431`: `dotfiles rollback zsh` → `dotfiles uninstall zsh`; add
      one line that overlay-merged config participates in the config-hash change detection.
- [x] `docs/ux-features.md`: add the `list`/`status` **built-in/override/custom** tag to
      the described UX (brief).
- [x] `docs/README.md` (index): add the missing **CI/CD Guide** link under Operations;
      confirm **Content Overlay** link present (it is).

### C. Template-context correctness (surgical) ✅ DONE (branch `docs/scrub-template-context`)
Fix the SAME context description everywhere it appears, using the §2.4 block verbatim as
the model. Files: `docs/architecture.md:201-212`, `docs/creating-modules.md:463-483`,
`docs/cli-reference.md:476-491`, and `docs/troubleshooting.md:499-503` (its "fix" is
itself wrong — the correct access is lowercase `.User.name`). (Overlaps §4.B by design;
if you ship the template-context PR separately, do these four together.)
- [x] All four blocks corrected to §2.4: lowercase `.User` keys, `.XDGConfigHome` added,
      `.Module` = config settings only (prompt answers via `.Env` `DOTFILES_PROMPT_*`),
      `.Secrets` an empty map (secrets via `get_secret`). F8 sweep clean.

### D. docs-site native pages & metadata (see §5 for mechanics — do not break the guard)
- [ ] Rewrite `docs-site/src/content/docs/guides/setup.mdx` — currently pure generator
      placeholder, yet it is the homepage "Get Started" target. Make it a real quick-setup
      (bootstrap one-liner, profile pick, the overlay path). Keep `title` frontmatter;
      internal links MUST be root-absolute `/slug/` (native-page rule, §5).
- [ ] Rewrite `docs-site/src/content/docs/index.mdx` — replace boilerplate hero/cards with
      a real product description (engine + overlay, key features). Fix the base-unsafe hero
      `link: guides/setup/` (`index.mdx:9`) to `/guides/setup/`. Keep body Card links
      root-absolute.
- [ ] Replace the placeholder description "Documentation for dotfiles" in all three synced
      spots: `docs-site/docs.manifest.json:4`, `docs-site/astro.config.mjs:32`,
      `index.mdx` frontmatter — with one real one-line description.
- [ ] Register the new **Extensibility** page (§4.G) as a docs-site page — manifest +
      `setup-docs.sh` + sidebar + `.gitignore`, in order (§5).
- [ ] (Optional) Consider a docs-site nav grouping so Overview/Installation/Quick
      Start/**Content Overlay**/**Extending** read as a "Getting started" cluster. If you
      touch the sidebar, keep manifest↔sidebar parity + order (§5).

### E. Top-level artifacts, ROADMAP, CONTRIBUTING
- [ ] **DELETE** `DOCUMENTATION_UPDATES.md` (376 lines) and `UX_ENHANCEMENT_SUMMARY.md`
      (288 lines) — one-off v2.1.0 summaries whose content is already in `CHANGELOG.md`
      (§6-D2, decided: delete). Remove any inbound links (grep first).
- [ ] `ROADMAP.md`: `:308` genericize `dotfiles add garygentry/my-module` →
      `you/my-module`. Re-verify statuses against what shipped (uninstall/rollback-history,
      schema validation partially advanced — see `plans/README.md` "Overlap" note); mark
      accordingly. Optionally add a one-line pointer to the content-overlay/extending docs.
- [ ] `CONTRIBUTING.md`: quick pass for stale build/test commands and to mention the
      overlay/plans workflow; no known hard errors — verify.
- [ ] `config.yml` / `config.overlay.example.yml` / `modules/ssh/install.sh` comments: fold
      the new ssh `key_item` default wording (§4.H) into these — they must show the new
      generic default, not `op://Personal/SSH Key`.

### F. Cross-file consistency sweeps (grep-driven; do once, repo-wide)
- [x] **Provider default**: no doc shows `provider: 1password` or `provider: ""` as the
      committed/default value. `rg -n "provider: *1password|provider: *\"\""` → all fixed
      to `noop` or clearly opt-in.
- [x] **`rollback` command**: `rg -n "dotfiles rollback"` → none (use `uninstall`).
- [x] **`version` command**: `rg -n "dotfiles version"` → none; remove the
      `cli-reference.md:493-503` section (no such command).
- [x] **ssh dependency chain**: `rg -n "1password.*ssh|ssh.*1password"` in docs → no
      dependency claims; ssh deps are `[]`.
- [x] **developer profile**: every listing matches `profiles/developer.yml`.
- [x] **Go version**: `rg -n "1\.22|golang:1\.22"` → standardize on ≥1.23.
- [x] **repo URL placeholder** (D4 → `garygentry`): replace `USER` in
      `installation.md`/`quick-start.md` raw URLs with `garygentry`; reserve
      `<your-fork>` only where a doc is explicitly about forking.
- [x] **template context**: `rg -n "\.User\.Name|\.Secrets\.|Module settings + prompts"` →
      reconciled with §2.4.

### G. Extensibility guide + hands-on tutorial (NEW page)
Create a dedicated, task-oriented page for building your own setup — the "how do I make
this mine?" companion to the conceptual `docs/content-overlay.md`. Proposed
`docs/extending.md`, slug `extending`, sidebar label "Extending". Register as a docs-site
page (§5 four-place sync + `.gitignore` + rerun `setup-docs.sh` + `check-docs.mjs`).

- [ ] **Concept intro** — what extensibility means here: a personal **content repo**
      (`my-dotfiles`) overlaying the generic engine; the three extension points (config
      overlay, content **profiles**, content **modules**: custom + override); secrets stay
      in a provider, never committed. Link `docs/content-overlay.md` for the deep model and
      `docs/creating-modules.md` for module-authoring reference (don't duplicate them).
- [ ] **Tutorial — build a `my-dotfiles` repo from scratch** (numbered, copy-pasteable,
      ending in a working `--dry-run`). Steps:
      1. `git init my-dotfiles`; create the layout (`config.yml`, `profiles/`, `modules/`).
      2. **Overlay `config.yml`** — set identity (`user.*`), opt into a provider if wanted
         (`secrets.provider`), tweak a module setting (e.g. `modules.git.default_branch`).
         Show the deep-merge (only changed keys needed).
      3. **A content profile** (`profiles/mine.yml`) selecting built-in + your own modules
         by name; set `profile: mine`. Note content-wins over a same-name built-in profile.
      4. **A CUSTOM module** — new `modules/hello/` (`module.yml` + `install.sh`) using
         `lib/helpers.sh` (`log_*`, `is_dry_run`, `pkg_install`) and the `DOTFILES_*` env;
         optionally a `files/` template showing `{{ .User.name }}` / `{{ .Module.<key> }}`
         / `index .Env "DOTFILES_PROMPT_*"` (correct per §2.4). Appears as tag **custom**.
      5. **An OVERRIDE module** — `modules/git/` with `name: git` to replace the built-in
         wholesale (precedence keyed on `name`, not dir); explain when to override vs
         customize, and that it's whole-module replacement (reproduce what you still want).
         Appears as tag **override**.
      6. **Point the engine at it**: `export DOTFILES_CONTENT_DIR=$PWD/my-dotfiles`, then
         `dotfiles list` (see custom/override tags) and `dotfiles install --profile mine
         --dry-run`. Show expected output.
      7. **Clean-machine bootstrap** from the repo:
         `curl … bootstrap.sh | bash -s -- --content-repo https://github.com/you/my-dotfiles.git`
         (public/secret-free recommended; private via agent/`--content-auth-cmd`). Link
         `docs/content-overlay.md#private-content-repos`.
      8. **Verify & troubleshoot**: the override-name gotcha (`name` must match), overlay
         not applied (`DOTFILES_CONTENT_DIR` unset), `dotfiles status` tags.
- [ ] Reuse the shipped example `docs/examples/content-repo/` as the tutorial's finished
      artifact — reference/link it (via a GitHub tree URL for docs-site link-safety, §5)
      rather than re-pasting all files. Keep the two in sync.
- [ ] Cross-link the new page from `docs/README.md` (index), root `README.md`,
      `docs/content-overlay.md`, and `docs/creating-modules.md`.

### H. ssh `key_item` default change (NOT docs-only — deliberate back-compat break) ✅ DONE (branch `fix/ssh-key-item-default`)
Per the guiding principle + §6-D1. Ship as its own PR (`fix/ssh-key-item-default`).
- [x] Change the code default in `modules/ssh/install.sh:27`
      (`${DOTFILES_SETTING_KEY_ITEM:-op://Personal/SSH Key}`) to the new generic
      (`op://Private/SSH Key`). Comments at `modules/ssh/install.sh:14-15,22-27` updated too.
- [x] Update `config.yml:29-30` comment and `config.overlay.example.yml:54` example to the
      new default; `test/bootstrap/ssh_key_item_test.sh` (the "unset → default"
      cases) now assert `op://Private/SSH Key/…`.
- [x] CHANGELOG `[Unreleased]` **Changed** entry explicitly flagging the changed default
      as a **breaking default** (only affects users of `key_source: 1password` who relied
      on the old default vault name; fix: set `modules.ssh.key_item`).
- [x] `go test ./...` + the hermetic ssh test green; `go vet` clean. (shellcheck/docker not
      available locally — CI's lint-shell covers it; the edited scripts are minimal comment/
      literal changes.)

---

## 5. docs-site mechanics (respect these or the build/guard breaks)

The site is Astro + Starlight, generated by "doc-site"; scaffold files are hash-tracked in
`.doc-site-scaffold.json` (hand-edits diverge the hash — acceptable, but be aware the
generator would skip regenerating a hand-edited file).

**Adding / renaming / removing a symlinked page requires editing FOUR places in sync,
then regenerating + checking:**
1. `docs-site/docs.manifest.json` `pages[]` (source of truth; unique slugs; order matters).
2. `docs-site/setup-docs.sh` — the matching `link_file "docs/<f>.md" "<slug>"` line.
3. `docs-site/astro.config.mjs` `sidebar[]` — matching `slug:`, **in manifest order**.
4. `docs-site/.gitignore` — `src/content/docs/<slug>.md` (the generated symlink is ignored).
Then run `sh docs-site/setup-docs.sh && node docs-site/check-docs.mjs`.

**Drift-guard rules (`docs-site/check-docs.mjs`) — what FAILS:**
- Any page (symlinked or native) missing a `title:` in frontmatter.
- A broken internal markdown link (resolved against the content dir / file dir with
  `.md`/`.mdx`/`/index` fallbacks). External (`http:`/`mailto:`/`#`/…) links are exempt.
- **Native pages** (`index.mdx`, `guides/setup.mdx`) must use **root-absolute `/slug/`**
  internal links — a relative `.md` or bare `foo/bar/` link fails. (Symlinked `docs/*.md`
  are exempt from this rule and may keep relative `./x.md` links for GitHub rendering.)
- Sidebar↔manifest mismatch (missing either side, or different order).
- Orphaned/dangling symlink, or a manifest symlink page with no link on disk (re-run
  `setup-docs.sh`). Duplicate slug / invalid manifest JSON = hard error (exit 2).

**Build/CI reality:**
- `.github/workflows/docs.yml` runs **only on push to `main`** (paths `docs/**`,
  `docs-site/**`), **not on PRs**, Node **22**. So there is no PR-time safety net for the
  site — validate locally before merging.
- The dev machine here is **Node 18**; Astro 7 needs **≥22**, so `npm run build`/`dev`
  can't run locally. `node docs-site/check-docs.mjs` is stdlib-only and DOES run on 18 —
  run it directly (symlinks are already materialized, or run `sh setup-docs.sh` first).
  If you can, use a Node-22 toolchain to run a full `npm ci && npm run build` before
  merging the `docs/scrub-site` group, since main will build it unguarded.

---

## 6. Decisions (resolved 2026-08-21)

- **D1 — `op://Personal/SSH Key` default → CHANGE IT (decided).** Per the guiding
  principle, replace the personal-flavored default with a neutral generic
  (recommended `op://Private/SSH Key` — 1Password's current default personal-vault name).
  This is a deliberate back-compat break, executed in §4.H (code + comments + docs + test +
  CHANGELOG). Confirm the exact new value during execution.
- **D2 — `DOCUMENTATION_UPDATES.md` & `UX_ENHANCEMENT_SUMMARY.md` → DELETE (decided).**
  Content already lives in `CHANGELOG.md`; remove both (not archived). See §4.E.
- **D3 — Author block** (`README.md:495-498`, real name/email). *Keep* — legitimate
  attribution on the maintainer's own repo. (Revisit only if the repo is ever handed off.)
- **D4 — Repo-URL placeholder** (`USER` vs `garygentry`). Standardize on **`garygentry`**
  (the real engine repo); reserve a clearly-marked `<your-fork>` only where a doc is
  explicitly about forking. Applied in §4.F.

---

## 7. Definition of done

- No doc presents `provider: 1password` (or `""`) as the committed/default provider; the
  overlay is the taught personalization path across the operational docs.
- Template-context descriptions match `internal/template/render.go` exactly (lowercase
  `.User` keys, `.Module` = settings, `.Secrets` empty, `.XDGConfigHome` present).
- CLI docs match the real surface: no `dotfiles version`, no `dotfiles rollback`;
  `--content-*` shown only as bootstrap flags; `render-template --module` documented;
  `list`/`status` tag column noted.
- No stale dependency chains, developer-profile listings, or Go-version references; repo
  URLs consistent.
- Architecture/creating-modules cover the overlay (config layering, content-wins
  discovery, unified render context).
- docs-site: the two native pages describe the real product; placeholder description
  replaced; the new **Extending** page is registered and renders; `node
  docs-site/check-docs.mjs` reports **OK — no drift**; a Node-22 `npm run build` succeeds
  (or is confirmed green post-merge on the docs workflow).
- **Extensibility page (§4.G) exists**: a new user can follow the tutorial end-to-end to
  build a `my-dotfiles` repo with a custom module and an override, ending in a working
  `--dry-run`; the shipped `docs/examples/content-repo/` is referenced as the finished result.
- **ssh `key_item` default changed (§4.H)**: the new generic default ships in code +
  comments + `config.yml`/example, the hermetic ssh test asserts the new value, and the
  CHANGELOG flags the deliberate breaking-default change. No doc shows `op://Personal/SSH Key`
  as the default any more.
- Top-level artifacts resolved: the two stale summaries deleted; ROADMAP genericized and
  status-checked.
- Each PR: CHANGELOG `[Unreleased]` updated; this plan's checkboxes/Status current;
  `go test -race ./...` + `go vet` still green (docs PRs are trivially green; the §4.H PR
  must pass the ssh test).

## 8. Verification commands
```bash
# repo-wide staleness sweeps (expect no hits after the scrub)
rg -n "provider: *1password|provider: *\"\"" README.md docs/
rg -n "dotfiles rollback|dotfiles version" docs/
rg -n "\.User\.Name" docs/
rg -n "1\.22|golang:1\.22" docs/
rg -n "op://Personal/SSH Key" .        # §4.H: none after the default change (except CHANGELOG note)
ls DOCUMENTATION_UPDATES.md UX_ENHANCEMENT_SUMMARY.md 2>/dev/null   # §4.E: expect "No such file"
# docs-site guard (runs on Node 18; materialize symlinks first)
sh docs-site/setup-docs.sh && node docs-site/check-docs.mjs
# full site build (needs Node >=22)
cd docs-site && npm ci && SITE=https://example.com BASE_PATH=/dotfiles/ npm run build
```

## 9. Workflow / conventions
Same as the rest of the repo — see `plans/README.md` §"Workflow": branch from `main`
(`docs/scrub-*`), SSH-signed commits via 1Password (unlock the app if signing fails),
open a PR, get CI green (unit / lint-shell / integration — all trivially pass for
docs-only), squash-merge, delete the branch, CHANGELOG under `[Unreleased]`. Remember the
**docs-site workflow runs only on push to `main`**, so validate the site locally (§5)
before merging the `docs/scrub-site` group.
