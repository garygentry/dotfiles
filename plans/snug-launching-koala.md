# Add new modules + update mechanism

## Context

Expanding the dotfiles system with new modules across several categories. Each module is assessed for installation robustness across Ubuntu, macOS, and Arch. Also adding a lightweight update mechanism so `dotfiles install --force` re-runs modules and updates tools.

---

## Robustness Assessment

| Module | Ubuntu | macOS | Arch | Install Method | Verdict |
|--------|--------|-------|------|----------------|---------|
| claude-code | native installer | native installer | native installer | `curl \| bash` (auto-updates) | **Include** |
| gemini-cli | npm | npm | npm | `npm i -g` (needs nodejs) | **Include** |
| ghostty | snap | brew | pacman | OS-specific | **Include** |
| zellij | apt/cargo | brew | pacman | pkg_install + cargo fallback | **Include** |
| fish | PPA+apt | brew | pacman | pkg_install (PPA on Ubuntu) | **Include** |
| zoxide | apt/script | brew | pacman | pkg_install + script fallback | **Include** |
| btop | apt | brew | pacman | pkg_install | **Include** |
| gh | apt (official repo) | brew | pacman | OS-specific repo setup | **Include** |
| awscli | official installer | brew | official installer | curl+unzip+install | **Include** |
| azure-cli | official script | brew | pacman | OS-specific | **Include** |
| gcloud | apt (official repo) | brew/installer | *no pacman* | OS-specific repo | **Include** (skip Arch) |
| rust | rustup script | rustup script | pacman (rustup) | curl script / pacman | **Include** |

All 12 are robust enough to include. gcloud skips Arch (no pacman package, AUR needs helper we don't support).

---

## Update Mechanism

**Approach:** Make install.sh scripts update-aware. When a tool is already installed, instead of just skipping, attempt an update. This works naturally with `dotfiles install --force` (which re-runs all modules).

**Convention for install scripts:**
```bash
if command -v tool &>/dev/null; then
    log_info "tool is already installed, updating..."
    # Run update command (brew upgrade, apt upgrade, rustup update, etc.)
else
    log_info "Installing tool..."
    # Fresh install
fi
```

**How updates are triggered:**
- `dotfiles install --force` re-runs all modules (existing mechanism)
- Module version bump causes re-run (idempotence detects change)
- No new Go code needed

Tools with built-in auto-updates (Claude Code native, Ghostty) just log that they're self-updating.

---

## Module Specifications

### 1. claude-code (AI CLI)

```
modules/claude-code/
├── module.yml
├── install.sh
└── verify.sh
```

- **Priority:** 50, **Deps:** [], **OS:** all, **Requires:** []
- **Install:** `curl -fsSL https://claude.ai/install.sh | bash` (native installer, no Node.js needed)
- **Update:** Re-run native installer (handles updates)
- **Verify:** `command -v claude`
- **Notes:** `["Run 'claude' to authenticate with your Anthropic account"]`

### 2. gemini-cli (AI CLI)

```
modules/gemini-cli/
├── module.yml
├── install.sh
└── verify.sh
```

- **Priority:** 50, **Deps:** [nodejs], **OS:** all, **Requires:** [npm]
- **Install:** `npm install -g @google/gemini-cli`
- **Update:** `npm update -g @google/gemini-cli`
- **Verify:** `command -v gemini`
- **Notes:** `["Run 'gemini' to authenticate with your Google account"]`

### 3. ghostty (Terminal)

```
modules/ghostty/
├── module.yml
├── install.sh
├── verify.sh
└── os/
    ├── ubuntu.sh
    ├── macos.sh
    └── arch.sh
```

- **Priority:** 35, **Deps:** [], **OS:** [ubuntu, macos, arch], **Requires:** []
- **Install (Ubuntu):** `sudo snap install ghostty` (no official apt package)
- **Install (macOS):** `brew install ghostty`
- **Install (Arch):** `pkg_install ghostty`
- **Update:** Package manager handles it (snap refresh, brew upgrade, pacman -Syu)
- **Verify:** `command -v ghostty`

### 4. zellij (Terminal Multiplexer)

```
modules/zellij/
├── module.yml
├── install.sh
└── verify.sh
```

- **Priority:** 35, **Deps:** [], **OS:** all, **Requires:** []
- **Install:** `pkg_install zellij` with cargo fallback (`cargo install zellij`) if pkg_install fails
- **Update:** Package manager or `cargo install zellij` (overwrites)
- **Verify:** `command -v zellij`

### 5. fish (Shell)

```
modules/fish/
├── module.yml
├── install.sh
├── verify.sh
└── os/
    └── ubuntu.sh
```

- **Priority:** 40, **Deps:** [], **OS:** all, **Requires:** []
- **Install (Ubuntu):** Add PPA (`ppa:fish-shell/release-4`), then `pkg_install fish`
- **Install (macOS/Arch):** `pkg_install fish`
- **Note:** Does NOT change default shell (that's the zsh module's job). Fish is available as `fish` command for users who want to run it explicitly or set it up themselves.
- **Verify:** `command -v fish`

### 6. zoxide (CLI)

```
modules/zoxide/
├── module.yml
├── install.sh
└── verify.sh
```

- **Priority:** 20, **Deps:** [], **OS:** all, **Requires:** []
- **Install:** `pkg_install zoxide` with fallback to official script: `curl -sS https://raw.githubusercontent.com/ajeetdsouza/zoxide/main/install.sh | bash`
- **Shell integration:** Add `eval "$(zoxide init zsh)"` to zshrc.tmpl (modify existing zsh template to include it if zoxide is available, like starship pattern)
- **Verify:** `command -v zoxide`

### 7. btop (CLI)

```
modules/btop/
├── module.yml
├── install.sh
└── verify.sh
```

- **Priority:** 20, **Deps:** [], **OS:** all, **Requires:** []
- **Install:** `pkg_install btop`
- **Verify:** `command -v btop`

### 8. gh (CLI)

```
modules/gh/
├── module.yml
├── install.sh
├── verify.sh
└── os/
    └── ubuntu.sh
```

- **Priority:** 20, **Deps:** [git], **OS:** all, **Requires:** [curl]
- **Install (Ubuntu):** Add official GitHub apt repo (keyring + sources), then `pkg_install gh`
- **Install (macOS):** `pkg_install gh`
- **Install (Arch):** `pkg_install github-cli`
- **Verify:** `command -v gh`
- **Notes:** `["Run 'gh auth login' to authenticate with GitHub"]`

### 9. awscli (Cloud CLI)

```
modules/awscli/
├── module.yml
├── install.sh
├── verify.sh
└── os/
    ├── ubuntu.sh
    └── macos.sh
```

- **Priority:** 50, **Deps:** [], **OS:** [ubuntu, macos, arch], **Requires:** [curl]
- **Install (Ubuntu/Arch):** Download zip, unzip, `sudo ./aws/install` (or `--update` if exists)
- **Install (macOS):** `brew install awscli`
- **Update:** `sudo ./aws/install --update` (Ubuntu/Arch), `brew upgrade awscli` (macOS)
- **Verify:** `command -v aws`
- **Notes:** `["Run 'aws configure' to set up your AWS credentials"]`

### 10. azure-cli (Cloud CLI)

```
modules/azure-cli/
├── module.yml
├── install.sh
├── verify.sh
└── os/
    └── ubuntu.sh
```

- **Priority:** 50, **Deps:** [], **OS:** [ubuntu, macos, arch], **Requires:** [curl]
- **Install (Ubuntu):** `curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash`
- **Install (macOS):** `brew install azure-cli`
- **Install (Arch):** `pkg_install azure-cli`
- **Update:** Re-run installer (Ubuntu), brew upgrade (macOS), pacman (Arch)
- **Verify:** `command -v az`
- **Notes:** `["Run 'az login' to authenticate with Azure"]`

### 11. gcloud (Cloud CLI)

```
modules/gcloud/
├── module.yml
├── install.sh
├── verify.sh
└── os/
    ├── ubuntu.sh
    └── macos.sh
```

- **Priority:** 50, **Deps:** [], **OS:** [ubuntu, macos] (no Arch — AUR only), **Requires:** [curl]
- **Install (Ubuntu):** Add Google Cloud apt repo (keyring + sources), then `pkg_install google-cloud-cli`
- **Install (macOS):** `brew install google-cloud-cli`
- **Update:** `gcloud components update` or package manager
- **Verify:** `command -v gcloud`
- **Notes:** `["Run 'gcloud init' to set up your Google Cloud configuration"]`

### 12. rust (Programming)

```
modules/rust/
├── module.yml
├── install.sh
└── verify.sh
```

- **Priority:** 50, **Deps:** [], **OS:** all, **Requires:** [curl]
- **Install:** `curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y` (non-interactive)
- **Update:** `rustup update` (built-in)
- **Verify:** `command -v rustc && command -v cargo`

### zshrc.tmpl update (zoxide integration)

Add zoxide init to the shared section of `modules/zsh/zshrc.tmpl`, following the existing starship pattern:

```bash
# zoxide (smarter cd)
if command -v zoxide &>/dev/null; then
    eval "$(zoxide init zsh)"
fi
```

---

## Implementation Order

Batch 1 — Simple pkg_install modules (no OS-specific scripts):
1. btop
2. zellij
3. zoxide (+ zshrc.tmpl update)

Batch 2 — Curl/script installers:
4. rust
5. claude-code

Batch 3 — npm-based:
6. gemini-cli

Batch 4 — OS-specific repo setup:
7. gh
8. fish
9. ghostty

Batch 5 — Cloud CLIs:
10. awscli
11. azure-cli
12. gcloud

---

## Files created (new modules: ~3-5 files each)

12 new module directories under `modules/`, plus one edit to `modules/zsh/zshrc.tmpl` for zoxide integration.

## Verification

```bash
export PATH="/usr/local/go/bin:$PATH"
go test ./internal/module/ ./cmd/dotfiles/
go build ./cmd/dotfiles/
# Verify module discovery finds all new modules:
./dotfiles list
```
