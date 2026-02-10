package dotfiles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsValidModuleName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"simple lowercase", "tmux", true},
		{"with hyphens", "my-module", true},
		{"with numbers", "node18", true},
		{"complex valid", "my-cool-module-2", true},
		{"uppercase", "MyModule", false},
		{"spaces", "my module", false},
		{"underscore", "my_module", false},
		{"leading hyphen", "-module", false},
		{"trailing hyphen", "module-", false},
		{"double hyphen", "my--module", false},
		{"empty", "", false},
		{"special chars", "my@module", false},
		{"dot", "my.module", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidModuleName(tt.input)
			if got != tt.want {
				t.Errorf("isValidModuleName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGenerateModuleYml(t *testing.T) {
	tests := []struct {
		name     string
		modName  string
		priority int
		depends  []string
		os       []string
		wantSubs []string
	}{
		{
			name:     "basic module",
			modName:  "test",
			priority: 50,
			depends:  []string{},
			os:       []string{},
			wantSubs: []string{"name: test", "priority: 50", "version: 1.0.0"},
		},
		{
			name:     "with dependencies",
			modName:  "tmux",
			priority: 35,
			depends:  []string{"git", "zsh"},
			os:       []string{},
			wantSubs: []string{"name: tmux", "priority: 35", "- git", "- zsh"},
		},
		{
			name:     "with OS filter",
			modName:  "macos-only",
			priority: 50,
			depends:  []string{},
			os:       []string{"macos"},
			wantSubs: []string{"name: macos-only", "- macos"},
		},
		{
			name:     "with custom priority",
			modName:  "early",
			priority: 5,
			depends:  []string{},
			os:       []string{},
			wantSubs: []string{"priority: 5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateModuleYml(tt.modName, tt.priority, tt.depends, tt.os)

			for _, want := range tt.wantSubs {
				if !strings.Contains(got, want) {
					t.Errorf("generateModuleYml() missing expected substring %q.\nGot:\n%s", want, got)
				}
			}

			// Verify it's valid YAML-like structure
			if !strings.Contains(got, "name:") {
				t.Error("generateModuleYml() should contain 'name:' field")
			}
			if !strings.Contains(got, "description:") {
				t.Error("generateModuleYml() should contain 'description:' field")
			}
		})
	}
}

func TestGenerateInstallScript(t *testing.T) {
	tests := []struct {
		name     string
		modName  string
		wantSubs []string
	}{
		{
			name:     "basic module",
			modName:  "test",
			wantSubs: []string{"#!/usr/bin/env bash", "test", "log_info", "set -euo pipefail"},
		},
		{
			name:     "module with hyphen",
			modName:  "my-tool",
			wantSubs: []string{"#!/usr/bin/env bash", "my-tool", "Installing my-tool"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateInstallScript(tt.modName)

			for _, want := range tt.wantSubs {
				if !strings.Contains(got, want) {
					t.Errorf("generateInstallScript() missing expected substring %q", want)
				}
			}

			// Verify script has proper structure
			lines := strings.Split(got, "\n")
			if !strings.HasPrefix(lines[0], "#!") {
				t.Error("Script should start with shebang")
			}
		})
	}
}

func TestGenerateVerifyScript(t *testing.T) {
	script := generateVerifyScript("test-module")

	requiredParts := []string{
		"#!/usr/bin/env bash",
		"verify.sh",
		"test-module",
		"set -euo pipefail",
		"log_info",
	}

	for _, part := range requiredParts {
		if !strings.Contains(script, part) {
			t.Errorf("generateVerifyScript() missing required part: %q", part)
		}
	}
}

func TestGenerateOSScript(t *testing.T) {
	tests := []struct {
		modName string
		osName  string
	}{
		{"test", "macos"},
		{"test", "ubuntu"},
		{"test", "arch"},
		{"my-tool", "macos"},
	}

	for _, tt := range tests {
		t.Run(tt.modName+"_"+tt.osName, func(t *testing.T) {
			got := generateOSScript(tt.modName, tt.osName)

			requiredParts := []string{
				"#!/usr/bin/env bash",
				tt.modName,
				tt.osName,
				"set -euo pipefail",
			}

			for _, part := range requiredParts {
				if !strings.Contains(got, part) {
					t.Errorf("generateOSScript() missing required part: %q", part)
				}
			}
		})
	}
}

func TestGenerateReadme(t *testing.T) {
	tests := []struct {
		name     string
		modName  string
		priority int
		depends  []string
		wantSubs []string
	}{
		{
			name:     "basic",
			modName:  "test",
			priority: 50,
			depends:  []string{},
			wantSubs: []string{"# test Module", "## Features", "## Installation", "dotfiles install test"},
		},
		{
			name:     "with dependencies",
			modName:  "tmux",
			priority: 35,
			depends:  []string{"git", "zsh"},
			wantSubs: []string{"# tmux Module", "depends on:", "- `git`", "- `zsh`"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateReadme(tt.modName, tt.priority, tt.depends)

			for _, want := range tt.wantSubs {
				if !strings.Contains(got, want) {
					t.Errorf("generateReadme() missing expected substring %q", want)
				}
			}
		})
	}
}

func TestGenerateExampleConfig(t *testing.T) {
	config := generateExampleConfig("test")

	requiredParts := []string{
		"Example configuration",
		"test",
		"symlink",
		"copy",
		"template",
	}

	for _, part := range requiredParts {
		if !strings.Contains(config, part) {
			t.Errorf("generateExampleConfig() missing required part: %q", part)
		}
	}
}

func TestNewCommandIntegration(t *testing.T) {
	// This test verifies the complete module generation workflow
	tmpDir := t.TempDir()
	t.Setenv("DOTFILES_DIR", tmpDir)

	moduleName := "test-module"
	moduleDir := filepath.Join(tmpDir, "modules", moduleName)

	// Simulate what the command does
	if err := os.MkdirAll(filepath.Join(moduleDir, "os"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(moduleDir, "files"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Generate files
	files := map[string]string{
		"module.yml":      generateModuleYml(moduleName, 50, []string{}, []string{}),
		"install.sh":      generateInstallScript(moduleName),
		"verify.sh":       generateVerifyScript(moduleName),
		"README.md":       generateReadme(moduleName, 50, []string{}),
		"files/example.conf": generateExampleConfig(moduleName),
		"os/macos.sh":     generateOSScript(moduleName, "macos"),
		"os/ubuntu.sh":    generateOSScript(moduleName, "ubuntu"),
		"os/arch.sh":      generateOSScript(moduleName, "arch"),
	}

	for name, content := range files {
		path := filepath.Join(moduleDir, name)
		perm := os.FileMode(0o644)
		if strings.HasSuffix(name, ".sh") {
			perm = 0o755
		}
		if err := os.WriteFile(path, []byte(content), perm); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	// Verify all files exist
	for name := range files {
		path := filepath.Join(moduleDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("file %s not created: %v", name, err)
		}
	}

	// Verify file permissions on scripts
	for name := range files {
		if strings.HasSuffix(name, ".sh") {
			path := filepath.Join(moduleDir, name)
			info, err := os.Stat(path)
			if err != nil {
				t.Errorf("stat %s: %v", name, err)
				continue
			}
			if info.Mode().Perm() != 0o755 {
				t.Errorf("%s has wrong permissions: got %o, want 0755", name, info.Mode().Perm())
			}
		}
	}

	// Verify module structure
	expectedDirs := []string{"os", "files"}
	for _, dir := range expectedDirs {
		path := filepath.Join(moduleDir, dir)
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			t.Errorf("directory %s not created properly", dir)
		}
	}
}
