# Plan: Comprehensive Documentation Scrub

**Status:** 📋 PLANNED — not started · **Branch:** `docs/scrub-*` per work group
**Scope:** All human-facing docs (repo markdown + the Astro/Starlight docs site) brought
into line with the merged content-overlay initiative (phases 1–4) and current engine
behavior. **Docs-only — no engine/behavior changes.** Multi-session; update the
checkboxes and Status line as you go and commit them with the work.

> **Execute this in a clean session.** It is self-contained: §2 is the authoritative
> "current truth" to align docs to; §4 is the itemized work with file:line anchors;
> §5 is the docs-site mechanics you must respect; §7 is the definition of done. Read
> `plans/README.md` first for the initiative context and the shared Workflow section.

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
- `modules.ssh.key_item` default `op://Personal/SSH Key` (used only when
  `key_source: 1password`; key type appended as the field). **Leave the code default as
  is** — changing it is a behavior change, out of scope for a docs scrub (see §6 D1).
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
3. **`docs/scrub-site`** — §4.D (rewrite the two native docs-site pages + site metadata).
   Respect §5. This is the only group touching `docs-site/`.
4. **`docs/scrub-artifacts`** — §4.E (top-level artifact cleanup, ROADMAP genericize,
   CONTRIBUTING check). Resolve the §6 decisions first.

Do §6 decisions up front (some gate the work). Update this plan's checkboxes + Status and
add a CHANGELOG `[Unreleased]` entry per PR.

---

## 4. Work items

### A. Central narrative & shipped defaults (across operational docs)
The recurring fix: (a) stop showing `provider: 1password` as the committed default; (b)
teach the **overlay** as the personalization path; (c) show/describe `ssh.key_source`.

- [ ] `README.md:172-174` config example → `provider: noop` (or drop the `secrets:` block
      and point to the overlay). `:18` & `:236` soften "Integrated/Seamless 1Password" to
      "optional 1Password (opt-in) secrets".
- [ ] `README.md:155-159` Modules table: fix ssh "Dependencies: 1password" → none;
      zsh "Zinit" → "Zinit or Oh My Zsh"; note the table is a sample of ~30 modules.
- [ ] `README.md:181-190` config example: add `key_source` under `ssh:` so readers learn
      generation is selectable.
- [ ] `README.md:163-186` "Configuration" section: reframe personalization to the overlay
      (link `docs/content-overlay.md` / `config.overlay.example.yml`) instead of editing
      `~/.dotfiles/config.yml` in place.
- [ ] `README.md:194-203` Developer Profile block → real `profiles/developer.yml` contents.
- [ ] `docs/quick-start.md:208-210` → `provider: noop`; `:30-33` sample `list` output →
      current modules/phrasing; `:167` "all modules" → curated set; add an overlay pointer
      in the Configuration section (198-222).
- [ ] `docs/installation.md`: add a short **content-overlay bootstrap** subsection
      (`--content-repo` / `--content-path` / `DOTFILES_CONTENT_DIR`) linking
      `docs/content-overlay.md`; fix the `USER` vs `garygentry` placeholder (see §4.F);
      align Go version (§2.5).
- [ ] `docs/cli-reference.md:568-570` → `provider: noop`; `:207-215` sample `list` →
      current modules + note the built-in/override/custom **tag column**; document
      `render-template --module`; add a pointer that `--content-*` are bootstrap flags.
- [ ] `docs/troubleshooting.md`: SSH section (130-157, 335, 574) → describe `key_source`
      (generate is the default, not a fallback); "Skip Secrets" (333-341) → note default is
      already `noop`; add an overlay-not-applied / override-name gotcha entry.
- [ ] `docs/ci-cd-guide.md`: replace `provider: ""` (53-54, 348-349, 653-655) with `noop`;
      mark nonexistent profiles (`server`/`ci`/`prod`/…) as "create your own"; add a
      content-overlay-in-CI note (`--content-repo` to materialize config in a container);
      align Go version (`:147`).

### B. Architecture / rationale / creating-modules (structural coverage)
- [ ] `docs/architecture.md`: fix the `Context` struct (201-212) per §2.4; replace the
      stale `1password→ssh→git` dependency example (362-378); note `noop` is the shipped
      default (187-193); **add** coverage of: config **overlay** layering, content-wins
      module **discovery** with built-in/override/custom tags (100-108), and the unified
      in-process vs `render-template` context (4C).
- [ ] `docs/design-rationale.md`: add the **engine vs content-overlay** separation as a
      rationale point; `:31` reword "SSH key generation" to reflect `key_source` gating.
- [ ] `docs/creating-modules.md`: fix the template-context section (463-474) and the
      gitconfig example (481-483) per §2.4 (lowercase `.User` keys, `.Module` = settings
      not prompts, `.Secrets` empty, add `.XDGConfigHome`); fix the priority/deps example
      (580-584) stale 1password chain; note default provider is `noop` near the
      `get_secret` example (373-378). (The overlay section 750-777 is correct — leave it.)
- [ ] `docs/idempotence.md:431`: `dotfiles rollback zsh` → `dotfiles uninstall zsh`; add
      one line that overlay-merged config participates in the config-hash change detection.
- [ ] `docs/ux-features.md`: add the `list`/`status` **built-in/override/custom** tag to
      the described UX (brief).
- [ ] `docs/README.md` (index): add the missing **CI/CD Guide** link under Operations;
      confirm **Content Overlay** link present (it is).

### C. Template-context correctness (surgical)
Fix the SAME context description everywhere it appears, using the §2.4 block verbatim as
the model. Files: `docs/architecture.md:201-212`, `docs/creating-modules.md:463-483`,
`docs/cli-reference.md:476-491`, and `docs/troubleshooting.md:499-503` (its "fix" is
itself wrong — the correct access is lowercase `.User.name`). (Overlaps §4.B by design;
if you ship the template-context PR separately, do these four together.)

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
- [ ] (Optional) Consider a docs-site nav grouping so Overview/Installation/Quick
      Start/**Content Overlay** read as a "Getting started" cluster. If you touch the
      sidebar, keep manifest↔sidebar parity + order (§5).

### E. Top-level artifacts, ROADMAP, CONTRIBUTING (resolve §6 first)
- [ ] `DOCUMENTATION_UPDATES.md` (376 lines) and `UX_ENHANCEMENT_SUMMARY.md` (288 lines):
      one-off v2.1.0 summaries whose content is already in `CHANGELOG.md`. Per §6-D2:
      delete (or move under a `docs/history/` archive). Remove any inbound links.
- [ ] `ROADMAP.md`: `:308` genericize `dotfiles add garygentry/my-module` →
      `you/my-module`. Re-verify statuses against what shipped (uninstall/rollback-history,
      schema validation partially advanced — see `plans/README.md` "Overlap" note); mark
      accordingly. Optionally add a one-line pointer to the content-overlay docs.
- [ ] `CONTRIBUTING.md`: quick pass for stale build/test commands and to mention the
      overlay/plans workflow; no known hard errors — verify.
- [ ] `config.yml` / `config.overlay.example.yml` / `modules/ssh/install.sh` comments: per
      §6-D1 decision on the `op://Personal/SSH Key` default wording.

### F. Cross-file consistency sweeps (grep-driven; do once, repo-wide)
- [ ] **Provider default**: no doc shows `provider: 1password` or `provider: ""` as the
      committed/default value. `rg -n "provider: *1password|provider: *\"\""` → all fixed
      to `noop` or clearly opt-in.
- [ ] **`rollback` command**: `rg -n "dotfiles rollback"` → none (use `uninstall`).
- [ ] **`version` command**: `rg -n "dotfiles version"` → none; remove the
      `cli-reference.md:493-503` section (no such command).
- [ ] **ssh dependency chain**: `rg -n "1password.*ssh|ssh.*1password"` in docs → no
      dependency claims; ssh deps are `[]`.
- [ ] **developer profile**: every listing matches `profiles/developer.yml`.
- [ ] **Go version**: `rg -n "1\.22|golang:1\.22"` → standardize on ≥1.23.
- [ ] **repo URL placeholder**: pick ONE — replace `USER` in
      `installation.md`/`quick-start.md` raw URLs with `garygentry` (real repo) OR make
      everything a clearly-marked `<your-fork>`; be consistent.
- [ ] **template context**: `rg -n "\.User\.Name|\.Secrets\.|Module settings + prompts"` →
      reconciled with §2.4.

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

## 6. Decisions to make first (confirm with the maintainer)

- **D1 — `op://Personal/SSH Key` default.** It's the committed fallback for
  `key_source: 1password` (`modules/ssh/install.sh:27`, mirrored in `config.yml:31`
  comment + `config.overlay.example.yml:54`). *Recommendation:* **do not change the code
  default** (backward-compat for existing 1Password users; only matters when
  `key_source: 1password`). Optionally reword the doc/comment to call it a
  personal-flavored **placeholder** users should override. Decide: reword-only vs
  leave-as-is vs (out-of-scope) change the default.
- **D2 — Fate of `DOCUMENTATION_UPDATES.md` & `UX_ENHANCEMENT_SUMMARY.md`.**
  *Recommendation:* delete both (content lives in `CHANGELOG.md`), or move to
  `docs/history/`. Decide delete vs archive.
- **D3 — Author block** (`README.md:495-498`, real name/email). It's legitimate
  attribution on the maintainer's own repo. *Recommendation:* keep; genericize only if the
  maintainer wants a public-facing generic engine.
- **D4 — Repo-URL placeholder** (`USER` vs `garygentry`). Pick one convention (§4.F).

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
  replaced; `node docs-site/check-docs.mjs` reports **OK — no drift**; a Node-22
  `npm run build` succeeds (or is confirmed green post-merge on the docs workflow).
- Top-level artifacts resolved per §6; ROADMAP genericized and status-checked.
- Each PR: CHANGELOG `[Unreleased]` updated; this plan's checkboxes/Status current;
  `go test ./...` + `go vet` still green (docs-only, but CI runs on every PR).

## 8. Verification commands
```bash
# repo-wide staleness sweeps (expect no hits after the scrub)
rg -n "provider: *1password|provider: *\"\"" README.md docs/
rg -n "dotfiles rollback|dotfiles version" docs/
rg -n "\.User\.Name" docs/
rg -n "1\.22|golang:1\.22" docs/
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
