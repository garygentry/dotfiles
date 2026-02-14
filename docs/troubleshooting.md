# Troubleshooting

Common issues and their solutions.

## Installation Issues

### Bootstrap Script Fails

**Problem:** Bootstrap script fails to download or execute.

**Solutions:**

```bash
# Check internet connectivity
ping -c 3 google.com

# Try manual installation instead
git clone https://github.com/garygentry/dotfiles.git ~/.dotfiles
cd ~/.dotfiles
go build -o bin/dotfiles .
./bin/dotfiles install
```

### Go Not Found

**Problem:** `go: command not found` error.

**Solutions:**

```bash
# Install Go manually
# macOS (with Homebrew)
brew install go

# Ubuntu/Debian
sudo apt update && sudo apt install -y golang-go

# Arch Linux
sudo pacman -S go

# Or download from official site
curl -fsSL https://go.dev/dl/go1.23.6.linux-amd64.tar.gz | sudo tar -C /usr/local -xz
export PATH="/usr/local/go/bin:$PATH"
```

### Build Fails

**Problem:** `go build` fails with errors.

**Solutions:**

```bash
# Ensure Go version is 1.22+
go version

# Clean and rebuild
go clean
go mod download
go build -o bin/dotfiles .

# Check for missing dependencies
go mod tidy
```

## Module Installation Issues

### Module Fails to Install

**Problem:** A module fails during installation.

**Debug Steps:**

```bash
# Run with verbose output
dotfiles install module-name -v

# Check state file for error details
cat ~/.dotfiles/.state/module-name.json

# Look for the error field
cat ~/.dotfiles/.state/module-name.json | jq .error
```

**Common Causes:**

1. **Missing System Requirements:**
```bash
# Check if required commands exist
command -v git  # Replace with required command
```

2. **Permission Issues:**
```bash
# Some operations need sudo
# Ensure you can run sudo commands
sudo -v
```

3. **Network Issues:**
```bash
# Check connectivity to package repos
curl -I https://github.com
```

### Package Installation Fails

**Problem:** `pkg_install` fails in module script.

**Solutions:**

```bash
# Update package manager cache
# macOS
brew update

# Ubuntu/Debian
sudo apt update

# Arch Linux
sudo pacman -Sy

# Then retry
dotfiles install module-name
```

### SSH Module Fails

**Problem:** SSH key generation or configuration fails.

**Debug:**

```bash
# Check existing SSH directory
ls -la ~/.ssh

# Check permissions
stat ~/.ssh

# Run with verbose output
dotfiles install ssh -v
```

**Solutions:**

```bash
# Backup and remove existing SSH config if corrupted
mv ~/.ssh ~/.ssh.backup
dotfiles install ssh

# Or remove just the config
rm ~/.ssh/config
dotfiles install ssh
```

### Git Module Fails

**Problem:** Git configuration fails.

**Debug:**

```bash
# Check current git config
git config --global --list

# Run with verbose output
dotfiles install git -v
```

**Solutions:**

```bash
# Reset git config
rm ~/.gitconfig
dotfiles install git

# Or edit config.yml with correct values
vim ~/.dotfiles/config.yml
```

### Zsh Module Fails

**Problem:** Zsh installation or Zinit setup fails.

**Debug:**

```bash
# Check if zsh is installed
command -v zsh

# Check Zinit directory
ls -la ~/.local/share/zinit

# Run with verbose output
dotfiles install zsh -v
```

**Solutions:**

```bash
# Remove Zinit and retry
rm -rf ~/.local/share/zinit
dotfiles install zsh

# If shell change fails, do it manually
chsh -s $(which zsh)
```

## Configuration Issues

### Config File Not Found

**Problem:** `config.yml not found` error.

**Solutions:**

```bash
# Check if file exists
ls -la ~/.dotfiles/config.yml

# Create from template if missing
cat > ~/.dotfiles/config.yml << 'EOF'
profile: developer

user:
  name: "Your Name"
  email: "your.email@example.com"
  github_user: "yourusername"
EOF
```

### Invalid YAML Syntax

**Problem:** YAML parsing errors.

**Solutions:**

```bash
# Validate YAML syntax
# Install yq if needed
brew install yq  # macOS
sudo apt install yq  # Ubuntu

# Check syntax
yq eval ~/.dotfiles/config.yml

# Common issues:
# - Inconsistent indentation (use spaces, not tabs)
# - Missing quotes around special characters
# - Unclosed quotes or brackets
```

### Profile Not Found

**Problem:** `profile 'name' not found` error.

**Solutions:**

```bash
# List available profiles
ls -la ~/.dotfiles/profiles/

# Create custom profile
cat > ~/.dotfiles/profiles/custom.yml << 'EOF'
modules:
  - git
  - zsh
EOF

# Or use a different profile
dotfiles install --profile developer
```

## Secrets Management Issues

### 1Password Not Authenticated

**Problem:** `not authenticated with 1Password` error.

**Solutions:**

```bash
# Check if op CLI is installed
command -v op

# Install 1Password CLI if missing
# macOS
brew install 1password-cli

# Ubuntu/Debian
curl -sS https://downloads.1password.com/linux/keys/1password.asc | sudo gpg --dearmor --output /usr/share/keyrings/1password-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/1password-archive-keyring.gpg] https://downloads.1password.com/linux/debian/$(dpkg --print-architecture) stable main" | sudo tee /etc/apt/sources.list.d/1password.list
sudo apt update && sudo apt install 1password-cli

# Sign in to 1Password
eval $(op signin)

# Retry installation
dotfiles install
```

### Secret Not Found

**Problem:** `secret not found` error.

**Solutions:**

```bash
# Verify secret reference format
# Correct: op://vault-name/item-name/field-name
# Example: op://Private/GitHub/token

# List vaults
op vault list

# List items in vault
op item list --vault "Private"

# Get specific item
op item get "GitHub" --vault "Private"
```

### Skip Secrets

**Problem:** Don't want to use 1Password.

**Solutions:**

```bash
# Remove secrets config from config.yml
# Modules will fall back to default behavior
# (e.g., SSH module will generate new keys instead of retrieving from 1Password)

# Edit config.yml and remove:
# secrets:
#   provider: 1password
#   account: my.1password.com
```

## State Management Issues

### Stale State

**Problem:** Module appears installed but isn't actually configured.

**Solutions:**

```bash
# Remove state file to force reinstall
rm ~/.dotfiles/.state/module-name.json

# Reinstall module
dotfiles install module-name
```

### Corrupted State

**Problem:** State file is corrupted or unreadable.

**Solutions:**

```bash
# Remove state file
rm ~/.dotfiles/.state/module-name.json

# Or remove all state
rm -rf ~/.dotfiles/.state/
mkdir -p ~/.dotfiles/.state/

# Reinstall
dotfiles install
```

## Platform-Specific Issues

### macOS Issues

**Xcode Command Line Tools Required:**
```bash
# Install if missing
xcode-select --install
```

**Homebrew Not Found:**
```bash
# Install Homebrew
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

**Permission Issues:**
```bash
# Fix Homebrew permissions
sudo chown -R $(whoami) /usr/local/*
```

### Ubuntu/Debian Issues

**Package Manager Lock:**
```bash
# Wait for other package operations to complete
# Or find and kill the process
ps aux | grep apt
sudo kill <process-id>

# Remove lock files (use with caution)
sudo rm /var/lib/dpkg/lock-frontend
sudo rm /var/lib/apt/lists/lock
sudo dpkg --configure -a
```

**PPA Errors:**
```bash
# Remove problematic PPA
sudo add-apt-repository --remove ppa:name/ppa
sudo apt update
```

### Arch Linux Issues

**Keyring Issues:**
```bash
# Reinitialize keyring
sudo pacman-key --init
sudo pacman-key --populate archlinux
sudo pacman -Sy archlinux-keyring
```

**Mirror Issues:**
```bash
# Update mirror list
sudo pacman -Sy reflector
sudo reflector --country 'United States' --age 12 --protocol https --sort rate --save /etc/pacman.d/mirrorlist
```

## Performance Issues

### Slow Installation

**Problem:** Installation takes a very long time.

**Causes:**

1. **Slow package mirrors** - Try different mirrors
2. **Network issues** - Check bandwidth
3. **Large downloads** - Some packages are big (Go, Zinit plugins, etc.)

**Solutions:**

```bash
# Use faster mirrors
# See platform-specific sections above

# Run one module at a time
dotfiles install ssh
dotfiles install git
dotfiles install zsh
```

### High Memory Usage

**Problem:** System runs out of memory during installation.

**Solutions:**

```bash
# Close other applications
# Increase swap space (Linux)
sudo fallocate -l 4G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

## Template Rendering Issues

### Template Syntax Error

**Problem:** Template rendering fails with syntax error.

**Debug:**

```bash
# Check template file
cat ~/.dotfiles/modules/module-name/files/template.tmpl

# Run with verbose output to see full error
dotfiles install module-name -v
```

**Common Issues:**

```go
// Missing closing brace
{{ .User.Name

// Wrong field name (case-sensitive)
{{ .user.name }}  // Should be .User.Name

// Using undefined field
{{ .NonExistent }}  // Field doesn't exist in context
```

### Missing Template Variables

**Problem:** Template expects variables that aren't set.

**Solutions:**

```bash
# Check config.yml has all required fields
vim ~/.dotfiles/config.yml

# Add missing fields
user:
  name: "Your Name"
  email: "your.email@example.com"
  github_user: "yourusername"
```

## Getting Help

### Enable Verbose Mode

Always start with verbose output:

```bash
dotfiles install -v
```

### Check Logs

```bash
# State files contain error messages
cat ~/.dotfiles/.state/module-name.json | jq .

# Script output is shown with -v flag
dotfiles install module-name -v 2>&1 | tee debug.log
```

### Create Debug Report

When reporting issues, include:

```bash
# System information
uname -a
go version

# Dotfiles version
cd ~/.dotfiles && git log -1 --oneline

# Module state
cat ~/.dotfiles/.state/module-name.json

# Verbose output
dotfiles install module-name -v 2>&1 | tee debug.log
```

### Ask for Help

If you're still stuck:

1. Check [GitHub Issues](https://github.com/garygentry/dotfiles/issues)
2. Search [GitHub Discussions](https://github.com/garygentry/dotfiles/discussions)
3. Create a new issue with debug information

## Known Issues

### Issue: Git signing fails on new systems

**Workaround:** Ensure SSH keys are generated before configuring Git signing.

```bash
# Install modules in order
dotfiles install 1password ssh git
```

### Issue: Zsh not set as default shell in containers

**Expected:** Containers may not allow `chsh`. The zsh module warns but doesn't fail.

```bash
# Manually set shell in container
chsh -s $(which zsh)
# Or just run zsh
zsh
```

### Issue: Some packages not available in older OS versions

**Workaround:** OS-specific scripts attempt to use PPAs or alternative sources, but some packages may not be available.

```bash
# Check your OS version
lsb_release -a  # Ubuntu/Debian
uname -r        # All systems
```

## Unattended Mode / CI-CD Issues

### Installation Hangs in CI/CD

**Problem:** Installation blocks waiting for input in automated environments.

**Solutions:**

```bash
# Use --unattended flag
dotfiles install --unattended

# For bootstrap script
curl -sfL https://url/to/bootstrap.sh | bash -s -- --unattended
```

### Modules Fail in Docker/CI

**Problem:** Some modules require interactive terminal or system features not available in containers.

**Solutions:**

```bash
# Use --skip-failed to continue installation
dotfiles install --unattended --skip-failed

# Or create a container-specific profile
cat > profiles/docker.yml << 'EOF'
modules:
  - git
  - zsh
  # Exclude modules that need GUI or special permissions
EOF

dotfiles install --unattended --profile docker
```

### Secrets Not Available in CI

**Problem:** 1Password prompts block installation in CI/CD pipelines.

**Solutions:**

```bash
# Unattended mode auto-skips secrets authentication
dotfiles install --unattended

# Or disable secrets in config
cat > config.yml << 'EOF'
profile: ci
secrets:
  provider: ""  # Disable secrets
EOF

# Use a profile without secrets-dependent modules
dotfiles install --unattended --profile ci
```

### Non-Interactive Stdin

**Problem:** Installation hangs when piped from curl or in automation.

**Solutions:**

```bash
# System auto-detects non-interactive stdin, but you can force it
dotfiles install --unattended

# Check if running in CI/CD environment
if [ -n "$CI" ]; then
  dotfiles install --unattended
fi
```

### Exit Code Handling

**Problem:** Need to handle failures in automated scripts.

**Solutions:**

```bash
#!/bin/bash
set -euo pipefail

# Installation with error handling
if dotfiles install --unattended --skip-failed; then
  echo "Installation successful"
else
  echo "Installation failed with exit code $?"
  dotfiles status  # Show what succeeded
  exit 1
fi

# Verify critical modules
for module in git zsh; do
  if ! dotfiles status | grep -q "${module}.*installed"; then
    echo "ERROR: ${module} not installed"
    exit 1
  fi
done
```

For comprehensive CI/CD integration examples, see the [CI/CD Guide](ci-cd-guide.md).

## See Also

- [CLI Reference](cli-reference.md) - Command documentation
- [Configuration](configuration.md) - Configuration options
- [Creating Modules](creating-modules.md) - Module development guide
- [CI/CD Guide](ci-cd-guide.md) - Automation and IaC integration
