# Plan: Eliminate Prompting for Auto-Included Dependencies

## Context

The dotfiles system currently has a UX issue where users are prompted for configuration options for modules they didn't explicitly select. This happens when a module is auto-included as a dependency of another module.

**Concrete Example:**
- User selects: `fish` (shell) + `starship` (prompt)
- System auto-includes: `zsh` (because starship declares it as a dependency)
- During installation: User gets prompted for zsh framework choice, plugin preset, and theme
- **Problem**: User chose fish, not zsh - these prompts are confusing and waste time

**Root Cause Analysis:**
1. **Incorrect dependency**: `starship` (cross-shell prompt) incorrectly depends only on `zsh`
2. **No context tracking**: System doesn't distinguish between "user explicitly selected this" vs "auto-included as dependency"
3. **Prompting happens during execution**: After dependency resolution, so can't skip prompts earlier

## Investigation Findings

### Current Architecture

**Dependency Resolution Flow:**
1. Module discovery from `modules/` directory
2. User selection (CLI args, profile, or interactive)
3. Dependency resolution using Kahn's algorithm (topological sort)
4. Module execution in dependency order
5. **Prompts shown during execution** (in `handlePrompts()`)

**Key Files:**
- `internal/module/resolver.go` - Dependency resolution with Kahn's algorithm
- `internal/module/runner.go` - Execution and prompting (lines 298-335)
- `internal/module/schema.go` - Module and prompt definitions
- `cmd/dotfiles/install.go` - Orchestration flow

### Dependency Patterns Found

Dependencies across modules:
- `starship → zsh` ⚠️ **PROBLEMATIC** (starship is cross-shell but only declares zsh)
- `zsh → git` ✓ Valid
- `git → ssh` ✓ Valid
- `lazygit → git` ✓ Valid
- `neovim → git` ✓ Valid
- `gh → git` ✓ Valid
- `fzf → ripgrep` ✓ Valid
- `gemini-cli → nodejs` ✓ Valid

### Prompting Behavior

**Current behavior:**
- All prompts shown during module execution, regardless of selection context
- `--unattended` flag uses defaults for all prompts
- No way to skip prompts for specific modules

**Example from zsh module** (`modules/zsh/module.yml:22-45`):
```yaml
prompts:
  - key: zsh_framework
    message: "Zsh plugin framework"
    default: "zinit"
    type: choice
    options: [zinit, ohmyzsh]

  - key: zsh_omz_plugins
    message: "Oh My Zsh plugin preset (ignored if using Zinit)"
    default: "standard"
    type: choice
    options: [minimal, standard, full]
```

These prompts are shown even if zsh was auto-included and the user chose fish as their shell.

## Recommended Solution: Multi-Layered Approach

### Layer 1: Fix Incorrect Dependencies (Immediate)

**What:** Remove zsh dependency from starship module

**Why:** Starship is described as "cross-shell prompt" but declares dependency only on zsh. This is incorrect - starship works with fish, bash, zsh, etc.

**Changes:**
```yaml
# modules/starship/module.yml
name: starship
description: "Install Starship cross-shell prompt (works with any shell)"
version: "1.1.0"  # Bump version
dependencies: []  # REMOVE zsh dependency
```

**Impact:**
- ✅ Fixes the concrete issue immediately
- ✅ Correct modeling of starship's capabilities
- ✅ No code changes needed, just configuration
- ⚠️ Breaking change for users expecting zsh auto-install

**Effort:** 15 minutes

### Layer 2: Track Installation Context (Core Solution)

**What:** Distinguish between explicitly-selected vs auto-included modules, skip prompts for auto-included ones

**Why:** Solves the general problem - any auto-included dependency uses sensible defaults without bothering the user

**Design:**

1. **Extend ExecutionPlan** to track explicit selections:
```go
// internal/module/resolver.go
type ExecutionPlan struct {
    Modules              []*Module
    Skipped              []*Module
    ExplicitlyRequested  map[string]bool  // NEW
}
```

2. **Modify Resolve()** to populate this tracking:
```go
func Resolve(allModules, requested []string, osName string) (*ExecutionPlan, error) {
    // ... existing resolution logic ...

    // NEW: Track which modules were explicitly requested
    explicitMap := make(map[string]bool)
    for _, name := range requested {
        explicitMap[name] = true
    }

    return &ExecutionPlan{
        Modules:             sortedModules,
        Skipped:             skippedModules,
        ExplicitlyRequested: explicitMap,  // NEW
    }, nil
}
```

3. **Pass context to runner**:
```go
// internal/module/runner.go
type RunConfig struct {
    // ... existing fields ...
    ExplicitModules map[string]bool  // NEW
}
```

4. **Skip prompts for auto-included modules**:
```go
// internal/module/runner.go (handlePrompts function)
func handlePrompts(cfg *RunConfig, mod *Module) (map[string]string, error) {
    answers := make(map[string]string, len(mod.Prompts))

    // NEW: Check if module was auto-included
    isAutoIncluded := !cfg.ExplicitModules[mod.Name]

    for _, p := range mod.Prompts {
        // Use defaults for unattended mode OR auto-included modules
        if cfg.Unattended || isAutoIncluded {
            answers[p.Key] = p.Default
            if isAutoIncluded && !cfg.Unattended {
                // Optional: Log that we're using defaults
                cfg.UI.Info(fmt.Sprintf("Using default for %s: %s", p.Message, p.Default))
            }
            continue
        }

        // ... existing interactive prompt logic ...
    }
    return answers, nil
}
```

5. **Wire through install.go**:
```go
// cmd/dotfiles/install.go (after line 146)
plan, err := module.Resolve(allModules, requested, sys.OS)
// ...

// NEW: Create run config with explicit module tracking
runCfg := &module.RunConfig{
    // ... existing fields ...
    ExplicitModules: plan.ExplicitlyRequested,
}
```

**Benefits:**
- ✅ Solves general problem for all modules
- ✅ Backward compatible (no module.yml changes needed)
- ✅ Clean separation of concerns
- ✅ User gets prompts for what they chose, defaults for dependencies

**Trade-offs:**
- ⚠️ Auto-included modules always use defaults (user can't override)
- ⚠️ If user wants to configure a dependency, they must explicitly select it

**Effort:** 4-6 hours

### Layer 3: Prompt Visibility Control (Optional Enhancement)

**What:** Add per-prompt metadata to control when prompts should be shown

**Why:** Provides fine-grained control - some prompts might make sense for auto-included modules, others don't

**Design:**

1. **Extend Prompt schema**:
```go
// internal/module/schema.go
type Prompt struct {
    Key      string   `yaml:"key"`
    Message  string   `yaml:"message"`
    Default  string   `yaml:"default"`
    Type     string   `yaml:"type"`
    Options  []string `yaml:"options"`
    ShowWhen string   `yaml:"show_when,omitempty"` // NEW: always|explicit_install|interactive
}
```

2. **Update module definitions**:
```yaml
# modules/zsh/module.yml
prompts:
  - key: zsh_framework
    message: "Zsh plugin framework"
    default: "zinit"
    type: choice
    options: [zinit, ohmyzsh]
    show_when: explicit_install  # NEW: Only show if user explicitly selected zsh
```

3. **Implement prompt filtering**:
```go
func shouldShowPrompt(p Prompt, cfg *RunConfig, mod *Module) bool {
    if cfg.Unattended {
        return false
    }

    switch p.ShowWhen {
    case "explicit_install":
        return cfg.ExplicitModules[mod.Name]
    case "interactive":
        return true
    case "always", "":
        return true
    default:
        return true
    }
}
```

**Benefits:**
- ✅ Fine-grained per-prompt control
- ✅ Module authors can mark framework-specific prompts
- ✅ Backward compatible (empty = always)

**Trade-offs:**
- ⚠️ Requires module.yml updates (but optional)
- ⚠️ More complex schema
- ⚠️ Might be over-engineering

**Effort:** 8-12 hours

## Chosen Approach: Full Solution (Layers 1 + 2 + 3)

User has selected the comprehensive approach implementing all three layers plus the `--prompt-dependencies` flag.

### Implementation Phases

**Phase 1: Fix Incorrect Dependencies**
1. Remove zsh dependency from starship module
2. Audit other modules for similar incorrect dependencies
3. Test starship with fish, bash, and zsh
4. Bump starship version to 1.1.0

**Phase 2: Installation Context Tracking**
1. Add `ExplicitlyRequested` field to ExecutionPlan
2. Modify `Resolve()` to populate this map
3. Add `ExplicitModules` to RunConfig
4. Update `handlePrompts()` to skip prompts for auto-included modules
5. Add tests for explicit vs auto-included behavior

**Phase 3: Prompt Visibility Control**
1. Add `ShowWhen` field to Prompt schema
2. Update zsh module prompts with `show_when: explicit_install`
3. Implement `shouldShowPrompt()` logic
4. Document the new prompt capability

**Phase 4: Force Configuration Flag**
1. Add `--prompt-dependencies` CLI flag
2. Pass flag through to RunConfig
3. Modify prompt logic to respect this override
4. Add tests for flag behavior
5. Document the flag in help text

## Critical Files to Modify

### Phase 1: Fix Dependencies
- `/home/gary/workspace/dotfiles/modules/starship/module.yml`
  - Remove `zsh` from dependencies
  - Update description
  - Bump version to 1.1.0

### Phase 2: Context Tracking
- `/home/gary/workspace/dotfiles/internal/module/resolver.go`
  - Add `ExplicitlyRequested map[string]bool` to ExecutionPlan struct
  - Modify `Resolve()` to populate this map from requested parameter
  - Return explicit tracking in plan

- `/home/gary/workspace/dotfiles/internal/module/runner.go`
  - Add `ExplicitModules map[string]bool` to RunConfig struct
  - Modify `handlePrompts()` to check if module is explicit or auto-included
  - Use defaults for auto-included modules (unless --prompt-dependencies)

- `/home/gary/workspace/dotfiles/cmd/dotfiles/install.go`
  - Extract ExplicitlyRequested from plan
  - Pass to RunConfig when creating runner
  - Add --prompt-dependencies flag definition

### Phase 3: Prompt Metadata
- `/home/gary/workspace/dotfiles/internal/module/schema.go`
  - Add `ShowWhen string` field to Prompt struct
  - Add validation for valid values (always, explicit_install, interactive)

- `/home/gary/workspace/dotfiles/modules/zsh/module.yml`
  - Add `show_when: explicit_install` to framework/plugin prompts
  - Keep framework-agnostic prompts as `always` (or omit)

- `/home/gary/workspace/dotfiles/internal/module/runner.go`
  - Add `shouldShowPrompt(p Prompt, cfg *RunConfig, mod *Module)` function
  - Update handlePrompts to use shouldShowPrompt check

### Phase 4: Force Config Flag
- `/home/gary/workspace/dotfiles/cmd/dotfiles/install.go`
  - Add `--prompt-dependencies` bool flag to install command
  - Pass through to RunConfig

- `/home/gary/workspace/dotfiles/internal/module/runner.go`
  - Add `PromptDependencies bool` to RunConfig struct
  - Update prompt logic: `if cfg.PromptDependencies || isExplicit { show prompt }`

## Verification Plan

### Phase 1: Dependency Fix
1. Test: `dotfiles install fish starship`
2. Verify: zsh is NOT auto-included
3. Verify: starship works with fish shell
4. Test: `dotfiles install bash starship`
5. Verify: starship works with bash
6. Test: `dotfiles install zsh starship`
7. Verify: Both install, starship integrates with zsh

### Phase 2: Context Tracking
1. Create test module `test-a` with prompts
2. Create test module `test-b` that depends on `test-a`
3. Test: `dotfiles install test-b`
4. Verify: test-a prompts use defaults (or show debug message with -v)
5. Verify: test-b prompts are interactive
6. Test: `dotfiles install test-a test-b`
7. Verify: Both modules show interactive prompts
8. Test: `dotfiles install test-b --prompt-dependencies`
9. Verify: Both modules show interactive prompts (flag override)

### Phase 3: Prompt Metadata
1. Update zsh module with `show_when: explicit_install` on framework prompts
2. Create module that depends on zsh
3. Test: `dotfiles install dependent-module`
4. Verify: zsh framework prompts are skipped (use defaults)
5. Test: `dotfiles install zsh dependent-module`
6. Verify: zsh framework prompts are shown
7. Test: `dotfiles install dependent-module --prompt-dependencies`
8. Verify: zsh framework prompts are shown (flag override)

### Phase 4: Full Integration
1. Test real-world scenario: `dotfiles install fish gh lazygit neovim`
2. Expected behavior:
   - fish: interactive prompts (explicit)
   - gh: interactive prompts (explicit)
   - lazygit: interactive prompts (explicit)
   - neovim: interactive prompts (explicit)
   - git: uses defaults (dependency of gh, lazygit, neovim)
   - ssh: uses defaults (dependency of git)
3. Test with flag: `dotfiles install fish gh --prompt-dependencies`
4. Expected: All modules including git and ssh show prompts

### Edge Cases
1. Test empty dependencies list
2. Test circular dependency detection (should still work)
3. Test OS filtering with explicit modules
4. Test update-only mode with explicit tracking
5. Test profiles that specify modules (should track as explicit)

## Implementation Details

### Layer 2: Prompt Logic Enhancement

```go
// internal/module/runner.go
func handlePrompts(cfg *RunConfig, mod *Module) (map[string]string, error) {
    answers := make(map[string]string, len(mod.Prompts))

    isExplicit := cfg.ExplicitModules[mod.Name]

    for _, p := range mod.Prompts {
        // Skip if unattended
        if cfg.Unattended {
            answers[p.Key] = p.Default
            continue
        }

        // Check show_when condition (Layer 3)
        if !shouldShowPrompt(p, cfg, mod, isExplicit) {
            answers[p.Key] = p.Default
            // Optional: Log for transparency
            if !isExplicit && cfg.Verbose {
                cfg.UI.Debug(fmt.Sprintf("Using default for %s.%s: %s (auto-included)", mod.Name, p.Key, p.Default))
            }
            continue
        }

        // Interactive prompt
        switch p.Type {
        case "input":
            answer, err = cfg.UI.PromptInput(p.Message, p.Default)
        case "confirm":
            answer, err = cfg.UI.PromptConfirm(p.Message, p.Default == "true")
        case "choice":
            answer, err = cfg.UI.PromptChoice(p.Message, p.Options)
        }

        if err != nil {
            return nil, err
        }
        answers[p.Key] = answer
    }
    return answers, nil
}

func shouldShowPrompt(p Prompt, cfg *RunConfig, mod *Module, isExplicit bool) bool {
    // --prompt-dependencies flag overrides everything
    if cfg.PromptDependencies {
        return true
    }

    // Check show_when field (Layer 3)
    switch p.ShowWhen {
    case "explicit_install":
        return isExplicit
    case "interactive":
        return true
    case "always", "":
        // Default: show for explicit, hide for auto-included
        return isExplicit
    default:
        return isExplicit
    }
}
```

### CLI Flag Addition

```go
// cmd/dotfiles/install.go
var (
    // ... existing flags ...
    promptDependencies bool
)

func init() {
    installCmd.Flags().BoolVar(&promptDependencies, "prompt-dependencies", false,
        "Show prompts for auto-included dependency modules (default: use defaults)")
}
```

### Updated RunConfig Structure

```go
// internal/module/runner.go
type RunConfig struct {
    UI                  RunnerUI
    SysInfo            *sysinfo.SystemInfo
    ConfigStore        *config.Store
    StateStore         *state.StateStore
    Force              bool
    SkipFailed         bool
    Unattended         bool
    Verbose            bool
    ExplicitModules    map[string]bool  // NEW: Tracks which modules were explicitly selected
    PromptDependencies bool             // NEW: Force prompts for dependencies
}
```

## Design Decisions

### Transitive Dependencies
- **Decision**: Only explicitly selected modules get prompts by default
- If user selects C → B → A (transitive chain)
- Only C prompts are shown interactively
- B and A use defaults (unless --prompt-dependencies)

### Transparency
- **Decision**: Use debug logging (requires --verbose) for default usage
- Avoids cluttering standard output
- Power users can see what's happening with -v flag

### Backward Compatibility
- **Decision**: Add migration notes in CHANGELOG
- Profiles using `modules: [starship]` and expecting zsh should add explicit zsh
- Breaking change but more correct behavior
- Version bump to 2.0.0 for major behavior change

### Default show_when Behavior
- **Decision**: Empty/omitted `show_when` defaults to "explicit_install" behavior
- More conservative default (don't spam users)
- Modules can explicitly use `show_when: always` if needed

## Additional Considerations

### Profile Handling
Modules specified in a profile should be treated as explicit selections:
```go
// cmd/dotfiles/install.go
// When loading profile
requested = append(requested, profile.Modules...)
// All profile modules are now in 'requested', so they'll be marked explicit
```

### Update Mode
When using `--update-only`, only actually-updated modules should show prompts:
```go
// If module is being updated AND explicit, show prompts
// If module is being updated but auto-included, use defaults
```

### Documentation Updates Needed
1. **README.md**: Add section on dependency behavior
2. **CHANGELOG.md**: Document breaking change in starship module
3. **Module authoring guide**: Document `show_when` field
4. **CLI help text**: Document `--prompt-dependencies` flag
5. **Migration guide**: Help users update profiles if they depended on transitive includes

### Testing Strategy
1. Unit tests for `shouldShowPrompt()` logic
2. Unit tests for explicit module tracking in `Resolve()`
3. Integration tests for full install flow
4. Regression tests for existing modules
5. Test coverage target: >80% for modified functions

### Rollout Plan
1. Implement Phase 1 (starship fix) as separate PR - quick win
2. Implement Phases 2-4 together in feature branch
3. Get user feedback on behavior before merging
4. Document in release notes as major version bump (2.0.0)
5. Provide migration script if needed

### Performance Impact
- Minimal: O(1) map lookups for explicit module checking
- No change to dependency resolution algorithm complexity
- Prompt display might be faster (fewer prompts shown)
