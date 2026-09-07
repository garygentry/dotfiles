package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// runComposePipeline replays exactly what install.go does to compose the layered
// config: estate-default (Load) → baseline+overlays (LoadProfileResolved) → host
// (LoadHostConfig), folded by ComposeModules. Kept in one place so a change to the
// pipeline is reflected in the R-CFG-4 and end-to-end tests together.
func runComposePipeline(t *testing.T, dotfilesDir, hostPath string) map[string]map[string]any {
	t.Helper()
	cfg, err := Load(dotfilesDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	_, layers, err := LoadProfileResolved(dotfilesDir, "", cfg.Profile)
	if err != nil {
		t.Fatalf("LoadProfileResolved: %v", err)
	}
	host, err := LoadHostConfig(hostPath)
	if err != nil {
		t.Fatalf("LoadHostConfig: %v", err)
	}
	return ComposeModules(cfg.Modules, layers, host)
}

// Phase 5 (ADR 0028 §D10) — layered config merge. These tests pin the merge
// semantics signed off in phase-5-design.md §D-b and the host-owned layer (§D-c).

// TestComposeModules_Precedence covers a full estate-default → baseline → overlay →
// host fold: scalar override across layers, a scalar overridden back to a falsy value
// (presence-wins), list replacement (not append), and nested-map deep-merge with the
// host winning the final say.
func TestComposeModules_Precedence(t *testing.T) {
	base := map[string]map[string]any{ // estate-default
		"claude-code": {
			"endpoint":  "https://estate.lg",
			"telemetry": true,
			"plugins":   []any{"core"},
			"limits":    map[string]any{"tokens": 1000, "depth": 3},
		},
	}
	baseline := map[string]map[string]any{
		"claude-code": {"endpoint": "https://baseline.lg"}, // scalar override
	}
	overlay := map[string]map[string]any{
		"claude-code": {
			"plugins": []any{"core", "obs"},       // list replacement
			"limits":  map[string]any{"depth": 5}, // nested-map deep-merge (tokens survives)
		},
	}
	host := map[string]map[string]any{
		"claude-code": {
			"endpoint":  "https://host.lg", // host wins the scalar
			"telemetry": false,             // presence-wins to a falsy value
		},
	}

	got := ComposeModules(base, []map[string]map[string]any{baseline, overlay}, host)
	cc := got["claude-code"]

	if cc["endpoint"] != "https://host.lg" {
		t.Errorf("endpoint = %v, want host wins", cc["endpoint"])
	}
	if cc["telemetry"] != false {
		t.Errorf("telemetry = %v, want false (presence-wins to falsy)", cc["telemetry"])
	}
	if !reflect.DeepEqual(cc["plugins"], []any{"core", "obs"}) {
		t.Errorf("plugins = %v, want overlay list (replacement, not append)", cc["plugins"])
	}
	limits, ok := cc["limits"].(map[string]any)
	if !ok {
		t.Fatalf("limits = %T, want map", cc["limits"])
	}
	if limits["depth"] != 5 {
		t.Errorf("limits.depth = %v, want 5 (overridden)", limits["depth"])
	}
	if limits["tokens"] != 1000 {
		t.Errorf("limits.tokens = %v, want 1000 (earlier-only key survives deep-merge)", limits["tokens"])
	}
}

// TestComposeModules_TypeConflict pins last-wins replacement across a type change
// (§D-b): a later scalar/list simply replaces a mismatched earlier value.
func TestComposeModules_TypeConflict(t *testing.T) {
	base := map[string]map[string]any{"m": {"k": "scalar"}}
	overlay := map[string]map[string]any{"m": {"k": []any{"a", "b"}}}

	got := ComposeModules(base, []map[string]map[string]any{overlay}, nil)
	if !reflect.DeepEqual(got["m"]["k"], []any{"a", "b"}) {
		t.Errorf("k = %v, want later list to replace the scalar", got["m"]["k"])
	}

	// And the reverse direction (list → scalar).
	got2 := ComposeModules(
		map[string]map[string]any{"m": {"k": []any{"a"}}},
		[]map[string]map[string]any{{"m": {"k": "scalar"}}}, nil)
	if got2["m"]["k"] != "scalar" {
		t.Errorf("k = %v, want later scalar to replace the list", got2["m"]["k"])
	}
}

// TestComposeModules_NewModule: a layer may introduce a module absent from the base.
func TestComposeModules_NewModule(t *testing.T) {
	base := map[string]map[string]any{"a": {"x": 1}}
	overlay := map[string]map[string]any{"b": {"y": 2}}
	got := ComposeModules(base, []map[string]map[string]any{overlay}, nil)
	if got["a"]["x"] != 1 || got["b"]["y"] != 2 {
		t.Errorf("got %v, want both a.x=1 and new b.y=2", got)
	}
}

// TestComposeModules_InputsUntouched: composing must not mutate any input layer,
// including nested maps and slices (the result owns clones).
func TestComposeModules_InputsUntouched(t *testing.T) {
	base := map[string]map[string]any{"m": {"nested": map[string]any{"a": 1}}}
	overlay := map[string]map[string]any{"m": {"nested": map[string]any{"b": 2}}}

	_ = ComposeModules(base, []map[string]map[string]any{overlay}, nil)

	if bn := base["m"]["nested"].(map[string]any); len(bn) != 1 || bn["a"] != 1 {
		t.Errorf("base nested mutated: %v", bn)
	}
	if on := overlay["m"]["nested"].(map[string]any); len(on) != 1 || on["b"] != 2 {
		t.Errorf("overlay nested mutated: %v", on)
	}
}

func TestLoadHostConfig(t *testing.T) {
	// Empty path and absent file both mean "no host overrides".
	if got, err := LoadHostConfig(""); err != nil || got != nil {
		t.Errorf("empty path: got %v, %v; want nil, nil", got, err)
	}
	if got, err := LoadHostConfig(filepath.Join(t.TempDir(), "nope.yml")); err != nil || got != nil {
		t.Errorf("absent file: got %v, %v; want nil, nil", got, err)
	}

	// A well-formed host file returns its modules bag.
	dir := t.TempDir()
	good := filepath.Join(dir, "local-config.yml")
	if err := os.WriteFile(good, []byte("modules:\n  claude-code:\n    endpoint: https://host.lg\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadHostConfig(good)
	if err != nil {
		t.Fatalf("good file: unexpected error %v", err)
	}
	if got["claude-code"]["endpoint"] != "https://host.lg" {
		t.Errorf("got %v, want endpoint from host file", got)
	}
}

// TestLoadHostConfig_MalformedHardFails: a broken host file is a hard error, never
// silently treated as empty (which would drop the overrides it exists to apply).
func TestLoadHostConfig_MalformedHardFails(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.yml")
	if err := os.WriteFile(bad, []byte("modules: [this, is, not, a, map]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadHostConfig(bad); err == nil {
		t.Error("malformed host config: want error, got nil")
	}
}

// TestLoadHostConfig_MultiDocumentRejected: a host file with a second YAML document
// (a stray `---`) is a hard error — half-applying it would silently drop every
// override after the first (T2), the opposite of the layer's stated contract.
func TestLoadHostConfig_MultiDocumentRejected(t *testing.T) {
	dir := t.TempDir()
	multi := filepath.Join(dir, "multi.yml")
	body := "modules:\n  cc:\n    endpoint: first\n---\nmodules:\n  cc:\n    endpoint: second\n"
	if err := os.WriteFile(multi, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadHostConfig(multi); err == nil {
		t.Error("multi-document host config: want error, got nil")
	}
}

// TestLoadHostConfig_IdentityFenced: identity fields are out of the host layer (§D-c);
// KnownFields makes a host file carrying user:/secrets: a hard error.
func TestLoadHostConfig_IdentityFenced(t *testing.T) {
	dir := t.TempDir()
	for _, field := range []string{"user:\n  email: x@y\n", "secrets:\n  provider: env\n"} {
		p := filepath.Join(dir, "id.yml")
		if err := os.WriteFile(p, []byte(field), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadHostConfig(p); err == nil {
			t.Errorf("host config with %q: want rejection, got nil", field)
		}
	}
}

// TestLoadProfileResolved_ConfigLayerOrder: profile `config:` blocks are collected in
// resolved extends order (parent before child) so the child wins when folded.
func TestLoadProfileResolved_ConfigLayerOrder(t *testing.T) {
	dir := t.TempDir()
	profiles := filepath.Join(dir, "profiles")
	if err := os.MkdirAll(profiles, 0755); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(profiles, name+".yml"), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("base", "modules: [git]\nconfig:\n  modules:\n    cc:\n      endpoint: base\n")
	write("child", "extends: [base]\nmodules: [zsh]\nconfig:\n  modules:\n    cc:\n      endpoint: child\n")

	modules, layers, err := LoadProfileResolved(dir, "", "child")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(modules, []string{"git", "zsh"}) {
		t.Errorf("modules = %v, want [git zsh]", modules)
	}
	if len(layers) != 2 {
		t.Fatalf("layers = %d, want 2", len(layers))
	}
	// Parent first, child second → folding last-wins yields the child's value.
	if layers[0]["cc"]["endpoint"] != "base" || layers[1]["cc"]["endpoint"] != "child" {
		t.Errorf("layer order wrong: %v", layers)
	}
	folded := ComposeModules(nil, layers, nil)
	if folded["cc"]["endpoint"] != "child" {
		t.Errorf("folded endpoint = %v, want child (last-wins)", folded["cc"]["endpoint"])
	}
}

// TestLoadProfileResolved_DiamondDeterministic: a profile reached twice through a
// diamond contributes its config layer once (first-seen), so an intervening override
// is not undone by a re-application of the shared ancestor.
func TestLoadProfileResolved_DiamondDeterministic(t *testing.T) {
	dir := t.TempDir()
	profiles := filepath.Join(dir, "profiles")
	if err := os.MkdirAll(profiles, 0755); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(profiles, name+".yml"), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	// a (k=1) is shared; b overrides k=2 and extends a; top extends [a, b].
	write("a", "modules: [ma]\nconfig:\n  modules:\n    cc:\n      k: 1\n")
	write("b", "extends: [a]\nmodules: [mb]\nconfig:\n  modules:\n    cc:\n      k: 2\n")
	write("top", "extends: [a, b]\nmodules: [mt]\n")

	_, layers, err := LoadProfileResolved(dir, "", "top")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(layers) != 2 {
		t.Fatalf("layers = %d, want 2 (a once, b once)", len(layers))
	}
	folded := ComposeModules(nil, layers, nil)
	if folded["cc"]["k"] != 2 {
		t.Errorf("folded k = %v, want 2 (b's override not undone by a re-application)", folded["cc"]["k"])
	}
}

// TestLoadProfileResolved_IdentityFencedInProfile: a `config:` block on a profile only
// accepts modules: — an identity field there is rejected by the KnownFields decoder.
func TestLoadProfileResolved_IdentityFencedInProfile(t *testing.T) {
	dir := t.TempDir()
	profiles := filepath.Join(dir, "profiles")
	if err := os.MkdirAll(profiles, 0755); err != nil {
		t.Fatal(err)
	}
	body := "modules: [git]\nconfig:\n  user:\n    email: x@y\n"
	if err := os.WriteFile(filepath.Join(profiles, "bad.yml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadProfileResolved(dir, "", "bad"); err == nil {
		t.Error("profile config with user: want rejection, got nil")
	}
}

// TestComposePipeline_EndToEnd exercises the whole layered stack the reconcile uses
// and asserts host beats overlay beats estate-default (D10 precedence, end to end).
func TestComposePipeline_EndToEnd(t *testing.T) {
	t.Setenv("DOTFILES_CONTENT_DIR", "")
	t.Setenv("DOTFILES_PROFILE", "")
	dir := t.TempDir()
	profiles := filepath.Join(dir, "profiles")
	if err := os.MkdirAll(profiles, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yml"),
		[]byte("profile: p\nmodules:\n  cc:\n    endpoint: estate\n    keep: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profiles, "p.yml"),
		[]byte("modules: [cc]\nconfig:\n  modules:\n    cc:\n      endpoint: overlay\n"), 0644); err != nil {
		t.Fatal(err)
	}
	host := filepath.Join(dir, "local-config.yml")
	if err := os.WriteFile(host,
		[]byte("modules:\n  cc:\n    endpoint: host\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got := runComposePipeline(t, dir, host)
	if got["cc"]["endpoint"] != "host" {
		t.Errorf("endpoint = %v, want host (host beats overlay beats estate)", got["cc"]["endpoint"])
	}
	if got["cc"]["keep"] != 1 {
		t.Errorf("keep = %v, want 1 (estate-default key survives all layers)", got["cc"]["keep"])
	}
}

// TestRCFG4_ProjectLocalSettingsUntouched pins R-CFG-4: the layered config pipeline is
// out-of-governance for project-local settings — it never reads or writes a project's
// .claude/settings.local.json. A full compose runs against a seeded project dir; the
// file must be byte-for-byte and mtime unchanged, and no new file may appear.
func TestRCFG4_ProjectLocalSettingsUntouched(t *testing.T) {
	t.Setenv("DOTFILES_CONTENT_DIR", "")
	t.Setenv("DOTFILES_PROFILE", "")
	dir := t.TempDir()
	profiles := filepath.Join(dir, "profiles")
	if err := os.MkdirAll(profiles, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yml"),
		[]byte("profile: p\nmodules:\n  cc:\n    endpoint: estate\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profiles, "p.yml"),
		[]byte("modules: [cc]\nconfig:\n  modules:\n    cc:\n      endpoint: overlay\n"), 0644); err != nil {
		t.Fatal(err)
	}
	host := filepath.Join(dir, "local-config.yml")
	if err := os.WriteFile(host, []byte("modules:\n  cc:\n    endpoint: host\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// A developer's project that happens to live where the reconcile runs.
	project := t.TempDir()
	claudeDir := filepath.Join(project, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	settings := filepath.Join(claudeDir, "settings.local.json")
	want := []byte(`{"permissions":{"allow":["Bash(git status)"]},"private":true}`)
	if err := os.WriteFile(settings, want, 0644); err != nil {
		t.Fatal(err)
	}
	fiBefore, err := os.Stat(settings)
	if err != nil {
		t.Fatal(err)
	}

	// Run the whole layered compose the reconcile uses.
	if got := runComposePipeline(t, dir, host); got["cc"]["endpoint"] != "host" {
		t.Fatalf("pipeline precedence broke: endpoint = %v", got["cc"]["endpoint"])
	}

	// The project-local settings file is untouched: same bytes, same mtime.
	gotBytes, err := os.ReadFile(settings)
	if err != nil {
		t.Fatalf("settings.local.json gone after compose: %v", err)
	}
	if string(gotBytes) != string(want) {
		t.Errorf("settings.local.json content changed:\n got %s\nwant %s", gotBytes, want)
	}
	fiAfter, err := os.Stat(settings)
	if err != nil {
		t.Fatal(err)
	}
	if !fiAfter.ModTime().Equal(fiBefore.ModTime()) {
		t.Errorf("settings.local.json mtime changed: %v -> %v", fiBefore.ModTime(), fiAfter.ModTime())
	}
	// No new files created under the project's .claude dir.
	entries, err := os.ReadDir(claudeDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "settings.local.json" {
		t.Errorf(".claude dir changed: %v, want only settings.local.json", entries)
	}
}
