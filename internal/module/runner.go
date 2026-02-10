package module

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/garygentry/dotfiles/internal/config"
	"github.com/garygentry/dotfiles/internal/secrets"
	"github.com/garygentry/dotfiles/internal/state"
	"github.com/garygentry/dotfiles/internal/sysinfo"
	"github.com/garygentry/dotfiles/internal/template"
)

// RunnerUI is the subset of ui functionality that the module runner requires.
// Defining it as an interface inside the module package avoids the import
// cycle between the module and ui packages. The concrete *ui.UI type does
// not directly satisfy this interface because its spinner methods use the
// concrete *ui.Spinner type; use UIAdapter to bridge the gap.
type RunnerUI interface {
	Info(msg string)
	Warn(msg string)
	Error(msg string)
	Success(msg string)
	Debug(msg string)

	PromptInput(msg string, defaultVal string) (string, error)
	PromptConfirm(msg string, defaultVal bool) (bool, error)
	PromptChoice(msg string, options []string) (string, error)

	StartSpinner(msg string) any
	StopSpinnerSuccess(s any, msg string)
	StopSpinnerFail(s any, msg string)
	StopSpinnerSkip(s any, msg string)
}

// RunConfig holds all configuration needed by the module runner.
type RunConfig struct {
	SysInfo    *sysinfo.SystemInfo
	Config     *config.Config
	UI         RunnerUI
	Secrets    secrets.Provider
	State      *state.Store
	DryRun     bool
	Unattended bool
	FailFast   bool
	Verbose    bool
}

// RunResult captures the outcome of running a single module.
type RunResult struct {
	Module   *Module
	Success  bool
	Skipped  bool
	Error    error
	Duration time.Duration
}

// Run is the main entry point for executing an ordered set of modules.
// It iterates over plan.Modules, running each one and collecting results.
// If cfg.FailFast is true the loop stops after the first failure.
func Run(cfg *RunConfig, plan *ExecutionPlan) []RunResult {
	results := make([]RunResult, 0, len(plan.Modules))

	for _, mod := range plan.Modules {
		result := runModule(cfg, mod)
		results = append(results, result)

		if cfg.FailFast && !result.Success && !result.Skipped {
			break
		}
	}

	return results
}

// runModule orchestrates the full lifecycle for a single module: prompts,
// env vars, scripts, file deployment, verification, and state recording.
//
// Scripts are executed without an active spinner to avoid the spinner
// goroutine colliding with subprocess output that writes directly to
// /dev/tty (e.g. sudo password prompts). A spinner is used only during
// file deployment which is Go-native and produces no terminal side-effects.
func runModule(cfg *RunConfig, mod *Module) RunResult {
	start := time.Now()

	// Show a static progress line while scripts run (no animated spinner).
	cfg.UI.Info(fmt.Sprintf("Installing %s...", mod.Name))

	// Step 1: Handle prompts.
	promptAnswers, err := handlePrompts(cfg, mod)
	if err != nil {
		cfg.UI.Error(fmt.Sprintf("Failed %s: %v", mod.Name, err))
		recordState(cfg, mod, "failed", err)
		return RunResult{Module: mod, Error: err, Duration: time.Since(start)}
	}

	// Step 2: Build environment variables.
	envVars := buildEnvVars(cfg, mod, promptAnswers)

	// Step 3: Build template context.
	tmplCtx := buildTemplateContext(cfg, mod, envVars)

	// Step 4: Run OS-specific script if it exists.
	osScript := filepath.Join(mod.Dir, "os", cfg.SysInfo.OS+".sh")
	if _, statErr := os.Stat(osScript); statErr == nil {
		if err := runScript(cfg, osScript, envVars); err != nil {
			cfg.UI.Error(fmt.Sprintf("Failed %s: os script error: %v", mod.Name, err))
			recordState(cfg, mod, "failed", err)
			return RunResult{Module: mod, Error: err, Duration: time.Since(start)}
		}
	}

	// Step 5: Run install.sh if it exists.
	installScript := filepath.Join(mod.Dir, "install.sh")
	if _, statErr := os.Stat(installScript); statErr == nil {
		if err := runScript(cfg, installScript, envVars); err != nil {
			cfg.UI.Error(fmt.Sprintf("Failed %s: install script error: %v", mod.Name, err))
			recordState(cfg, mod, "failed", err)
			return RunResult{Module: mod, Error: err, Duration: time.Since(start)}
		}
	}

	// Step 6: Deploy files (use spinner here — Go-native, no subprocess writes).
	spinner := cfg.UI.StartSpinner(fmt.Sprintf("Deploying %s files...", mod.Name))
	if err := deployFiles(cfg, mod, tmplCtx); err != nil {
		cfg.UI.StopSpinnerFail(spinner, fmt.Sprintf("Failed %s: file deployment error: %v", mod.Name, err))
		recordState(cfg, mod, "failed", err)
		return RunResult{Module: mod, Error: err, Duration: time.Since(start)}
	}
	cfg.UI.StopSpinnerSuccess(spinner, fmt.Sprintf("Deployed %s files", mod.Name))

	// Step 7: Run verify.sh if it exists.
	verifyScript := filepath.Join(mod.Dir, "verify.sh")
	if _, statErr := os.Stat(verifyScript); statErr == nil {
		if err := runScript(cfg, verifyScript, envVars); err != nil {
			cfg.UI.Error(fmt.Sprintf("Failed %s: verify script error: %v", mod.Name, err))
			recordState(cfg, mod, "failed", err)
			return RunResult{Module: mod, Error: err, Duration: time.Since(start)}
		}
	}

	// Step 8: Record success in state store.
	recordState(cfg, mod, "installed", nil)

	// Step 9: Print final result.
	cfg.UI.Success(fmt.Sprintf("Installed %s", mod.Name))

	return RunResult{
		Module:   mod,
		Success:  true,
		Duration: time.Since(start),
	}
}

// handlePrompts processes module prompts. In unattended mode, defaults are
// used. Otherwise the UI is used to prompt the user interactively. Returns
// a map of prompt key -> answer value.
func handlePrompts(cfg *RunConfig, mod *Module) (map[string]string, error) {
	answers := make(map[string]string, len(mod.Prompts))

	for _, p := range mod.Prompts {
		if cfg.Unattended {
			answers[p.Key] = p.Default
			continue
		}

		var answer string
		var err error

		switch p.Type {
		case "confirm":
			defaultBool := strings.EqualFold(p.Default, "true") || strings.EqualFold(p.Default, "yes") || strings.EqualFold(p.Default, "y")
			var confirmed bool
			confirmed, err = cfg.UI.PromptConfirm(p.Message, defaultBool)
			if err == nil {
				if confirmed {
					answer = "true"
				} else {
					answer = "false"
				}
			}
		case "choice":
			answer, err = cfg.UI.PromptChoice(p.Message, p.Options)
		default: // "input" or any other type
			answer, err = cfg.UI.PromptInput(p.Message, p.Default)
		}

		if err != nil {
			return nil, fmt.Errorf("prompt %q: %w", p.Key, err)
		}
		answers[p.Key] = answer
	}

	return answers, nil
}

// buildEnvVars constructs the full DOTFILES_* environment variable map
// passed to scripts and available during module execution.
func buildEnvVars(cfg *RunConfig, mod *Module, promptAnswers map[string]string) map[string]string {
	binPath, _ := os.Executable()

	env := map[string]string{
		"DOTFILES_OS":          cfg.SysInfo.OS,
		"DOTFILES_ARCH":        cfg.SysInfo.Arch,
		"DOTFILES_PKG_MGR":     cfg.SysInfo.PkgMgr,
		"DOTFILES_HAS_SUDO":    boolToStr(cfg.SysInfo.HasSudo),
		"DOTFILES_HOME":        cfg.SysInfo.HomeDir,
		"DOTFILES_DIR":         cfg.SysInfo.DotfilesDir,
		"DOTFILES_BIN":         binPath,
		"DOTFILES_MODULE_DIR":  mod.Dir,
		"DOTFILES_MODULE_NAME": mod.Name,
		"DOTFILES_INTERACTIVE": boolToStr(!cfg.Unattended),
		"DOTFILES_DRY_RUN":     boolToStr(cfg.DryRun),
		"DOTFILES_VERBOSE":     boolToStr(cfg.Verbose),
	}

	// Add prompt answers as DOTFILES_PROMPT_<UPPER_KEY>.
	for key, value := range promptAnswers {
		envKey := "DOTFILES_PROMPT_" + strings.ToUpper(key)
		env[envKey] = value
	}

	// Add user config values as DOTFILES_USER_*.
	if cfg.Config.User.Name != "" {
		env["DOTFILES_USER_NAME"] = cfg.Config.User.Name
	}
	if cfg.Config.User.Email != "" {
		env["DOTFILES_USER_EMAIL"] = cfg.Config.User.Email
	}
	if cfg.Config.User.GithubUser != "" {
		env["DOTFILES_USER_GITHUB_USER"] = cfg.Config.User.GithubUser
	}

	return env
}

// buildTemplateContext creates a template.Context from the current run
// configuration and environment variables for rendering template files.
func buildTemplateContext(cfg *RunConfig, mod *Module, envVars map[string]string) *template.Context {
	// Build module-specific settings map.
	modSettings := make(map[string]any)
	if settings, ok := cfg.Config.Modules[mod.Name]; ok {
		for k, v := range settings {
			modSettings[k] = v
		}
	}

	// Build user map from config.
	userMap := map[string]string{
		"name":        cfg.Config.User.Name,
		"email":       cfg.Config.User.Email,
		"github_user": cfg.Config.User.GithubUser,
	}

	// Build secrets map if provider is available.
	secretsMap := make(map[string]string)

	return &template.Context{
		User:        userMap,
		OS:          cfg.SysInfo.OS,
		Arch:        cfg.SysInfo.Arch,
		Home:        cfg.SysInfo.HomeDir,
		DotfilesDir: cfg.SysInfo.DotfilesDir,
		Module:      modSettings,
		Secrets:     secretsMap,
		Env:         envVars,
	}
}

// runScript executes a shell script with the set -euo pipefail preamble
// and sources lib/helpers.sh before sourcing the actual script. In dry-run
// mode it logs what would be executed instead.
func runScript(cfg *RunConfig, scriptPath string, envVars map[string]string) error {
	if cfg.DryRun {
		cfg.UI.Info(fmt.Sprintf("[dry-run] Would run script: %s", scriptPath))
		return nil
	}

	cfg.UI.Debug(fmt.Sprintf("Running script: %s", scriptPath))

	helpersPath := filepath.Join(cfg.SysInfo.DotfilesDir, "lib", "helpers.sh")

	// Build a wrapper script that:
	// 1. Enables strict mode
	// 2. Sources the shared helpers library if it exists
	// 3. Sources the actual module script
	var wrapper strings.Builder
	wrapper.WriteString("set -euo pipefail\n")
	wrapper.WriteString(fmt.Sprintf("if [ -f %q ]; then source %q; fi\n", helpersPath, helpersPath))
	wrapper.WriteString(fmt.Sprintf("source %q\n", scriptPath))

	cmd := exec.Command("bash", "-c", wrapper.String())

	// Set all environment variables on the command. Start with the current
	// process environment and layer DOTFILES_* vars on top.
	cmd.Env = os.Environ()
	for k, v := range envVars {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	// In interactive mode, connect stdin/stdout/stderr directly to the
	// terminal so commands like chsh can prompt for passwords.
	if !cfg.Unattended {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("script %s failed: %w", filepath.Base(scriptPath), err)
		}
		return nil
	}

	// Non-interactive / unattended: capture combined output and surface it
	// only on failure or when verbose logging is enabled.
	output, err := cmd.CombinedOutput()
	if len(output) > 0 && cfg.Verbose {
		cfg.UI.Debug(fmt.Sprintf("Script output:\n%s", string(output)))
	}

	if err != nil {
		// Show script output on failure so the user can diagnose the problem.
		// Only print here if verbose mode didn't already show it above.
		if len(output) > 0 && !cfg.Verbose {
			cfg.UI.Info(string(output))
		}
		return fmt.Errorf("script %s failed: %w", filepath.Base(scriptPath), err)
	}

	return nil
}

// deployFiles processes each FileEntry in the module, creating symlinks,
// copying files, or rendering templates as specified.
func deployFiles(cfg *RunConfig, mod *Module, tmplCtx *template.Context) error {
	for _, f := range mod.Files {
		src := filepath.Join(mod.Dir, f.Source)
		dest := expandHome(f.Dest, cfg.SysInfo.HomeDir)

		if cfg.DryRun {
			cfg.UI.Info(fmt.Sprintf("[dry-run] Would deploy %s -> %s (%s)", f.Source, dest, f.Type))
			continue
		}

		cfg.UI.Debug(fmt.Sprintf("Deploying %s -> %s (%s)", f.Source, dest, f.Type))

		// Ensure the destination directory exists.
		destDir := filepath.Dir(dest)
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", destDir, err)
		}

		switch f.Type {
		case "symlink":
			if err := deploySymlink(src, dest); err != nil {
				return fmt.Errorf("symlink %s -> %s: %w", src, dest, err)
			}
		case "copy":
			if err := deployCopy(src, dest); err != nil {
				return fmt.Errorf("copy %s -> %s: %w", src, dest, err)
			}
		case "template":
			if err := template.RenderToFile(src, dest, tmplCtx); err != nil {
				return fmt.Errorf("template %s -> %s: %w", src, dest, err)
			}
		default:
			return fmt.Errorf("unknown file type %q for %s", f.Type, f.Source)
		}
	}

	return nil
}

// deploySymlink creates a symbolic link at dest pointing to src. If dest
// already exists it is removed first.
func deploySymlink(src, dest string) error {
	// Resolve to absolute path for the symlink target.
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return err
	}

	// Remove existing file/symlink at dest.
	if _, err := os.Lstat(dest); err == nil {
		if err := os.Remove(dest); err != nil {
			return fmt.Errorf("removing existing %s: %w", dest, err)
		}
	}

	return os.Symlink(absSrc, dest)
}

// deployCopy copies the file at src to dest, preserving permissions.
func deployCopy(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	destFile, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode().Perm())
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}

// recordState persists the module's installation outcome to the state store.
func recordState(cfg *RunConfig, mod *Module, status string, runErr error) {
	ms := &state.ModuleState{
		Name:        mod.Name,
		Version:     mod.Version,
		Status:      status,
		InstalledAt: time.Now(),
		OS:          cfg.SysInfo.OS,
	}

	if runErr != nil {
		ms.Error = runErr.Error()
	}

	if err := cfg.State.Set(ms); err != nil {
		cfg.UI.Warn(fmt.Sprintf("Failed to save state for %s: %v", mod.Name, err))
	}
}

// expandHome replaces a leading ~ in path with the provided home directory.
func expandHome(path, homeDir string) string {
	if path == "~" {
		return homeDir
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

// boolToStr converts a bool to the string "true" or "false".
func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
