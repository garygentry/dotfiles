package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain clears DOTFILES_CONTENT_DIR so an ambient overlay on the developer's
// machine cannot leak into tests that don't set it explicitly.
func TestMain(m *testing.M) {
	os.Unsetenv("DOTFILES_CONTENT_DIR")
	os.Exit(m.Run())
}

func TestMergeConfig(t *testing.T) {
	base := &Config{
		Profile: "developer",
		Secrets: SecretsConfig{Provider: "noop"},
		User:    UserConfig{Name: "Base"},
		Modules: map[string]map[string]any{
			"ssh": {"key_type": "ed25519", "key_source": "generate"},
		},
	}
	overlay := &Config{
		Secrets: SecretsConfig{Account: "acct.example.com"},
		User:    UserConfig{Email: "e@x"},
		Modules: map[string]map[string]any{
			"ssh": {"key_source": "agent"},     // per-key override
			"git": {"default_branch": "trunk"}, // new module
		},
	}
	mergeConfig(base, overlay)

	if base.Profile != "developer" {
		t.Errorf("Profile = %q, want unchanged developer", base.Profile)
	}
	if base.Secrets.Provider != "noop" {
		t.Errorf("Provider = %q, want unchanged noop", base.Secrets.Provider)
	}
	if base.Secrets.Account != "acct.example.com" {
		t.Errorf("Account = %q, want overlay value", base.Secrets.Account)
	}
	if base.User.Name != "Base" || base.User.Email != "e@x" {
		t.Errorf("User = %+v, want Name preserved + Email overlaid", base.User)
	}
	if base.Modules["ssh"]["key_type"] != "ed25519" {
		t.Errorf("ssh.key_type = %v, want preserved ed25519", base.Modules["ssh"]["key_type"])
	}
	if base.Modules["ssh"]["key_source"] != "agent" {
		t.Errorf("ssh.key_source = %v, want agent", base.Modules["ssh"]["key_source"])
	}
	if base.Modules["git"]["default_branch"] != "trunk" {
		t.Errorf("git.default_branch = %v, want trunk", base.Modules["git"]["default_branch"])
	}
}

func TestResolveContentDir(t *testing.T) {
	if got := ResolveContentDir(); got != "" {
		t.Errorf("ResolveContentDir with no env = %q, want empty", got)
	}
	t.Setenv("DOTFILES_CONTENT_DIR", "/tmp/my-content")
	if got := ResolveContentDir(); got != "/tmp/my-content" {
		t.Errorf("ResolveContentDir = %q, want /tmp/my-content", got)
	}
}

func TestLoad_ContentOverlay(t *testing.T) {
	dir := t.TempDir()
	base := `
profile: developer
secrets:
  provider: noop
modules:
  ssh:
    key_type: ed25519
    key_source: generate
  git:
    default_branch: main
`
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte(base), 0644); err != nil {
		t.Fatal(err)
	}

	content := t.TempDir()
	overlay := `
secrets:
  provider: 1password
  account: my.1password.com
user:
  email: ada@example.com
modules:
  ssh:
    key_source: agent
`
	if err := os.WriteFile(filepath.Join(content, "config.yml"), []byte(overlay), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTFILES_CONTENT_DIR", content)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.ContentDir != content {
		t.Errorf("ContentDir = %q, want %q", cfg.ContentDir, content)
	}
	// Overlay wins where it sets a value.
	if cfg.Secrets.Provider != "1password" || cfg.Secrets.Account != "my.1password.com" {
		t.Errorf("secrets = %+v, want overlay 1password/my.1password.com", cfg.Secrets)
	}
	if cfg.User.Email != "ada@example.com" {
		t.Errorf("User.Email = %q, want overlay value", cfg.User.Email)
	}
	if cfg.Modules["ssh"]["key_source"] != "agent" {
		t.Errorf("ssh.key_source = %v, want agent (overlay)", cfg.Modules["ssh"]["key_source"])
	}
	// Base preserved where the overlay is silent.
	if cfg.Profile != "developer" {
		t.Errorf("Profile = %q, want developer (base)", cfg.Profile)
	}
	if cfg.Modules["ssh"]["key_type"] != "ed25519" {
		t.Errorf("ssh.key_type = %v, want ed25519 (base)", cfg.Modules["ssh"]["key_type"])
	}
	if cfg.Modules["git"]["default_branch"] != "main" {
		t.Errorf("git.default_branch = %v, want main (base)", cfg.Modules["git"]["default_branch"])
	}
}

func TestLoad_NoOverlayWhenUnset(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte("profile: minimal\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.ContentDir != "" {
		t.Errorf("ContentDir = %q, want empty when DOTFILES_CONTENT_DIR unset", cfg.ContentDir)
	}
	if cfg.Profile != "minimal" {
		t.Errorf("Profile = %q, want minimal", cfg.Profile)
	}
}

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

func TestLoadProfile_NotFound(t *testing.T) {
	dir := t.TempDir()

	_, err := LoadProfile(dir, "nosuchprofile")
	if err == nil {
		t.Fatal("LoadProfile() returned no error for a missing profile")
	}
	if !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("error = %v, want one wrapping ErrProfileNotFound", err)
	}
	// The message must name the file that was looked for; "profile not found" alone
	// leaves the user guessing which of several plausible paths was tried.
	want := filepath.Join(dir, "profiles", "nosuchprofile.yml")
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to mention %q", err.Error(), want)
	}
}

func TestProfileIsPath(t *testing.T) {
	cases := map[string]bool{
		"minimal":                    false,
		"developer":                  false,
		"gnet-lg":                    false,
		"":                           false,
		"profiles/test.yml":          true,
		"./test.yml":                 true,
		"/etc/dotfiles/server.yml":   true,
		"~/workspace/gnet-lg/p.yml":  true,
		"test.yml":                   true,
		"test.yaml":                  true,
		"TEST.YML":                   true,
		"/opt/profiles/no-extension": true,
	}
	for input, want := range cases {
		if got := ProfileIsPath(input); got != want {
			t.Errorf("ProfileIsPath(%q) = %t, want %t", input, got, want)
		}
	}
}

func TestResolveProfilePath(t *testing.T) {
	dir := t.TempDir()

	// A bare name resolves inside the dotfiles repo, as it always has.
	if got, want := ResolveProfilePath(dir, "minimal"), filepath.Join(dir, "profiles", "minimal.yml"); got != want {
		t.Errorf("ResolveProfilePath(bare) = %q, want %q", got, want)
	}

	// An absolute path is taken literally — the point of the feature is that a profile
	// can live outside the dotfiles repo.
	abs := filepath.Join(dir, "elsewhere", "gnet-lg.yml")
	if got := ResolveProfilePath(dir, abs); got != abs {
		t.Errorf("ResolveProfilePath(abs) = %q, want %q", got, abs)
	}

	// A leading ~ expands rather than being treated as a directory literally named "~".
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory available")
	}
	if got, want := ResolveProfilePath(dir, "~/p.yml"), filepath.Join(home, "p.yml"); got != want {
		t.Errorf("ResolveProfilePath(~) = %q, want %q", got, want)
	}
}

func TestLoadProfile_FromPathOutsideDotfilesDir(t *testing.T) {
	dotfilesDir := t.TempDir()
	projectDir := t.TempDir() // stands in for a repo like gnet-lg

	profilePath := filepath.Join(projectDir, "gnet-lg.yml")
	profileYAML := `
modules:
  - git
  - 1password
  - uv
`
	if err := os.WriteFile(profilePath, []byte(profileYAML), 0644); err != nil {
		t.Fatal(err)
	}

	modules, err := LoadProfile(dotfilesDir, profilePath)
	if err != nil {
		t.Fatalf("LoadProfile(path) error: %v", err)
	}

	expected := []string{"git", "1password", "uv"}
	if len(modules) != len(expected) {
		t.Fatalf("len(modules) = %d, want %d", len(modules), len(expected))
	}
	for i, mod := range modules {
		if mod != expected[i] {
			t.Errorf("modules[%d] = %q, want %q", i, mod, expected[i])
		}
	}

	// A path that does not exist must still be ErrProfileNotFound, so the caller can
	// fail fast on it rather than silently installing everything.
	if _, err := LoadProfile(dotfilesDir, filepath.Join(projectDir, "absent.yml")); !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("error = %v, want one wrapping ErrProfileNotFound", err)
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
