package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
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

	if cfg.Profile != "minimal" {
		t.Errorf("Profile = %q, want default %q", cfg.Profile, "minimal")
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

	modules, err := LoadProfile(dir, "", "test")
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

	_, err := LoadProfile(dir, "", "nosuchprofile")
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
	if got, want := ResolveProfilePath(dir, "", "minimal"), filepath.Join(dir, "profiles", "minimal.yml"); got != want {
		t.Errorf("ResolveProfilePath(bare) = %q, want %q", got, want)
	}

	// An absolute path is taken literally — the point of the feature is that a profile
	// can live outside the dotfiles repo.
	abs := filepath.Join(dir, "elsewhere", "gnet-lg.yml")
	if got := ResolveProfilePath(dir, "", abs); got != abs {
		t.Errorf("ResolveProfilePath(abs) = %q, want %q", got, abs)
	}

	// A leading ~ expands rather than being treated as a directory literally named "~".
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory available")
	}
	if got, want := ResolveProfilePath(dir, "", "~/p.yml"), filepath.Join(home, "p.yml"); got != want {
		t.Errorf("ResolveProfilePath(~) = %q, want %q", got, want)
	}
}

func TestResolveProfilePath_ContentDirOverrides(t *testing.T) {
	engine := t.TempDir()
	content := t.TempDir()
	if err := os.MkdirAll(filepath.Join(engine, "profiles"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(engine, "profiles", "mine.yml"), []byte("modules: [git]\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// With no content profile of that name, resolves to the engine's.
	if got, want := ResolveProfilePath(engine, content, "mine"), filepath.Join(engine, "profiles", "mine.yml"); got != want {
		t.Errorf("no content profile: got %q, want %q", got, want)
	}

	// A content profile of the same name overrides the built-in.
	if err := os.MkdirAll(filepath.Join(content, "profiles"), 0755); err != nil {
		t.Fatal(err)
	}
	cp := filepath.Join(content, "profiles", "mine.yml")
	if err := os.WriteFile(cp, []byte("modules: [zsh]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := ResolveProfilePath(engine, content, "mine"); got != cp {
		t.Errorf("content override: got %q, want %q", got, cp)
	}

	// A name that exists only in the content dir also resolves there.
	custom := filepath.Join(content, "profiles", "custom.yml")
	if err := os.WriteFile(custom, []byte("modules: [tmux]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := ResolveProfilePath(engine, content, "custom"); got != custom {
		t.Errorf("content-only: got %q, want %q", got, custom)
	}
}

func TestLoadProfile_ContentDirOverride(t *testing.T) {
	engine := t.TempDir()
	content := t.TempDir()
	if err := os.MkdirAll(filepath.Join(engine, "profiles"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(engine, "profiles", "dev.yml"), []byte("modules:\n  - git\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(content, "profiles"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(content, "profiles", "dev.yml"), []byte("modules:\n  - zsh\n  - tmux\n"), 0644); err != nil {
		t.Fatal(err)
	}

	mods, err := LoadProfile(engine, content, "dev")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if len(mods) != 2 || mods[0] != "zsh" || mods[1] != "tmux" {
		t.Errorf("got %v, want [zsh tmux] from content override", mods)
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

	modules, err := LoadProfile(dotfilesDir, "", profilePath)
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
	if _, err := LoadProfile(dotfilesDir, "", filepath.Join(projectDir, "absent.yml")); !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("error = %v, want one wrapping ErrProfileNotFound", err)
	}
}

// writeProfile is a tiny helper for the extends tests below — the flat profile tests
// above deliberately spell out the writes to stay legible in isolation.
func writeProfile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "profiles"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "profiles", name+".yml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

// TestLoadProfile_ExtendsSingle covers the smallest extends composition: one child
// extends one parent, both contributing modules. The child's modules follow the
// parent's, in first-seen order.
func TestLoadProfile_ExtendsSingle(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "baseline", "modules:\n  - git\n  - tmux\n")
	writeProfile(t, dir, "workstation", "extends: [baseline]\nmodules:\n  - zsh\n  - neovim\n")

	got, err := LoadProfile(dir, "", "workstation")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	want := []string{"git", "tmux", "zsh", "neovim"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestLoadProfile_ExtendsAlias covers the common migration shape: an alias profile
// with only `extends:` and no `modules:` of its own resolves to exactly its parent's
// set. This is the shape a fleet-floor rename uses to preserve caller behavior when
// a profile name changes without changing its module contents.
func TestLoadProfile_ExtendsAlias(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "baseline", "modules:\n  - git\n  - tmux\n  - starship\n")
	writeProfile(t, dir, "alias", "extends: [baseline]\n")

	got, err := LoadProfile(dir, "", "alias")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	direct, err := LoadProfile(dir, "", "baseline")
	if err != nil {
		t.Fatalf("LoadProfile baseline: %v", err)
	}
	if !reflect.DeepEqual(got, direct) {
		t.Errorf("alias resolved to %v, want set-equal to baseline %v", got, direct)
	}
}

// TestLoadProfile_ExtendsDiamond covers the diamond case: two parents extend a shared
// grandparent. Modules from the shared grandparent must appear exactly once, and each
// parent's own additions must appear once (first-seen wins). This is the shape a
// multi-role host uses when it expresses `extends: [gpu, workstation]` on a profile
// whose parents both extend baseline.
func TestLoadProfile_ExtendsDiamond(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "baseline", "modules:\n  - git\n  - tmux\n")
	writeProfile(t, dir, "workstation", "extends: [baseline]\nmodules:\n  - zsh\n")
	writeProfile(t, dir, "gpu", "extends: [baseline]\nmodules:\n  - nvtop\n")
	writeProfile(t, dir, "host", "extends: [workstation, gpu]\n")

	got, err := LoadProfile(dir, "", "host")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	// baseline modules first (via workstation), then zsh, then baseline seen again via
	// gpu (skipped as duplicates), then nvtop.
	want := []string{"git", "tmux", "zsh", "nvtop"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestLoadProfile_ExtendsMultiLevel covers chains deeper than one level (a >  b > c).
// The engine handles arbitrary depth via the recursive resolver.
func TestLoadProfile_ExtendsMultiLevel(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "c", "modules:\n  - alpha\n")
	writeProfile(t, dir, "b", "extends: [c]\nmodules:\n  - beta\n")
	writeProfile(t, dir, "a", "extends: [b]\nmodules:\n  - gamma\n")

	got, err := LoadProfile(dir, "", "a")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	want := []string{"alpha", "beta", "gamma"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestLoadProfile_ExtendsCycle covers direct and indirect cycles. The engine must
// fail at load with ErrProfileCycle, not recurse until the stack blows up.
func TestLoadProfile_ExtendsCycle(t *testing.T) {
	dir := t.TempDir()

	// Direct self-cycle: a extends a.
	writeProfile(t, dir, "a", "extends: [a]\n")
	if _, err := LoadProfile(dir, "", "a"); !errors.Is(err, ErrProfileCycle) {
		t.Errorf("self-cycle: err = %v, want one wrapping ErrProfileCycle", err)
	}

	// Indirect cycle: b extends c, c extends b.
	writeProfile(t, dir, "b", "extends: [c]\n")
	writeProfile(t, dir, "c", "extends: [b]\n")
	if _, err := LoadProfile(dir, "", "b"); !errors.Is(err, ErrProfileCycle) {
		t.Errorf("indirect cycle: err = %v, want one wrapping ErrProfileCycle", err)
	}
}

// TestLoadProfile_FlatBehavior pins the flat-profile (no `extends:`) shape: the module
// list is returned in file order; duplicates within a single list are collapsed (a
// small behaviour change vs. the pre-extends form, documented on LoadProfile); an
// empty list resolves to nil.
func TestLoadProfile_FlatBehavior(t *testing.T) {
	t.Run("ordered no duplicates", func(t *testing.T) {
		dir := t.TempDir()
		writeProfile(t, dir, "flat", "modules:\n  - git\n  - zsh\n  - neovim\n")
		got, err := LoadProfile(dir, "", "flat")
		if err != nil {
			t.Fatalf("LoadProfile: %v", err)
		}
		if want := []string{"git", "zsh", "neovim"}; !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})
	t.Run("duplicates within one list are deduped", func(t *testing.T) {
		// Pre-extends: this returned [git, tmux, git]. Post-extends: dedupe runs on
		// every load, so the flat form loses the duplicate too. Downstream treats
		// the list as a set, so this is safe — but pinned here so anyone reasoning
		// about the caller sees the actual behaviour, not the old one.
		dir := t.TempDir()
		writeProfile(t, dir, "dup", "modules:\n  - git\n  - tmux\n  - git\n")
		got, err := LoadProfile(dir, "", "dup")
		if err != nil {
			t.Fatalf("LoadProfile: %v", err)
		}
		if want := []string{"git", "tmux"}; !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v (duplicate should be collapsed)", got, want)
		}
	})
}

// TestLoadProfile_ExtendsMissingParent pins the three properties the missing-parent
// error path must have: it wraps ErrProfileNotFound (not ErrProfileCycle), the error
// mentions the child profile that referenced the missing parent (so a deep chain
// isn't opaque), and the returned slice is nil (not a silent empty set).
func TestLoadProfile_ExtendsMissingParent(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "child", "extends: [nosuch]\nmodules:\n  - git\n")

	mods, err := LoadProfile(dir, "", "child")
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("err = %v, want one wrapping ErrProfileNotFound", err)
	}
	if errors.Is(err, ErrProfileCycle) {
		t.Errorf("err = %v, unexpectedly matches ErrProfileCycle", err)
	}
	if mods != nil {
		t.Errorf("modules = %v, want nil on error", mods)
	}
	if !strings.Contains(err.Error(), "child") {
		t.Errorf("err = %q, want it to name the originating child profile", err.Error())
	}
	if !strings.Contains(err.Error(), "nosuch") {
		t.Errorf("err = %q, want it to name the missing parent", err.Error())
	}
}

// TestLoadProfile_ExtendsAcrossContentBoundary pins that content-over-engine precedence
// applies to *parents* pulled via `extends:`, not just to the top-level profile. Without
// this, a refactor of ResolveProfilePath could quietly change parent resolution while the
// rest of the extends suite (which writes everything to a single dir) stayed green.
func TestLoadProfile_ExtendsAcrossContentBoundary(t *testing.T) {
	engine := t.TempDir()
	content := t.TempDir()
	writeProfile(t, engine, "baseline", "modules:\n  - engine-baseline-only\n")
	writeProfile(t, content, "baseline", "modules:\n  - content-baseline-only\n")
	writeProfile(t, content, "child", "extends: [baseline]\nmodules:\n  - childmod\n")

	got, err := LoadProfile(engine, content, "child")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	// The content-dir baseline wins over the engine baseline (matches
	// ResolveProfilePath's precedence when the parent is a bare name).
	want := []string{"content-baseline-only", "childmod"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v (content baseline should shadow engine baseline)", got, want)
	}
}

// TestLoadProfile_ExtendsRejectsUnknownField covers the highest-severity silent-failure
// class the extends primitive would otherwise introduce: a typo of `extend:` (singular)
// or any other unrecognised top-level key would parse silently under yaml.Unmarshal's
// default mode, dropping the inheritance and installing the wrong module set.
// KnownFields(true) rejects them at parse time.
func TestLoadProfile_ExtendsRejectsUnknownField(t *testing.T) {
	dir := t.TempDir()

	t.Run("singular extend typo", func(t *testing.T) {
		writeProfile(t, dir, "typo", "extend: [baseline]\nmodules: [git]\n")
		if _, err := LoadProfile(dir, "", "typo"); err == nil {
			t.Errorf("LoadProfile: expected error on `extend:` typo, got nil")
		}
	})
	t.Run("unrecognised top-level key", func(t *testing.T) {
		writeProfile(t, dir, "unknown", "requires: [foo]\nmodules: [git]\n")
		if _, err := LoadProfile(dir, "", "unknown"); err == nil {
			t.Errorf("LoadProfile: expected error on unknown top-level key, got nil")
		}
	})
}

// TestLoadProfile_EmptyExtendsElement covers the case where an `extends:` list contains
// an empty string (`extends: [""]` or `extends: [baseline, ""]`) — a straightforward
// authoring mistake that would otherwise yield a confusing "profile not found:
// /…/profiles/.yml" error. Reject at load with a clear message.
func TestLoadProfile_EmptyExtendsElement(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "baseline", "modules: [git]\n")
	writeProfile(t, dir, "child", "extends: [\"\"]\nmodules: [zsh]\n")
	if _, err := LoadProfile(dir, "", "child"); err == nil {
		t.Errorf("LoadProfile: expected error on empty parent name, got nil")
	}
}

// TestLoadProfile_EmptyProfileFile covers a profile file that declares neither
// `extends:` nor `modules:`. Almost always a typo; silent empty membership is exactly
// the outcome we don't want.
func TestLoadProfile_EmptyProfileFile(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "empty", "# nothing here\n")
	if _, err := LoadProfile(dir, "", "empty"); err == nil {
		t.Errorf("LoadProfile: expected error on all-empty profile, got nil")
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
