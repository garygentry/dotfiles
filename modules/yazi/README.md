# yazi

Installs the [yazi](https://yazi-rs.github.io) terminal file manager (`yazi` and `ya`) without sudo, either through mise or from the upstream release.

## With `modules.yazi.tools`

This setting maps a tool to an exact, quoted version. It must include yazi itself, and can add preview helpers such as `glow` or `7zip`:

```yaml
modules:
  yazi:
    tools:
      yazi: "26.9.1"
      # or "github:sxyazi/yazi[matching=musl]": "26.9.1"
```

The module declares these tools in `~/.config/mise/conf.d/yazi.toml`, installs them through mise (which needs the `mise` module) and removes its own release install. If mise has no build for this platform (a lockfile without one), it falls back to the path below.

## Without `tools`

- **macOS with Homebrew:** installs through Homebrew.
- **Otherwise:** installs the upstream release zip (musl on Linux, apple-darwin on macOS) into `~/.local/share/yazi` and links `yazi` and `ya` from `~/.local/bin`. The zip is verified against the release asset's sha256 digest from the GitHub API, using `GITHUB_TOKEN` when set. A missing digest fails the install unless `DOTFILES_ALLOW_UNVERIFIED=1`.

It installs when no runnable `yazi` and `ya` are found. `DOTFILES_YAZI_VERSION` (default: latest) also reinstalls this module's own copy at a different version.

## Config

The module ships no config. Deploy `~/.config/yazi/{yazi,keymap,theme}.toml` from a content overlay. `~/.config/yazi/package.toml` must be a copy, because `ya pkg` rewrites it.

## Previews

Previews use external tools when present: `file`, `fd`, `rg`, `fzf`, `jq`, `7zip`, and `chafa` >= 1.16 for block-character images. This module installs none of them unless you list them in `tools`.
