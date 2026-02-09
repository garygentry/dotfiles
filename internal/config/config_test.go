package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()

	configYAML := `
profile: server

secrets:
  provider: vault
  account: vault.example.com

user:
  name: Test User
  email: test@example.com
  github_user: testuser

modules:
  git:
    default_branch: main
  ssh:
    key_type: ed25519
`
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Profile != "server" {
		t.Errorf("Profile = %q, want %q", cfg.Profile, "server")
	}
	if cfg.DotfilesDir != dir {
		t.Errorf("DotfilesDir = %q, want %q", cfg.DotfilesDir, dir)
	}
	if cfg.Secrets.Provider != "vault" {
		t.Errorf("Secrets.Provider = %q, want %q", cfg.Secrets.Provider, "vault")
	}
	if cfg.Secrets.Account != "vault.example.com" {
		t.Errorf("Secrets.Account = %q, want %q", cfg.Secrets.Account, "vault.example.com")
	}
	if cfg.User.Name != "Test User" {
		t.Errorf("User.Name = %q, want %q", cfg.User.Name, "Test User")
	}
	if cfg.User.Email != "test@example.com" {
		t.Errorf("User.Email = %q, want %q", cfg.User.Email, "test@example.com")
	}
	if cfg.User.GithubUser != "testuser" {
		t.Errorf("User.GithubUser = %q, want %q", cfg.User.GithubUser, "testuser")
	}
	if len(cfg.Modules) != 2 {
		t.Errorf("len(Modules) = %d, want 2", len(cfg.Modules))
	}
}

func TestLoad_Defaults(t *testing.T) {
	dir := t.TempDir()

	// Minimal config with no profile set.
	configYAML := `
modules: {}
`
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Profile != "developer" {
		t.Errorf("Profile = %q, want default %q", cfg.Profile, "developer")
	}
}

func TestLoadProfile(t *testing.T) {
	dir := t.TempDir()

	profileDir := filepath.Join(dir, "profiles")
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		t.Fatal(err)
	}

	profileYAML := `
modules:
  - git
  - zsh
  - neovim
`
	if err := os.WriteFile(filepath.Join(profileDir, "test.yml"), []byte(profileYAML), 0644); err != nil {
		t.Fatal(err)
	}

	modules, err := LoadProfile(dir, "test")
	if err != nil {
		t.Fatalf("LoadProfile() error: %v", err)
	}

	expected := []string{"git", "zsh", "neovim"}
	if len(modules) != len(expected) {
		t.Fatalf("len(modules) = %d, want %d", len(modules), len(expected))
	}
	for i, mod := range modules {
		if mod != expected[i] {
			t.Errorf("modules[%d] = %q, want %q", i, mod, expected[i])
		}
	}
}

func TestEnvVarOverrides(t *testing.T) {
	dir := t.TempDir()

	configYAML := `
profile: server

secrets:
  provider: vault
  account: vault.example.com
`
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DOTFILES_PROFILE", "minimal")
	t.Setenv("DOTFILES_SECRETS_PROVIDER", "bitwarden")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Profile != "minimal" {
		t.Errorf("Profile = %q, want %q (env override)", cfg.Profile, "minimal")
	}
	if cfg.Secrets.Provider != "bitwarden" {
		t.Errorf("Secrets.Provider = %q, want %q (env override)", cfg.Secrets.Provider, "bitwarden")
	}
	// Account should remain unchanged.
	if cfg.Secrets.Account != "vault.example.com" {
		t.Errorf("Secrets.Account = %q, want %q", cfg.Secrets.Account, "vault.example.com")
	}
}

func TestGetModuleSetting(t *testing.T) {
	cfg := &Config{
		Modules: map[string]map[string]any{
			"ssh": {
				"key_type": "ed25519",
			},
			"git": {
				"default_branch": "main",
			},
		},
	}

	// Existing module and key.
	val, ok := cfg.GetModuleSetting("ssh", "key_type")
	if !ok {
		t.Fatal("GetModuleSetting(ssh, key_type) returned false, want true")
	}
	if val != "ed25519" {
		t.Errorf("GetModuleSetting(ssh, key_type) = %v, want %q", val, "ed25519")
	}

	// Existing module, missing key.
	_, ok = cfg.GetModuleSetting("ssh", "nonexistent")
	if ok {
		t.Error("GetModuleSetting(ssh, nonexistent) returned true, want false")
	}

	// Missing module.
	_, ok = cfg.GetModuleSetting("docker", "anything")
	if ok {
		t.Error("GetModuleSetting(docker, anything) returned true, want false")
	}
}
