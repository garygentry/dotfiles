# homebrew Module

The Homebrew package manager for macOS and Linux.

## Features

- Installs Homebrew package manager
- Supports macOS (Intel and Apple Silicon)
- Supports Linux (Ubuntu and Arch)
- Configures shell integration
- Runs health checks with `brew doctor`

## Prerequisites

- curl
- git
- Sudo access for package installation

## Installation

```bash
dotfiles install homebrew
```

## Configuration

Homebrew installs to:
- macOS (Apple Silicon): `/opt/homebrew`
- macOS (Intel): `/usr/local`
- Linux: `/home/linuxbrew/.linuxbrew`

## Platform Support

- ✅ macOS (Intel and Apple Silicon)
- ✅ Ubuntu
- ✅ Arch Linux

## Notes

- On first install, Homebrew may take several minutes to download and compile
- The module runs `brew update` after installation
- Shell integration is automatically configured
- This is a foundation module - many other modules depend on Homebrew for package installation
