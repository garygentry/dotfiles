package dotfiles

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/garygentry/dotfiles/internal/config"
	"github.com/garygentry/dotfiles/internal/module"
	"github.com/garygentry/dotfiles/internal/secrets"
	"github.com/garygentry/dotfiles/internal/state"
	"github.com/garygentry/dotfiles/internal/sysinfo"
	"github.com/garygentry/dotfiles/internal/ui"
	"github.com/spf13/cobra"
)

var (
	unattended bool
	failFast   bool
	force      bool
	skipFailed bool
	updateOnly bool
)

var installCmd = &cobra.Command{
	Use:   "install [modules...]",
	Short: "Install and configure dotfiles modules",
	Long: `Install runs the specified modules (or all modules if none specified)
through a 5-phase flow: config loading, secret authentication, dependency
resolution, module execution, and summary output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		u := ui.New(verbose)

		// Phase 1: System detection and config loading.
		u.Info("Detecting system...")
		sys, err := sysinfo.Detect()
		if err != nil {
			return fmt.Errorf("system detection: %w", err)
		}
		u.Success(fmt.Sprintf("System: %s/%s (pkg: %s)", sys.OS, sys.Arch, sys.PkgMgr))

		// Auto-enable unattended mode when stdin is not interactive (e.g. curl | bash).
		if !sys.IsInteractive && !unattended {
			u.Info("Non-interactive stdin detected, using default values for prompts")
			unattended = true
		}

		cfg, err := config.Load(sys.DotfilesDir)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		u.Debug(fmt.Sprintf("Profile: %s", cfg.Profile))

		// Determine requested modules: CLI args > profile > all.
		requested := args
		if len(requested) == 0 {
			profileModules, err := config.LoadProfile(sys.DotfilesDir, cfg.Profile)
			if err != nil {
				u.Debug(fmt.Sprintf("No profile %q found, using all modules", cfg.Profile))
			} else {
				requested = profileModules
			}
		}

		// Phase 2: Secrets authentication.
		provider := secrets.NewProvider(cfg.Secrets.Provider, cfg.Secrets.Account)
		if cfg.Secrets.Provider != "" && !provider.Available() {
			u.Warn(fmt.Sprintf("Secrets provider %q is configured but not available (is the CLI installed?), continuing without secrets", cfg.Secrets.Provider))
		} else if provider.Available() && !dryRun {
			if provider.IsAuthenticated() {
				u.Success(fmt.Sprintf("Authenticated with %s", provider.Name()))
			} else if unattended {
				u.Info("Skipping secrets authentication (unattended mode)")
			} else {
				setupNow, promptErr := u.PromptConfirm(
					fmt.Sprintf("%s is not authenticated. Set up now?", provider.Name()),
					false,
				)
				if promptErr != nil {
					u.Warn("Could not read input, continuing without secrets")
				} else if setupNow {
					if err := provider.Authenticate(); err != nil {
						u.Warn(fmt.Sprintf("Authentication failed: %v (continuing without secrets)", err))
					} else {
						u.Success(fmt.Sprintf("Authenticated with %s", provider.Name()))
					}
				} else {
					u.Info("Skipping secrets. Modules that use secrets will fall back to defaults.")
					u.Info("Run 'dotfiles install' later to set up 1Password.")
				}
			}
		}

		// Phase 3: Module discovery and dependency resolution.
		modulesDir := filepath.Join(sys.DotfilesDir, "modules")
		allModules, err := module.Discover(modulesDir)
		if err != nil {
			return fmt.Errorf("module discovery: %w", err)
		}
		if len(allModules) == 0 {
			u.Warn("No modules found in " + modulesDir)
			return nil
		}

		// Interactive module selection when no CLI args provided.
		if len(args) == 0 && !unattended {
			options := make([]module.MultiSelectOption, 0, len(allModules))
			for _, m := range allModules {
				if !m.SupportsOS(sys.OS) {
					continue
				}
				desc := m.Description
				if desc == "" {
					desc = "no description"
				}
				options = append(options, module.MultiSelectOption{
					Value:       m.Name,
					Label:       m.Name,
					Description: desc,
				})
			}

			selected, selErr := u.PromptMultiSelect("Select modules to install", options, requested)
			if selErr != nil {
				if errors.Is(selErr, module.ErrUserCancelled) {
					u.Info("Module selection cancelled")
					return nil
				}
				return fmt.Errorf("module selection: %w", selErr)
			}

			if len(selected) == 0 {
				u.Warn("No modules selected, nothing to do")
				return nil
			}

			requested = selected
		}

		plan, err := module.Resolve(allModules, requested, sys.OS)
		if err != nil {
			return fmt.Errorf("dependency resolution: %w", err)
		}

		// Show auto-included dependencies if the user made an interactive selection.
		if len(args) == 0 && !unattended && len(requested) > 0 {
			requestedSet := make(map[string]bool, len(requested))
			for _, name := range requested {
				requestedSet[name] = true
			}
			var autoIncluded []string
			for _, m := range plan.Modules {
				if !requestedSet[m.Name] {
					autoIncluded = append(autoIncluded, m.Name)
				}
			}
			if len(autoIncluded) > 0 {
				u.Info(fmt.Sprintf("Auto-including dependencies: %s", strings.Join(autoIncluded, ", ")))
			}
		}

		// Filter modules for update-only mode
		if updateOnly {
			stateStore := state.NewStore(filepath.Join(sys.DotfilesDir, ".state"))
			var updatableModules []*module.Module
			var skippedNew []*module.Module

			for _, m := range plan.Modules {
				existingState, _ := stateStore.Get(m.Name)
				if existingState != nil && existingState.Status == "installed" {
					updatableModules = append(updatableModules, m)
				} else {
					skippedNew = append(skippedNew, m)
				}
			}

			plan.Modules = updatableModules
			plan.Skipped = append(plan.Skipped, skippedNew...)

			if len(skippedNew) > 0 {
				var names []string
				for _, m := range skippedNew {
					names = append(names, m.Name)
				}
				u.Info(fmt.Sprintf("Skipping new modules (--update-only): %s", strings.Join(names, ", ")))
			}
		}

		u.PrintExecutionPlan(plan.Modules, plan.Skipped)

		if dryRun {
			u.Info("Dry-run mode: no changes will be made")
			return nil
		}

		// Phase 4: Module execution.
		runCfg := &module.RunConfig{
			SysInfo:    sys,
			Config:     cfg,
			UI:         u,
			Secrets:    provider,
			State:      state.NewStore(filepath.Join(sys.DotfilesDir, ".state")),
			DryRun:     dryRun,
			Unattended: unattended,
			FailFast:   failFast,
			Verbose:    verbose,
			Force:      force,
			SkipFailed: skipFailed,
			UpdateOnly: updateOnly,
		}

		results := module.Run(runCfg, plan)

		// Phase 5: Summary output.
		var succeeded, failed, skipped int
		for _, r := range results {
			switch {
			case r.Skipped:
				skipped++
			case r.Success:
				succeeded++
			default:
				failed++
				u.Error(fmt.Sprintf("  %s: %v", r.Module.Name, r.Error))
			}
		}
		skipped += len(plan.Skipped)

		elapsed := time.Since(start).Round(time.Millisecond)
		u.Info(fmt.Sprintf("Completed in %s: %d succeeded, %d failed, %d skipped",
			elapsed, succeeded, failed, skipped))

		// Display post-run notes from modules that ran successfully.
		var allNotes []string
		for _, r := range results {
			if r.Success && !r.Skipped && len(r.Notes) > 0 {
				for _, note := range r.Notes {
					allNotes = append(allNotes, fmt.Sprintf("[%s] %s", r.Module.Name, note))
				}
			}
		}
		if len(allNotes) > 0 {
			u.Info("")
			u.Warn("Post-install notes:")
			for _, note := range allNotes {
				u.Warn("  " + note)
			}
		}

		if failed > 0 {
			return fmt.Errorf("%d module(s) failed", failed)
		}
		return nil
	},
}

func init() {
	installCmd.Flags().BoolVar(&unattended, "unattended", false, "Run without prompts, using defaults")
	installCmd.Flags().BoolVar(&failFast, "fail-fast", false, "Stop on first module failure")
	installCmd.Flags().BoolVar(&force, "force", false, "Force reinstall all modules even if up-to-date")
	installCmd.Flags().BoolVar(&skipFailed, "skip-failed", false, "Skip modules that failed previously")
	installCmd.Flags().BoolVar(&updateOnly, "update-only", false, "Only update existing modules, don't install new ones")
	rootCmd.AddCommand(installCmd)
}
