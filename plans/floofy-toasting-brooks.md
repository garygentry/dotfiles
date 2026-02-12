# Full Unattended Mode Support for IaC/CI-CD

## Context

The dotfiles system needs to support **fully automated, zero-prompt installation** for Infrastructure as Code (IaC) scenarios like Terraform, Ansible, Docker, and CI/CD pipelines.

**Current State**: The `--unattended` flag mostly works for the `install` command, but has critical gaps:

1. **`uninstall` command blocks in unattended mode** - Has 2 confirmation prompts that only check `--force`, not `--unattended`
2. **`--unattended` is not global** - Defined only in install.go, not available to other commands
3. **Documentation gap** - No guide for IaC/CI-CD usage, bootstrap command doesn't mention `--unattended`

**Goal**: Make the entire system work seamlessly in automated environments with zero user interaction.

## Approach

1. **Make `--unattended` a global flag** like `--verbose` (available to all commands)
2. **Fix `uninstall` command** to respect `--unattended` flag
3. **Create comprehensive IaC/CI-CD documentation** with Terraform, Docker, Ansible examples
4. **Update README** with unattended mode examples

This ensures any command can run without prompts when `--unattended` is set.

## Implementation Steps

### 1. Make `--unattended` a Global Flag

**File**: `cmd/dotfiles/root.go`

Add `unattended` variable alongside existing global flags (after line 7):
```go
var (
    verbose    bool
    dryRun     bool
    logJSON    bool
    unattended bool  // NEW
)
```

Register as persistent flag in `init()` (after line 24):
```go
rootCmd.PersistentFlags().BoolVar(&unattended, "unattended", false, "Run without prompts, using defaults")
```

### 2. Update `install` Command to Use Global Flag

**File**: `cmd/dotfiles/install.go`

**Remove** local `unattended` variable:
- Line 20: Delete `unattended bool` from var block
- Line 264: Delete flag registration in `init()`

**Keep** auto-detection logic (lines 46-50) - it will now set the global variable:
```go
if !sys.IsInteractive && !unattended {
    u.Info("Non-interactive stdin detected, using default values for prompts")
    unattended = true  // Sets global variable
}
```

### 3. Fix `uninstall` Command

**File**: `cmd/dotfiles/uninstall.go`

**Change 1** - Line 78 (no rollback operations confirmation):
```go
// BEFORE
if !dryRun && !uninstallForce {
    confirm, err := u.PromptConfirm("Remove module state anyway?", false)
    if err != nil || !confirm {
        return fmt.Errorf("uninstall cancelled")
    }
}

// AFTER - Add unattended check
if !dryRun && !uninstallForce && !unattended {
    confirm, err := u.PromptConfirm("Remove module state anyway?", false)
    if err != nil || !confirm {
        return fmt.Errorf("uninstall cancelled")
    }
}
// In unattended mode, automatically proceed with state removal
```

**Change 2** - Line 108 (proceed with uninstall confirmation):
```go
// BEFORE
if !uninstallForce {
    confirm, err := u.PromptConfirm(fmt.Sprintf("Proceed with uninstall of %s?", moduleName), false)
    if err != nil || !confirm {
        return fmt.Errorf("uninstall cancelled")
    }
}

// AFTER - Add unattended check
if !uninstallForce && !unattended {
    confirm, err := u.PromptConfirm(fmt.Sprintf("Proceed with uninstall of %s?", moduleName), false)
    if err != nil || !confirm {
        return fmt.Errorf("uninstall cancelled")
    }
}
// In unattended mode, automatically proceed
```

### 4. Update README.md

**File**: `README.md`

**Add section** after line 28 (after Quick Start installation):
```markdown
### Automated/Unattended Installation

For IaC, CI/CD, or automated server provisioning:

\`\`\`bash
# Download and run in unattended mode
curl -sfL https://raw.githubusercontent.com/garygentry/dotfiles/main/bootstrap.sh | bash -s -- --unattended

# With specific profile
curl -sfL https://raw.githubusercontent.com/garygentry/dotfiles/main/bootstrap.sh | bash -s -- --unattended --profile minimal
\`\`\`

This installs all modules using defaults with zero interactive prompts. [CI/CD Guide →](docs/ci-cd-guide.md)
```

**Update Usage section** (around line 68):
```markdown
# Run without prompts (works with all commands)
dotfiles install --unattended
dotfiles uninstall git --unattended
```

### 5. Create CI/CD Guide

**File**: `docs/ci-cd-guide.md` (NEW FILE)

Create comprehensive guide covering:

**Sections**:
1. **Overview** - How `--unattended` works
2. **Quick Start** - Basic unattended installation
3. **Use Cases**:
   - Terraform / CloudFormation (EC2 user data)
   - Docker images (Dockerfile example)
   - Ansible playbooks
   - GitHub Actions
   - Packer templates
4. **Configuration** - Pre-creating config.yml, environment variables
5. **Handling Failures** - `--skip-failed`, `--fail-fast`, `--dry-run`
6. **Secrets Management** - Pre-auth 1Password, skip secrets modules
7. **Verification** - Exit codes, status checks
8. **Best Practices** - Profiles, testing, logging
9. **Troubleshooting** - Common CI/CD issues
10. **Examples** - Complete working code snippets

See plan agent output for full content template.

### 6. Update CLI Reference

**File**: `docs/cli-reference.md`

**Add to Global Flags section**:
```markdown
- `--unattended`: Run without interactive prompts, using defaults (ideal for CI/CD and IaC)
```

**Update `dotfiles uninstall` section**:
```markdown
**Flags:**
- `--force`: Skip confirmation prompts and continue on errors
- `--unattended`: Skip confirmation prompts (NEW)

**Examples:**
\`\`\`bash
# Unattended mode (auto-confirm)
dotfiles uninstall git --unattended
\`\`\`
```

### 7. Update Troubleshooting Guide

**File**: `docs/troubleshooting.md`

**Add section** at end:
```markdown
## Unattended Mode / CI-CD

### Installation hangs in CI/CD

**Problem**: Installation blocks waiting for input

**Solution**:
\`\`\`bash
dotfiles install --unattended
\`\`\`

### Modules fail in Docker/CI

**Problem**: Some modules require interactive terminal

**Solution**:
\`\`\`bash
dotfiles install --unattended --skip-failed
\`\`\`

### Secrets not available in CI

**Problem**: 1Password prompts block

**Solution**: Unattended mode auto-skips secrets authentication. Use a profile without secrets-dependent modules.
```

## Files Modified

| File | Change |
|------|--------|
| `cmd/dotfiles/root.go` | Add global `unattended` flag (2 lines added) |
| `cmd/dotfiles/install.go` | Remove local flag, use global (2 lines removed) |
| `cmd/dotfiles/uninstall.go` | Add `!unattended` checks to 2 prompts (lines 78, 108) |
| `README.md` | Add unattended installation section, update usage |
| `docs/cli-reference.md` | Document global `--unattended`, update uninstall flags |
| `docs/troubleshooting.md` | Add CI/CD troubleshooting section |
| `docs/ci-cd-guide.md` | NEW - Comprehensive IaC/CI-CD guide with examples |

## Verification

### Manual Testing

```bash
# Test install (should work already)
dotfiles install --unattended --dry-run

# Test uninstall (should not prompt after fix)
dotfiles install git --unattended
dotfiles uninstall git --unattended --dry-run
dotfiles uninstall git --unattended

# Test bootstrap with unattended
curl -sfL https://raw.githubusercontent.com/.../bootstrap.sh | bash -s -- --unattended --dry-run

# Test that prompts still work in interactive mode
dotfiles install git
dotfiles uninstall git
```

### Automated Test Script

Create and run test:
```bash
#!/bin/bash
set -euo pipefail

echo "Testing unattended mode..."

# Simulate CI environment
export CI=true

# Test bootstrap
if curl -sfL https://raw.githubusercontent.com/.../bootstrap.sh | bash -s -- --unattended --dry-run; then
  echo "✓ Bootstrap works"
else
  echo "✗ Bootstrap failed"
  exit 1
fi

# Test all commands
dotfiles install --unattended --dry-run || exit 1
dotfiles uninstall git --unattended --dry-run || exit 1
dotfiles status --unattended || exit 1

echo "All tests passed!"
```

### Expected Behavior

**With `--unattended`**:
- ✅ No prompts anywhere
- ✅ Uses default values for all module prompts
- ✅ Skips secrets authentication
- ✅ Proceeds with uninstall automatically
- ✅ Works in `curl | bash` scenarios

**Without `--unattended`** (backward compatibility):
- ✅ Interactive prompts still appear
- ✅ User can make choices
- ✅ Secrets auth is attempted
- ✅ Uninstall asks for confirmation

## Critical Files

- `cmd/dotfiles/root.go` - Add global flag here
- `cmd/dotfiles/install.go` - Remove local flag definition (lines 20, 264)
- `cmd/dotfiles/uninstall.go` - Fix prompts (lines 78, 108)
- `README.md` - Update Quick Start with unattended examples
- `docs/ci-cd-guide.md` - New comprehensive IaC guide
