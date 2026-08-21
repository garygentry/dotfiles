package module

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeModule creates <root>/<name>/module.yml with the given body and returns
// the module directory. The body should be a valid module.yml unless the test
// wants a parse/validation failure.
func writeModule(t *testing.T, root, name, body string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "module.yml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// validModule returns a minimal valid module.yml with the given name and a
// description marker so tests can tell which root a module came from.
func validModule(name, marker string) string {
	return "name: " + name + "\ndescription: " + marker + "\nversion: \"1.0\"\n"
}

func byName(mods []*Module) map[string]*Module {
	m := make(map[string]*Module, len(mods))
	for _, mod := range mods {
		m[mod.Name] = mod
	}
	return m
}

func TestDiscoverSingleRootUnchanged(t *testing.T) {
	root := t.TempDir()
	writeModule(t, root, "git", validModule("git", "engine-git"))
	writeModule(t, root, "zsh", validModule("zsh", "engine-zsh"))

	mods, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(mods) != 2 {
		t.Fatalf("got %d modules, want 2", len(mods))
	}
	// Discover leaves Source empty; only DiscoverRoots classifies.
	for _, m := range mods {
		if m.Source != "" {
			t.Errorf("Discover set Source=%q on %q, want empty", m.Source, m.Name)
		}
	}
}

func TestDiscoverMissingRootErrors(t *testing.T) {
	// Discover keeps its existing behavior: a missing root is an error.
	_, err := Discover(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("Discover on a missing dir: want error, got nil")
	}
}

func TestDiscoverRootsTwoRootsOverrideAndCustom(t *testing.T) {
	engine := t.TempDir()
	content := t.TempDir()

	writeModule(t, engine, "git", validModule("git", "engine-git"))
	writeModule(t, engine, "zsh", validModule("zsh", "engine-zsh"))
	// content overrides git and adds a custom module.
	writeModule(t, content, "git", validModule("git", "content-git"))
	writeModule(t, content, "mymod", validModule("mymod", "content-mymod"))

	mods, err := DiscoverRoots([]string{engine, content})
	if err != nil {
		t.Fatalf("DiscoverRoots: %v", err)
	}

	m := byName(mods)
	if len(m) != 3 {
		t.Fatalf("got %d modules, want 3 (git, zsh, mymod)", len(m))
	}

	// content-wins: the git module is the content one.
	if got := m["git"].Description; got != "content-git" {
		t.Errorf("git description = %q, want content-git (content should win)", got)
	}
	if got := m["git"].Source; got != SourceOverride {
		t.Errorf("git Source = %q, want %q", got, SourceOverride)
	}
	if got := m["zsh"].Source; got != SourceBuiltin {
		t.Errorf("zsh Source = %q, want %q", got, SourceBuiltin)
	}
	if got := m["mymod"].Source; got != SourceCustom {
		t.Errorf("mymod Source = %q, want %q", got, SourceCustom)
	}
}

func TestDiscoverRootsMissingContentRootIsEngineOnly(t *testing.T) {
	engine := t.TempDir()
	writeModule(t, engine, "git", validModule("git", "engine-git"))

	// Second root does not exist — skipped, not an error.
	mods, err := DiscoverRoots([]string{engine, filepath.Join(t.TempDir(), "nope")})
	if err != nil {
		t.Fatalf("DiscoverRoots: %v", err)
	}
	if len(mods) != 1 || mods[0].Name != "git" {
		t.Fatalf("got %v, want just engine git", mods)
	}
	if mods[0].Source != SourceBuiltin {
		t.Errorf("git Source = %q, want %q", mods[0].Source, SourceBuiltin)
	}
}

func TestDiscoverRootsMalformedContentModuleErrorsWithPath(t *testing.T) {
	engine := t.TempDir()
	content := t.TempDir()
	writeModule(t, engine, "git", validModule("git", "engine-git"))
	// Malformed YAML in the content root.
	badDir := writeModule(t, content, "broken", "name: [this is: not valid")

	_, err := DiscoverRoots([]string{engine, content})
	if err == nil {
		t.Fatal("DiscoverRoots with malformed content module: want error, got nil")
	}
	wantPath := filepath.Join(badDir, "module.yml")
	if got := err.Error(); !strings.Contains(got, wantPath) {
		t.Errorf("error %q does not name the offending file %q", got, wantPath)
	}
}

func TestModuleRoots(t *testing.T) {
	dotfiles := t.TempDir()

	// No content dir → single engine root.
	roots := ModuleRoots(dotfiles, "")
	if len(roots) != 1 || roots[0] != filepath.Join(dotfiles, "modules") {
		t.Fatalf("no content dir: got %v, want [%s/modules]", roots, dotfiles)
	}

	// Content dir set but no modules/ subdir → still single engine root.
	content := t.TempDir()
	roots = ModuleRoots(dotfiles, content)
	if len(roots) != 1 {
		t.Fatalf("content without modules/: got %v, want single root", roots)
	}

	// Content dir with a modules/ subdir → two roots, content last.
	if err := os.MkdirAll(filepath.Join(content, "modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	roots = ModuleRoots(dotfiles, content)
	if len(roots) != 2 {
		t.Fatalf("content with modules/: got %v, want two roots", roots)
	}
	if roots[0] != filepath.Join(dotfiles, "modules") || roots[1] != filepath.Join(content, "modules") {
		t.Errorf("root order wrong: %v (engine must be first, content last)", roots)
	}
}
