---
title: "Extending"
---

# Extending: make this setup your own

This is the hands-on companion to the conceptual [Content Overlay](content-overlay.md)
guide. It walks you through building a personal **content repo** (`my-dotfiles`) from
scratch — one that overlays the generic engine with your identity, your profile, a custom
module, and an override of a built-in — ending in a working `--dry-run`.

## What "extending" means here

The engine in this repo is intentionally **generic**: it ships conservative, personal-free
defaults and knows nothing about you. You extend it with an optional **content overlay** — a
directory (`$DOTFILES_CONTENT_DIR`) laid out just like the engine repo and **deep-merged**
over it. There are three extension points:

1. **Config overlay** — your `config.yml` overrides only the keys you set (identity, a
   provider opt-in, per-module settings).
2. **Content profiles** — your `profiles/*.yml` select which modules to install; a content
   profile wins over a same-name built-in.
3. **Content modules** — your `modules/*/` add brand-new modules (**custom**) or replace a
   built-in wholesale by matching its `name:` (**override**).

Secrets never get committed — they stay in a provider (e.g. 1Password) and reach scripts via
the `get_secret` helper. For the deep model see [Content Overlay](content-overlay.md); for
module-authoring reference see [Creating Modules](creating-modules.md). This page won't
duplicate those — it's the build-it walkthrough.

A finished version of everything below ships in the repo at
[`docs/examples/content-repo/`](https://github.com/garygentry/dotfiles/tree/main/docs/examples/content-repo)
— use it as the reference result.

## Tutorial: build a `my-dotfiles` repo

### 1. Create the layout

```bash
git init my-dotfiles
cd my-dotfiles
mkdir -p profiles modules
```

The overlay mirrors the engine's layout:

```
my-dotfiles/
├── config.yml        # deep-merged over the engine's config.yml
├── profiles/         # your profiles (content-wins over built-ins by name)
└── modules/          # your custom + override modules
```

### 2. Overlay `config.yml`

Set only the keys you want to change — everything else falls through to the engine's
defaults.

```yaml
# my-dotfiles/config.yml
profile: mine                 # use the content profile from profiles/mine.yml (step 3)

user:                         # identity, consumed by git/ssh etc.
  name: "Ada Lovelace"
  email: "ada@example.com"
  github_user: "ada"

modules:
  git:
    default_branch: main      # per-key merge: change one setting, keep the rest
  ssh:
    key_source: agent         # use an external SSH agent; no local key managed
```

> Keep real identity here only if the repo is **private**. A public content repo can leave
> `user.*` blank and rely on interactive prompts or `DOTFILES_USER_*` env vars. To opt into a
> secrets provider, add `secrets: { provider: 1password }` (the engine default is `noop`).

### 3. A content profile

A profile is the set of modules to install. It can mix built-in engine modules and your own
content modules by name.

```yaml
# my-dotfiles/profiles/mine.yml
modules:
  - ssh
  - git        # overridden by modules/git/ below (same-name, content-wins)
  - zsh
  - hello      # a custom content module (step 4)
```

Reference it from `config.yml` with `profile: mine`. Because a content profile of the same
name **wins** over a built-in one, naming this file `developer.yml` would *override* the
engine's developer profile instead of adding a new one — so pick a fresh name unless you mean
to replace.

### 4. A custom module

A custom module is a brand-new module the engine doesn't ship. Author it exactly like a
built-in: it inherits `set -euo pipefail`, the [`lib/helpers.sh`](creating-modules.md)
functions (`log_*`, `is_dry_run`, `pkg_install`, …), and the `DOTFILES_*` environment.

```yaml
# my-dotfiles/modules/hello/module.yml
name: hello
description: "Example custom module shipped from a content overlay"
version: "1.0.0"
priority: 50
dependencies: []
os: [macos, ubuntu, arch]
```

```bash
# my-dotfiles/modules/hello/install.sh
#!/usr/bin/env bash

if is_dry_run; then
    log_info "[dry-run] Would greet from the hello content module"
    return 0
fi

log_success "Hello from your content overlay, ${DOTFILES_USER_NAME:-friend}!"
```

If the module deploys a template file (`files/*.tmpl`), remember the real context fields
(see [Creating Modules](creating-modules.md)): `{{ .User.name }}` (lowercase keys),
`{{ .Module.<key> }}` for this module's `config.yml` settings, and
`{{ index .Env "DOTFILES_PROMPT_MY_KEY" }}` for prompt answers. In `dotfiles list`, `hello`
is tagged **custom**.

### 5. An override module

To replace a built-in module wholesale, add a module whose `name:` **exactly matches** the
built-in's. Precedence is keyed on `name` (which defaults to the directory name), not the
directory itself. An override is a **whole-module replacement**, not a merge — the built-in's
`module.yml` and `install.sh` do not run, so reproduce anything you still want.

```yaml
# my-dotfiles/modules/git/module.yml
name: git                     # MUST match the built-in to override it
description: "Custom git configuration (content override)"
version: "1.0.0"
priority: 30
dependencies: [ssh]
os: [macos, ubuntu, arch]
```

```bash
# my-dotfiles/modules/git/install.sh
#!/usr/bin/env bash

if is_dry_run; then
    log_info "[dry-run] Would apply custom git configuration (content override)"
    return 0
fi

[[ -n "${DOTFILES_USER_NAME:-}" ]]  && git config --global user.name  "$DOTFILES_USER_NAME"
[[ -n "${DOTFILES_USER_EMAIL:-}" ]] && git config --global user.email "$DOTFILES_USER_EMAIL"
git config --global init.defaultBranch "${DOTFILES_SETTING_DEFAULT_BRANCH:-main}"
git config --global pull.rebase true

log_success "Applied custom git configuration (content override)"
```

**Override vs. customize:** reach for an override only when you need to change the module's
*behavior*. If you just want to tweak a setting the module already exposes (like
`default_branch`), do that in `config.yml` (step 2) instead — no override needed. In
`dotfiles list`, `git` is tagged **override**.

### 6. Point the engine at your overlay

```bash
export DOTFILES_CONTENT_DIR="$PWD/my-dotfiles"

# The Source column now appears, tagging modules built-in / override / custom:
dotfiles list

# Preview the install without changing anything:
dotfiles install --profile mine --dry-run
```

Expected: `dotfiles list` shows a **Source** column with `git` as `override`, `hello` as
`custom`, and everything else `built-in`; the dry-run resolves `ssh → git → zsh → hello` in
dependency order and prints each step's `[dry-run] Would …` line.

### 7. Bootstrap from the repo on a clean machine

Commit and push `my-dotfiles`, then on a fresh machine point the bootstrap script at it — it
clones the overlay and persists `DOTFILES_CONTENT_DIR` for you:

```bash
curl -sfL https://raw.githubusercontent.com/garygentry/dotfiles/main/bootstrap.sh | bash -s -- \
  --content-repo https://github.com/you/my-dotfiles.git
```

Keep the content repo **public and secret-free** when you can. For a private overlay, use an
ambient SSH/1Password agent or `--content-auth-cmd`; see
[Content Overlay → Private content repos](content-overlay.md#private-content-repos).

### 8. Verify & troubleshoot

- **Overlay not applied** (no Source column, identity missing) → `DOTFILES_CONTENT_DIR` isn't
  set/exported. `echo "$DOTFILES_CONTENT_DIR"` and re-export it.
- **Override ignored** (built-in still runs, or your module shows as `custom` not `override`)
  → the `name:` in `module.yml` doesn't match the built-in's. Fix `name:` (directory name is
  irrelevant to precedence).
- **Check the tags** any time with `dotfiles status`, which shows the same Source column.

## The finished result

The working example that mirrors this tutorial lives at
[`docs/examples/content-repo/`](https://github.com/garygentry/dotfiles/tree/main/docs/examples/content-repo)
(config, `profiles/mine.yml`, a `hello` custom module, and a `git` override). Copy it as a
starting point and adapt.

## See also

- [Content Overlay](content-overlay.md) — the two-repo model, first-install flow, and private-repo auth
- [Creating Modules](creating-modules.md) — the full module-authoring reference (schema, prompts, templates, helpers)
- [CLI Reference](cli-reference.md) — `list` / `status` / `install` and the bootstrap `--content-*` flags
