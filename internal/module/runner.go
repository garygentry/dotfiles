package module

import (
	"context"
	"errors"
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

// ErrUserCancelled is returned when the user cancels an interactive prompt.
var ErrUserCancelled = errors.New("operation cancelled by user")

// MultiSelectOption represents a single option in a multi-select prompt.
type MultiSelectOption struct {
	Value       string
	Label       string
	Description string
}

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

	PromptMultiSelect(msg string, options []MultiSelectOption, preSelected []string) ([]string, error)
}

// RunConfig holds all configuration needed by the module runner.
type RunConfig struct {
	SysInfo            *sysinfo.SystemInfo
	Config             *config.Config
	UI                 RunnerUI
	Secrets            secrets.Provider
	State              *state.Store
	DryRun             bool
	Unattended         bool
	FailFast           bool
	Verbose            bool
	ScriptTimeout      time.Duration       // Default timeout for scripts (0 = use default)
	Force              bool                // Force reinstall even if up-to-date
	SkipFailed         bool                // Skip modules that failed previously
	UpdateOnly         bool                // Only update existing modules, don't install new
	ExplicitModules    map[string]bool     // Tracks which modules were explicitly selected (not auto-included)
	PromptDependencies bool                // Force prompts for auto-included dependencies
}

// ExecutionDecision represents the runner's decision about whether to execute a module.
type ExecutionDecision int

const (
	ExecutionSkip         ExecutionDecision = iota // Already up-to-date
	ExecutionInstallFresh                          // No previous state
	ExecutionInstallRetry                          // Failed previously
	ExecutionUpdateModule                          // Module changed
	ExecutionUpdateConfig                          // Config changed
	ExecutionForce                                 // --force flag
)

// RunResult captures the outcome of running a single module.
type RunResult struct {
	Module   *Module
	Success  bool
	Skipped  bool
	Error    error
	Duration time.Duration
	Notes    []string // Post-install notes from module definition
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

// shouldRunModule determines whether a module needs to be executed based on
// existing state, checksums, and configuration. This enables idempotent
// installations where modules are only re-run when necessary.
func shouldRunModule(mod *Module, existingState *state.ModuleState, cfg *RunConfig) (ExecutionDecision, string) {
	// No state = fresh install
	if existingState == nil {
		return ExecutionInstallFresh, "no previous installation"
	}

	// Force flag = always run
	if cfg.Force {
		return ExecutionForce, "--force flag set"
	}

	// Failed previously = retry (unless --skip-failed)
	if existingState.Status == "failed" {
		if cfg.SkipFailed {
			return ExecutionSkip, "failed previously, --skip-failed set"
		}
		return ExecutionInstallRetry, "retrying failed installation"
	}

	// Check module checksum (scripts/definition changed?)
	currentChecksum, err := ComputeModuleChecksum(mod)
	if err != nil {
		// If we can't compute checksum, be safe and re-run
		return ExecutionUpdateModule, fmt.Sprintf("checksum error: %v", err)
	}
	if existingState.Checksum != "" && currentChecksum != existingState.Checksum {
		return ExecutionUpdateModule, "module definition/scripts changed"
	}

	// Check config hash (user settings changed?)
	currentConfigHash := ComputeConfigHash(mod, cfg.Config)
	if existingState.ConfigHash != "" && currentConfigHash != existingState.ConfigHash {
		return ExecutionUpdateConfig, "user config values changed"
	}

	// Check version
	if mod.Version != existingState.Version {
		return ExecutionUpdateModule, "module version changed"
	}

	// Everything matches = skip
	return ExecutionSkip, "already installed and up-to-date"
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

	// Check if module needs running (idempotence check)
	existingState, _ := cfg.State.Get(mod.Name)
	decision, reason := shouldRunModule(mod, existingState, cfg)

	if decision == ExecutionSkip {
		cfg.UI.Info(fmt.Sprintf("✓ %s (skipped: %s)", mod.Name, reason))
		return RunResult{Module: mod, Success: true, Skipped: true, Duration: time.Since(start)}
	}

	// Show a static progress line while scripts run (no animated spinner).
	action := "Installing"
	if existingState != nil && existingState.Status == "installed" {
		action = "Updating"
	}
	cfg.UI.Info(fmt.Sprintf("%s %s (%s)...", action, mod.Name, reason))

	// Initialize state for operation recording
	// Preserve InstalledAt timestamp for updates
	installedAt := time.Now()
	if existingState != nil && !existingState.InstalledAt.IsZero() {
		installedAt = existingState.InstalledAt
	}

	modState := &state.ModuleState{
		Name:        mod.Name,
		Version:     mod.Version,
		Status:      "installing",
		InstalledAt: installedAt,
		OS:          cfg.SysInfo.OS,
	}

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
		modState.RecordOperation(state.Operation{
			Type:   "script_run",
			Action: "executed",
			Path:   osScript,
		})
		if err := runScript(cfg, mod, osScript, envVars); err != nil {
			cfg.UI.Error(fmt.Sprintf("Failed %s: os script error: %v", mod.Name, err))
			return handleInstallFailure(cfg, modState, mod, err, start)
		}
	}

	// Step 5: Run install.sh if it exists.
	installScript := filepath.Join(mod.Dir, "install.sh")
	if _, statErr := os.Stat(installScript); statErr == nil {
		modState.RecordOperation(state.Operation{
			Type:   "script_run",
			Action: "executed",
			Path:   installScript,
		})
		if err := runScript(cfg, mod, installScript, envVars); err != nil {
			cfg.UI.Error(fmt.Sprintf("Failed %s: install script error: %v", mod.Name, err))
			return handleInstallFailure(cfg, modState, mod, err, start)
		}
	}

	// Step 6: Deploy files (use spinner here — Go-native, no subprocess writes).
	spinner := cfg.UI.StartSpinner(fmt.Sprintf("Deploying %s files...", mod.Name))
	deployedCount, skippedCount, err := deployFiles(cfg, mod, tmplCtx, modState, existingState)
	if err != nil {
		cfg.UI.StopSpinnerFail(spinner, fmt.Sprintf("Failed %s: file deployment error: %v", mod.Name, err))
		return handleInstallFailure(cfg, modState, mod, err, start)
	}

	// Build informative message about file operations
	var fileMsg string
	if deployedCount > 0 && skippedCount > 0 {
		fileMsg = fmt.Sprintf("Deployed %d file(s), %d unchanged", deployedCount, skippedCount)
	} else if deployedCount > 0 {
		fileMsg = fmt.Sprintf("Deployed %d file(s)", deployedCount)
	} else if skippedCount > 0 {
		fileMsg = fmt.Sprintf("All %d file(s) unchanged", skippedCount)
	} else {
		fileMsg = "No files to deploy"
	}
	cfg.UI.StopSpinnerSuccess(spinner, fileMsg)

	// Step 7: Run verify.sh if it exists.
	verifyScript := filepath.Join(mod.Dir, "verify.sh")
	if _, statErr := os.Stat(verifyScript); statErr == nil {
		modState.RecordOperation(state.Operation{
			Type:   "script_run",
			Action: "executed",
			Path:   verifyScript,
		})
		if err := runScript(cfg, mod, verifyScript, envVars); err != nil {
			cfg.UI.Error(fmt.Sprintf("Failed %s: verify script error: %v", mod.Name, err))
			return handleInstallFailure(cfg, modState, mod, err, start)
		}
	}

	// Step 8: Record success in state store with operations and checksums.
	recordStateWithChecksums(cfg, modState, mod, "installed", nil)

	// Step 9: Print final result.
	action = "Installed"
	if existingState != nil && existingState.Status == "installed" {
		action = "Updated"
	}
	cfg.UI.Success(fmt.Sprintf("%s %s", action, mod.Name))

	return RunResult{
		Module:   mod,
		Success:  true,
		Duration: time.Since(start),
		Notes:    mod.Notes,
	}
}

// shouldShowPrompt determines whether a prompt should be shown interactively
// based on the module's selection context and the prompt's ShowWhen field.
func shouldShowPrompt(p Prompt, cfg *RunConfig, mod *Module, isExplicit bool) bool {
	// --prompt-dependencies flag overrides everything
	if cfg.PromptDependencies {
		return true
	}

	// Check show_when field (empty means use default behavior)
	switch p.ShowWhen {
	case "explicit_install":
		// Only show for explicitly selected modules
		return isExplicit
	case "interactive":
		// Always show in interactive mode
		return true
	case "always":
		// Always show this prompt
		return true
	case "":
		// Default: only show for explicit modules (same as explicit_install)
		return isExplicit
	default:
		// Unknown value, default to explicit_install behavior
		return isExplicit
	}
}

// handlePrompts processes module prompts. In unattended mode, defaults are
// used. For auto-included dependencies (not explicitly selected), defaults are
// also used unless --prompt-dependencies is set. Otherwise the UI is used to
// prompt the user interactively. Returns a map of prompt key -> answer value.
func handlePrompts(cfg *RunConfig, mod *Module) (map[string]string, error) {
	answers := make(map[string]string, len(mod.Prompts))

	// Check if this module was explicitly selected
	isExplicit := cfg.ExplicitModules != nil && cfg.ExplicitModules[mod.Name]

	for _, p := range mod.Prompts {
		// Always use defaults in unattended mode
		if cfg.Unattended {
			answers[p.Key] = p.Default
			continue
		}

		// Check if we should show this prompt interactively
		if !shouldShowPrompt(p, cfg, mod, isExplicit) {
			answers[p.Key] = p.Default
			// Log for transparency when using defaults for auto-included modules
			if !isExplicit && cfg.Verbose {
				cfg.UI.Debug(fmt.Sprintf("Using default for %s.%s: %s (auto-included dependency)", mod.Name, p.Key, p.Default))
			}
			continue
		}

		// Show interactive prompt
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
// mode it logs what would be executed instead. Scripts are executed with a
// timeout (default 5 minutes, configurable per-module via timeout field).
func runScript(cfg *RunConfig, mod *Module, scriptPath string, envVars map[string]string) error {
	if cfg.DryRun {
		cfg.UI.Info(fmt.Sprintf("[dry-run] Would run script: %s", scriptPath))
		return nil
	}

	cfg.UI.Debug(fmt.Sprintf("Running script: %s", scriptPath))

	// Determine timeout: module-specific > config > default (5 minutes)
	timeout := cfg.ScriptTimeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	if mod.Timeout != "" {
		if parsed, err := time.ParseDuration(mod.Timeout); err == nil {
			timeout = parsed
		} else {
			cfg.UI.Warn(fmt.Sprintf("Invalid timeout %q in module %s, using default", mod.Timeout, mod.Name))
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	helpersPath := filepath.Join(cfg.SysInfo.DotfilesDir, "lib", "helpers.sh")

	// Build a wrapper script that:
	// 1. Enables strict mode
	// 2. Sources the shared helpers library if it exists
	// 3. Sources the actual module script
	var wrapper strings.Builder
	wrapper.WriteString("set -euo pipefail\n")
	wrapper.WriteString(fmt.Sprintf("if [ -f %q ]; then source %q; fi\n", helpersPath, helpersPath))
	wrapper.WriteString(fmt.Sprintf("source %q\n", scriptPath))

	cmd := exec.CommandContext(ctx, "bash", "-c", wrapper.String())

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
			if ctx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("script %s timed out after %v", filepath.Base(scriptPath), timeout)
			}
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
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("script %s timed out after %v", filepath.Base(scriptPath), timeout)
		}
		// Show script output on failure so the user can diagnose the problem.
		// Only print here if verbose mode didn't already show it above.
		if len(output) > 0 && !cfg.Verbose {
			cfg.UI.Info(string(output))
		}
		return fmt.Errorf("script %s failed: %w", filepath.Base(scriptPath), err)
	}

	return nil
}

// shouldDeployFile determines whether a file needs to be deployed based on
// existing state, source hash, and destination state. This enables file-level
// idempotence where files are only deployed when necessary.
func shouldDeployFile(fileEntry FileEntry, src, dest, sourceHash string,
	existingFile *state.FileState, cfg *RunConfig) (bool, string) {

	if cfg.Force {
		return true, "force flag set"
	}

	if existingFile == nil {
		return true, "not previously deployed"
	}

	// Source content changed = must redeploy
	if sourceHash != existingFile.SourceHash {
		return true, "source file changed"
	}

	// Check if destination exists
	if _, err := os.Lstat(dest); os.IsNotExist(err) {
		return true, "destination file missing"
	}

	// For symlinks: check if pointing to correct location
	if fileEntry.Type == "symlink" {
		target, err := os.Readlink(dest)
		if err != nil {
			return true, "symlink read error"
		}
		expectedTarget, _ := filepath.Abs(src)
		if target != expectedTarget {
			return true, "symlink points to wrong location"
		}
		return false, "symlink already correct"
	}

	// For copy/template: check file content hash
	currentHash, err := ComputeFileHash(dest)
	if err != nil {
		return true, "destination hash error"
	}

	// Destination matches our deployed content = unchanged
	if currentHash == existingFile.DeployedHash {
		return false, "destination unchanged since deployment"
	}

	// Content changed but source didn't = user modified
	// Don't redeploy (protect user changes)
	return false, "user modified (source unchanged)"
}

// deployFiles processes each FileEntry in the module, creating symlinks,
// copying files, or rendering templates as specified.
// Operations are recorded in modState for rollback capability.
// File-level idempotence: files are only deployed when source changed or dest is missing.
// Returns (deployedCount, skippedCount, error).
func deployFiles(cfg *RunConfig, mod *Module, tmplCtx *template.Context, modState *state.ModuleState, existingState *state.ModuleState) (int, int, error) {
	// Build map of previously deployed files for quick lookup
	existingFiles := make(map[string]*state.FileState)
	if existingState != nil {
		for i := range existingState.FileStates {
			fs := &existingState.FileStates[i]
			existingFiles[fs.Dest] = fs
		}
	}

	var deployedCount, skippedCount int

	for _, f := range mod.Files {
		src := filepath.Join(mod.Dir, f.Source)
		dest := expandHome(f.Dest, cfg.SysInfo.HomeDir)

		// Compute source hash for change detection
		sourceHash, err := ComputeFileHash(src)
		if err != nil {
			return 0, 0, fmt.Errorf("computing hash for %s: %w", src, err)
		}

		// Check if deployment needed
		existingFile := existingFiles[dest]
		needsDeploy, reason := shouldDeployFile(f, src, dest, sourceHash, existingFile, cfg)

		if !needsDeploy {
			cfg.UI.Debug(fmt.Sprintf("Skipping %s: %s", dest, reason))
			skippedCount++

			// Carry forward existing state with updated check time
			modState.FileStates = append(modState.FileStates, state.FileState{
				Source:       f.Source,
				Dest:         dest,
				Type:         f.Type,
				DeployedAt:   existingFile.DeployedAt,
				SourceHash:   sourceHash,
				DeployedHash: existingFile.DeployedHash,
				UserModified: reason == "user modified (source unchanged)",
				LastChecked:  time.Now(),
			})
			continue
		}

		if cfg.DryRun {
			cfg.UI.Info(fmt.Sprintf("[dry-run] Would deploy %s -> %s (%s): %s", f.Source, dest, f.Type, reason))
			continue
		}

		// Backup user-modified files before overwriting
		if existingFile != nil && existingFile.UserModified {
			if err := createBackup(dest, cfg, mod.Name); err != nil {
				cfg.UI.Warn(fmt.Sprintf("Backup failed for %s: %v", dest, err))
			}
		}

		cfg.UI.Debug(fmt.Sprintf("Deploying %s -> %s (%s): %s", f.Source, dest, f.Type, reason))

		// Ensure the destination directory exists.
		destDir := filepath.Dir(dest)
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return 0, 0, fmt.Errorf("creating directory %s: %w", destDir, err)
		}

		// Record directory creation if it was created
		if _, err := os.Stat(destDir); err == nil {
			modState.RecordOperation(state.Operation{
				Type:   "dir_create",
				Action: "created",
				Path:   destDir,
			})
		}

		// Check if file exists before deploying (for backup/modification tracking)
		fileExisted := false
		if _, err := os.Lstat(dest); err == nil {
			fileExisted = true
		}

		var deployedHash string

		switch f.Type {
		case "symlink":
			if err := deploySymlink(src, dest); err != nil {
				return 0, 0, fmt.Errorf("symlink %s -> %s: %w", src, dest, err)
			}
			// For symlinks, we use source hash as deployed hash
			deployedHash = sourceHash

			modState.RecordOperation(state.Operation{
				Type:   "file_deploy",
				Action: "symlinked",
				Path:   dest,
				Metadata: map[string]string{
					"source":       src,
					"type":         "symlink",
					"file_existed": fmt.Sprintf("%v", fileExisted),
					"source_hash":  sourceHash,
				},
			})

		case "copy":
			if err := deployCopy(src, dest); err != nil {
				return 0, 0, fmt.Errorf("copy %s -> %s: %w", src, dest, err)
			}
			// For copies, compute the deployed file's hash
			deployedHash, err = ComputeFileHash(dest)
			if err != nil {
				return 0, 0, fmt.Errorf("computing deployed hash for %s: %w", dest, err)
			}

			action := "created"
			if fileExisted {
				action = "modified"
			}
			modState.RecordOperation(state.Operation{
				Type:   "file_deploy",
				Action: action,
				Path:   dest,
				Metadata: map[string]string{
					"source":      src,
					"type":        "copy",
					"source_hash": sourceHash,
				},
			})

		case "template":
			if err := template.RenderToFile(src, dest, tmplCtx); err != nil {
				return 0, 0, fmt.Errorf("template %s -> %s: %w", src, dest, err)
			}
			// For templates, compute the rendered file's hash
			deployedHash, err = ComputeFileHash(dest)
			if err != nil {
				return 0, 0, fmt.Errorf("computing deployed hash for %s: %w", dest, err)
			}

			action := "created"
			if fileExisted {
				action = "modified"
			}
			modState.RecordOperation(state.Operation{
				Type:   "file_deploy",
				Action: action,
				Path:   dest,
				Metadata: map[string]string{
					"source":      src,
					"type":        "template",
					"source_hash": sourceHash,
				},
			})

		default:
			return 0, 0, fmt.Errorf("unknown file type %q for %s", f.Type, f.Source)
		}

		// File was successfully deployed
		deployedCount++

		// Record file state for idempotence tracking
		modState.FileStates = append(modState.FileStates, state.FileState{
			Source:       f.Source,
			Dest:         dest,
			Type:         f.Type,
			DeployedAt:   time.Now(),
			SourceHash:   sourceHash,
			DeployedHash: deployedHash,
			UserModified: false,
			LastChecked:  time.Now(),
		})
	}

	return deployedCount, skippedCount, nil
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

// recordStateWithOps persists the module state including recorded operations.
func recordStateWithOps(cfg *RunConfig, modState *state.ModuleState, status string, runErr error) {
	modState.Status = status
	modState.UpdatedAt = time.Now()

	if runErr != nil {
		modState.Error = runErr.Error()
	}

	if err := cfg.State.Set(modState); err != nil {
		cfg.UI.Warn(fmt.Sprintf("Failed to save state for %s: %v", modState.Name, err))
	}
}

// recordStateWithChecksums persists the module state including checksums and config hash
// for idempotence tracking. Used after successful installation/update.
func recordStateWithChecksums(cfg *RunConfig, modState *state.ModuleState, mod *Module, status string, runErr error) {
	modState.Status = status
	modState.UpdatedAt = time.Now()

	if runErr != nil {
		modState.Error = runErr.Error()
	}

	// Compute and store checksums for change detection
	if checksum, err := ComputeModuleChecksum(mod); err == nil {
		modState.Checksum = checksum
	} else {
		cfg.UI.Debug(fmt.Sprintf("Failed to compute checksum for %s: %v", mod.Name, err))
	}

	modState.ConfigHash = ComputeConfigHash(mod, cfg.Config)

	if err := cfg.State.Set(modState); err != nil {
		cfg.UI.Warn(fmt.Sprintf("Failed to save state for %s: %v", modState.Name, err))
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

// handleInstallFailure handles installation failures by offering rollback options.
// In interactive mode, prompts the user to [S]kip or [U]ndo.
// In unattended mode, records failure and continues.
func handleInstallFailure(cfg *RunConfig, modState *state.ModuleState, mod *Module, installErr error, start time.Time) RunResult {
	// Record failure
	recordStateWithOps(cfg, modState, "failed", installErr)

	// In unattended mode, just return the error
	if cfg.Unattended {
		return RunResult{Module: mod, Error: installErr, Duration: time.Since(start)}
	}

	// Check if rollback is possible
	if !modState.CanRollback() {
		cfg.UI.Warn("No operations to rollback")
		return RunResult{Module: mod, Error: installErr, Duration: time.Since(start)}
	}

	// Show rollback options
	cfg.UI.Info("")
	cfg.UI.Warn(fmt.Sprintf("Installation of %s failed with %d recorded operations", mod.Name, len(modState.Operations)))
	cfg.UI.Info("Options:")
	cfg.UI.Info("  [S]kip - Leave partial installation as-is")
	cfg.UI.Info("  [U]ndo - Rollback changes and clean up")

	// Prompt for action
	choice, promptErr := cfg.UI.PromptChoice("What would you like to do?", []string{"skip", "undo"})
	if promptErr != nil {
		cfg.UI.Warn("Failed to read input, skipping rollback")
		return RunResult{Module: mod, Error: installErr, Duration: time.Since(start)}
	}

	if choice == "undo" {
		cfg.UI.Info("Rolling back changes...")

		// Execute rollback operations
		rollbackCount := 0
		rollbackErrors := 0

		for i := len(modState.Operations) - 1; i >= 0; i-- {
			op := modState.Operations[i]
			if err := executeRollbackOp(cfg, op); err != nil {
				cfg.UI.Warn(fmt.Sprintf("Rollback operation %d failed: %v", i, err))
				rollbackErrors++
			} else {
				rollbackCount++
			}
		}

		// Remove state after rollback
		if err := cfg.State.Remove(mod.Name); err != nil {
			cfg.UI.Warn(fmt.Sprintf("Failed to remove state: %v", err))
		}

		if rollbackErrors > 0 {
			cfg.UI.Warn(fmt.Sprintf("Rolled back %d/%d operations (%d errors)", 
				rollbackCount, len(modState.Operations), rollbackErrors))
		} else {
			cfg.UI.Success(fmt.Sprintf("Successfully rolled back %d operations", rollbackCount))
		}
	} else {
		cfg.UI.Info("Skipping rollback, partial installation preserved")
	}

	return RunResult{Module: mod, Error: installErr, Duration: time.Since(start)}
}

// executeRollbackOp executes a single rollback operation.
func executeRollbackOp(cfg *RunConfig, op state.Operation) error {
	cfg.UI.Debug(fmt.Sprintf("Rolling back: %s %s %s", op.Type, op.Action, op.Path))

	switch op.Type {
	case "file_deploy":
		return rollbackFileOp(op)
	case "dir_create":
		return rollbackDirOp(op)
	case "script_run":
		// Scripts cannot be automatically rolled back
		cfg.UI.Debug(fmt.Sprintf("Script rollback not supported: %s", op.Path))
		return nil
	case "package_install":
		// Packages are not automatically removed
		cfg.UI.Debug(fmt.Sprintf("Package rollback not supported: %s", op.Path))
		return nil
	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}
}

// rollbackFileOp rolls back a file deployment operation.
func rollbackFileOp(op state.Operation) error {
	switch op.Action {
	case "created", "symlinked":
		// Remove the file/symlink
		if err := os.Remove(op.Path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing %s: %w", op.Path, err)
		}
		return nil

	case "modified":
		// Restore from backup if available
		if backup := op.Metadata["backup_path"]; backup != "" {
			data, err := os.ReadFile(backup)
			if err != nil {
				return fmt.Errorf("reading backup %s: %w", backup, err)
			}
			if err := os.WriteFile(op.Path, data, 0o644); err != nil {
				return fmt.Errorf("restoring %s: %w", op.Path, err)
			}
			os.Remove(backup)
		}
		return nil

	default:
		return fmt.Errorf("unknown file action: %s", op.Action)
	}
}

// rollbackDirOp rolls back a directory creation operation.
func rollbackDirOp(op state.Operation) error {
	// Only remove if empty
	entries, err := os.ReadDir(op.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already gone
		}
		return fmt.Errorf("reading directory %s: %w", op.Path, err)
	}

	if len(entries) == 0 {
		if err := os.Remove(op.Path); err != nil {
			return fmt.Errorf("removing directory %s: %w", op.Path, err)
		}
	}

	return nil
}
