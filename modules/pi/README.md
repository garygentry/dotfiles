# pi

Installs the `pi` coding agent globally through npm, at a pinned version.

## Version

Set `modules.pi.version` in your content overlay's `config.yml`. If it's unset, the module installs npm's `latest` and warns.

This module never runs `pi update`. A manual `pi update` is corrected back to the pin on the next install.

## Authentication

Run `pi` once interactively to authenticate. Auth and trust state are local to the machine and aren't managed here.

## When the install can't complete

A missing npm or a failed global install (no registry, or offline) degrades to a warning and success, so a locked-down host still completes `dotfiles install` (ADR 0025 D6). `verify.sh` is the gate that `pi` is actually present.

Two cases are hard failures: a resolved version that's off the pin, and a WSL `/mnt` provenance miss.

The `nodejs` dependency pins npm's global prefix to `~/.local` and enforces pi's `engines.node` floor (`modules.nodejs.min_version`). This module checks pi's own `engines.node` again and fails if it isn't met.
