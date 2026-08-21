# Architecture Initiative — Roadmap & Active Plans

This is the index for an ongoing **architecture initiative** on the dotfiles engine.
It is separate from the product-feature backlog in the repo-root `ROADMAP.md`
(uninstall, schema validation, etc.); some of those items overlap with work done
here and are noted below.

**New to this repo / a fresh session? Start here**, then open the specific plan
file for whatever is `PLANNED`.

---

## The two arcs

1. **Re-run safety** — running `dotfiles install` again on an installed machine must
   never break or regress it. (Done.)
2. **Depersonalize + external content** — the generic repo ships conservative,
   unopinionated defaults; each user keeps their personalization (config, profiles,
   modules) in an optional **content overlay** directory (`$DOTFILES_CONTENT_DIR`)
   laid out just like the repo. Delivered incrementally in 4 phases, each strictly
   backward-compatible (no content dir set → behavior identical to today).

The **content overlay** is the spine of arc 2: a user content dir with the same
layout as the engine (`config.yml`, `profiles/`, `modules/`) that overlays it. The
engine only ever **consumes a local content directory**; how that directory is
**acquired** (local, git repo, network path) is a separate, later concern.

---

## Status

| Item | Status | Where |
|---|---|---|
| **Re-run / rollback safety** | ✅ Done | PR #4 (in v2.2.0) |
| **Content overlay — phase 1: config** | ✅ Done | v2.2.0 |
| **Content overlay — phase 2: profiles** | ✅ Done | PR #6 |
| **Content overlay — phase 3: modules** | 📋 Planned | `plans/phase3-content-modules.md` |
| **Content overlay — phase 4: acquisition/packaging + cleanups** | 📋 Planned | `plans/phase4-acquisition-packaging.md` |

### Phase summary (arc 2)
- **P1 config** — `<content>/config.yml` deep-merged over the repo's; conservative
  committed defaults (`provider: noop`, `ssh.key_source: generate`).
- **P2 profiles** — bare profile names resolve from `<content>/profiles/` first
  (content-wins); overlay announced; typo'd content dir warned.
- **P3 modules** — discover `<content>/modules/` with same-name **content-wins**
  precedence (custom + override modules). *Next up.*
- **P4 acquisition/packaging** — fetch a content repo/path (public/private/network)
  into the content dir + auth ordering; the two-repo "my-dotfiles" story; plus the
  deferred cleanups (secret-ref parameterization, template-context unification,
  dead-config).

## Releases
- **v2.2.0** (2026-08-21) — fresh-WSL install fixes, ssh `key_source`, re-run/rollback
  safety, and content overlay phase 1. Baseline before the phase 3+ module work.

## Overlap with the product `ROADMAP.md`
- Its **1.2 Module Schema Validation** — partly advanced (ssh `verify.sh` now truly
  fails; content-dir structure is validated on load). Full unknown-key linting still
  open there.
- Its **1.1 Module Uninstall** — this initiative fixed rollback-history integrity
  (PR #4), which uninstall depends on, but the uninstall UX itself remains a product
  item.

---

## Workflow (how everything here ships)
- Branch from `main` (`track3/…`, `track4/…`). Build: `go build -o bin/dotfiles .`
  (Go at `~/.local/go/bin`; `make` may be absent on some machines).
- Commits are **SSH-signed via 1Password** (`op-ssh-sign-wsl.exe`); the Windows
  1Password app must be unlocked or signing fails with "failed to fill whole buffer"
  — unlock and retry. Local `git log` shows `N` (no `allowedSignersFile`); GitHub
  still verifies. Use the repo's standard `Co-Authored-By` / `Claude-Session`
  commit trailers.
- Open a PR, wait for CI (unit, lint-shell, integration ubuntu+arch), **squash**
  merge, delete the branch. CHANGELOG entries under `[Unreleased]`.
- **Every change stays backward-compatible**: with `$DOTFILES_CONTENT_DIR` unset,
  behavior is exactly as before.
- Keep the relevant `plans/*.md` checkboxes + Status line current as you work.

## Files
- `plans/phase3-content-modules.md` — next up.
- `plans/phase4-acquisition-packaging.md` — after phase 3.
- `plans/README.md` — this index.
- Root `ROADMAP.md` — separate product-feature backlog.
