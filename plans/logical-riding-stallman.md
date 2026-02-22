# Starship Preset Selection Prompt

## Context

The starship module currently deploys a static `starship.toml` (as a symlink) with no
user configuration. Separately, the `zsh_prompt` question in the zsh module used to
offer "starship" alongside OMZ themes — but that has since been scoped to ohmyzsh-only
via `depends_on`, so zinit users are never prompted about their prompt appearance.

The request: add a preset selection prompt to the **starship module itself**, independent
of which zsh framework is used. Starship ships official presets (`starship preset
<name>`) that cover a wide range of styles; the user should be able to choose one at
install time instead of always getting the custom `starship.toml`.

---

## Architecture Decision

**Key constraint**: `install.sh` runs *before* files are deployed. If we wrote the
preset to `~/.config/starship.toml` in install.sh and the `files:` section still had a
symlink for `starship.toml`, the file deployment would overwrite install.sh's work.

**Solution**: Remove `starship.toml` from the `files:` section entirely. `install.sh`
takes sole ownership of writing `~/.config/starship.toml`:
- preset == "custom" → copy `$DOTFILES_MODULE_DIR/starship.toml` (our curated config)
- preset == anything else → run `starship preset <name> > ~/.config/starship.toml`

The repo's `starship.toml` is kept as the "custom" source; it's just no longer
symlinked automatically.

---

## Implementation

### 1. `modules/starship/module.yml`

- Add `starship_preset` prompt, `show_when: explicit_install`, default `"custom"`
- Options: `custom` + all 11 official presets (in a sensible order)
- Remove the `files:` section

```yaml
prompts:
  - key: starship_preset
    message: "Starship prompt preset (custom = dotfiles curated config)"
    default: "custom"
    type: choice
    options:
      - custom
      - no-nerd-font
      - plain-text
      - pure-preset
      - tokyo-night
      - catppuccin-powerline
      - gruvbox-rainbow
      - jetpack
      - onehalf-dark
      - pastel-powerline
      - bracketed-segments
      - solarized-osaka
    show_when: explicit_install
```

No `depends_on` needed — starship preset is independent of zsh framework.

### 2. `modules/starship/install.sh`

After the binary is installed, add a config block:

```bash
# Apply starship preset or custom config
_preset="${DOTFILES_PROMPT_STARSHIP_PRESET:-custom}"
_starship_cfg="${DOTFILES_HOME}/.config/starship.toml"

if is_dry_run; then
    log_info "[dry-run] Would apply starship preset: ${_preset}"
elif [[ "$_preset" == "custom" ]]; then
    cp "${DOTFILES_MODULE_DIR}/starship.toml" "$_starship_cfg"
    log_success "Applied custom starship config"
else
    starship preset "$_preset" > "$_starship_cfg"
    log_success "Applied starship preset: ${_preset}"
fi
```

The copy/write is safe on re-runs (idempotent) — it just overwrites.
`--force` reinstall triggers install.sh again, so the preset is re-applied.

---

## Critical Files

| File | Change |
|------|--------|
| `modules/starship/module.yml` | Add `starship_preset` prompt; remove `files:` section |
| `modules/starship/install.sh` | Add preset application block after binary install |

No Go code changes required — `DOTFILES_PROMPT_STARSHIP_PRESET` is automatically
passed to scripts via the existing `buildEnvVars` mechanism in `runner.go`.

---

## Verification

```bash
# Build fresh binary
go build -o /tmp/dotfiles .

# Test 1: custom preset (default) — should copy our starship.toml
dotfiles install starship --unattended
cat ~/.config/starship.toml   # should match modules/starship/starship.toml

# Test 2: official preset
DOTFILES_PROMPT_STARSHIP_PRESET=tokyo-night dotfiles install starship --unattended
cat ~/.config/starship.toml   # should show tokyo-night palette/config

# Test 3: dry-run shows correct log
dotfiles install starship --unattended --dry-run
# should see: [dry-run] Would apply starship preset: custom

# Test 4: interactive prompt (zinit user) — only starship_preset asked
dotfiles install starship
# should prompt once: "Starship prompt preset"
# should NOT prompt for OMZ plugins or prompt theme (those are zsh-module prompts)
```
