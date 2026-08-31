package module

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/garygentry/dotfiles/internal/config"
	"github.com/garygentry/dotfiles/internal/secrets"
	"github.com/garygentry/dotfiles/internal/state"
	"github.com/garygentry/dotfiles/internal/sysinfo"
)

// testUI is a minimal RunnerUI implementation for testing that records
// calls but does not produce terminal output.
type testUI struct {
	infos     []string
	warns     []string
	errs      []string
	successes []string
	debugs    []string
	verbose   bool
}

func (t *testUI) Info(msg string)    { t.infos = append(t.infos, msg) }
func (t *testUI) Warn(msg string)    { t.warns = append(t.warns, msg) }
func (t *testUI) Error(msg string)   { t.errs = append(t.errs, msg) }
func (t *testUI) Success(msg string) { t.successes = append(t.successes, msg) }
func (t *testUI) Debug(msg string)   { t.debugs = append(t.debugs, msg) }

func (t *testUI) PromptInput(_ string, defaultVal string) (string, error) {
	return defaultVal, nil
}
func (t *testUI) PromptConfirm(_ string, defaultVal bool) (bool, error) {
	return defaultVal, nil
}
func (t *testUI) PromptChoice(_ string, options []string) (string, error) {
	if len(options) > 0 {
		return options[0], nil
	}
	return "", nil
}

func (t *testUI) StartSpinner(_ string) any          { return nil }
func (t *testUI) StopSpinnerSuccess(_ any, _ string) {}
func (t *testUI) StopSpinnerFail(_ any, _ string)    {}
func (t *testUI) StopSpinnerSkip(_ any, _ string)    {}

func (t *testUI) PromptMultiSelect(_ string, _ []MultiSelectOption, preSelected []string) ([]string, error) {
	return preSelected, nil
}

func (t *testUI) PrintCollapsedOutput(scriptName, output string)                   {}
func (t *testUI) StartProgressBar(total int) ProgressTracker                       { return nil }
func (t *testUI) UpdateProgress(p ProgressTracker, current int, moduleName string) {}
func (t *testUI) RecordModuleTime(p ProgressTracker, duration time.Duration)       {}
func (t *testUI) FinishProgress(p ProgressTracker, summary *ProgressSummary)       {}

// newTestRunConfig returns a RunConfig suitable for unit tests. It uses
// temp directories for state and dotfiles, a test UI, and the noop secrets
// provider.
func newTestRunConfig(t *testing.T) *RunConfig {
	t.Helper()

	stateDir := t.TempDir()
	dotfilesDir := t.TempDir()

	homeDir := t.TempDir()

	return &RunConfig{
		SysInfo: &sysinfo.SystemInfo{
			OS:            "linux",
			Arch:          "amd64",
			PkgMgr:        "apt",
			HasSudo:       true,
			User:          "testuser",
			HomeDir:       homeDir,
			XDGConfigHome: filepath.Join(homeDir, ".config"),
			DotfilesDir:   dotfilesDir,
		},
		Config: &config.Config{
			Profile:     "test",
			DotfilesDir: dotfilesDir,
			User: config.UserConfig{
				Name:       "Test User",
				Email:      "test@example.com",
				GithubUser: "testuser",
			},
			Modules: make(map[string]map[string]any),
		},
		UI:         &testUI{verbose: true},
		Secrets:    secrets.NewProvider("", ""),
		State:      state.NewStore(stateDir),
		DryRun:     false,
		Unattended: true,
		FailFast:   false,
		Verbose:    true,
	}
}

func TestBuildEnvVars(t *testing.T) {
	cfg := newTestRunConfig(t)

	mod := &Module{
		Name: "test-module",
		Dir:  "/tmp/modules/test-module",
	}

	promptAnswers := map[string]string{
		"editor":    "vim",
		"use_color": "true",
	}

	env := buildEnvVars(cfg, mod, promptAnswers)

	// Verify standard DOTFILES_* variables.
	checks := map[string]string{
		"DOTFILES_OS":              "linux",
		"DOTFILES_ARCH":            "amd64",
		"DOTFILES_PKG_MGR":         "apt",
		"DOTFILES_HAS_SUDO":        "true",
		"DOTFILES_HOME":            cfg.SysInfo.HomeDir,
		"DOTFILES_XDG_CONFIG_HOME": cfg.SysInfo.XDGConfigHome,
		"DOTFILES_DIR":             cfg.SysInfo.DotfilesDir,
		"DOTFILES_MODULE_DIR":      "/tmp/modules/test-module",
		"DOTFILES_MODULE_NAME":     "test-module",
		"DOTFILES_INTERACTIVE":     "false", // Unattended=true => interactive=false
		"DOTFILES_DRY_RUN":         "false",
		"DOTFILES_VERBOSE":         "true",
	}

	for key, want := range checks {
		got, ok := env[key]
		if !ok {
			t.Errorf("expected env var %s to be present", key)
			continue
		}
		if got != want {
			t.Errorf("env[%s] = %q, want %q", key, got, want)
		}
	}

	// Verify prompt answers are uppercased and prefixed.
	if got := env["DOTFILES_PROMPT_EDITOR"]; got != "vim" {
		t.Errorf("env[DOTFILES_PROMPT_EDITOR] = %q, want %q", got, "vim")
	}
	if got := env["DOTFILES_PROMPT_USE_COLOR"]; got != "true" {
		t.Errorf("env[DOTFILES_PROMPT_USE_COLOR] = %q, want %q", got, "true")
	}

	// DOTFILES_BIN should be present (value depends on test binary).
	if _, ok := env["DOTFILES_BIN"]; !ok {
		t.Error("expected DOTFILES_BIN to be present")
	}

	// Verify user config values are exported.
	if got := env["DOTFILES_USER_NAME"]; got != "Test User" {
		t.Errorf("env[DOTFILES_USER_NAME] = %q, want %q", got, "Test User")
	}
	if got := env["DOTFILES_USER_EMAIL"]; got != "test@example.com" {
		t.Errorf("env[DOTFILES_USER_EMAIL] = %q, want %q", got, "test@example.com")
	}
	if got := env["DOTFILES_USER_GITHUB_USER"]; got != "testuser" {
		t.Errorf("env[DOTFILES_USER_GITHUB_USER] = %q, want %q", got, "testuser")
	}
}

func TestBuildEnvVarsModuleSettings(t *testing.T) {
	cfg := newTestRunConfig(t)
	cfg.Config.Modules["ssh"] = map[string]any{
		"key_source": "agent",
		"key_type":   "ed25519",
		"enabled":    true,
		"count":      3,
	}

	mod := &Module{Name: "ssh", Dir: "/tmp/modules/ssh"}

	env := buildEnvVars(cfg, mod, nil)

	// Settings are exposed as DOTFILES_SETTING_*, with scalars stringified.
	checks := map[string]string{
		"DOTFILES_SETTING_KEY_SOURCE": "agent",
		"DOTFILES_SETTING_KEY_TYPE":   "ed25519",
		"DOTFILES_SETTING_ENABLED":    "true",
		"DOTFILES_SETTING_COUNT":      "3",
	}
	for key, want := range checks {
		if got, ok := env[key]; !ok {
			t.Errorf("expected env var %s to be present", key)
		} else if got != want {
			t.Errorf("env[%s] = %q, want %q", key, got, want)
		}
	}

	// Prompt answers stay in their own namespace, independent of settings.
	if _, ok := env["DOTFILES_PROMPT_KEY_SOURCE"]; ok {
		t.Error("settings must not appear under the DOTFILES_PROMPT_ prefix")
	}

	// Settings from a different module must not leak in.
	cfg2 := newTestRunConfig(t)
	cfg2.Config.Modules["git"] = map[string]any{"default_branch": "main"}
	env2 := buildEnvVars(cfg2, mod, nil) // mod is "ssh", not "git"
	if _, ok := env2["DOTFILES_SETTING_DEFAULT_BRANCH"]; ok {
		t.Error("settings from another module leaked into env")
	}
}

// TestBuildEnvVarsGitDefaultBranch is the positive counterpart to the leak check
// above: the git module's own default_branch setting must reach its install
// script as DOTFILES_SETTING_DEFAULT_BRANCH (config key that 4D made live).
func TestBuildEnvVarsGitDefaultBranch(t *testing.T) {
	cfg := newTestRunConfig(t)
	cfg.Config.Modules["git"] = map[string]any{"default_branch": "trunk"}

	gitMod := &Module{Name: "git", Dir: "/tmp/modules/git"}
	env := buildEnvVars(cfg, gitMod, nil)

	if got := env["DOTFILES_SETTING_DEFAULT_BRANCH"]; got != "trunk" {
		t.Errorf("env[DOTFILES_SETTING_DEFAULT_BRANCH] = %q, want %q", got, "trunk")
	}
}

// setupCopyModule creates a module with a single copy-type file under the
// config's dotfiles dir and returns the module plus the absolute source/dest.
func setupCopyModule(t *testing.T, cfg *RunConfig, content string) (*Module, string, string) {
	t.Helper()
	modDir := filepath.Join(cfg.SysInfo.DotfilesDir, "modules", "mymod")
	if err := os.MkdirAll(filepath.Join(modDir, "files"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(modDir, "files", "foo.conf")
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	mod := &Module{
		Name:  "mymod",
		Dir:   modDir,
		Files: []FileEntry{{Source: "files/foo.conf", Dest: "~/foo.conf", Type: "copy"}},
	}
	dest := filepath.Join(cfg.SysInfo.HomeDir, "foo.conf")
	return mod, src, dest
}

func countFileDeployOps(ms *state.ModuleState, dest string) int {
	n := 0
	for _, op := range ms.Operations {
		if op.Type == "file_deploy" && op.Path == dest {
			n++
		}
	}
	return n
}

// A clean re-run that skips an unchanged file must still record a rollback
// operation, or a later uninstall cannot remove the file.
func TestDeployFiles_SkipRecordsRollbackOp(t *testing.T) {
	cfg := newTestRunConfig(t)
	mod, _, dest := setupCopyModule(t, cfg, "v1")
	tctx := buildTemplateContext(cfg, mod, buildEnvVars(cfg, mod, nil))

	ms1 := &state.ModuleState{Name: mod.Name}
	deployed, skipped, err := deployFiles(cfg, mod, tctx, ms1, nil)
	if err != nil {
		t.Fatalf("first deploy: %v", err)
	}
	if deployed != 1 || skipped != 0 {
		t.Fatalf("first deploy counts: deployed=%d skipped=%d, want 1/0", deployed, skipped)
	}

	ms2 := &state.ModuleState{Name: mod.Name}
	deployed, skipped, err = deployFiles(cfg, mod, tctx, ms2, ms1)
	if err != nil {
		t.Fatalf("second deploy: %v", err)
	}
	if deployed != 0 || skipped != 1 {
		t.Fatalf("second deploy counts: deployed=%d skipped=%d, want 0/1", deployed, skipped)
	}
	if got := countFileDeployOps(ms2, dest); got != 1 {
		t.Fatalf("skip run recorded %d file_deploy ops for %s, want 1", got, dest)
	}
	if !ms2.CanRollback() {
		t.Fatal("state after a clean re-run reports CanRollback()==false")
	}
}

// When the user edits a deployed file AND the source also changes before the
// next run, the user's copy must be backed up (not silently overwritten) and the
// operation must carry backup_path so rollback can restore it.
func TestDeployFiles_BacksUpUserEditWhenSourceAlsoChanged(t *testing.T) {
	cfg := newTestRunConfig(t)
	mod, src, dest := setupCopyModule(t, cfg, "v1")
	tctx := buildTemplateContext(cfg, mod, buildEnvVars(cfg, mod, nil))

	ms1 := &state.ModuleState{Name: mod.Name}
	if _, _, err := deployFiles(cfg, mod, tctx, ms1, nil); err != nil {
		t.Fatalf("first deploy: %v", err)
	}

	if err := os.WriteFile(dest, []byte("user-edited"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}

	ms2 := &state.ModuleState{Name: mod.Name}
	deployed, _, err := deployFiles(cfg, mod, tctx, ms2, ms1)
	if err != nil {
		t.Fatalf("second deploy: %v", err)
	}
	if deployed != 1 {
		t.Fatalf("expected redeploy (source changed), deployed=%d", deployed)
	}

	var backupPath, action string
	for _, op := range ms2.Operations {
		if op.Type == "file_deploy" && op.Path == dest {
			action = op.Action
			backupPath = op.Metadata["backup_path"]
		}
	}
	if action != "modified" {
		t.Errorf("op action = %q, want modified", action)
	}
	if backupPath == "" {
		t.Fatal("no backup_path recorded for overwritten user edit")
	}
	if data, rerr := os.ReadFile(backupPath); rerr != nil {
		t.Fatalf("reading backup: %v", rerr)
	} else if string(data) != "user-edited" {
		t.Errorf("backup content = %q, want %q", string(data), "user-edited")
	}
	if got, _ := os.ReadFile(dest); string(got) != "v2" {
		t.Errorf("dest content = %q, want %q", string(got), "v2")
	}
}

func countDirCreateOps(ms *state.ModuleState, dir string) int {
	n := 0
	for _, op := range ms.Operations {
		if op.Type == "dir_create" && op.Path == dir {
			n++
		}
	}
	return n
}

// Deploying a file straight into a directory that already exists (e.g. $HOME)
// must NOT record a dir_create op — otherwise uninstall proposes
// "Remove directory: $HOME" in its rollback plan. Regression test for #22.
func TestDeployFiles_NoDirCreateOpForExistingDir(t *testing.T) {
	cfg := newTestRunConfig(t)
	mod, _, _ := setupCopyModule(t, cfg, "v1") // deploys to ~/foo.conf; parent is HomeDir (pre-existing)
	tctx := buildTemplateContext(cfg, mod, buildEnvVars(cfg, mod, nil))

	ms := &state.ModuleState{Name: mod.Name}
	if _, _, err := deployFiles(cfg, mod, tctx, ms, nil); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	home := cfg.SysInfo.HomeDir
	if got := countDirCreateOps(ms, home); got != 0 {
		t.Errorf("recorded %d dir_create ops for pre-existing home %s, want 0", got, home)
	}
	for _, instr := range ms.RollbackInstructions() {
		if instr == "Remove directory: "+home {
			t.Errorf("rollback plan proposes removing pre-existing home dir: %q", instr)
		}
	}
}

// Deploying into a directory the install actually creates DOES record a
// dir_create op, so uninstall can clean up the (empty) directory it made.
func TestDeployFiles_RecordsDirCreateForNewDir(t *testing.T) {
	cfg := newTestRunConfig(t)

	modDir := filepath.Join(cfg.SysInfo.DotfilesDir, "modules", "mymod")
	if err := os.MkdirAll(filepath.Join(modDir, "files"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(modDir, "files", "foo.conf")
	if err := os.WriteFile(src, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Dest lives in a subdirectory that does not exist yet.
	mod := &Module{
		Name:  "mymod",
		Dir:   modDir,
		Files: []FileEntry{{Source: "files/foo.conf", Dest: "~/newsub/foo.conf", Type: "copy"}},
	}
	tctx := buildTemplateContext(cfg, mod, buildEnvVars(cfg, mod, nil))
	newDir := filepath.Join(cfg.SysInfo.HomeDir, "newsub")

	ms := &state.ModuleState{Name: mod.Name}
	if _, _, err := deployFiles(cfg, mod, tctx, ms, nil); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	if got := countDirCreateOps(ms, newDir); got != 1 {
		t.Errorf("recorded %d dir_create ops for newly created %s, want 1", got, newDir)
	}
}

// A module whose files deploy under a directory that is a DANGLING symlink (the
// state hal was left in after an engine upgrade: ~/.config/nvim -> the previous
// repo's files) must not fail: os.MkdirAll over the link would error with
// EEXIST. deployFiles reconciles the symlinked parent (a link into a managed
// root carries no unique content, so no backup) and creates a real directory.
func TestDeployFiles_ReconcilesDanglingSymlinkParentDir(t *testing.T) {
	cfg := newTestRunConfig(t)

	modDir := filepath.Join(cfg.SysInfo.DotfilesDir, "modules", "nvimcfg")
	if err := os.MkdirAll(filepath.Join(modDir, "files"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(modDir, "files", "init.lua")
	if err := os.WriteFile(src, []byte("NVIM-CONFIG"), 0o644); err != nil {
		t.Fatal(err)
	}

	// The parent dir of the dest is a dangling symlink into a (now-gone) managed
	// path — exactly what an engine upgrade leaves behind.
	parent := filepath.Join(cfg.SysInfo.HomeDir, "nvimcfg")
	danglingTarget := filepath.Join(cfg.SysInfo.DotfilesDir, "gone", "old-nvim")
	if err := os.Symlink(danglingTarget, parent); err != nil {
		t.Fatal(err)
	}

	mod := &Module{
		Name:  "nvimcfg",
		Dir:   modDir,
		Files: []FileEntry{{Source: "files/init.lua", Dest: "~/nvimcfg/init.lua", Type: "copy"}},
	}
	tctx := buildTemplateContext(cfg, mod, buildEnvVars(cfg, mod, nil))

	ms := &state.ModuleState{Name: mod.Name}
	deployed, _, err := deployFiles(cfg, mod, tctx, ms, nil)
	if err != nil {
		t.Fatalf("deploy over dangling-symlink parent: %v", err)
	}
	if deployed != 1 {
		t.Fatalf("deployed=%d, want 1", deployed)
	}

	// The parent is now a real directory, not a symlink.
	fi, err := os.Lstat(parent)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatal("parent dir is still a symlink after reconciliation")
	}
	if !fi.IsDir() {
		t.Fatalf("parent is not a directory (mode %v)", fi.Mode())
	}
	if got, _ := os.ReadFile(filepath.Join(parent, "init.lua")); string(got) != "NVIM-CONFIG" {
		t.Errorf("deployed file content = %q, want NVIM-CONFIG", string(got))
	}
}

func TestRunEmptyPlan(t *testing.T) {
	cfg := newTestRunConfig(t)

	plan := &ExecutionPlan{
		Modules: []*Module{},
		Skipped: []*Module{},
	}

	results := Run(cfg, plan)

	if len(results) != 0 {
		t.Errorf("Run with empty plan returned %d results, want 0", len(results))
	}
}

func TestDeploySymlink(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a source file.
	srcPath := filepath.Join(tmpDir, "source.conf")
	if err := os.WriteFile(srcPath, []byte("config-content"), 0o644); err != nil {
		t.Fatalf("writing source file: %v", err)
	}

	// Deploy a symlink.
	destPath := filepath.Join(tmpDir, "dest", "link.conf")
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		t.Fatalf("creating dest dir: %v", err)
	}

	if err := deploySymlink(srcPath, destPath); err != nil {
		t.Fatalf("deploySymlink: %v", err)
	}

	// Verify symlink target.
	target, err := os.Readlink(destPath)
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}

	absSrc, _ := filepath.Abs(srcPath)
	if target != absSrc {
		t.Errorf("symlink target = %q, want %q", target, absSrc)
	}

	// Verify the symlink resolves to the correct content.
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("reading through symlink: %v", err)
	}
	if string(data) != "config-content" {
		t.Errorf("content through symlink = %q, want %q", string(data), "config-content")
	}
}

// Migrating from the old model (a config symlinked straight out of the repo) to
// the new model (a rendered template/copy at the same dest) must replace the
// symlink with a real file WITHOUT following it — following would clobber and
// re-dirty the repo source. A link into a managed root holds no unique content,
// so no backup is taken.
func TestDeployFiles_MigratesLegacySymlinkIntoManagedRoot(t *testing.T) {
	cfg := newTestRunConfig(t)

	modDir := filepath.Join(cfg.SysInfo.DotfilesDir, "modules", "mymod")
	if err := os.MkdirAll(filepath.Join(modDir, "files"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The repo file the OLD symlink points at.
	repoSrc := filepath.Join(modDir, "files", "legacy.conf")
	if err := os.WriteFile(repoSrc, []byte("REPO-ORIGINAL"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The NEW template source managed at the same destination.
	newSrc := filepath.Join(modDir, "files", "app.tmpl")
	if err := os.WriteFile(newSrc, []byte("NEW-CONTENT"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(cfg.SysInfo.HomeDir, "app.conf")
	if err := os.Symlink(repoSrc, dest); err != nil {
		t.Fatal(err)
	}

	mod := &Module{
		Name:  "mymod",
		Dir:   modDir,
		Files: []FileEntry{{Source: "files/app.tmpl", Dest: "~/app.conf", Type: "template"}},
	}
	tctx := buildTemplateContext(cfg, mod, buildEnvVars(cfg, mod, nil))

	// Prior state records the legacy symlink deployment (type change forces redeploy).
	prior := &state.ModuleState{Name: mod.Name, FileStates: []state.FileState{{
		Source: "files/legacy.conf", Dest: dest, Type: "symlink",
	}}}

	ms := &state.ModuleState{Name: mod.Name}
	deployed, _, err := deployFiles(cfg, mod, tctx, ms, prior)
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if deployed != 1 {
		t.Fatalf("deployed=%d, want 1", deployed)
	}

	fi, err := os.Lstat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatal("dest is still a symlink after migration")
	}
	if got, _ := os.ReadFile(dest); string(got) != "NEW-CONTENT" {
		t.Errorf("dest = %q, want NEW-CONTENT", string(got))
	}
	// The write must NOT have followed the old link back into the repo.
	if got, _ := os.ReadFile(repoSrc); string(got) != "REPO-ORIGINAL" {
		t.Errorf("repo source was modified via write-through: %q", string(got))
	}
	for _, op := range ms.Operations {
		if op.Type == "file_deploy" && op.Path == dest && op.Metadata["backup_path"] != "" {
			t.Errorf("unexpected backup for a managed legacy symlink: %s", op.Metadata["backup_path"])
		}
	}
}

// A legacy symlink pointing OUTSIDE any managed root may reference real user
// content, so its resolved content is backed up before the link is replaced, and
// the pointed-at file itself is never disturbed.
func TestDeployFiles_MigratesExternalSymlinkWithBackup(t *testing.T) {
	cfg := newTestRunConfig(t)

	modDir := filepath.Join(cfg.SysInfo.DotfilesDir, "modules", "mymod")
	if err := os.MkdirAll(filepath.Join(modDir, "files"), 0o755); err != nil {
		t.Fatal(err)
	}
	newSrc := filepath.Join(modDir, "files", "app.conf")
	if err := os.WriteFile(newSrc, []byte("NEW-CONTENT"), 0o644); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "user-real.conf")
	if err := os.WriteFile(external, []byte("USER-REAL"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(cfg.SysInfo.HomeDir, "app.conf")
	if err := os.Symlink(external, dest); err != nil {
		t.Fatal(err)
	}

	mod := &Module{
		Name:  "mymod",
		Dir:   modDir,
		Files: []FileEntry{{Source: "files/app.conf", Dest: "~/app.conf", Type: "copy"}},
	}
	tctx := buildTemplateContext(cfg, mod, buildEnvVars(cfg, mod, nil))

	ms := &state.ModuleState{Name: mod.Name}
	if _, _, err := deployFiles(cfg, mod, tctx, ms, nil); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	if got, _ := os.ReadFile(dest); string(got) != "NEW-CONTENT" {
		t.Errorf("dest = %q, want NEW-CONTENT", string(got))
	}
	if got, _ := os.ReadFile(external); string(got) != "USER-REAL" {
		t.Errorf("external target modified: %q", string(got))
	}
	var backupPath string
	for _, op := range ms.Operations {
		if op.Type == "file_deploy" && op.Path == dest {
			backupPath = op.Metadata["backup_path"]
		}
	}
	if backupPath == "" {
		t.Fatal("no backup recorded when migrating an external symlink")
	}
	if got, _ := os.ReadFile(backupPath); string(got) != "USER-REAL" {
		t.Errorf("backup content = %q, want USER-REAL", string(got))
	}
}

// A dangling legacy symlink (its repo target already deleted, e.g. after an
// engine update removed the old source) must be removed cleanly — never
// re-created via write-through — and the new file deployed.
func TestDeployFiles_MigratesDanglingSymlink(t *testing.T) {
	cfg := newTestRunConfig(t)

	modDir := filepath.Join(cfg.SysInfo.DotfilesDir, "modules", "mymod")
	if err := os.MkdirAll(filepath.Join(modDir, "files"), 0o755); err != nil {
		t.Fatal(err)
	}
	newSrc := filepath.Join(modDir, "files", "app.tmpl")
	if err := os.WriteFile(newSrc, []byte("NEW"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Points into the managed repo, but the target does not exist.
	danglingTarget := filepath.Join(modDir, "files", "gone.conf")
	dest := filepath.Join(cfg.SysInfo.HomeDir, "app.conf")
	if err := os.Symlink(danglingTarget, dest); err != nil {
		t.Fatal(err)
	}

	mod := &Module{
		Name:  "mymod",
		Dir:   modDir,
		Files: []FileEntry{{Source: "files/app.tmpl", Dest: "~/app.conf", Type: "template"}},
	}
	tctx := buildTemplateContext(cfg, mod, buildEnvVars(cfg, mod, nil))

	ms := &state.ModuleState{Name: mod.Name}
	if _, _, err := deployFiles(cfg, mod, tctx, ms, nil); err != nil {
		t.Fatalf("deploy: %v", err)
	}
	fi, err := os.Lstat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatal("dest is still a symlink")
	}
	if got, _ := os.ReadFile(dest); string(got) != "NEW" {
		t.Errorf("dest = %q, want NEW", string(got))
	}
	if _, err := os.Stat(danglingTarget); !os.IsNotExist(err) {
		t.Error("write-through re-created the dangling target inside the repo")
	}
}

func TestIsInsideManagedRoot(t *testing.T) {
	cfg := newTestRunConfig(t)
	cfg.Config.ContentDir = t.TempDir()
	outside := t.TempDir()

	cases := []struct {
		path string
		want bool
	}{
		{filepath.Join(cfg.SysInfo.DotfilesDir, "modules", "x", "y"), true},
		{filepath.Join(cfg.Config.ContentDir, "modules", "x"), true},
		{filepath.Join(outside, "elsewhere"), false},
		{"/etc/passwd", false},
	}
	for _, c := range cases {
		if got := isInsideManagedRoot(c.path, cfg); got != c.want {
			t.Errorf("isInsideManagedRoot(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestDeploySymlinkReplacesExisting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two source files.
	src1 := filepath.Join(tmpDir, "v1.conf")
	src2 := filepath.Join(tmpDir, "v2.conf")
	if err := os.WriteFile(src1, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src2, []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}

	destPath := filepath.Join(tmpDir, "link.conf")

	// Create initial symlink.
	if err := deploySymlink(src1, destPath); err != nil {
		t.Fatalf("first deploySymlink: %v", err)
	}

	// Replace with second symlink.
	if err := deploySymlink(src2, destPath); err != nil {
		t.Fatalf("second deploySymlink: %v", err)
	}

	// Should now point to v2.
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("reading through symlink: %v", err)
	}
	if string(data) != "v2" {
		t.Errorf("content = %q, want %q", string(data), "v2")
	}
}

func TestDeployCopy(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "source.txt")
	if err := os.WriteFile(srcPath, []byte("copied-content"), 0o755); err != nil {
		t.Fatal(err)
	}

	destPath := filepath.Join(tmpDir, "dest.txt")
	if err := deployCopy(srcPath, destPath); err != nil {
		t.Fatalf("deployCopy: %v", err)
	}

	// Verify content.
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("reading dest: %v", err)
	}
	if string(data) != "copied-content" {
		t.Errorf("content = %q, want %q", string(data), "copied-content")
	}

	// Verify permissions are preserved.
	info, err := os.Stat(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("permissions = %o, want %o", info.Mode().Perm(), 0o755)
	}
}

func TestExpandPaths(t *testing.T) {
	home := "/home/testuser"
	xdgDefault := "/home/testuser/.config"
	xdgCustom := "/tmp/xdg-config"

	tests := []struct {
		name          string
		input         string
		xdgConfigHome string
		want          string
	}{
		{"home dir tilde", "~/.bashrc", xdgDefault, "/home/testuser/.bashrc"},
		{"home dir trailing slash", "~/", xdgDefault, "/home/testuser"},
		{"bare tilde", "~", xdgDefault, "/home/testuser"},
		{"absolute path", "/etc/config", xdgDefault, "/etc/config"},
		{"relative path", "relative/path", xdgDefault, "relative/path"},
		{"xdg config default", "~/.config/tmux/tmux.conf", xdgDefault, "/home/testuser/.config/tmux/tmux.conf"},
		{"xdg config custom", "~/.config/tmux/tmux.conf", xdgCustom, "/tmp/xdg-config/tmux/tmux.conf"},
		{"xdg config nested", "~/.config/nvim/init.lua", xdgCustom, "/tmp/xdg-config/nvim/init.lua"},
		{"xdg config root file", "~/.config/starship.toml", xdgCustom, "/tmp/xdg-config/starship.toml"},
		{"non-config dot path", "~/.local/bin", xdgCustom, "/home/testuser/.local/bin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandPaths(tt.input, home, tt.xdgConfigHome)
			if got != tt.want {
				t.Errorf("expandPaths(%q, %q, %q) = %q, want %q", tt.input, home, tt.xdgConfigHome, got, tt.want)
			}
		})
	}
}

func TestBoolToStr(t *testing.T) {
	if got := boolToStr(true); got != "true" {
		t.Errorf("boolToStr(true) = %q, want %q", got, "true")
	}
	if got := boolToStr(false); got != "false" {
		t.Errorf("boolToStr(false) = %q, want %q", got, "false")
	}
}

func TestRunDryRunSkipsScripts(t *testing.T) {
	cfg := newTestRunConfig(t)
	cfg.DryRun = true

	// Create a module directory with a fake install.sh that would fail
	// if it were actually executed.
	modDir := t.TempDir()
	installScript := filepath.Join(modDir, "install.sh")
	if err := os.WriteFile(installScript, []byte("exit 1"), 0o755); err != nil {
		t.Fatal(err)
	}

	mod := &Module{
		Name: "dry-run-mod",
		Dir:  modDir,
	}

	plan := &ExecutionPlan{
		Modules: []*Module{mod},
	}

	results := Run(cfg, plan)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Success {
		t.Errorf("expected success in dry-run mode, got error: %v", results[0].Error)
	}
}

func TestRunDeploysFiles(t *testing.T) {
	cfg := newTestRunConfig(t)

	// Create module dir with a source file.
	modDir := t.TempDir()
	srcContent := "symlinked-content"
	if err := os.WriteFile(filepath.Join(modDir, "bashrc"), []byte(srcContent), 0o644); err != nil {
		t.Fatal(err)
	}

	destPath := filepath.Join(cfg.SysInfo.HomeDir, ".bashrc")

	mod := &Module{
		Name: "file-deploy-mod",
		Dir:  modDir,
		Files: []FileEntry{
			{Source: "bashrc", Dest: "~/.bashrc", Type: "symlink"},
		},
	}

	plan := &ExecutionPlan{
		Modules: []*Module{mod},
	}

	results := Run(cfg, plan)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Success {
		t.Fatalf("expected success, got error: %v", results[0].Error)
	}

	// Verify the symlink was created.
	target, err := os.Readlink(destPath)
	if err != nil {
		t.Fatalf("readlink %s: %v", destPath, err)
	}
	absSrc, _ := filepath.Abs(filepath.Join(modDir, "bashrc"))
	if target != absSrc {
		t.Errorf("symlink target = %q, want %q", target, absSrc)
	}
}

func TestRunFailFastStopsOnError(t *testing.T) {
	cfg := newTestRunConfig(t)
	cfg.FailFast = true

	// First module has an install.sh that fails.
	modDir1 := t.TempDir()
	if err := os.WriteFile(filepath.Join(modDir1, "install.sh"), []byte("exit 1"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Second module would succeed (no scripts, no files).
	modDir2 := t.TempDir()

	plan := &ExecutionPlan{
		Modules: []*Module{
			{Name: "will-fail", Dir: modDir1},
			{Name: "would-succeed", Dir: modDir2},
		},
	}

	results := Run(cfg, plan)

	// With FailFast, only the first module should have run.
	if len(results) != 1 {
		t.Fatalf("expected 1 result with FailFast, got %d", len(results))
	}
	if results[0].Success {
		t.Error("expected first module to fail")
	}
	if results[0].Error == nil {
		t.Error("expected error to be set")
	}
}

func TestScriptTimeout(t *testing.T) {
	cfg := newTestRunConfig(t)

	// Create a module with a script that sleeps for 10 seconds.
	modDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(modDir, "install.sh"), []byte("sleep 10"), 0o755); err != nil {
		t.Fatal(err)
	}

	mod := &Module{
		Name:    "slow-module",
		Dir:     modDir,
		Timeout: "1s", // Set a 1-second timeout
	}

	plan := &ExecutionPlan{
		Modules: []*Module{mod},
	}

	results := Run(cfg, plan)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// The module should have failed due to timeout.
	if results[0].Success {
		t.Error("expected module to fail due to timeout")
	}
	if results[0].Error == nil {
		t.Error("expected error to be set")
	}

	// Verify the error message mentions timeout.
	if results[0].Error != nil {
		errMsg := results[0].Error.Error()
		if !contains(errMsg, "timed out") && !contains(errMsg, "timeout") {
			t.Errorf("expected timeout error, got: %v", results[0].Error)
		}
	}
}

func TestScriptTimeoutDefault(t *testing.T) {
	cfg := newTestRunConfig(t)
	cfg.ScriptTimeout = 1 * time.Second // Set config-level timeout

	// Create a module WITHOUT a module-specific timeout.
	modDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(modDir, "install.sh"), []byte("sleep 10"), 0o755); err != nil {
		t.Fatal(err)
	}

	mod := &Module{
		Name: "slow-module-no-override",
		Dir:  modDir,
		// No Timeout field - should use config default
	}

	plan := &ExecutionPlan{
		Modules: []*Module{mod},
	}

	results := Run(cfg, plan)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Should timeout using the config-level timeout.
	if results[0].Success {
		t.Error("expected module to fail due to timeout")
	}
	if results[0].Error == nil {
		t.Error("expected error to be set")
	}
}

func TestScriptNoTimeoutForQuickScripts(t *testing.T) {
	cfg := newTestRunConfig(t)

	// Create a module with a quick script.
	modDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(modDir, "install.sh"), []byte("echo 'quick'"), 0o755); err != nil {
		t.Fatal(err)
	}

	mod := &Module{
		Name:    "quick-module",
		Dir:     modDir,
		Timeout: "5m", // Long timeout, but script finishes quickly
	}

	plan := &ExecutionPlan{
		Modules: []*Module{mod},
	}

	results := Run(cfg, plan)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Should succeed.
	if !results[0].Success {
		t.Errorf("expected success, got error: %v", results[0].Error)
	}
}

func TestRunNotesOnSuccess(t *testing.T) {
	cfg := newTestRunConfig(t)

	// Module with notes but no scripts or files — should succeed and carry notes.
	modDir := t.TempDir()
	mod := &Module{
		Name:  "noted-mod",
		Dir:   modDir,
		Notes: []string{"Remember to log out and back in"},
	}

	plan := &ExecutionPlan{
		Modules: []*Module{mod},
	}

	results := Run(cfg, plan)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Success {
		t.Fatalf("expected success, got error: %v", results[0].Error)
	}
	if len(results[0].Notes) != 1 || results[0].Notes[0] != "Remember to log out and back in" {
		t.Errorf("Notes = %v, want [Remember to log out and back in]", results[0].Notes)
	}
}

func TestRunNotesEmptyOnSkip(t *testing.T) {
	cfg := newTestRunConfig(t)

	modDir := t.TempDir()
	mod := &Module{
		Name:    "skip-noted-mod",
		Version: "1.0.0",
		Dir:     modDir,
		Notes:   []string{"You should not see this"},
	}

	// Pre-populate state so the module gets skipped.
	checksum, _ := ComputeModuleChecksum(mod)
	configHash := ComputeConfigHash(mod, cfg.Config)
	cfg.State.Set(&state.ModuleState{
		Name:       mod.Name,
		Version:    mod.Version,
		Status:     "installed",
		Checksum:   checksum,
		ConfigHash: configHash,
	})

	plan := &ExecutionPlan{
		Modules: []*Module{mod},
	}

	results := Run(cfg, plan)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Skipped {
		t.Errorf("expected module to be skipped")
	}
	if len(results[0].Notes) != 0 {
		t.Errorf("Notes = %v, want empty for skipped module", results[0].Notes)
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// sequentialUI is a testUI variant whose PromptChoice returns answers in
// order from a pre-supplied slice, making prompt ordering deterministic.
type sequentialUI struct {
	testUI
	choiceAnswers []string
	choiceIdx     int
	promptsCalled []string
}

func (s *sequentialUI) PromptChoice(msg string, _ []string) (string, error) {
	s.promptsCalled = append(s.promptsCalled, msg)
	if s.choiceIdx < len(s.choiceAnswers) {
		answer := s.choiceAnswers[s.choiceIdx]
		s.choiceIdx++
		return answer, nil
	}
	return "", nil
}

func TestHandlePromptsZinitSkipsOMZQuestions(t *testing.T) {
	ui := &sequentialUI{choiceAnswers: []string{"zinit"}}
	cfg := newTestRunConfig(t)
	cfg.Unattended = false
	cfg.UI = ui
	cfg.ExplicitModules = map[string]bool{"zsh": true}

	mod := &Module{
		Name: "zsh",
		Prompts: []Prompt{
			{Key: "zsh_framework", Message: "Zsh plugin framework", Default: "zinit", Type: "choice", Options: []string{"zinit", "ohmyzsh"}, ShowWhen: "explicit_install"},
			{Key: "zsh_omz_plugins", Message: "Oh My Zsh plugin preset", Default: "standard", Type: "choice", Options: []string{"minimal", "standard", "full"}, ShowWhen: "explicit_install", DependsOn: &PromptDependency{Key: "zsh_framework", Value: "ohmyzsh"}},
			{Key: "zsh_prompt", Message: "Prompt theme", Default: "starship", Type: "choice", Options: []string{"starship", "robbyrussell"}, ShowWhen: "explicit_install", DependsOn: &PromptDependency{Key: "zsh_framework", Value: "ohmyzsh"}},
		},
	}

	answers, err := handlePrompts(cfg, mod)
	if err != nil {
		t.Fatalf("handlePrompts error: %v", err)
	}

	// Only the framework question should have been asked
	if len(ui.promptsCalled) != 1 {
		t.Errorf("prompts called = %v, want only [Zsh plugin framework]", ui.promptsCalled)
	}
	if answers["zsh_framework"] != "zinit" {
		t.Errorf("zsh_framework = %q, want zinit", answers["zsh_framework"])
	}
	// OMZ prompts should be empty (dependency not met)
	if answers["zsh_omz_plugins"] != "" {
		t.Errorf("zsh_omz_plugins = %q, want empty (skipped)", answers["zsh_omz_plugins"])
	}
	if answers["zsh_prompt"] != "" {
		t.Errorf("zsh_prompt = %q, want empty (skipped)", answers["zsh_prompt"])
	}
}

func TestScriptUsesSudo(t *testing.T) {
	tests := []struct {
		name      string
		script    string
		pkgMgr    string
		currentOS string
		want      bool
	}{
		{
			name:      "unguarded sudo detected on any OS",
			script:    "#!/bin/bash\nsudo apt-get update\n",
			pkgMgr:    "apt",
			currentOS: "macos",
			want:      true,
		},
		{
			name:      "sudo inside is_ubuntu guard skipped on macOS",
			script:    "#!/bin/bash\nif is_ubuntu; then\n  sudo dpkg -i foo.deb\nfi\n",
			pkgMgr:    "brew",
			currentOS: "macos",
			want:      false,
		},
		{
			name:      "sudo inside is_ubuntu guard detected on linux",
			script:    "#!/bin/bash\nif is_ubuntu; then\n  sudo dpkg -i foo.deb\nfi\n",
			pkgMgr:    "apt",
			currentOS: "linux",
			want:      true,
		},
		{
			name:      "pkg_install with apt triggers sudo",
			script:    "#!/bin/bash\npkg_install git\n",
			pkgMgr:    "apt",
			currentOS: "linux",
			want:      true,
		},
		{
			name:      "pkg_install with brew does not trigger sudo",
			script:    "#!/bin/bash\npkg_install git\n",
			pkgMgr:    "brew",
			currentOS: "macos",
			want:      false,
		},
		{
			name:      "comment with sudo is ignored",
			script:    "#!/bin/bash\n# sudo apt-get update\necho hello\n",
			pkgMgr:    "apt",
			currentOS: "linux",
			want:      false,
		},
		{
			name:      "no sudo at all",
			script:    "#!/bin/bash\necho hello\n",
			pkgMgr:    "apt",
			currentOS: "linux",
			want:      false,
		},
		{
			name: "nested if inside guarded block",
			script: `#!/bin/bash
if is_ubuntu; then
  if [ -f /tmp/foo ]; then
    sudo dpkg -i bar.deb
  fi
fi
`,
			pkgMgr:    "apt",
			currentOS: "macos",
			want:      false,
		},
		{
			name: "sudo after guarded block exits",
			script: `#!/bin/bash
if is_ubuntu; then
  echo ubuntu only
fi
sudo something
`,
			pkgMgr:    "brew",
			currentOS: "macos",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			scriptPath := filepath.Join(tmpDir, "test.sh")
			if err := os.WriteFile(scriptPath, []byte(tt.script), 0o755); err != nil {
				t.Fatal(err)
			}

			got := scriptUsesSudo(scriptPath, tt.pkgMgr, tt.currentOS)
			if got != tt.want {
				t.Errorf("scriptUsesSudo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScriptUsesSudoGitInstallOnMacOS(t *testing.T) {
	// Test against the actual git/install.sh which has sudo inside an
	// is_ubuntu guard — should return false on macOS with brew.
	gitInstall := filepath.Join("../../modules/git/install.sh")
	if _, err := os.Stat(gitInstall); os.IsNotExist(err) {
		t.Skip("modules/git/install.sh not found")
	}

	got := scriptUsesSudo(gitInstall, "brew", "macos")
	if got {
		t.Error("scriptUsesSudo(git/install.sh, brew, macos) = true, want false; sudo is inside is_ubuntu guard")
	}
}

func TestHandleLegacyPathsBackup(t *testing.T) {
	cfg := newTestRunConfig(t)

	// Create a legacy file
	legacyFile := filepath.Join(cfg.SysInfo.HomeDir, ".tmux.conf")
	if err := os.WriteFile(legacyFile, []byte("legacy-content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create backup directory
	if err := os.MkdirAll(filepath.Join(cfg.SysInfo.DotfilesDir, ".backups"), 0o755); err != nil {
		t.Fatal(err)
	}

	mod := &Module{
		Name: "tmux",
		Dir:  t.TempDir(),
		LegacyPaths: []LegacyPath{
			{Path: "~/.tmux.conf", Action: "backup", Reason: "Shadows XDG config"},
		},
	}

	modState := &state.ModuleState{Name: mod.Name}

	if err := handleLegacyPaths(cfg, mod, modState); err != nil {
		t.Fatalf("handleLegacyPaths: %v", err)
	}

	// Legacy file should be removed
	if _, err := os.Stat(legacyFile); !os.IsNotExist(err) {
		t.Error("expected legacy file to be removed after backup")
	}

	// Should have recorded a legacy_cleanup operation
	found := false
	for _, op := range modState.Operations {
		if op.Type == "legacy_cleanup" && op.Path == legacyFile {
			found = true
		}
	}
	if !found {
		t.Error("expected legacy_cleanup operation to be recorded")
	}
}

func TestHandleLegacyPathsWarn(t *testing.T) {
	cfg := newTestRunConfig(t)
	ui := cfg.UI.(*testUI)

	// Create a legacy file
	legacyFile := filepath.Join(cfg.SysInfo.HomeDir, ".gitconfig")
	if err := os.WriteFile(legacyFile, []byte("legacy"), 0o644); err != nil {
		t.Fatal(err)
	}

	mod := &Module{
		Name: "git",
		Dir:  t.TempDir(),
		LegacyPaths: []LegacyPath{
			{Path: "~/.gitconfig", Action: "warn", Reason: "May override managed config"},
		},
	}

	modState := &state.ModuleState{Name: mod.Name}

	if err := handleLegacyPaths(cfg, mod, modState); err != nil {
		t.Fatalf("handleLegacyPaths: %v", err)
	}

	// File should still exist
	if _, err := os.Stat(legacyFile); err != nil {
		t.Error("expected legacy file to still exist for warn action")
	}

	// Should have a warning
	foundWarn := false
	for _, w := range ui.warns {
		if strings.Contains(w, ".gitconfig") {
			foundWarn = true
		}
	}
	if !foundWarn {
		t.Errorf("expected warning about .gitconfig, got warns: %v", ui.warns)
	}
}

func TestHandleLegacyPathsNonexistent(t *testing.T) {
	cfg := newTestRunConfig(t)

	mod := &Module{
		Name: "tmux",
		Dir:  t.TempDir(),
		LegacyPaths: []LegacyPath{
			{Path: "~/.tmux.conf", Action: "backup"},
		},
	}

	modState := &state.ModuleState{Name: mod.Name}

	// Should succeed without error — file doesn't exist
	if err := handleLegacyPaths(cfg, mod, modState); err != nil {
		t.Fatalf("handleLegacyPaths: %v", err)
	}

	if len(modState.Operations) != 0 {
		t.Errorf("expected no operations for nonexistent file, got %d", len(modState.Operations))
	}
}

func TestHandleLegacyPathsDryRun(t *testing.T) {
	cfg := newTestRunConfig(t)
	cfg.DryRun = true

	legacyFile := filepath.Join(cfg.SysInfo.HomeDir, ".tmux.conf")
	if err := os.WriteFile(legacyFile, []byte("legacy"), 0o644); err != nil {
		t.Fatal(err)
	}

	mod := &Module{
		Name: "tmux",
		Dir:  t.TempDir(),
		LegacyPaths: []LegacyPath{
			{Path: "~/.tmux.conf", Action: "backup"},
		},
	}

	modState := &state.ModuleState{Name: mod.Name}

	if err := handleLegacyPaths(cfg, mod, modState); err != nil {
		t.Fatalf("handleLegacyPaths: %v", err)
	}

	// File should still exist in dry-run mode
	if _, err := os.Stat(legacyFile); err != nil {
		t.Error("expected legacy file to still exist in dry-run mode")
	}
}

func TestHandleLegacyPathsPromptUnattended(t *testing.T) {
	cfg := newTestRunConfig(t)
	cfg.Unattended = true

	legacyFile := filepath.Join(cfg.SysInfo.HomeDir, ".config", "nvim", "init.vim")
	if err := os.MkdirAll(filepath.Dir(legacyFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyFile, []byte("set nocompatible"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create backup directory
	if err := os.MkdirAll(filepath.Join(cfg.SysInfo.DotfilesDir, ".backups"), 0o755); err != nil {
		t.Fatal(err)
	}

	mod := &Module{
		Name: "neovim",
		Dir:  t.TempDir(),
		LegacyPaths: []LegacyPath{
			{Path: "~/.config/nvim/init.vim", Action: "prompt", Reason: "Conflicts with Lua config"},
		},
	}

	modState := &state.ModuleState{Name: mod.Name}

	if err := handleLegacyPaths(cfg, mod, modState); err != nil {
		t.Fatalf("handleLegacyPaths: %v", err)
	}

	// In unattended mode, prompt falls back to backup — file should be removed
	if _, err := os.Stat(legacyFile); !os.IsNotExist(err) {
		t.Error("expected legacy file to be removed (prompt falls back to backup in unattended mode)")
	}
}

func TestHandlePromptsOhmyzshShowsAllQuestions(t *testing.T) {
	ui := &sequentialUI{choiceAnswers: []string{"ohmyzsh", "full", "robbyrussell"}}
	cfg := newTestRunConfig(t)
	cfg.Unattended = false
	cfg.UI = ui
	cfg.ExplicitModules = map[string]bool{"zsh": true}

	mod := &Module{
		Name: "zsh",
		Prompts: []Prompt{
			{Key: "zsh_framework", Message: "Zsh plugin framework", Default: "zinit", Type: "choice", Options: []string{"zinit", "ohmyzsh"}, ShowWhen: "explicit_install"},
			{Key: "zsh_omz_plugins", Message: "Oh My Zsh plugin preset", Default: "standard", Type: "choice", Options: []string{"minimal", "standard", "full"}, ShowWhen: "explicit_install", DependsOn: &PromptDependency{Key: "zsh_framework", Value: "ohmyzsh"}},
			{Key: "zsh_prompt", Message: "Prompt theme", Default: "starship", Type: "choice", Options: []string{"starship", "robbyrussell"}, ShowWhen: "explicit_install", DependsOn: &PromptDependency{Key: "zsh_framework", Value: "ohmyzsh"}},
		},
	}

	answers, err := handlePrompts(cfg, mod)
	if err != nil {
		t.Fatalf("handlePrompts error: %v", err)
	}

	// All three prompts should have been asked
	if len(ui.promptsCalled) != 3 {
		t.Errorf("prompts called = %v, want all 3", ui.promptsCalled)
	}
	if answers["zsh_framework"] != "ohmyzsh" {
		t.Errorf("zsh_framework = %q, want ohmyzsh", answers["zsh_framework"])
	}
	if answers["zsh_omz_plugins"] != "full" {
		t.Errorf("zsh_omz_plugins = %q, want full", answers["zsh_omz_plugins"])
	}
	if answers["zsh_prompt"] != "robbyrussell" {
		t.Errorf("zsh_prompt = %q, want robbyrussell", answers["zsh_prompt"])
	}
}
