package dotfiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/garygentry/dotfiles/internal/module"
	"github.com/garygentry/dotfiles/internal/state"
	"github.com/garygentry/dotfiles/internal/ui"
)

func TestLoadAdditionsManifest(t *testing.T) {
	dir := t.TempDir()

	// Empty path → empty allowlist, no error (no manifest configured).
	if got, err := loadAdditionsManifest(""); err != nil || len(got) != 0 {
		t.Fatalf("empty path: got %v err %v; want empty,nil", got, err)
	}

	// Absent file → empty allowlist, no error (host simply has no additions).
	if got, err := loadAdditionsManifest(filepath.Join(dir, "nope.yml")); err != nil || len(got) != 0 {
		t.Fatalf("absent file: got %v err %v; want empty,nil", got, err)
	}

	// Empty file → empty allowlist, no error.
	empty := filepath.Join(dir, "empty.yml")
	if err := os.WriteFile(empty, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := loadAdditionsManifest(empty); err != nil || len(got) != 0 {
		t.Fatalf("empty file: got %v err %v; want empty,nil", got, err)
	}

	// Valid manifest → the listed modules.
	valid := filepath.Join(dir, "valid.yml")
	if err := os.WriteFile(valid, []byte("modules:\n  - foo\n  - bar\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := loadAdditionsManifest(valid)
	if err != nil {
		t.Fatalf("valid: unexpected err %v", err)
	}
	if !got["foo"] || !got["bar"] || len(got) != 2 {
		t.Fatalf("valid: got %v; want {foo,bar}", got)
	}

	// Malformed (unknown key — the `module:` singular footgun) → HARD error, so a
	// broken manifest can never be treated as "nothing to protect".
	bad := filepath.Join(dir, "bad.yml")
	if err := os.WriteFile(bad, []byte("module:\n  - foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadAdditionsManifest(bad); err == nil {
		t.Fatal("malformed manifest: expected error, got nil")
	}

	// Multi-document manifest → HARD error: a stray `---` would silently drop every
	// protection after the first document, wrongly pruning those modules (T2).
	multi := filepath.Join(dir, "multi.yml")
	if err := os.WriteFile(multi, []byte("modules:\n  - foo\n---\nmodules:\n  - bar\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadAdditionsManifest(multi); err == nil {
		t.Fatal("multi-document manifest: expected error, got nil")
	}
}

func TestComputePruneCandidates(t *testing.T) {
	store := state.NewStore(t.TempDir())
	set := func(name, status string) {
		if err := store.Set(&state.ModuleState{Name: name, Status: status}); err != nil {
			t.Fatal(err)
		}
	}
	set("keep", "installed")      // in effective set
	set("protected", "installed") // in manifest
	set("extra", "installed")     // → candidate
	set("failed", "failed")       // not installed → never a candidate

	effective := map[string]bool{"keep": true}
	protect := map[string]bool{"protected": true}

	cands, err := computePruneCandidates(store, effective, protect)
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 1 || cands[0] != "extra" {
		t.Fatalf("got %v; want [extra]", cands)
	}
}

func TestPruneOptIn(t *testing.T) {
	dir := t.TempDir()
	if hasPruneOptIn(dir) {
		t.Fatal("fresh host should not be opted in")
	}
	if err := recordPruneOptIn(dir); err != nil {
		t.Fatal(err)
	}
	if !hasPruneOptIn(dir) {
		t.Fatal("after record, host should be opted in")
	}
	if _, err := os.Stat(pruneOptInPath(dir)); err != nil {
		t.Fatalf("marker file missing: %v", err)
	}
}

func TestRemoveModuleForPrune(t *testing.T) {
	dir := t.TempDir()
	store := state.NewStore(dir)
	u := ui.New(false)

	// A module that "created" a file: record the op so rollback removes it.
	target := filepath.Join(dir, "deployed.conf")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ms := &state.ModuleState{Name: "victim", Status: "installed"}
	ms.RecordOperation(state.Operation{Type: "file_deploy", Action: "created", Path: target})
	if err := store.Set(ms); err != nil {
		t.Fatal(err)
	}

	// dry-run: no-op — file stays, state stays.
	dryRun = true
	if errs := removeModuleForPrune(u, store, ms); len(errs) != 0 {
		t.Fatalf("dry-run errs: %v", errs)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatal("dry-run must not remove the file")
	}
	if got, _ := store.Get("victim"); got == nil {
		t.Fatal("dry-run must not remove state")
	}

	// real run: file removed, state dropped.
	dryRun = false
	if errs := removeModuleForPrune(u, store, ms); len(errs) != 0 {
		t.Fatalf("real errs: %v", errs)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("file should be removed, stat err: %v", err)
	}
	if got, _ := store.Get("victim"); got != nil {
		t.Fatal("state should be removed")
	}
}

// A failed rollback must PRESERVE state so the module stays a prune candidate for
// the next reconcile — never an untracked orphan (file left, state gone).
func TestRemoveModuleForPrunePreservesStateOnRollbackError(t *testing.T) {
	dir := t.TempDir()
	store := state.NewStore(dir)
	u := ui.New(false)
	dryRun = false

	// Record a "created" op whose Path is a NON-EMPTY directory: rollback does
	// os.Remove(path), which fails on a non-empty dir → an op error.
	busyDir := filepath.Join(dir, "busy")
	if err := os.MkdirAll(busyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(busyDir, "child"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ms := &state.ModuleState{Name: "stuck", Status: "installed"}
	ms.RecordOperation(state.Operation{Type: "file_deploy", Action: "created", Path: busyDir})
	if err := store.Set(ms); err != nil {
		t.Fatal(err)
	}

	errs := removeModuleForPrune(u, store, ms)
	if len(errs) == 0 {
		t.Fatal("expected a rollback error for a non-empty dir")
	}
	if got, _ := store.Get("stuck"); got == nil {
		t.Fatal("state must be PRESERVED on rollback error (no orphaning)")
	}
}

// Prune must remove only what the pruned module still owns (#60): the issue's
// case is a stale neovim record whose init.lua nvim-config now deploys.
func TestRemoveModuleForPruneKeepsPathsNoLongerItsOwn(t *testing.T) {
	dir := t.TempDir()
	store := state.NewStore(filepath.Join(dir, ".state"))
	u := ui.New(false)
	dryRun = false

	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	hash := func(p string) string {
		h, err := module.ComputeFileHash(p)
		if err != nil {
			t.Fatal(err)
		}
		return h
	}

	shared := write("init.lua", "nvim-config's")
	edited := write("edited.conf", "as deployed")
	editedHash := hash(edited)
	write("edited.conf", "edited by hand")
	unchanged := write("unchanged.conf", "as deployed")
	src := write("src.conf", "source")
	other := write("other.conf", "other source")
	goodLink := filepath.Join(dir, "good.link")
	movedLink := filepath.Join(dir, "moved.link")
	replacedLink := filepath.Join(dir, "replaced.link")
	for _, l := range []struct{ path, target string }{{goodLink, src}, {movedLink, other}} {
		if err := os.Symlink(l.target, l.path); err != nil {
			t.Fatal(err)
		}
	}
	write("replaced.link", "a real file now")

	owner := &state.ModuleState{Name: "nvim-config", Status: "installed"}
	owner.RecordOperation(state.Operation{Type: "file_deploy", Action: "created", Path: shared})
	ms := &state.ModuleState{Name: "neovim", Status: "installed", FileStates: []state.FileState{
		{Dest: edited, DeployedHash: editedHash},
		{Dest: unchanged, DeployedHash: hash(unchanged)},
	}}
	ms.RecordOperation(state.Operation{Type: "file_deploy", Action: "created", Path: shared})
	ms.RecordOperation(state.Operation{Type: "file_deploy", Action: "created", Path: edited})
	ms.RecordOperation(state.Operation{Type: "file_deploy", Action: "created", Path: unchanged})
	for _, l := range []string{goodLink, movedLink, replacedLink} {
		ms.RecordOperation(state.Operation{Type: "file_deploy", Action: "symlinked", Path: l,
			Metadata: map[string]string{"source": src}})
	}
	ms.RecordOperation(state.Operation{Type: "script_run", Action: "executed", Path: "install.sh"})
	for _, s := range []*state.ModuleState{owner, ms} {
		if err := store.Set(s); err != nil {
			t.Fatal(err)
		}
	}

	if errs := removeModuleForPrune(u, store, ms); len(errs) != 0 {
		t.Fatalf("errs: %v", errs)
	}
	for _, p := range []string{shared, edited, movedLink, replacedLink} {
		if _, err := os.Lstat(p); err != nil {
			t.Errorf("%s must be kept: %v", filepath.Base(p), err)
		}
	}
	for _, p := range []string{unchanged, goodLink} {
		if _, err := os.Lstat(p); !os.IsNotExist(err) {
			t.Errorf("%s must be removed, lstat err: %v", filepath.Base(p), err)
		}
	}
	if got, _ := store.Get("neovim"); got != nil {
		t.Error("state should be removed")
	}
	if !leftInstalled(ms) {
		t.Error("leftInstalled: a module that ran install.sh leaves software behind")
	}
	if leftInstalled(owner) {
		t.Error("leftInstalled: a files-only module leaves nothing behind")
	}
}
