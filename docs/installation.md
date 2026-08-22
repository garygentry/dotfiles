---
title: "Installation"
---

# Installation

This guide covers different ways to install the dotfiles management system.

## Quick Install (Recommended)

The fastest way to get started is using the bootstrap script:

```bash
curl -sfL https://raw.githubusercontent.com/garygentry/dotfiles/main/bootstrap.sh | bash
```

This will:
1. Detect your operating system and architecture
2. Install Git (if not present)
3. Install Go 1.23.6 (if not present)
4. Clone the repository to `~/.dotfiles`
5. Build the dotfiles CLI
6. Run the installation

## Manual Installation

### Prerequisites

- **Git** - Version control system
- **Go** - Version 1.23 or later
- **Bash** - Version 4.0 or later

### Step 1: Clone the Repository

```bash
git clone https://github.com/garygentry/dotfiles.git ~/.dotfiles
cd ~/.dotfiles
```

### Step 2: Build the CLI

```bash
go build -o bin/dotfiles .
```

### Step 3: Run Installation

```bash
./bin/dotfiles install
```

## Platform-Specific Notes

### macOS

The bootstrap script will use Homebrew to install dependencies if available. If Homebrew is not installed, it will install Go from the official tarball.

```bash
# Install Homebrew first (optional but recommended)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Then run bootstrap
curl -sfL https://raw.githubusercontent.com/garygentry/dotfiles/main/bootstrap.sh | bash
```

### Ubuntu/Debian

The bootstrap script will use `apt` to install system packages. You may need sudo privileges for system package installation.

```bash
# Ensure system is up to date
sudo apt update && sudo apt upgrade -y

# Run bootstrap
curl -sfL https://raw.githubusercontent.com/garygentry/dotfiles/main/bootstrap.sh | bash
```

### Arch Linux

The bootstrap script will use `pacman` to install system packages.

```bash
# Update system
sudo pacman -Syu

# Run bootstrap
curl -sfL https://raw.githubusercontent.com/garygentry/dotfiles/main/bootstrap.sh | bash
```

## Configuration

### Environment Variables

You can customize the installation by setting environment variables before running the bootstrap script:

```bash
# Custom installation directory
export DOTFILES_DIR="$HOME/my-dotfiles"

# Custom repository URL
export DOTFILES_REPO="https://github.com/<your-fork>/dotfiles.git"

# Custom Go version
export GO_VERSION="1.23.6"

# Run bootstrap
curl -sfL https://raw.githubusercontent.com/garygentry/dotfiles/main/bootstrap.sh | bash
```

### Custom Profile

To install a specific profile during bootstrap:

```bash
curl -sfL https://raw.githubusercontent.com/garygentry/dotfiles/main/bootstrap.sh | bash -s -- --profile minimal
```

### Content Overlay

The engine is generic; your personal identity, secrets choice, and any custom
profiles/modules live in an optional **content overlay** directory that is deep-merged
over the repo. Point the bootstrap script at your content repo and it will clone it and
persist `DOTFILES_CONTENT_DIR` for you:

```bash
# Materialize a content overlay from your own repo during bootstrap
curl -sfL https://raw.githubusercontent.com/garygentry/dotfiles/main/bootstrap.sh | bash -s -- \
  --content-repo https://github.com/you/my-dotfiles.git
```

Or use an already-cloned overlay by exporting the directory before installing:

```bash
export DOTFILES_CONTENT_DIR="$HOME/my-dotfiles"
```

The `--content-*` flags (`--content-repo`, `--content-ref`, `--content-path`,
`--content-dir`, `--content-auth-cmd`, `--no-persist-content-dir`) are **bootstrap
flags**, not `dotfiles` subcommand flags. See the [Content Overlay guide](content-overlay.md)
for the full model, private-repo auth, and layout.

## Verification

After installation, verify the CLI is working:

```bash
# Check version
~/.dotfiles/bin/dotfiles --version

# List available modules
~/.dotfiles/bin/dotfiles list

# View help
~/.dotfiles/bin/dotfiles --help
```

## Adding to PATH

For easier access, add the dotfiles binary to your PATH:

```bash
# Add to your shell rc file (~/.zshrc, ~/.bashrc, etc.)
export PATH="$HOME/.dotfiles/bin:$PATH"

# Reload shell
source ~/.zshrc  # or ~/.bashrc
```

Then you can run:

```bash
dotfiles install
dotfiles list
```

## Uninstallation

To remove the dotfiles system:

```bash
# Remove the repository
rm -rf ~/.dotfiles

# Remove state directory
rm -rf ~/.dotfiles/.state

# Manually remove any symlinked configuration files if desired
```

Note: This does not uninstall packages or tools installed by modules. You'll need to remove those manually if desired.

## Next Steps

- [Quick Start Guide](quick-start.md) - Get started with common tasks
- [Creating Modules](creating-modules.md) - Build and extend modules
- [CLI Reference](cli-reference.md) - Full command documentation
