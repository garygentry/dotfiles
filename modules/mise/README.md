# mise

Installs the [mise](https://mise.jdx.dev) binary and wires its shims onto `PATH`. Tools come from config, never from this module.

## Install

The binary goes to `~/.local/bin/mise` without sudo, verified against the release `SHASUMS256.txt`. Pin it with `modules.mise.version`. If it's unset, the module installs the latest version and warns. mise's own self-update isn't used, so bump the pin instead.

A mise already on `PATH` from elsewhere (Homebrew, a distro package) is used as it is. The pin doesn't apply to it, and the module warns if the versions differ.

## Declaring tools

`modules.mise.tools` maps a tool to an exact, quoted version:

```yaml
modules:
  mise:
    tools:
      ripgrep: "14.1.1"
```

Quote every version: an unquoted `1.10` is a YAML float and arrives as `1.1`. The engine ships an empty list.

Any module can declare its own tools the same way, with a `tools:` setting plus `mise_sync_tools "$DOTFILES_MODULE_NAME"` in its `install.sh`, so a tool set follows whichever profile selects that module. Each module's tools go to `~/.config/mise/conf.d/<module>.toml`. After a module is uninstalled or pruned, its file is removed the next time the mise module runs. Your own `~/.config/mise/config.toml` (`mise use -g`) is left alone.

## Settings

| Setting | Effect |
|---|---|
| `modules.mise.version` | Pins the mise binary. |
| `modules.mise.tools` | Tools to install, as above. |
| `modules.mise.settings` | Rendered to `~/.config/mise/conf.d/dotfiles-settings.toml` (e.g. `trusted_config_paths`, `minimum_release_age`, `github_attestations`). |
| `modules.mise.lockfile` | Path to a `mise.lock`; see below. |
| `modules.mise.github_token_command` | A command whose stdout is a GitHub token. Used only for the tools in `modules.mise.tools`, to avoid the 60 req/h anonymous API limit when resolving versions without a lockfile. Other modules' tools don't use it. |
| `modules.mise.activate` | Set to `false` to use shims alone; see below. |

## Lockfile

`modules.mise.lockfile` is a path, relative to the content overlay or else the engine directory, to a `mise.lock` made with `mise lock --global --platform ...`. When it's set:

- The engine owns `~/.config/mise/mise.lock`.
- Every module's declared tools install with `--locked`, using checksummed URLs and no GitHub API calls.
- Lock the union of all declared tools, and declare global tools in config rather than with `mise use -g`.

## PATH

Shims (`~/.local/share/mise/shims`) go on `PATH` for every shell, through managed blocks in `~/.zshenv`, `~/.profile` and the top of `~/.bashrc`. `~/.bash_profile` or `~/.bash_login` also gets a block when one exists, since bash login shells read that instead. This covers `ssh host cmd`, `bash -lc`, cron-less scripts and agent tool shells.

Interactive zsh also runs `mise activate` for per-directory env and hooks. Set `modules.mise.activate: false` to use shims alone.

systemd units and cron entries don't read shell rc files. Call tools by absolute path or via `~/.local/bin/mise exec -- <cmd>`.
