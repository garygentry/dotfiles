package module

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/garygentry/dotfiles/internal/config"
	"github.com/garygentry/dotfiles/internal/secrets"
	"github.com/garygentry/dotfiles/internal/state"
	"github.com/garygentry/dotfiles/internal/sysinfo"
)

// testUI is a minimal RunnerUI implementation for testing that records
// calls but does not produce terminal output.
type testUI struct {
	infos    []string
	warns    []string
	errs     []string
	successes []string
	debugs   []string
	verbose  bool
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

// newTestRunConfig returns a RunConfig suitable for unit tests. It uses
// temp directories for state and dotfiles, a test UI, and the noop secrets
// provider.
func newTestRunConfig(t *testing.T) *RunConfig {
	t.Helper()

	stateDir := t.TempDir()
	dotfilesDir := t.TempDir()

	return &RunConfig{
		SysInfo: &sysinfo.SystemInfo{
			OS:          "linux",
			Arch:        "amd64",
			PkgMgr:      "apt",
			HasSudo:     true,
			User:        "testuser",
			HomeDir:     t.TempDir(),
			DotfilesDir: dotfilesDir,
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
		"DOTFILES_OS":          "linux",
		"DOTFILES_ARCH":        "amd64",
		"DOTFILES_PKG_MGR":     "apt",
		"DOTFILES_HAS_SUDO":    "true",
		"DOTFILES_HOME":        cfg.SysInfo.HomeDir,
		"DOTFILES_DIR":         cfg.SysInfo.DotfilesDir,
		"DOTFILES_MODULE_DIR":  "/tmp/modules/test-module",
		"DOTFILES_MODULE_NAME": "test-module",
		"DOTFILES_INTERACTIVE": "false", // Unattended=true => interactive=false
		"DOTFILES_DRY_RUN":     "false",
		"DOTFILES_VERBOSE":     "true",
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

func TestExpandHome(t *testing.T) {
	home := "/home/testuser"

	tests := []struct {
		input string
		want  string
	}{
		{"~/.bashrc", "/home/testuser/.bashrc"},
		{"~/", "/home/testuser"},
		{"~", "/home/testuser"},
		{"/etc/config", "/etc/config"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		got := expandHome(tt.input, home)
		if got != tt.want {
			t.Errorf("expandHome(%q, %q) = %q, want %q", tt.input, home, got, tt.want)
		}
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
