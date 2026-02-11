# Idempotence Quick Reference

## TL;DR

✅ **Safe to run `dotfiles install` multiple times** - only updates what changed

## Common Commands

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

## Status Symbols

| Symbol | Meaning |
|--------|---------|
| `✓` | Up-to-date, nothing to do |
| `•` | Needs update (version/changed/config) |
| `⚠` | User modified files |
| `!` | Failed previously |

## Why Things Run or Skip

### Module runs when:
- First time installing
- Module version changed
- Module scripts changed (`install.sh`, `verify.sh`, `os/*.sh`)
- User config changed (`config.yml` values for this module)
- Previously failed (and not using `--skip-failed`)
- `--force` flag used

### Module skips when:
- Already installed
- No changes detected
- Failed previously + `--skip-failed` flag
- New module + `--update-only` flag

### File deploys when:
- Source file changed
- Destination missing
- Symlink points to wrong location

### File skips when:
- Already correct (hash matches)
- User modified (source unchanged)

## Backups

**Location:** `~/.dotfiles/.backups/<timestamp>/`

**When created:** Before overwriting user-modified files

**Restore manually:**
```bash
# Find backup
ls -lt ~/.dotfiles/.backups/

# Copy back
cp ~/.dotfiles/.backups/20260211-143022/.zshrc ~/.zshrc
```

## Troubleshooting

| Problem | Solution |
|---------|----------|
| Always re-runs | Old state, run once to record checksums |
| Want to reinstall | Use `--force` flag |
| Skip failed module | Use `--skip-failed` flag |
| Only update existing | Use `--update-only` flag |

## Examples

```bash
# Daily workflow
cd ~/dotfiles && git pull && dotfiles install

# Test module changes
vim modules/zsh/install.sh
dotfiles status          # Shows "• changed"
dotfiles install         # Only zsh runs

# Retry failed module
dotfiles install broken-module --force

# Skip known broken module
dotfiles install --skip-failed
```

## Performance

- Skipped modules: <100ms
- Skipped files: <10ms each
- Hash computation: Cached

## Full Documentation

See [IDEMPOTENCE.md](IDEMPOTENCE.md) for complete details.
