# Plan: Interactive Module Selection

## Context

The install command currently runs modules determined by either CLI args or a static profile YAML file, with no opportunity for the user to review or adjust the selection before execution. This plan adds an interactive multiselect prompt at startup that shows all available modules with profile-selected ones pre-checked, allowing the user to confirm or adjust before proceeding. It also assesses the current CLI architecture against the patterns in `docs/cli-implementation-notes.md`.

## Architecture Assessment

**What's already solid:**
- RunnerUI interface decouples module runner from ui pkg (same pattern as the doc's WizardPrompter)
- TTY detection + `--unattended` flag with auto-detection from non-interactive stdin
- Catppuccin Mocha color palette centralized in ui.go
- Graceful degradation (spinner/logging methods adapt to TTY vs non-TTY)

**Gaps vs reference doc (address incrementally, not all at once):**
- No multiselect/checkbox capability (blocking for this feature)
- No cancellation handling on prompts (nice to add alongside multiselect)
- No `NO_COLOR`/`FORCE_COLOR` support (out of scope)
- Flat architecture vs 4-layer (current is fine for project size)

## Library Choice: `charmbracelet/huh`

A proper multiselect with arrow key navigation + spacebar toggling requires raw terminal mode. Building this from scratch would be 300+ lines of platform-sensitive code. `charmbracelet/huh` provides exactly this with ~15 lines of integration code.

- Provides `MultiSelect` field with keyboard navigation, spacebar toggle, Enter to confirm
- Built on bubbletea/lipgloss with custom theme support (can match Catppuccin Mocha)
- Returns `huh.ErrUserAborted` on Ctrl+C for clean cancellation handling
- MIT license, actively maintained, standard Go CLI library

**Trade-off:** Adds transitive deps (bubbletea, lipgloss, termenv, x/term). Binary size increases ~2-3 MB. Worth it for the UX gain and future interactive capabilities.

## Implementation Steps

### 1. Add huh dependency
```
go get github.com/charmbracelet/huh@latest
```

### 2. Add sentinel error and MultiSelectOption to module package
**File:** [runner.go](internal/module/runner.go)

Add `ErrUserCancelled` sentinel and `MultiSelectOption` struct. Add `PromptMultiSelect` to the `RunnerUI` interface:

```go
var ErrUserCancelled = errors.New("operation cancelled by user")

type MultiSelectOption struct {
    Value       string
    Label       string
    Description string
}

type RunnerUI interface {
    // ... existing methods ...
    PromptMultiSelect(msg string, options []MultiSelectOption, preSelected []string) ([]string, error)
}
```

### 3. Create huh theme helper
**File:** [theme.go](internal/ui/theme.go) (new)

Map the existing Catppuccin Mocha colors to a `*huh.Theme` using lipgloss styles. This keeps theme logic separate from ui.go and ensures the multiselect visually matches the rest of the CLI.

### 4. Implement PromptMultiSelect on *UI
**File:** [ui.go](internal/ui/ui.go)

- **TTY mode:** Use `huh.NewForm` with `huh.NewMultiSelect[string]`, applying the Catppuccin theme. Translate `huh.ErrUserAborted` to `module.ErrUserCancelled`.
- **Non-TTY mode:** Print `[MULTISELECT]` prefix and return preSelected values unchanged (same as unattended).

### 5. Move auto-unattended detection earlier in install flow
**File:** [install.go](cmd/dotfiles/install.go)

Move lines 109-113 (auto-unattended when stdin is non-interactive) to right after system detection (after line 38), before secrets auth and module discovery. This ensures `unattended` is correctly set before the multiselect guard.

### 6. Insert interactive module selection into install flow
**File:** [install.go](cmd/dotfiles/install.go)

Between module discovery (line 95) and `module.Resolve` (line 97), add:

```
if no CLI args given && !unattended:
    build MultiSelectOption list from allModules (filtered by OS)
    call u.PromptMultiSelect("Select modules to install", options, requested)
    if ErrUserCancelled: print info message, return nil
    if empty selection: warn, return nil
    requested = selected
```

After `Resolve`, compute auto-included dependencies (diff between user selection and resolved plan) and show an info message listing them.

### 7. Update test mocks
**File:** [runner_test.go](internal/module/runner_test.go)

Add `PromptMultiSelect` to `testUI` struct that returns preSelected values.

**File:** [ui_test.go](internal/ui/ui_test.go)

Add non-TTY test for `PromptMultiSelect` verifying it returns preSelected values and prints `[MULTISELECT]` output.

## Files Modified

| File | Change |
|------|--------|
| `go.mod` | Add `charmbracelet/huh` dependency |
| [internal/module/runner.go](internal/module/runner.go) | Add `ErrUserCancelled`, `MultiSelectOption`, extend `RunnerUI` interface |
| [internal/ui/ui.go](internal/ui/ui.go) | Add `PromptMultiSelect` method |
| `internal/ui/theme.go` | New: Catppuccin Mocha huh theme |
| [cmd/dotfiles/install.go](cmd/dotfiles/install.go) | Move auto-unattended earlier, add multiselect step, dependency note |
| [internal/module/runner_test.go](internal/module/runner_test.go) | Update `testUI` mock |
| [internal/ui/ui_test.go](internal/ui/ui_test.go) | Add non-TTY multiselect test |

## Unattended / Non-Interactive Behavior

The multiselect is skipped entirely when any of these hold:
- `--unattended` flag is set
- stdin is non-interactive (auto-detected, piped input, CI)
- CLI args were provided (`dotfiles install ssh git`)

In all cases, the existing behavior is preserved: profile modules are used.

## Verification

1. `go build ./...` compiles successfully
2. `go test ./...` passes (updated mocks satisfy interface)
3. `dotfiles install` interactively - shows multiselect with profile modules checked, arrow/spacebar works, Enter confirms
4. Ctrl+C during multiselect - clean exit with "cancelled" message
5. `dotfiles install --unattended` - no prompt, uses profile defaults
6. `echo "" | dotfiles install` - auto-detects non-interactive, no prompt
7. `dotfiles install ssh git` - no prompt, uses CLI args directly
8. Select module with dependencies (e.g., only `zsh`) - auto-includes `git`, `ssh` with info message
9. `dotfiles install --dry-run` - shows multiselect, prints plan, no execution
