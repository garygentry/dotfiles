# herdr

Installs the upstream [herdr](https://herdr.dev) release binary to `~/.local/bin` through `herdr.dev/install.sh`, SHA-256 verified and without sudo. On macOS it uses Homebrew when `brew` is present.

## Updates

herdr owns its updates through `herdr update`. This module only installs when the binary is missing.

## Config

The module ships no `config.toml`. Deploy one from a content overlay as type `copy`, because herdr rewrites it at runtime (onboarding, the settings UI, `herdr config reset-keys`).
