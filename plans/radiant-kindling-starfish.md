# Roadmap Tier 1 Implementation Plan

## Context

Following the ROADMAP.md audit, this plan implements the three genuine gaps in Tier 1 (Core Completeness). Two items turned out to be already done:

- ~~Fully implement module uninstall~~ — `cmd/dotfiles/uninstall.go` is complete with full rollback
- ~~State schema migration~~ — `NeedsMigration()` and `MigrateFileStatesFromOperations()` already exist in `internal/state/state.go`

Remaining work:

1. **Module schema validation** — `ParseModuleYAML()` does zero validation; typos in field values silently pass through
2. **`download_file` helper + checksum verification** — no centralized download helper exists; modules invent their own patterns
3. **ARM64 fixes** — `lazygit` and `awscli` hard-code `Linux_x86_64`; all other modules either use `$DOTFILES_ARCH` or dynamic installer scripts

A lightweight fourth item is also included:

4. **Formalize state schema versioning** — add a `schema_version` field to `ModuleState` to complement the existing migration code

---

## Item 1 — Module Schema Validation

### What to build

A `validateModule(*Module)` function that checks field values after YAML parse. Called in two places:
- **`Discover()`** in `internal/module/discovery.go` — issues a warning for invalid modules but continues (graceful degradation)
- **`dotfiles validate`** subcommand — strict mode, exits 1 if any module is invalid

### Validation rules

**Required fields:**
- `name` — non-empty, matches `^[a-z0-9]+(-[a-z0-9]+)*$`
- `description` — non-empty
- `version` — non-empty, ideally semver but any non-empty string is acceptable

**Value constraints:**
- `files[].type` — must be one of `symlink`, `copy`, `template`
- `prompts[].type` — must be one of `input`, `confirm`, `choice`
- `prompts[].options` — required and non-empty when `type == "choice"`
- `prompts[].show_when` — if set, must be one of `always`, `explicit_install`, `interactive`
- `prompts[].depends_on.key` — must reference an existing prompt key in the same module
- `os[]` values — if set, must be one of `darwin`, `ubuntu`, `arch`
- `timeout` — if set, must parse successfully via `time.ParseDuration`

**File existence check (validate subcommand only):**
- Each `files[].source` path must exist relative to the module directory

**Unknown keys:**
- YAML unmarshaling uses strict mode (`yaml.Decoder.KnownFields(true)`) in the validate subcommand to catch typos like `dependancies`

### Files to modify

- **`internal/module/schema.go`** — add `validateModule(m *Module) []string` returning a slice of error strings
- **`internal/module/discovery.go`** — call `validateModule` after parse; log warnings via `log.Warn()` for each issue; skip module if name is empty
- **`cmd/dotfiles/validate.go`** — new file; `dotfiles validate` subcommand
- **`cmd/dotfiles/root.go`** — register `validateCmd`

### `dotfiles validate` CLI behaviour

```
$ dotfiles validate
✓ 1password  (v1.0.0)
✓ git        (v1.0.0)
✗ my-module  — files[0].type "link" is not valid (must be symlink, copy, or template)
✗ zsh        — prompts[2].options must be non-empty for type "choice"

2 modules failed validation.
$ echo $?
1
```

Flags:
- `--json` — output as JSON array of `{module, errors[]}` for CI integration
- `--strict` — also warn on unknown YAML keys (uses `KnownFields(true)`)

---

## Item 2 — `download_file` Helper + Checksum Verification

### What to build

A `download_file` function in `lib/helpers.sh`:

```bash
# download_file URL DEST [SHA256]
#   Downloads URL to DEST using curl.
#   If SHA256 is provided, verifies the checksum and aborts on mismatch.
#   Idempotent: skips download if DEST already exists and checksum matches.
#   Dry-run safe.
download_file() {
    local url="$1" dest="$2" expected_sha="${3:-}"
    ...
}
```

Behaviour:
- If `DEST` already exists and `SHA256` provided → verify hash; skip download if matches, re-download if not
- If `DEST` already exists and no `SHA256` → skip with `log_info`
- In dry-run → log what would happen, return 0
- After download, verify checksum if provided; `log_error` and `return 1` on mismatch
- Uses `curl -fsSL` for downloads; falls back to `wget -qO-` if curl not available
- Checksum tool: `sha256sum` (Linux) or `shasum -a 256` (macOS)

### Files to modify

- **`lib/helpers.sh`** — add `download_file` after `github_clone` (currently ~line 242)
- **`modules/lazygit/install.sh`** — replace hard-coded URL with `download_file` + arch detection
- **`modules/awscli/install.sh`** — replace hard-coded URL with `download_file` + arch detection

### `lazygit` fix

```bash
# Before:
LAZYGIT_VERSION="0.44.1"
curl -Lo /tmp/lazygit.tar.gz \
  "https://github.com/jesseduffield/lazygit/releases/download/v${LAZYGIT_VERSION}/lazygit_${LAZYGIT_VERSION}_Linux_x86_64.tar.gz"

# After:
LAZYGIT_VERSION="0.44.1"
ARCH="Linux_x86_64"
[[ "${DOTFILES_ARCH}" == "arm64" ]] && ARCH="Linux_arm64"
download_file \
  "https://github.com/jesseduffield/lazygit/releases/download/v${LAZYGIT_VERSION}/lazygit_${LAZYGIT_VERSION}_${ARCH}.tar.gz" \
  "/tmp/lazygit.tar.gz"
```

### `awscli` fix

```bash
# Before:
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o /tmp/awscliv2.zip

# After:
ARCH="x86_64"
[[ "${DOTFILES_ARCH}" == "arm64" ]] && ARCH="aarch64"
download_file \
  "https://awscli.amazonaws.com/awscli-exe-linux-${ARCH}.zip" \
  "/tmp/awscliv2.zip"
```

---

## Item 3 — State Schema Versioning

Minor addition to complement existing migration code.

### What to build

Add `SchemaVersion int` to `ModuleState` in `internal/state/state.go`.

- New states written with `SchemaVersion: 1`
- `NeedsMigration()` (already exists) additionally checks `SchemaVersion == 0`
- After running `MigrateFileStatesFromOperations()` (already exists), set `SchemaVersion = 1` and re-save
- This is purely additive: JSON files without this field unmarshal with the Go zero value (0), which triggers migration

### Files to modify

- **`internal/state/state.go`** — add `SchemaVersion int` field, update `NeedsMigration()`, update `MigrateFileStatesFromOperations()` to set version after migration
- **`internal/state/state_test.go`** — add test asserting that an old-format state file (without `schema_version`) triggers migration and is re-written with `schema_version: 1`

---

## Verification

```bash
export PATH="/usr/local/go/bin:$PATH"

# Build
go build -o dotfiles ./cmd/dotfiles/

# Validate all modules (expect clean output)
./dotfiles validate

# Test validate detects a bad field value (manually break a module.yml temporarily)
echo "  files:\n  - source: foo\n    dest: ~/bar\n    type: INVALID" >> modules/git/module.yml
./dotfiles validate   # should show error for git
git checkout modules/git/module.yml  # restore

# Test download_file helper (syntax check)
bash -n lib/helpers.sh

# Test dry-run on a module that uses download_file
./dotfiles install lazygit --dry-run

# Build and run unit tests
go test ./internal/...

# Verify state schema_version field in a fresh state file
./dotfiles install neovim --dry-run   # no state written
./dotfiles status                     # confirm existing states still load correctly
```

---

## Implementation Order

1. `internal/state/state.go` — add `SchemaVersion` (5 min, minimal risk)
2. `internal/module/schema.go` — add `validateModule()` (30 min)
3. `internal/module/discovery.go` — call `validateModule()` with warn-on-error (10 min)
4. `cmd/dotfiles/validate.go` — new validate subcommand (30 min)
5. `cmd/dotfiles/root.go` — register validateCmd (2 min)
6. `lib/helpers.sh` — add `download_file` helper (20 min)
7. `modules/lazygit/install.sh` — use `download_file` + ARM64 (10 min)
8. `modules/awscli/install.sh` — use `download_file` + ARM64 (10 min)
9. Tests — state migration test, validation unit tests (20 min)
