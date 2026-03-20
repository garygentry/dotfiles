# Fix neovim install.sh failing when ~/.config/nvim exists as non-directory

## Context

On a fresh Ubuntu machine, the neovim module fails during `install.sh` with:
```
mkdir: cannot create directory '/home/gary/.config/nvim': File exists
```

The script checks `[[ ! -d "$_nvim_config" ]]` before running `mkdir -p`. This fails when something exists at `~/.config/nvim` that isn't a directory (e.g. a file or broken symlink from a previous dotfiles run or package install). The `-d` test returns false (not a directory), so it tries `mkdir -p` which fails because the path is occupied by a non-directory entry.

## Fix

**File**: `modules/neovim/install.sh:22-30`

Before creating the directory, check if a non-directory entry exists at the path and remove it:

```bash
_nvim_config="${DOTFILES_XDG_CONFIG_HOME}/nvim"
if [[ ! -d "$_nvim_config" ]]; then
    if is_dry_run; then
        log_info "[dry-run] Would create directory: ${_nvim_config}"
    else
        # Remove stale file/symlink if something non-directory exists
        if [[ -e "$_nvim_config" || -L "$_nvim_config" ]]; then
            log_warn "Removing non-directory at ${_nvim_config}"
            rm -f "$_nvim_config"
        fi
        mkdir -p "$_nvim_config"
        log_success "Created ${_nvim_config}"
    fi
fi
```

The `-L` check catches broken symlinks (which `-e` misses).

## Verification

1. Create a file at `~/.config/nvim` to simulate the failure: `touch ~/.config/nvim` (or `ln -s /nonexistent ~/.config/nvim`)
2. Run the dotfiles installer targeting the neovim module
3. Confirm it logs the warning, removes the stale entry, creates the directory, and succeeds
