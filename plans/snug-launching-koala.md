# Fix: git-delta sudo prompt + post-install notes

## Context

On a clean Ubuntu user install, two UX issues:
1. **git-delta silently fails** instead of prompting for sudo password — the `has_sudo` gate checks for *passwordless* sudo (`sudo -n true`) and bails before the user gets a chance to authenticate interactively, unlike `chsh` which prompts naturally.
2. **No post-install guidance** — after zsh/starship install, the new shell isn't active until the user logs out and back in, but nothing tells them this.

---

## Fix 1: git-delta sudo prompt

**File:** `modules/git/install.sh` (line 91)

Change the `has_sudo` gate to allow interactive sudo prompting. In interactive mode, skip the gate and let `sudo dpkg -i` prompt naturally (stdin is already connected by the runner). In non-interactive mode, keep the existing `has_sudo` check.

```bash
# Before:
if ! has_sudo; then

# After:
if ! is_interactive && ! has_sudo; then
```

One line change. `is_interactive()` is already available in `lib/helpers.sh:63`. The runner already connects stdin for interactive mode (`runner.go:459`).

---

## Fix 2: Post-install notes system

### 2a. Add `Notes` field to Module struct
**File:** `internal/module/schema.go`

Add `Notes []string \`yaml:"notes"\`` to the `Module` struct (after `Timeout`, before `Dir`). Backward compatible — omitted YAML fields result in nil slice.

### 2b. Add `Notes` to RunResult and populate on success
**File:** `internal/module/runner.go`

- Add `Notes []string` to `RunResult` struct
- In `runModule()` success return (line 286), set `Notes: mod.Notes`
- Skipped and failed modules get no notes (correct behavior)

### 2c. Display notes after summary
**File:** `cmd/dotfiles/install.go`

After the summary line (line 234), collect and display notes from successful non-skipped results:

```go
var allNotes []string
for _, r := range results {
    if r.Success && !r.Skipped && len(r.Notes) > 0 {
        for _, note := range r.Notes {
            allNotes = append(allNotes, fmt.Sprintf("[%s] %s", r.Module.Name, note))
        }
    }
}
if len(allNotes) > 0 {
    u.Info("")
    u.Warn("Post-install notes:")
    for _, note := range allNotes {
        u.Warn("  " + note)
    }
}
```

### 2d. Add notes to zsh module
**File:** `modules/zsh/module.yml`

```yaml
notes:
  - "Run 'exec zsh' or log out and back in to activate your new shell"
```

Starship does **not** need its own note — it depends on zsh and activates via zshrc. The zsh note covers both.

---

## Tests

### Update existing test YAML in `schema_test.go`
Add `notes:` to the `TestParseModuleYAML` test YAML and assert `m.Notes` is parsed correctly. Also verify that `TestParseModuleYAML_DefaultName` (no notes in YAML) results in nil `Notes`.

### Add runner test in `runner_test.go`
Add `TestRunNotesOnSuccess` — create a module with `Notes: []string{"test note"}`, no scripts, run it, assert `result.Notes` contains the note. Verify a skipped module returns empty notes.

---

## Files modified (6)
1. `modules/git/install.sh` — interactive-aware sudo gate
2. `internal/module/schema.go` — Notes field on Module
3. `internal/module/runner.go` — Notes on RunResult, populate on success
4. `cmd/dotfiles/install.go` — display post-install notes
5. `modules/zsh/module.yml` — add notes
6. `internal/module/schema_test.go` — test Notes parsing
7. `internal/module/runner_test.go` — test Notes in RunResult

## Verification
```bash
export PATH="/usr/local/go/bin:$PATH"
go test ./internal/module/ ./cmd/dotfiles/
go build ./cmd/dotfiles/
```
