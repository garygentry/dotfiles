---
title: "Content Overlay & Packaging"
---

# Content Overlay: the two-repo model

This project ships as a **generic engine**: conservative, unopinionated defaults
that work for the widest audience out of the box. Your personalization —
identity, the modules you want, custom or overridden modules — lives in a
**separate content directory** that _overlays_ the engine. No fork required.

That's the two-repo model:

| Repo | What it is | Who owns it |
|---|---|---|
| **engine** (`garygentry/dotfiles`) | the Go+shell machinery and generic default modules | this project |
| **content** (`my-dotfiles`, yours) | your `config.yml`, `profiles/`, and `modules/` overlay | you |

The engine only ever **consumes a local content directory**, pointed to by the
`DOTFILES_CONTENT_DIR` environment variable. How that directory is **acquired**
(a git repo, a local path, a network share) is a thin, separate concern handled
by the bootstrap — the engine core never gains git/network/auth logic.

> **Backward compatible by construction.** With `DOTFILES_CONTENT_DIR` unset and
> no content flags, the engine behaves exactly as it does from its own committed
> `config.yml`. Everything below is additive and opt-in.

---

## What lives in the content directory

A content directory is laid out just like the engine:

```
~/.config/dotfiles/          # your content dir (DOTFILES_CONTENT_DIR)
  config.yml                 # deep-merged OVER the engine's config.yml
  profiles/
    mine.yml                 # your module sets (content-wins over same-name built-ins)
  modules/
    hello/                   # a CUSTOM module (a new name)
    git/                     # an OVERRIDE of a built-in (same name, wins wholesale)
```

- **`config.yml`** is **deep-merged** over the engine's: scalars override when
  set, and module settings merge per key, so you can change one
  `modules.<name>.<key>` without redefining the whole module. Only include the
  keys you want to change.
- **`profiles/*.yml`** resolve **content-first**: a bare profile name (`profile:
  mine`, or `--profile mine`) is looked up in your content `profiles/` before the
  engine's. A content profile with the same name as a built-in (e.g.
  `developer.yml`) **replaces** it.
- **`modules/*`** are discovered alongside the engine's with **same-name
  content-wins** precedence: a new name is added as a `custom` module; a name
  that matches a built-in **overrides** it (whole-module replacement, not a
  merge). `dotfiles list` and `dotfiles status` tag every module `built-in`,
  `override`, or `custom`.

See [Creating Modules → Custom & override modules](creating-modules.md#custom--override-modules-via-the-content-overlay)
for the module authoring details, and
[`config.overlay.example.yml`](https://github.com/garygentry/dotfiles/blob/main/config.overlay.example.yml)
for an annotated `config.yml`.

### Keep secrets out of the content repo

Real secrets never belong in the content repo. Modules fetch them from your
secret provider (e.g. 1Password) **at install time** via `get_secret`; the
content repo only carries *references* (like an `op://…` path), never the secret
values. This is what makes a **public, secret-free** content repo safe — and
recommended (see [private repos](#private-content-repos) for why).

---

## A minimal example content repo

A complete, copy-me template lives at
[`docs/examples/content-repo/`](https://github.com/garygentry/dotfiles/tree/main/docs/examples/content-repo). It contains a `config.yml`,
a `profiles/mine.yml`, a **custom** `hello` module, and an **override** of the
built-in `git` module. Try it against a checkout of the engine:

```bash
DOTFILES_CONTENT_DIR="$(pwd)/docs/examples/content-repo" \
  ./bin/dotfiles install --profile mine --dry-run --unattended
```

```
[INFO] Using content overlay: .../docs/examples/content-repo

[INFO] Execution Plan
----------------------------------------

  Install (4):
  1. ssh - Configure SSH keys and settings
  2. hello - Example custom module shipped from a content overlay
  3. git - Custom git configuration (content override)
  4. zsh - Install and configure Zsh with plugin manager (Zinit or Oh My Zsh)
```

Note `hello` (a `custom` module) and `git` running the overlay's **override**
description rather than the built-in one.

---

## Clean-machine first install

On a fresh machine, `bootstrap.sh` installs the engine's prerequisites (git,
Go), clones/builds the engine, **materializes your content directory**, exports
`DOTFILES_CONTENT_DIR`, and runs the install — in one command.

### From a public content repo (recommended)

```bash
curl -sfL https://raw.githubusercontent.com/garygentry/dotfiles/main/bootstrap.sh \
  | bash -s -- --content-repo https://github.com/you/my-dotfiles.git
```

Pin a ref and/or choose where the content lands (default `~/.config/dotfiles`):

```bash
curl -sfL .../bootstrap.sh | bash -s -- \
  --content-repo https://github.com/you/my-dotfiles.git \
  --content-ref v1 \
  --content-dir ~/.config/dotfiles
```

### From a local or network path

If the content already exists on the machine (a synced folder, a network share,
a checkout you made by hand), use it **in place** — no clone:

```bash
curl -sfL .../bootstrap.sh | bash -s -- --content-path ~/my-dotfiles
```

### What the bootstrap does with it

1. **Resolve → materialize**: clone (or idempotently **pull** on re-runs) a
   `--content-repo`, or use a `--content-path` directory in place.
2. **Point the engine at it**: export `DOTFILES_CONTENT_DIR` for this run.
3. **Persist for future shells**: append the `export` line to `~/.zshenv`
   (idempotently), and print the exact line. Opt out with
   `--no-persist-content-dir`.
4. **Install**: every non-content flag (`--profile`, `--unattended`,
   `--dry-run`, …) is forwarded verbatim to `dotfiles install`.

Re-running the same bootstrap is safe: a git source is pulled rather than
re-cloned, and `~/.zshenv` is not appended to twice. With **no** content flags
and no pre-set `DOTFILES_CONTENT_DIR`, the bootstrap behaves exactly as before.

The content flags (`--content-repo`, `--content-ref`, `--content-path`,
`--content-dir`, `--content-auth-cmd`, `--no-persist-content-dir`) each also have
a `DOTFILES_CONTENT_*` environment-variable equivalent; see the header of
[`bootstrap.sh`](https://github.com/garygentry/dotfiles/blob/main/bootstrap.sh).

---

## Private content repos

Because the content is cloned **before** the engine runs, any authentication
material must already be present when the clone happens. The simplest and
strongly preferred path is an **ambient agent** — a running SSH agent or the
1Password SSH agent — which needs no extra flags:

```bash
# With an SSH/1Password agent already available on the machine:
curl -sfL .../bootstrap.sh | bash -s -- \
  --content-repo git@github.com:you/my-dotfiles.git
```

If you must start or unlock a credential first, run a setup step **before** the
clone with `--content-auth-cmd`:

```bash
curl -sfL .../bootstrap.sh | bash -s -- \
  --content-repo git@github.com:you/my-dotfiles.git \
  --content-auth-cmd 'eval "$(ssh-agent -s)" && ssh-add ~/.ssh/id_ed25519'
```

### The chicken-and-egg (read this)

This only bites when the auth material itself would live **inside** the private
content repo — you can't use a secret to clone the repo that contains that
secret. The honest fix is to **not** put that material in the content repo:

- Prefer a **public, secret-free** content repo. It carries only *references*
  to secrets; the real values are fetched from 1Password at install time.
- Keep the bootstrap credential ambient (an agent) or supply it out-of-band via
  `--content-auth-cmd` — never by committing it to the content repo.

Treat the private-repo path as advanced. For most people, public + secret-free
is simpler *and* safer.

---

## Making it stick

After the first bootstrap, `DOTFILES_CONTENT_DIR` is exported in `~/.zshenv`, so
every future `dotfiles install` (and re-run of the bootstrap) automatically uses
your content. To set it up by hand instead:

```bash
echo 'export DOTFILES_CONTENT_DIR="$HOME/.config/dotfiles"' >> ~/.zshenv
```

---

## See also

- [`config.overlay.example.yml`](https://github.com/garygentry/dotfiles/blob/main/config.overlay.example.yml) — annotated overlay `config.yml`.
- [`docs/examples/content-repo/`](https://github.com/garygentry/dotfiles/tree/main/docs/examples/content-repo) — the copy-me example repo.
- [Creating Modules](creating-modules.md) — authoring custom/override modules.
- [Installation](installation.md) and [Quick Start](quick-start.md) — the no-overlay path.
