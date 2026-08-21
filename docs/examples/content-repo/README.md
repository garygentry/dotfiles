# Example content repo (`my-dotfiles`)

A minimal, copy-me template for a personal **content overlay** — the second repo
in the [two-repo model](../../content-overlay.md). It overlays the generic
dotfiles engine so your personalization lives outside the engine, no fork
required.

```
content-repo/
  config.yml            # overlays the engine's config.yml (deep-merged)
  profiles/
    mine.yml            # a content profile (selected by config.yml)
  modules/
    hello/              # a CUSTOM module (new name the engine doesn't ship)
      module.yml
      install.sh
    git/                # an OVERRIDE of the built-in git module (same name)
      module.yml
      install.sh
```

## Use it

Copy this directory to a repo of your own and point the engine at it:

```bash
# Try it locally against a checkout of the engine
DOTFILES_CONTENT_DIR="$(pwd)/docs/examples/content-repo" ./bin/dotfiles install --dry-run

# Or, on a clean machine, publish it as (ideally public + secret-free) my-dotfiles
# and bootstrap straight from it:
curl -sfL https://raw.githubusercontent.com/garygentry/dotfiles/main/bootstrap.sh \
  | bash -s -- --content-repo https://github.com/you/my-dotfiles.git
```

## What each piece shows

- **`config.yml`** — deep-merged over the engine's; only the keys you change.
  Picks `profile: mine`.
- **`profiles/mine.yml`** — the module set for this machine: built-in modules
  (`ssh`, `git`, `zsh`) plus the custom `hello` module, by name.
- **`modules/hello/`** — a brand-new module the engine doesn't ship (a `custom`
  module).
- **`modules/git/`** — same `name` as the built-in `git`, so it **overrides** it
  wholesale (`override`, content-wins). `dotfiles list` / `status` tag each
  module `built-in`, `override`, or `custom`.

Keep **real secrets out of this repo** — fetch them from 1Password at
install time. See [`docs/content-overlay.md`](../../content-overlay.md).
