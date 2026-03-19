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

func (t *testUI) PromptMultiSelect(_ string, _ []MultiSelectOption, preSelected []string) ([]string, error) {
	return preSelected, nil
}

func (t *testUI) PrintCollapsedOutput(scriptName, output string) {}
func (t *testUI) StartProgressBar(total int) ProgressTracker     { return nil }
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
		"DOTFILES_OS":          "linux",
		"DOTFILES_ARCH":        "amd64",
		"DOTFILES_PKG_MGR":     "apt",
		"DOTFILES_HAS_SUDO":    "true",
		"DOTFILES_HOME":           cfg.SysInfo.HomeDir,
		"DOTFILES_XDG_CONFIG_HOME": cfg.SysInfo.XDGConfigHome,
		"DOTFILES_DIR":            cfg.SysInfo.DotfilesDir,
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
