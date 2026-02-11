# Add Oh My Zsh support to zsh module

## Context

User wants Oh My Zsh as an alternative to the existing Zinit plugin manager. OMZ and Zinit are mutually exclusive — they both manage the shell initialization, so you can't source both. Rather than a separate module (which would create zshrc deployment conflicts), the zsh module gains a framework choice prompt and the zshrc becomes a Go template with conditional sections.

---

## Design

### Prompts (added to `modules/zsh/module.yml`)

1. **`zsh_framework`** — choice: `zinit` (default), `ohmyzsh`
2. **`zsh_omz_plugins`** — choice: `minimal`, `standard` (default), `full`
   - minimal: git, z
   - standard: git, z, docker, colored-man-pages, command-not-found
   - full: git, z, docker, colored-man-pages, command-not-found, sudo, extract, history, aliases, web-search
3. **`zsh_prompt`** — choice: `starship` (default), `robbyrussell`, `agnoster`

All 3 prompts shown. If zinit chosen, prompts 2-3 ignored by template (zinit always uses starship + its own plugins). Defaults preserve current behavior for existing/unattended installs.

### zshrc → zshrc.tmpl (Go template)

Convert from static file (symlink) to rendered template. Structure:

```
{{- $framework := index .Env "DOTFILES_PROMPT_ZSH_FRAMEWORK" -}}
{{- $prompt := index .Env "DOTFILES_PROMPT_ZSH_PROMPT" -}}
{{- $plugins := index .Env "DOTFILES_PROMPT_ZSH_OMZ_PLUGINS" -}}

{{- if eq $framework "ohmyzsh" }}
# Oh My Zsh setup
export ZSH="$HOME/.oh-my-zsh"
ZSH_THEME="{{ theme based on $prompt }}"
plugins=( {{ plugin list based on $plugins }} )
source $ZSH/oh-my-zsh.sh
{{- else }}
# Zinit setup (current content, unchanged)
{{- end }}

# === Shared config (both frameworks) ===
# History, shell options, key bindings, PATH
# Source aliases.zsh, functions.zsh
# Starship init (if prompt=starship or framework=zinit)
# Source .zshrc.local
```

Key details:
- OMZ handles completions internally, so completion settings only render for zinit
- When OMZ + starship: `ZSH_THEME=""` (empty disables OMZ theme, starship takes over)
- When OMZ + robbyrussell/agnoster: normal OMZ theme, no starship init

### install.sh changes

Read `DOTFILES_PROMPT_ZSH_FRAMEWORK` env var. Conditionally:
- **zinit**: clone Zinit (current logic, unchanged)
- **ohmyzsh**: `git clone https://github.com/ohmyzsh/ohmyzsh.git ~/.oh-my-zsh`
- Both: shared logic (install zsh, create config dir, chsh) stays the same

### verify.sh changes

Read `DOTFILES_PROMPT_ZSH_FRAMEWORK`. Conditionally check:
- **zinit**: `~/.local/share/zinit/zinit.git` exists
- **ohmyzsh**: `~/.oh-my-zsh` directory exists
- `.zshrc` check: accept both symlink and regular file (template produces a file)
- Shared checks unchanged (zsh binary, aliases.zsh, functions.zsh)

### module.yml file entry change

```yaml
# Before:
- source: zshrc
  dest: "~/.zshrc"
  type: symlink

# After:
- source: zshrc.tmpl
  dest: "~/.zshrc"
  type: template
```

---

## Files to modify/create

1. **`modules/zsh/module.yml`** — add 3 prompts, change zshrc file entry to template type
2. **`modules/zsh/zshrc`** → rename to **`modules/zsh/zshrc.tmpl`** — Go template with conditional zinit/omz sections
3. **`modules/zsh/install.sh`** — conditional Zinit/OMZ installation
4. **`modules/zsh/verify.sh`** — conditional framework verification

No Go code changes needed.

## Verification

```bash
export PATH="/usr/local/go/bin:$PATH"
go test ./internal/module/ ./cmd/dotfiles/
go build ./cmd/dotfiles/
# Manual: dotfiles install zsh --force (test both zinit and ohmyzsh paths)
```
