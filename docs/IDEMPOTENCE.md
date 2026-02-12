# Idempotence System

The dotfiles system is fully idempotent - you can safely run `dotfiles install` multiple times without unnecessary work or data loss.

## Quick Reference

### Common Commands

```bash
# Normal install (skips unchanged)
dotfiles install

# After git pull (only changed modules run)
git pull && dotfiles install

# Force reinstall everything
dotfiles install --force

# Skip failed modules
dotfiles install --skip-failed

# Only update existing (no new installs)
dotfiles install --update-only

# Check what needs updating
dotfiles status
```

### Status Symbols

| Symbol | Meaning |
|--------|---------|
| `✓` | Up-to-date, nothing to do |
| `•` | Needs update (version/changed/config) |
| `⚠` | User modified files |
| `!` | Failed previously |

### Why Things Run or Skip

**Module runs when:**
- First time installing
- Module version changed
- Module scripts changed (`install.sh`, `verify.sh`, `os/*.sh`)
- User config changed (`config.yml` values for this module)
- Previously failed (and not using `--skip-failed`)
- `--force` flag used

**Module skips when:**
- Already installed and no changes detected
- Failed previously + `--skip-failed` flag
- New module + `--update-only` flag

**File deploys when:**
- Source file changed
- Destination missing
- Symlink points to wrong location

**File skips when:**
- Already correct (hash matches)
- User modified (source unchanged)

---

## Overview

**Idempotent** means running the same command multiple times produces the same result as running it once. The dotfiles system now:

- ✅ **Skips unchanged modules** - Only runs what actually changed
- ✅ **Skips unchanged files** - Only deploys what's different
- ✅ **Protects user modifications** - Backs up before overwriting
- ✅ **Detects all changes** - Version, config, scripts, and files
- ✅ **Clear feedback** - Always shows why something ran or was skipped

## How It Works

### Module-Level Idempotence

Before running a module, the system checks:

1. **Module checksum** - SHA256 of module.yml + all scripts
2. **Config hash** - SHA256 of user config affecting this module
3. **Version** - Module version field
4. **Status** - Previous installation status

**Execution decisions:**
- ✓ **Skip** - Already up-to-date (nothing changed)
- → **Install fresh** - No previous installation
- → **Install retry** - Failed previously (unless --skip-failed)
- → **Update module** - Module definition/scripts/version changed
- → **Update config** - User config values changed
- → **Force** - --force flag overrides all checks

### File-Level Idempotence

Before deploying each file, the system checks:

1. **Source hash** - Did the source file change?
2. **Destination exists** - Is the file already deployed?
3. **Symlink target** - Does symlink point to correct location?
4. **Content hash** - Does deployed content match source?
5. **User modifications** - Did user change the deployed file?

**Deployment decisions:**
- ✓ **Skip** - File already correct
- → **Deploy** - Source changed or destination missing
- ⚠ **Skip with warning** - User modified (source unchanged)
- → **Backup & deploy** - User modified + source changed

## Usage

### Basic Idempotent Install

```bash
# First run - installs everything
dotfiles install

# Second run - skips everything (nothing changed)
dotfiles install
# Output: ✓ git (skipped: already up-to-date)
#         ✓ zsh (skipped: already up-to-date)
```

### After Updating Repository

```bash
# Pull latest changes
git pull

# Install updates - only changed modules run
dotfiles install
# Output: ✓ git (skipped: already up-to-date)
#         → zsh (updating: source file changed)
#         ✓ zsh (updated in 2.3s)
```

### Force Reinstall

```bash
# Force reinstall even if up-to-date
dotfiles install --force

# Force specific module
dotfiles install git --force
```

### Skip Failed Modules

```bash
# Skip modules that failed previously
dotfiles install --skip-failed
# Output: ✓ git (skipped: already up-to-date)
#         ✓ broken-module (skipped: failed previously, --skip-failed set)
#         → zsh (installing...)
```

### Update Only (No New Installs)

```bash
# Only update already-installed modules
dotfiles install --update-only
# Output: Skipping new modules (--update-only): neovim, tmux
#         → git (updating: config changed)
#         ✓ zsh (skipped: already up-to-date)
```

## Status Command

The enhanced status command shows what needs updating:

```bash
dotfiles status
```

**Output:**
```
  Name     Version  Status     Update     Installed   OS
  ----     -------  ------     ------     ---------   --
  git      1.0.0    installed  ✓          2 days ago  ubuntu
  zsh      1.2.0    installed  • changed  1 week ago  ubuntu
  neovim   2.1.0    failed     ! failed   3 days ago  ubuntu
  tmux     1.0.0    installed  ⚠ modified 1 week ago  ubuntu

  Update status:  ✓ up-to-date  • needs update  ⚠ user modified  ! failed

Total: 4 modules (3 installed, 1 failed, 1 need update, 1 user modified)

Run 'dotfiles install' to update out-of-date modules
Run 'dotfiles install --force neovim' to retry failed modules
```

**Update status meanings:**
- `✓` **up-to-date** - Module and files match, nothing to do
- `• version` - Module version changed
- `• changed` - Module scripts/definition changed
- `• config` - User config affecting this module changed
- `⚠ modified` - User modified deployed files
- `! failed` - Installation failed previously

## Backup System

When a module update would overwrite user-modified files, the system automatically creates backups.

### Backup Location

```
~/.dotfiles/.backups/
├── 20260211-143022/        # Timestamp directory
│   ├── .zshrc              # Backed up file
│   └── .zshrc.meta.json    # Metadata
└── 20260211-150133/
    ├── .gitconfig
    └── .gitconfig.meta.json
```

### Backup Metadata

Each backup includes a JSON metadata file:

```json
{
  "original_path": "/home/user/.zshrc",
  "backup_time": "2026-02-11T14:30:22Z",
  "content_hash": "abc123...",
  "reason": "user-modified file overwritten by module update",
  "module": "zsh"
}
```

### Restoring Backups

Backups are manual - the system won't automatically restore them. To restore:

```bash
# Find your backup
ls -lt ~/.dotfiles/.backups/

# Copy back to original location
cp ~/.dotfiles/.backups/20260211-143022/.zshrc ~/.zshrc

# Or diff to see what changed
diff ~/.zshrc ~/.dotfiles/.backups/20260211-143022/.zshrc
```

## Change Detection Details

### Module Checksum

Computed from:
- `module.yml` content
- `install.sh` content (if exists)
- `verify.sh` content (if exists)
- All `os/*.sh` scripts (if exist)

Any change to these files triggers a re-run.

### Config Hash

Computed from:
- `user.name`, `user.email`, `user.github_user` (affects templates)
- Module-specific config from `config.modules.<module-name>`

Changes to these values trigger a re-run.

### File Hash

SHA256 of file content. Used to:
- Detect source file changes
- Detect user modifications
- Skip unchanged files

## Performance

The system is designed to be fast:

- **Skipped modules**: <100ms (state check only)
- **Skipped files**: <10ms per file (hash comparison)
- **Hash computation**: Cached within single run

## Edge Cases

### User Deleted Deployed File

**Behavior:** File will be redeployed on next run

```bash
rm ~/.gitconfig
dotfiles install git
# Output: → git (updating: destination file missing)
```

### User Modified Symlink

**Behavior:** Symlink recreated with warning

```bash
rm ~/.zshrc && echo "custom" > ~/.zshrc
dotfiles install zsh
# Output: ⚠ Backed up user-modified file: ~/.zshrc → ~/.dotfiles/.backups/...
#         → zsh (updating: symlink points to wrong location)
```

### Module Version Downgrade

**Behavior:** Treated as update (re-runs installation)

### Template Variables Changed

**Behavior:** Template re-rendered with new values

### Failed Module with Partial State

**Behavior:** Can retry (default) or skip (--skip-failed)

## Limitations

### Scripts Always Run on Update

Module scripts (install.sh, verify.sh, os/*.sh) always run when a module needs updating. The scripts themselves should be idempotent (using checks like `command -v`, `pkg_installed`, etc.).

### Package Installs

Package manager operations (apt, brew, etc.) are naturally idempotent, but rollback only provides informational messages - you must manually remove packages if needed.

### Concurrent Installs

Running multiple `dotfiles install` commands simultaneously is not supported and may cause state corruption.

## Best Practices

1. **Run regularly** - Safe to run after every `git pull`
2. **Check status** - Use `dotfiles status` to see what needs updating
3. **Review backups** - Periodically check `~/.dotfiles/.backups/` for important changes
4. **Write idempotent scripts** - Module scripts should handle being run multiple times
5. **Test updates** - Use `--dry-run` to preview changes

## Troubleshooting

### Module Always Re-runs

**Cause:** Module state missing checksums (old installation)

**Fix:** Run once - checksums will be recorded

### File Always Redeploys

**Cause:** File state missing hashes (old installation)

**Fix:** Run once - hashes will be recorded

### "User modified" but I didn't change it

**Cause:** File changed by another tool/process

**Fix:** This is expected - the system detects any changes, not just manual edits

### Want to force reinstall

**Solution:** Use `--force` flag

```bash
dotfiles install git --force
```

## Migration from Old Installations

The system is backward compatible. Old installations without checksums will:

1. First run: Module runs normally, checksums recorded
2. Second run: Module skips (now has checksums)

No manual migration needed!

## Technical Details

For developers wanting to understand the implementation:

- **State schema**: `internal/state/state.go` - ModuleState + FileState
- **Hash functions**: `internal/module/hash.go` - SHA256 computation
- **Decision logic**: `internal/module/runner.go` - shouldRunModule, shouldDeployFile
- **Backup system**: `internal/module/backup.go` - Timestamped backups
- **Tests**: `internal/module/*_test.go` - Comprehensive coverage

## Examples

### Daily Workflow

```bash
# Morning: update dotfiles
cd ~/dotfiles
git pull
dotfiles install
# Only changed modules run

# Check status
dotfiles status
# See what's up-to-date, what needs attention
```

### Testing Module Changes

```bash
# Edit a module
vim modules/zsh/install.sh

# See what changed
dotfiles status
# Output shows "• changed"

# Install with changes
dotfiles install
# Only zsh runs
```

### Recovering from Mistakes

```bash
# Accidentally installed broken config
dotfiles install

# Check backups
ls -lt ~/.dotfiles/.backups/

# Restore old version
cp ~/.dotfiles/.backups/20260211-143022/.zshrc ~/.zshrc

# Or rollback entire module
dotfiles rollback zsh
```
