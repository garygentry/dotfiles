# Plan: Content Overlay — Phase 3 (Modules)

**Status:** DONE (T1–T6) — PR open, awaiting CI · **Branch:** `track3/content-modules` (from `main`)
**Tracks:** Content-overlay model, phase 3 of 4. Multi-session plan — update the
checkboxes and the Status line as you go, and commit the updated plan with the work.

---

## 1. Context (read first — this is self-contained)

This repo is a Go + shell **dotfiles engine**. It is being taught an optional
**content overlay**: a user directory (`$DOTFILES_CONTENT_DIR`) laid out exactly
like the repo (`config.yml`, `profiles/`, `modules/`) whose contents overlay the
generic repo, so personal customizations live outside it.

Delivered so far:
- **Phase 1 (v2.2.0):** `<content>/config.yml` is deep-merged over the repo's
  `config.yml`. Resolution + merge in `internal/config/config.go`
  (`ResolveContentDir`, `loadOverlay`, `mergeConfig`; `Config.ContentDir`).
- **Phase 2 (PR #6):** bare profile names resolve from `<content>/profiles/`
  before the engine's, content-wins. See `ResolveProfilePath(dotfilesDir,
  contentDir, name)` and `LoadProfile(...)` in `internal/config/config.go`.

**This phase (3):** discover **modules** from `<content>/modules/` in addition to
the engine's `modules/`, with **same-name content-wins precedence** — so a user
can add custom modules AND override a built-in module by dropping a same-named
module dir in their content dir. This is what makes a content profile able to
reference the user's own modules.

### Guiding principles (non-negotiable)
- **Backward compatible, always.** With no `$DOTFILES_CONTENT_DIR`, behavior must
  be byte-for-byte identical to today (single engine module root).
- **Simplest, most robust.** No cleverness. Whole-module **replacement** by name
  (NOT patching/merging a module). Deterministic precedence.
- **Additive.** Nothing existing should change behavior when the overlay is unused.

### Why this is contained (already-portable pieces — do NOT re-solve)
- `Module.Dir` is location-agnostic (`internal/module/schema.go`): a module dir
  can live anywhere.
- `ComputeModuleChecksum`, file deployment, and `lib/helpers.sh` sourcing all key
  off `mod.Dir` / `DOTFILES_DIR` and already work for an externally-located module.
- So the ONLY coupling to fix is **discovery** (single root) and **resolution**
  (flat name namespace). Everything downstream already works.

### The coupling points to change (verified file:line, may drift — re-grep)
- `internal/module/discovery.go:14` — `Discover(modulesDir string)` scans exactly
  one directory, non-recursively.
- Four hardwired call sites build `filepath.Join(sys.DotfilesDir, "modules")`:
  - `cmd/dotfiles/install.go` (~line 116)
  - `cmd/dotfiles/list.go` (~line 26)
  - `cmd/dotfiles/status.go` (~line 39)
  - `cmd/dotfiles/validate.go` (~line 41)
  - (`cmd/dotfiles/new.go` scaffolds into `<DotfilesDir>/modules` — leave targeting
    the engine for now; note as a future option, not this phase.)
- `internal/module/resolver.go:38-40` — `moduleMap` keyed by `m.Name`; a second
  same-name module silently overwrites (last-writer-wins). Phase 3 makes this a
  **deliberate** content-over-engine precedence with visibility.

---

## 2. Design

### Discovery
Add multi-root discovery. Preferred shape:

```go
// DiscoverRoots discovers modules across an ordered list of roots. Later roots
// override earlier ones by module name (so pass [engineModules, contentModules]).
// Missing roots are skipped. Returns modules plus a record of which name came
// from which root (for override visibility).
func DiscoverRoots(roots []string) ([]*Module, error)
```

- Keep the existing `Discover(dir)` (single root) working — implement it in terms
  of `DiscoverRoots` or keep both. Existing tests must keep passing.
- Precedence: **content wins**. Build the map engine-first, then let content
  entries replace by name. Record overrides so `list`/`status` can show them.

### Root resolution helper
One place that computes the ordered roots, used by every command:

```go
// ModuleRoots returns the ordered module roots: the engine's built-in modules
// first, then the content dir's modules/ if a content dir is set and it exists.
func ModuleRoots(dotfilesDir, contentDir string) []string
```

### Resolver
`Resolve`/`expandDependencies` already key by name; once `DiscoverRoots` has
applied precedence, the resolver sees ONE module per name — no resolver change
needed for correctness. Add a guard/log if you detect a name collision that is
NOT an intentional content override (defensive; optional).

### Validation ("yields proper structure")
- A malformed `module.yml` in the content dir must produce a clear error naming
  the file (existing `validateModule` already errors; make sure `DiscoverRoots`
  surfaces it with the path, not a silent skip — mirror `discovery.go` behavior).
- If `<content>/modules/` is absent, that's fine (no custom modules).

### Override visibility (nice-to-have, keep small)
`dotfiles list` / `status` should mark each module's source:
`built-in` | `override` (content shadows a built-in) | `custom` (content-only).
Small column or tag. If it balloons, split to its own task/PR.

---

## 3. Task breakdown (each is a small, self-contained commit)

- [x] **T1 — Multi-root discovery + tests.** Add `DiscoverRoots(roots)` and
      `ModuleRoots(dotfilesDir, contentDir)` in `internal/module`. Content-wins
      precedence; missing roots skipped; malformed module.yml errors with path.
      Unit tests: two roots, override by name, content-only module, missing
      content root = engine only, malformed content module errors. Keep
      `Discover` behavior intact. *Acceptance:* `go test ./internal/module/`.
- [x] **T2 — Wire `install`.** `cmd/dotfiles/install.go` discovers via
      `ModuleRoots(sys.DotfilesDir, cfg.ContentDir)`. A content profile can now
      reference a content module end to end. *Acceptance:* dry-run install of a
      content profile that lists a content-only module shows it in the plan.
- [x] **T3 — Wire `list`, `status`, `validate`.** Same `ModuleRoots` source in
      `cmd/dotfiles/list.go`, `status.go`, `validate.go`. *Acceptance:* `list`
      shows content + override modules; `validate` checks them.
- [x] **T4 — Override visibility.** `list`/`status` tag built-in / override /
      custom. *Acceptance:* a content `git` override shows as "override"; a
      content-only module shows as "custom". (Split out if large.)
- [x] **T5 — Docs + example + CHANGELOG.** Add a `modules/` note to
      `config.overlay.example.yml` (or a short `docs/` note) showing a content
      module and an override. CHANGELOG under `[Unreleased]` → "Content overlay
      (phase 3: modules)". *Acceptance:* docs build / render fine.
- [x] **T6 — End-to-end (real install).** `scripts/testuser.sh` needs root
      (`useradd`), which the dev session lacked (no passwordless sudo), so the
      same two scenarios were proven with a hermetic *real, non-dry-run* install
      against a synthetic engine+content dir (own `HOME`/`DOTFILES_DIR`, modules
      that only write markers — no packages/sudo/network): (a) content `git`
      override + custom `mymod` both executed (markers `CONTENT-GIT`/
      `CONTENT-MYMOD`), the override dropped git's `ssh` dep, and `list` tagged
      them `override`/`custom`; (b) with no content dir, engine `git` ran
      (`ENGINE-GIT`), no Source column, `mymod` unknown — identical to before.
      Output recorded in the PR. CI's Docker integration tests exercise the full
      testuser-style flow. Optional manual harness run left for a root shell.

Tasks may be one PR or a few small ones — reviewer's discretion. Keep each commit
green.

---

## 4. Definition of done
- A `$DOTFILES_CONTENT_DIR` with `modules/mymod/` (custom) and `modules/git/`
  (override) → install uses the content `git` over the engine `git`, and installs
  `mymod`; a content profile can list both.
- With `$DOTFILES_CONTENT_DIR` unset, everything is exactly as before.
- `go test -race ./...` green; `go vet` clean; integration tests (Ubuntu + Arch)
  green.
- CHANGELOG updated; plan checkboxes updated; PR opened, signed, CI green.

## 5. Risks / watch-outs
- **Silent override surprise.** Make overrides visible (T4) so a user isn't
  confused about why a built-in behaves differently.
- **Two content roots later.** Out of scope now (one content dir). Keep
  `DiscoverRoots` ordered-list based so multiple roots are a future config change,
  not a redesign.
- **`new.go` scaffolding** still targets the engine `modules/`. Fine for now;
  a `--content` target is a future nicety, not this phase.
- **Don't touch acquisition** (cloning a content repo, auth) — that's phase 4.

---

## 6. Workflow / conventions (how this repo ships)
- Branch from `main`: `track3/content-modules`. Build binary with
  `go build -o bin/dotfiles .` (Go lives at `~/.local/go/bin`; `make` may be
  absent on some machines).
- Commits are **SSH-signed via 1Password** (`op-ssh-sign-wsl.exe`); the Windows
  1Password app must be unlocked or signing fails with "failed to fill whole
  buffer" — retry after unlocking. Local `git log` shows `N` for signatures
  (no `allowedSignersFile`); GitHub still verifies them.
- End commit messages with the repo's standard `Co-Authored-By` / `Claude-Session`
  trailers.
- Open a PR, wait for CI (unit, lint-shell, integration ubuntu+arch), **squash**
  merge, delete the branch. CHANGELOG entries go under `[Unreleased]`.
- Update THIS file's checkboxes + Status line as tasks complete, and include it in
  the PR.

## 7. How to resume in any session
1. `git checkout main && git pull`, then `git checkout track3/content-modules`
   (or create it from `main` if starting).
2. Read this file; find the first unchecked task in §3.
3. Do it, test, commit (updating the checkbox), push.
4. When all tasks are checked and §4 is met, open/finish the PR.
