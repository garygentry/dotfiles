package module

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseModuleYAML(t *testing.T) {
	dir := t.TempDir()
	moduleDir := filepath.Join(dir, "zsh")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}

	yml := `name: zsh
description: Zsh shell configuration
version: "1.0"
priority: 10
dependencies:
  - base
os:
  - macos
  - ubuntu
requires:
  - git
files:
  - source: zshrc
    dest: ~/.zshrc
    type: symlink
  - source: zshenv.tmpl
    dest: ~/.zshenv
    type: template
prompts:
  - key: theme
    message: "Pick a theme"
    default: powerlevel10k
    type: choice
    options:
      - powerlevel10k
      - starship
tags:
  - shell
  - cli
`
	ymlPath := filepath.Join(moduleDir, "module.yml")
	if err := os.WriteFile(ymlPath, []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := ParseModuleYAML(ymlPath)
	if err != nil {
		t.Fatalf("ParseModuleYAML returned error: %v", err)
	}

	if m.Name != "zsh" {
		t.Errorf("Name = %q, want %q", m.Name, "zsh")
	}
	if m.Description != "Zsh shell configuration" {
		t.Errorf("Description = %q, want %q", m.Description, "Zsh shell configuration")
	}
	if m.Version != "1.0" {
		t.Errorf("Version = %q, want %q", m.Version, "1.0")
	}
	if m.Priority != 10 {
		t.Errorf("Priority = %d, want %d", m.Priority, 10)
	}
	if len(m.Dependencies) != 1 || m.Dependencies[0] != "base" {
		t.Errorf("Dependencies = %v, want [base]", m.Dependencies)
	}
	if len(m.OS) != 2 || m.OS[0] != "macos" || m.OS[1] != "ubuntu" {
		t.Errorf("OS = %v, want [macos ubuntu]", m.OS)
	}
	if len(m.Requires) != 1 || m.Requires[0] != "git" {
		t.Errorf("Requires = %v, want [git]", m.Requires)
	}
	if len(m.Files) != 2 {
		t.Fatalf("Files length = %d, want 2", len(m.Files))
	}
	if m.Files[0].Source != "zshrc" || m.Files[0].Dest != "~/.zshrc" || m.Files[0].Type != "symlink" {
		t.Errorf("Files[0] = %+v, unexpected", m.Files[0])
	}
	if m.Files[1].Source != "zshenv.tmpl" || m.Files[1].Dest != "~/.zshenv" || m.Files[1].Type != "template" {
		t.Errorf("Files[1] = %+v, unexpected", m.Files[1])
	}
	if len(m.Prompts) != 1 {
		t.Fatalf("Prompts length = %d, want 1", len(m.Prompts))
	}
	p := m.Prompts[0]
	if p.Key != "theme" || p.Message != "Pick a theme" || p.Default != "powerlevel10k" || p.Type != "choice" {
		t.Errorf("Prompts[0] = %+v, unexpected", p)
	}
	if len(p.Options) != 2 || p.Options[0] != "powerlevel10k" || p.Options[1] != "starship" {
		t.Errorf("Prompts[0].Options = %v, unexpected", p.Options)
	}
	if len(m.Tags) != 2 || m.Tags[0] != "shell" || m.Tags[1] != "cli" {
		t.Errorf("Tags = %v, want [shell cli]", m.Tags)
	}
	if m.Dir != moduleDir {
		t.Errorf("Dir = %q, want %q", m.Dir, moduleDir)
	}
}

func TestParseModuleYAML_DefaultName(t *testing.T) {
	dir := t.TempDir()
	moduleDir := filepath.Join(dir, "vim")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}

	yml := `description: Vim editor config
files:
  - source: vimrc
    dest: ~/.vimrc
    type: symlink
`
	ymlPath := filepath.Join(moduleDir, "module.yml")
	if err := os.WriteFile(ymlPath, []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := ParseModuleYAML(ymlPath)
	if err != nil {
		t.Fatalf("ParseModuleYAML returned error: %v", err)
	}

	if m.Name != "vim" {
		t.Errorf("Name = %q, want %q (derived from directory)", m.Name, "vim")
	}
	if m.Priority != 50 {
		t.Errorf("Priority = %d, want default 50", m.Priority)
	}
}

func TestSupportsOS(t *testing.T) {
	tests := []struct {
		name     string
		osList   []string
		queryOS  string
		expected bool
	}{
		{
			name:     "empty OS list supports any OS",
			osList:   nil,
			queryOS:  "macos",
			expected: true,
		},
		{
			name:     "empty OS list supports unknown OS",
			osList:   []string{},
			queryOS:  "freebsd",
			expected: true,
		},
		{
			name:     "matching OS",
			osList:   []string{"macos", "ubuntu"},
			queryOS:  "ubuntu",
			expected: true,
		},
		{
			name:     "non-matching OS",
			osList:   []string{"macos", "ubuntu"},
			queryOS:  "arch",
			expected: false,
		},
		{
			name:     "single OS match",
			osList:   []string{"macos"},
			queryOS:  "macos",
			expected: true,
		},
		{
			name:     "single OS no match",
			osList:   []string{"macos"},
			queryOS:  "ubuntu",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Module{OS: tt.osList}
			got := m.SupportsOS(tt.queryOS)
			if got != tt.expected {
				t.Errorf("SupportsOS(%q) = %v, want %v (OS list: %v)", tt.queryOS, got, tt.expected, tt.osList)
			}
		})
	}
}

func TestDiscover(t *testing.T) {
	dir := t.TempDir()

	// Module "git" with higher priority (lower number = higher priority).
	gitDir := filepath.Join(dir, "git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	gitYML := `name: git
description: Git configuration
priority: 10
files:
  - source: gitconfig
    dest: ~/.gitconfig
    type: symlink
`
	if err := os.WriteFile(filepath.Join(gitDir, "module.yml"), []byte(gitYML), 0o644); err != nil {
		t.Fatal(err)
	}

	// Module "zsh" with lower priority (higher number).
	zshDir := filepath.Join(dir, "zsh")
	if err := os.MkdirAll(zshDir, 0o755); err != nil {
		t.Fatal(err)
	}
	zshYML := `name: zsh
description: Zsh config
priority: 30
os:
  - macos
  - ubuntu
files:
  - source: zshrc
    dest: ~/.zshrc
    type: symlink
`
	if err := os.WriteFile(filepath.Join(zshDir, "module.yml"), []byte(zshYML), 0o644); err != nil {
		t.Fatal(err)
	}

	// A directory without module.yml should be skipped.
	noModDir := filepath.Join(dir, "random")
	if err := os.MkdirAll(noModDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// A regular file in the modules directory should be skipped.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	modules, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}

	if len(modules) != 2 {
		t.Fatalf("Discover returned %d modules, want 2", len(modules))
	}

	// First module should be "git" (priority 10).
	if modules[0].Name != "git" {
		t.Errorf("modules[0].Name = %q, want %q", modules[0].Name, "git")
	}
	if modules[0].Priority != 10 {
		t.Errorf("modules[0].Priority = %d, want 10", modules[0].Priority)
	}
	if modules[0].Dir != gitDir {
		t.Errorf("modules[0].Dir = %q, want %q", modules[0].Dir, gitDir)
	}

	// Second module should be "zsh" (priority 30).
	if modules[1].Name != "zsh" {
		t.Errorf("modules[1].Name = %q, want %q", modules[1].Name, "zsh")
	}
	if modules[1].Priority != 30 {
		t.Errorf("modules[1].Priority = %d, want 30", modules[1].Priority)
	}
	if modules[1].Dir != zshDir {
		t.Errorf("modules[1].Dir = %q, want %q", modules[1].Dir, zshDir)
	}
}

func TestDiscoverSortByNameWhenPriorityEqual(t *testing.T) {
	dir := t.TempDir()

	// Two modules with the same priority, should sort by name.
	for _, name := range []string{"beta", "alpha"} {
		modDir := filepath.Join(dir, name)
		if err := os.MkdirAll(modDir, 0o755); err != nil {
			t.Fatal(err)
		}
		yml := "priority: 50\n"
		if err := os.WriteFile(filepath.Join(modDir, "module.yml"), []byte(yml), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	modules, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}

	if len(modules) != 2 {
		t.Fatalf("Discover returned %d modules, want 2", len(modules))
	}

	if modules[0].Name != "alpha" {
		t.Errorf("modules[0].Name = %q, want %q", modules[0].Name, "alpha")
	}
	if modules[1].Name != "beta" {
		t.Errorf("modules[1].Name = %q, want %q", modules[1].Name, "beta")
	}
}
