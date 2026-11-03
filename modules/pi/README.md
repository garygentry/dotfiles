# pi

Installs the `pi` coding agent globally through npm, at a pinned version.

## Version

Set `modules.pi.version` in your content overlay's `config.yml`. If it's unset, the module installs npm's `latest` and warns.

This module never runs `pi update`. A manual `pi update` is corrected back to the pin on the next install.

## Authentication

Run `pi` once interactively to authenticate. Auth and trust state are local to the machine and aren't managed here.

## When the install can't complete

If npm is missing or the global install fails (no registry, or offline), `install.sh` warns and doesn't stop (ADR 0025 D6). `verify.sh` runs next and is the gate: if `pi` is missing or off its pin, the module is recorded as failed and `dotfiles install` exits non-zero. Other modules still install.

After npm succeeds, `install.sh` fails if:

- `pi` isn't on `PATH` (a WSL `/mnt` path doesn't count)
- `pi` isn't runnable
- its version is off the pin
- node is below pi's own `engines.node`, when both versions are readable

`verify.sh` also fails if `pi` doesn't resolve in a login shell. A miss in a non-login shell is only a warning.

The `nodejs` dependency pins npm's global prefix to `~/.local` and enforces a node floor through `modules.nodejs.min_version`.
