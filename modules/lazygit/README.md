# lazygit

Installs [lazygit](https://github.com/jesseduffield/lazygit) without sudo or a system package, either through mise or from the upstream release.

## With `modules.lazygit.tools`

This setting maps a tool to an exact, quoted version:

```yaml
modules:
  lazygit:
    tools:
      lazygit: "0.65.1"
```

The module declares it in `~/.config/mise/conf.d/lazygit.toml`, installs it through mise (which needs the `mise` module) and removes its own release install. If mise has no build for this platform (a lockfile without one), it falls back to the path below.

## Without `tools`

- **macOS with Homebrew:** installs through Homebrew.
- **Otherwise:** installs the upstream release tarball into `~/.local/share/lazygit` and links it from `~/.local/bin`. The tarball is verified against the release's `checksums.txt`. A missing checksum fails the install unless `DOTFILES_ALLOW_UNVERIFIED=1`.

It installs when no runnable `lazygit` is found. `DOTFILES_LAZYGIT_VERSION` (default: latest) also reinstalls this module's own copy at a different version.

## Config

The module ships no config. Deploy `~/.config/lazygit/config.yml` from a content overlay.
