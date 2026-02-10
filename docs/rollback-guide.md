# Rollback and Uninstall Guide

This guide explains how the dotfiles system tracks operations and enables safe uninstallation and rollback of modules.

## Overview

Starting with version 1.1.0, the dotfiles system records all operations performed during module installation. This operation history enables:

- **Uninstalling modules** and reversing their changes
- **Automatic rollback** when installation fails mid-way
- **Audit trail** of what files and directories were modified

## Operation Recording

During installation, the system tracks:

### File Deployments
- **Created**: New files or symlinks
- **Modified**: Existing files that were changed
- **Backed up**: Original files that were backed up before modification

### Directory Creation
- **Created**: New directories that were made
- Includes empty directory tracking for safe cleanup

### Script Execution
- **Executed**: Install scripts that ran
- Note: Scripts cannot be automatically rolled back

### Package Installation
- **Installed**: Packages added via system package manager
- Note: Packages are not automatically removed during rollback

## Viewing Rollback Plan

Before uninstalling, you can see what operations will be reversed:

```bash
# Show rollback plan for a module
dotfiles uninstall git --dry-run
```

Output example:
```
Rollback plan (5 operations):
  1. Remove: /home/user/.gitconfig
  2. Restore /home/user/.bashrc from /home/user/.bashrc.backup
  3. Remove directory: /home/user/.config/git
  4. Package was installed: git (manual removal may be needed)
  5. Script was executed: install.sh (manual cleanup may be needed)
```

## Uninstalling Modules

### Basic Uninstall

```bash
dotfiles uninstall <module>
```

The command will:
1. Show the rollback plan
2. Ask for confirmation
3. Execute rollback operations in reverse order
4. Remove the module from state

### Force Uninstall

Skip confirmation prompts:

```bash
dotfiles uninstall <module> --force
```

Use this when:
- Running in scripts or CI/CD
- You're confident about the uninstall
- Continuing despite errors

### Dry-run Mode

Preview what would be uninstalled without making changes:

```bash
dotfiles uninstall <module> --dry-run
```

### Multiple Modules

Uninstall several modules at once:

```bash
dotfiles uninstall git zsh tmux
```

## Rollback Operations

### File Operations

**Created files/symlinks**: Removed completely
```
Remove: /home/user/.gitconfig
```

**Modified files**: Restored from backup (if available)
```
Restore /home/user/.bashrc from /home/user/.bashrc.backup
```

**No backup**: Warning shown, file left as-is
```
File was modified: /home/user/.zshrc (no backup available)
```

### Directory Operations

**Created directories**: Removed if empty
```
Remove directory: /home/user/.config/module
```

**Non-empty directories**: Left in place with warning
```
Directory not empty, keeping: /home/user/.config (3 files)
```

### Script Operations

Scripts cannot be automatically rolled back:
```
Script was executed: install.sh (manual cleanup may be needed)
```

You may need to manually:
- Remove configuration added by scripts
- Undo system-level changes
- Clean up artifacts

### Package Operations

Packages are not automatically removed:
```
Package was installed: git (manual removal may be needed)
```

To remove packages manually:
```bash
# macOS
brew uninstall <package>

# Ubuntu/Debian
sudo apt remove <package>

# Arch Linux
sudo pacman -R <package>
```

## Automatic Rollback on Failure

If a module installation fails mid-way, you'll see an interactive prompt:

```
[ERROR] Failed to install module: script execution failed

Options:
  [S]kip - Leave partial installation as-is
  [U]ndo - Rollback changes and clean up

What would you like to do?
```

### Skip Option

Leaves the partial installation:
- Files that were deployed remain
- State is marked as "failed"
- You can retry installation later
- Useful for debugging

### Undo Option

Rolls back all operations:
- Removes deployed files
- Restores backed-up files
- Removes created directories (if empty)
- Cleans up module state
- Returns system to pre-installation state

### Unattended Mode

In `--unattended` mode:
- No prompt is shown
- Failed state is recorded
- Partial installation is left as-is
- Use `dotfiles uninstall` to clean up later

## State Management

### Viewing Module State

```bash
# Show all installed modules
dotfiles status

# List available modules
dotfiles list
```

### State Location

Module state is stored at:
```
$DOTFILES_DIR/.state/<module>.json
```

Each state file contains:
- Module name and version
- Installation timestamp
- Installation status
- Operating system
- Complete operation history

### Manual State Inspection

```bash
# View state for a specific module
cat $DOTFILES_DIR/.state/git.json | jq .
```

Example state file:
```json
{
  "name": "git",
  "version": "1.0.0",
  "status": "installed",
  "installed_at": "2026-02-10T12:00:00Z",
  "updated_at": "2026-02-10T12:00:05Z",
  "os": "ubuntu",
  "operations": [
    {
      "type": "file_deploy",
      "action": "created",
      "path": "/home/user/.gitconfig",
      "timestamp": "2026-02-10T12:00:01Z",
      "metadata": {
        "source": "/path/to/dotfiles/modules/git/gitconfig",
        "type": "symlink"
      }
    }
  ]
}
```

## Best Practices

### Before Uninstalling

1. **Check dependencies**: Verify no other modules depend on this one
2. **Review rollback plan**: Use `--dry-run` to see what will change
3. **Backup important files**: If you've customized module files
4. **Check for manual changes**: Review files that can't be auto-removed

### After Uninstalling

1. **Verify removal**: Check that files were actually removed
2. **Clean up packages**: Manually remove packages if desired
3. **Remove custom configs**: Check for files you added after installation

### Module Development

When creating modules:

1. **Use file deployments**: Prefer declarative file deployments over scripts
2. **Minimize script logic**: Keep install scripts simple
3. **Document manual steps**: Note any manual cleanup needed
4. **Test rollback**: Verify uninstall works correctly

## Troubleshooting

### Uninstall fails with errors

```bash
# Continue uninstalling despite errors
dotfiles uninstall <module> --force
```

### State file exists but module "not installed"

```bash
# Check state file
cat $DOTFILES_DIR/.state/<module>.json

# Manually remove state
rm $DOTFILES_DIR/.state/<module>.json
```

### Files not removed during uninstall

Possible causes:
- Files were modified after installation
- Files are owned by root
- Permissions prevent deletion

Manual cleanup:
```bash
# Check file ownership
ls -la <path>

# Remove with sudo if needed
sudo rm <path>
```

### Directory not removed

Directories are only removed if empty:
```bash
# Check directory contents
ls -la <directory>

# Remove manually if desired
rm -rf <directory>
```

### No operations recorded

Modules installed before operation recording was implemented have empty operation lists. To uninstall:

1. Review module files manually
2. Use `--force` to remove state anyway
3. Manually clean up deployed files

## Advanced Topics

### Partial Rollback

Currently not supported. You can:
- Manually remove specific files
- Edit state JSON to remove specific operations
- Reinstall the module

### Rollback Hooks

Future enhancement. Currently:
- No pre/post rollback hooks
- No custom rollback scripts
- Use module install scripts for setup; uninstall for teardown

### Cross-Machine Rollback

Operation paths are absolute and machine-specific. When syncing across machines:
- State files are not portable
- Reinstall modules on each machine
- Use profiles for machine-specific configuration

## Related Documentation

- [Architecture Guide](architecture.md) - System design and operation recording
- [Module Development](module-development.md) - Creating rollback-friendly modules
- [CLI Reference](cli.md) - Full uninstall command documentation
- [Troubleshooting](troubleshooting.md) - Common issues and solutions
