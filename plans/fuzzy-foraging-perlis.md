# Idempotence Enhancement Plan
## Making Dotfiles Bulletproof for Re-runs and Updates

**Date:** 2026-02-10
**Status:** Ready for Implementation
**Priority:** HIGH - Core reliability feature

---

## Context

### Problem Statement

The current dotfiles system is **not idempotent**. Running `dotfiles install` twice causes unnecessary work and potential data loss:

- **Always re-runs modules** even if already installed and up-to-date
- **Always overwrites files** without checking if they're already correct
- **Silently overwrites user modifications** with no backup
- **No version/content change detection** - can't tell if module needs updating
- State is written AFTER execution, never checked BEFORE

### User Need

> "Important feature to be able to run on any state of previous run, get latest from repo, and update the older install with the latest and not trash or break anything on the current."

**Requirements:**
1. Safe to run on any previous installation state
2. Pull latest repo changes and update only what changed
3. Never break or trash existing installations
4. Detect and skip modules/files already in correct state
5. Protect user modifications (backup before overwrite)
6. Efficient (don't redo unnecessary work)

### Current State Investigation

**What works well:**
- Module scripts are already idempotent (use `command -v` checks, `pkg_installed` helpers)
- Rollback system tracks operations
- Package managers naturally idempotent (apt, brew, pacman)

**Critical gaps:**
- **Module-level**: Install command never checks `state.Get()` before running modules
- **File-level**: `deployFiles()` always overwrites (symlinks removed/recreated, copies truncated with `O_TRUNC`)
- **No content comparison**: Files deployed even if already correct
- **No backups**: User modifications silently lost
- **Checksum field exists but unused**: `ModuleState.Checksum` is defined but never populated

---

## Solution Design

### High-Level Approach

Transform the system from **"always re-run everything"** to **"smart change detection with skip-when-correct"**:

1. **Module-level idempotence**: Check state before running, skip if up-to-date
2. **File-level idempotence**: Check content hashes, skip if unchanged
3. **User modification protection**: Backup before overwriting user changes
4. **Version/content tracking**: Detect module updates, re-run only when needed
5. **Flexible control**: CLI flags for force, skip-failed, update-only modes

---

## Implementation Plan

### Phase 1: State Schema Extensions (Week 1)

**Goal:** Add fields for change detection without breaking existing installations

**Files to modify:**
- `internal/state/state.go`

**Changes:**

```go
type ModuleState struct {
    Name        string      `json:"name"`
    Version     string      `json:"version"`
    Status      string      `json:"status"`
    InstalledAt time.Time   `json:"installed_at"`
    UpdatedAt   time.Time   `json:"updated_at"`
    OS          string      `json:"os"`
    Error       string      `json:"error,omitempty"`

    // Enhanced change detection
    Checksum    string      `json:"checksum"`              // SHA256 of module.yml + scripts
    ConfigHash  string      `json:"config_hash"`           // Hash of user config for module
    FileStates  []FileState `json:"file_states,omitempty"` // Per-file tracking

    Operations  []Operation `json:"operations,omitempty"`  // Existing rollback metadata
}

// NEW: Per-file deployment tracking
type FileState struct {
    Source       string    `json:"source"`        // Relative path in module dir
    Dest         string    `json:"dest"`          // Absolute destination path
    Type         string    `json:"type"`          // symlink, copy, template
    DeployedAt   time.Time `json:"deployed_at"`   // When deployed
    SourceHash   string    `json:"source_hash"`   // SHA256 of source at deploy time
    DeployedHash string    `json:"deployed_hash"` // SHA256 of deployed content
    UserModified bool      `json:"user_modified"` // True if user changed dest
    LastChecked  time.Time `json:"last_checked"`  // Last verification time
}
```

**Why:**
- `Checksum`: Detect when module definition/scripts change → triggers re-run
- `ConfigHash`: Detect when user's config.yml values change → triggers re-run
- `FileStates`: Per-file tracking enables granular skip/deploy decisions
- `SourceHash`: Detect when source file in repo changes
- `DeployedHash`: Reference for detecting user modifications
- All fields `omitempty` for backward compatibility

**Migration Function:**
```go
func migrateOldState(existing *ModuleState, mod *Module, cfg *RunConfig) {
    if len(existing.FileStates) == 0 && existing.Status == "installed" {
        // Retroactively build FileStates by scanning deployed files
        // Compute checksums for the first time
        // Save migrated state
    }
}
```

---

### Phase 2: Hash Computation Utilities (Week 1)

**Goal:** Create reliable change detection system

**New file:** `internal/module/hash.go`

**Functions:**

```go
// computeModuleChecksum: SHA256 of module.yml + install.sh + verify.sh + os/*.sh
// Changes to any definition or script will change this hash
func computeModuleChecksum(mod *Module) (string, error)

// computeConfigHash: SHA256 of user.* + modules[mod.Name] from config.yml
// Changes to config values for this module will change this hash
func computeConfigHash(mod *Module, cfg *config.Config) string

// computeFileHash: SHA256 of file content
// Standard file hashing for content comparison
func computeFileHash(path string) (string, error)
```

**Why:**
- SHA256 provides reliable change detection
- Module checksum detects definition changes
- Config hash detects user setting changes
- File hashes enable idempotent deployment

**Integration:** Populate `ModuleState.Checksum` and `ModuleState.ConfigHash` during state recording

---

### Phase 3: Module-Level Idempotence (Week 2)

**Goal:** Skip modules that don't need re-running

**File to modify:** `internal/module/runner.go`

**Decision Algorithm:**

```go
type ExecutionDecision int
const (
    SKIP          ExecutionDecision = iota  // Already up-to-date
    INSTALL_FRESH                            // No previous state
    INSTALL_RETRY                            // Failed previously
    UPDATE_MODULE                            // Module changed
    UPDATE_CONFIG                            // Config changed
    INSTALL_FORCE                            // --force flag
)

func shouldRunModule(mod *Module, existingState *state.ModuleState, cfg *RunConfig) (ExecutionDecision, string) {
    // No state = fresh install
    if existingState == nil {
        return INSTALL_FRESH, "no previous installation"
    }

    // Force flag = always run
    if cfg.Force {
        return INSTALL_FORCE, "--force flag set"
    }

    // Failed previously = retry (unless --skip-failed)
    if existingState.Status == "failed" {
        if cfg.SkipFailed {
            return SKIP, "failed previously, --skip-failed set"
        }
        return INSTALL_RETRY, "retrying failed installation"
    }

    // Check module checksum (scripts/definition changed?)
    currentChecksum, _ := computeModuleChecksum(mod)
    if currentChecksum != existingState.Checksum {
        return UPDATE_MODULE, "module definition/scripts changed"
    }

    // Check config hash (user settings changed?)
    currentConfigHash := computeConfigHash(mod, cfg.Config)
    if currentConfigHash != existingState.ConfigHash {
        return UPDATE_CONFIG, "user config values changed"
    }

    // Check version
    if mod.Version != existingState.Version {
        return UPDATE_MODULE, "module version changed"
    }

    // Everything matches = skip
    return SKIP, "already installed and up-to-date"
}
```

**Integration in `runModule()`:**

```go
func runModule(cfg *RunConfig, mod *Module) RunResult {
    start := time.Now()

    // NEW: Check if module needs running
    existingState, _ := cfg.State.Get(mod.Name)
    decision, reason := shouldRunModule(mod, existingState, cfg)

    if decision == SKIP {
        cfg.UI.Info(fmt.Sprintf("✓ %s (skipped: %s)", mod.Name, reason))
        return RunResult{Module: mod, Success: true, Skipped: true, Duration: time.Since(start)}
    }

    cfg.UI.Info(fmt.Sprintf("Installing %s (%s)...", mod.Name, reason))

    // ... rest of existing runModule logic ...
}
```

**Why:**
- Reduces unnecessary work (scripts, file deployments)
- Clear feedback about why modules run or skip
- Respects force/skip-failed flags
- Detects all types of changes (version, definition, config)

---

### Phase 4: File-Level Idempotence (Week 2-3)

**Goal:** Only deploy files that need deployment

**File to modify:** `internal/module/runner.go` - function `deployFiles()`

**Smart Deployment Algorithm:**

```go
func shouldDeployFile(fileEntry FileEntry, src, dest, sourceHash string,
                      existing *state.FileState, cfg *RunConfig) (bool, string) {

    if cfg.Force {
        return true, "force flag set"
    }

    if existing == nil {
        return true, "not previously deployed"
    }

    // Source content changed = must redeploy
    if sourceHash != existing.SourceHash {
        return true, "source file changed"
    }

    // Check if destination exists
    if _, err := os.Lstat(dest); os.IsNotExist(err) {
        return true, "destination file missing"
    }

    // For symlinks: check if pointing to correct location
    if fileEntry.Type == "symlink" {
        target, _ := os.Readlink(dest)
        expectedTarget, _ := filepath.Abs(src)
        if target != expectedTarget {
            return true, "symlink points to wrong location"
        }
        return false, "symlink already correct"  // SKIP
    }

    // For copy/template: check file content hash
    currentHash, _ := computeFileHash(dest)

    // Destination matches our deployed content = unchanged
    if currentHash == existing.DeployedHash {
        return false, "destination unchanged since deployment"  // SKIP
    }

    // Content changed but source didn't = user modified
    // Don't redeploy (protect user changes)
    return false, "user modified (source unchanged)"  // SKIP
}
```

**Modified `deployFiles()` function:**

```go
func deployFiles(cfg *RunConfig, mod *Module, tmplCtx *template.Context,
                 modState *state.ModuleState, existingState *state.ModuleState) error {

    // Build map of previously deployed files
    existingFiles := make(map[string]*state.FileState)
    if existingState != nil {
        for i := range existingState.FileStates {
            fs := &existingState.FileStates[i]
            existingFiles[fs.Dest] = fs
        }
    }

    for _, f := range mod.Files {
        src := filepath.Join(mod.Dir, f.Source)
        dest := expandHome(f.Dest, cfg.SysInfo.HomeDir)

        // Compute source hash
        sourceHash, err := computeFileHash(src)
        if err != nil {
            return fmt.Errorf("computing hash for %s: %w", src, err)
        }

        // Check if deployment needed
        existingFile := existingFiles[dest]
        needsDeploy, reason := shouldDeployFile(f, src, dest, sourceHash, existingFile, cfg)

        if !needsDeploy {
            cfg.UI.Debug(fmt.Sprintf("Skipping %s: %s", dest, reason))

            // Carry forward existing state
            modState.FileStates = append(modState.FileStates, state.FileState{
                Source:       f.Source,
                Dest:         dest,
                Type:         f.Type,
                DeployedAt:   existingFile.DeployedAt,
                SourceHash:   sourceHash,
                DeployedHash: existingFile.DeployedHash,
                UserModified: existingFile.UserModified,
                LastChecked:  time.Now(),
            })
            continue
        }

        cfg.UI.Debug(fmt.Sprintf("Deploying %s -> %s: %s", f.Source, dest, reason))

        // Handle user modifications (backup if changed)
        if existingFile != nil && existingFile.UserModified && !cfg.Force {
            if err := createBackup(dest, cfg); err != nil {
                cfg.UI.Warn(fmt.Sprintf("Backup failed for %s: %v", dest, err))
            }
        }

        // Deploy the file (existing deploySymlink/deployCopy/template logic)
        deployedHash, err := deployFileActual(cfg, mod, f, src, dest, tmplCtx)
        if err != nil {
            return err
        }

        // Record in state
        modState.FileStates = append(modState.FileStates, state.FileState{
            Source:       f.Source,
            Dest:         dest,
            Type:         f.Type,
            DeployedAt:   time.Now(),
            SourceHash:   sourceHash,
            DeployedHash: deployedHash,
            UserModified: false,
            LastChecked:  time.Now(),
        })

        // Record operation for rollback
        modState.RecordOperation(state.Operation{
            Type:   "file_deploy",
            Action: determineAction(existingFile),
            Path:   dest,
            Metadata: map[string]string{
                "source":      src,
                "type":        f.Type,
                "source_hash": sourceHash,
            },
        })
    }

    return nil
}
```

**Why:**
- Skips files that are already correct (symlink pointing to right target, copy/template with unchanged content)
- Protects user modifications when source hasn't changed
- Only redeploys when necessary (source changed, dest missing, dest incorrect)
- Provides clear feedback about why each file was deployed or skipped

---

### Phase 5: Backup System (Week 3)

**Goal:** Protect user modifications from being lost

**New file:** `internal/module/backup.go`

**Backup Strategy:**

```go
// createBackup backs up a file before overwriting it
func createBackup(filePath string, cfg *RunConfig) error {
    if cfg.DryRun {
        cfg.UI.Debug(fmt.Sprintf("[dry-run] Would backup: %s", filePath))
        return nil
    }

    // Create backup with timestamp
    backupRoot := filepath.Join(cfg.SysInfo.DotfilesDir, ".backups")
    timestamp := time.Now().Format("20060102-150405")

    // Preserve directory structure: .backups/<timestamp>/<relative-path>
    relPath, _ := filepath.Rel(cfg.SysInfo.HomeDir, filePath)
    backupPath := filepath.Join(backupRoot, timestamp, relPath)

    // Copy file to backup location
    if err := copyFile(filePath, backupPath); err != nil {
        return fmt.Errorf("copying to backup: %w", err)
    }

    // Write metadata alongside backup
    meta := BackupMetadata{
        OriginalPath: filePath,
        BackupTime:   time.Now(),
        ContentHash:  computeFileHash(filePath),
        Reason:       "user-modified file overwritten by module update",
    }
    saveBackupMetadata(backupPath+".meta.json", meta)

    cfg.UI.Warn(fmt.Sprintf("Backed up user-modified file: %s", filePath))
    return nil
}

type BackupMetadata struct {
    OriginalPath string    `json:"original_path"`
    BackupTime   time.Time `json:"backup_time"`
    ContentHash  string    `json:"content_hash"`
    Reason       string    `json:"reason"`
}
```

**Backup Location:** `~/.dotfiles/.backups/<timestamp>/<relative-path>`

**Why:**
- Timestamped backups prevent collisions
- Preserves directory structure for easy restoration
- Metadata tracks original location and reason
- User can manually restore if needed
- Future: Add `dotfiles restore` command to automate restoration

---

### Phase 6: CLI Flags (Week 3)

**Goal:** Give users control over execution behavior

**File to modify:** `cmd/dotfiles/install.go`

**New Flags:**

```go
var (
    unattended bool
    failFast   bool
    force      bool   // NEW: Force reinstall even if up-to-date
    skipFailed bool   // NEW: Skip modules that failed previously
    updateOnly bool   // NEW: Only update existing, don't install new
    checkOnly  bool   // NEW: Check status without making changes
)

installCmd.Flags().BoolVar(&force, "force", false, "Force reinstall all modules")
installCmd.Flags().BoolVar(&skipFailed, "skip-failed", false, "Skip modules that failed previously")
installCmd.Flags().BoolVar(&updateOnly, "update-only", false, "Only update existing modules")
installCmd.Flags().BoolVar(&checkOnly, "check", false, "Check status without changes")
```

**Flag Semantics:**
- `--force`: Skip all idempotence checks, always reinstall everything
- `--skip-failed`: Don't retry previously failed modules (useful for clean runs)
- `--update-only`: Only process modules with existing state (ignore new modules in repo)
- `--check`: Report what would be done without actually doing it (stronger than --dry-run)

**Integration:** Pass flags to `RunConfig` and use in decision algorithms

---

### Phase 7: User Experience Enhancements (Week 4)

**Goal:** Clear feedback about what's happening and why

**Enhanced Output:**

```
Installing modules...
  ✓ git (skipped: already up-to-date)
  → zsh (updating: source files changed)
    • Deploying 2 of 3 files (1 unchanged)
    ⚠ Backed up ~/.zshrc (user modified)
  ✓ zsh (updated in 2.3s)
  → neovim (installing: new module)
  ✓ neovim (installed in 8.1s)

Summary:
  2 skipped, 1 updated, 1 installed
  1 backup created in ~/.dotfiles/.backups/20260210-143022/
  Total time: 10.4s
```

**Status Command Enhancement:**

```bash
$ dotfiles status

Installed modules:
  git      v1.0.0  ✓ up-to-date
  zsh      v1.2.0  • needs update (source changed)
  neovim   v2.1.0  ! failed (2026-02-10)
  tmux     v1.0.0  ⚠ user modified (2 files)

4 modules installed (1 failed, 1 needs update, 1 user modified)

Run 'dotfiles install' to update out-of-date modules
Run 'dotfiles install --force neovim' to retry failed module
```

---

## Critical Files Summary

### Files to Modify:
1. **internal/state/state.go** - Add `FileState`, `ConfigHash`, extend `ModuleState`
2. **internal/module/runner.go** - Add `shouldRunModule()`, modify `runModule()`, enhance `deployFiles()`
3. **cmd/dotfiles/install.go** - Add CLI flags (force, skip-failed, update-only, check)
4. **cmd/dotfiles/status.go** - Enhance output to show update status

### Files to Create:
1. **internal/module/hash.go** - Hash computation utilities
2. **internal/module/backup.go** - Backup system
3. **internal/module/hash_test.go** - Hash tests
4. **internal/module/backup_test.go** - Backup tests

---

## Testing Strategy

### Unit Tests:
- `shouldRunModule()` decision tree (all branches)
- `shouldDeployFile()` decision tree (all branches)
- Hash computation (deterministic, detects changes)
- Backup creation (preserves content, correct metadata)

### Integration Tests:
- Fresh install → state written correctly
- Re-run with no changes → everything skipped
- Update module.yml → module re-runs
- Modify user config → module re-runs with new values
- User modifies deployed file → backup created on next run
- `--force` → everything reinstalls
- `--skip-failed` → failed modules skipped

### Test Scenarios:
```bash
# Scenario 1: Fresh install
dotfiles install git
# Verify: State written with checksums, files deployed

# Scenario 2: Re-run (idempotent)
dotfiles install git
# Verify: "skipped: already up-to-date", no file operations

# Scenario 3: Update module
echo "# comment" >> modules/git/install.sh
dotfiles install git
# Verify: "updating: module definition changed", re-runs scripts

# Scenario 4: User modification
echo "custom" >> ~/.gitconfig
dotfiles install git
# Verify: "skipped: user modified (source unchanged)"

# Scenario 5: Both changed
echo "# comment" >> modules/git/install.sh
echo "custom" >> ~/.gitconfig
dotfiles install git
# Verify: Backup created, file deployed with warning

# Scenario 6: Force reinstall
dotfiles install git --force
# Verify: Always runs even if up-to-date
```

---

## Rollout Plan

### Phase 1-2 (Week 1): Foundation
- Extend state schema with backward compatibility
- Add hash computation utilities
- Deploy and test with existing installations
- **Validation:** No breaking changes, state files upgraded gracefully

### Phase 3-4 (Week 2): Core Idempotence
- Module-level skip logic
- File-level skip logic
- **Validation:** Modules skip when appropriate, files only deploy when needed

### Phase 5-6 (Week 3): Safety & Control
- Backup system
- CLI flags
- **Validation:** User modifications protected, flags work correctly

### Phase 7 (Week 4): Polish
- Enhanced UX feedback
- Status command improvements
- Documentation updates
- **Validation:** Clear feedback, intuitive behavior

---

## Success Criteria

### Functional Requirements:
✅ Running `dotfiles install` twice does nothing the second time (idempotent)
✅ Pull latest changes + install only updates what changed
✅ User modifications preserved (backed up before overwrite)
✅ Clear feedback about what's happening and why
✅ `--force` flag overrides all skip logic
✅ Failed modules can be skipped or retried

### Performance Requirements:
✅ Skipped modules take <100ms (state check only)
✅ Skipped files take <10ms per file (hash comparison)
✅ Hash computation cached within single run

### Reliability Requirements:
✅ Backward compatible with existing state files
✅ Migration path for old installations
✅ No data loss on upgrades
✅ All existing tests still pass

---

## Verification Commands

```bash
# Build and test
make build
go test ./internal/state ./internal/module

# Test idempotence
./bin/dotfiles install git
./bin/dotfiles install git  # Should skip

# Test update detection
echo "# comment" >> modules/git/install.sh
./bin/dotfiles install git  # Should re-run

# Test user modification protection
echo "custom" >> ~/.gitconfig
./bin/dotfiles install git  # Should skip with message

# Test force flag
./bin/dotfiles install git --force  # Should always run

# Test status command
./bin/dotfiles status  # Should show update status
```

---

## Edge Cases

### Handled:
- User deleted deployed file → Redeploy
- User modified symlink → Recreate (with warning)
- Module version downgrade → Treat as update
- Multiple modules deploy to same dest → Error with clear message
- Template variables changed → Re-render
- Failed module with partial state → Offer rollback or skip
- Concurrent installs → Document as unsupported (low risk)

### Known Limitations:
- Scripts always run on update (rely on script idempotence)
- OS-specific scripts run on fresh install only (assume idempotent)
- Package installs informational in rollback (manual removal needed)

---

## Next Steps

After plan approval:
1. Create feature branch: `git checkout -b feature/idempotence`
2. Implement Phase 1 (state schema)
3. Test backward compatibility
4. Implement Phases 2-4 (core idempotence)
5. Test against existing modules
6. Implement Phases 5-7 (safety & UX)
7. Update documentation
8. Create PR for review

---

**Estimated Total Effort:** 4 weeks (160-200 hours)
**Risk Level:** Medium (requires careful state migration)
**Value:** HIGH - Transforms system from "run once" to "run anytime"
