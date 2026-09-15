package dotfiles

import (
	"errors"
	"fmt"
	"os"
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
	failFast           bool
	force              bool
	skipFailed         bool
	updateOnly         bool
	promptDependencies bool
	profile            string
	hostConfig         string
	metricsTextfile    string
)

var installCmd = &cobra.Command{
	Use:   "install [modules...]",
	Short: "Install and configure dotfiles modules",
	Long: `Install runs the specified modules (or all modules if none specified)
through a 5-phase flow: config loading, secret authentication, dependency
resolution, module execution, and summary output.`,
	RunE: func(cmd *cobra.Command, args []string) (rerr error) {
		start := time.Now()
		u := ui.New(verbose)

		// Phase 7 (D13): reconcile observability. The emitter runs as a defer so it
		// fires on every return path AND on a panic (so a crashed run never publishes
		// a green exit_status). It emits ONLY for a full-profile reconcile — the
		// operation Phase 7 observes — never for a targeted `install <mod>` or a
		// no-profile fallback: because the estate sets DOTFILES_METRICS_TEXTFILE
		// globally, any ad-hoc `install <x>` would otherwise overwrite the shared
		// .prom with drift/armed=0 and a fresh timestamp, masking the real reconcile's
		// signal. drift/armed/dir/profile/full are captured as the run proceeds; the
		// defer reads their final values. Also skipped when off or on a dry-run.
		var (
			metricsDir     string
			metricsProfile string
			metricsDrift   int
			metricsArmed   bool
			metricsFull    bool
		)
		defer func() {
			r := recover()
			if metricsTextfile == "" || dryRun || !metricsFull {
				if r != nil {
					panic(r) // preserve the crash; we just weren't emitting
				}
				return
			}
			writeReconcileMetrics(u, metricsTextfile, reconcileMetrics{
				start:       start,
				exitStatus:  boolToExit(rerr != nil || r != nil),
				driftCount:  metricsDrift,
				pruneArmed:  metricsArmed,
				dotfilesDir: metricsDir,
				profile:     metricsProfile,
			})
			if r != nil {
				panic(r) // re-raise after publishing exit_status=1
			}
		}()

		// Phase 1: System detection and config loading.
		u.Info("Detecting system...")
		sys, err := sysinfo.Detect()
		if err != nil {
			return fmt.Errorf("system detection: %w", err)
		}
		metricsDir = sys.DotfilesDir // for the pin_sha git read at emit time
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
		// A profile named on the command line or in the environment was asked for
		// deliberately; one that only comes from config.yml is a default. The
		// difference matters below: falling back to *every module* is a reasonable
		// default and a terrible answer to an explicit request.
		explicitProfile := profile != "" || os.Getenv("DOTFILES_PROFILE") != ""
		if profile != "" {
			cfg.Profile = profile
		}
		u.Debug(fmt.Sprintf("Profile: %s (explicit: %t)", cfg.Profile, explicitProfile))
		metricsProfile = cfg.Profile // the RESOLVED profile, not just the flag (metrics info label)

		// Surface the content overlay, and catch a mistyped DOTFILES_CONTENT_DIR
		// (set but no such directory) rather than silently ignoring it.
		if cfg.ContentDir != "" {
			if fi, statErr := os.Stat(cfg.ContentDir); statErr != nil || !fi.IsDir() {
				u.Warn(fmt.Sprintf("DOTFILES_CONTENT_DIR is set to %q but no such directory exists; the overlay is ignored.", cfg.ContentDir))
			} else {
				u.Info(fmt.Sprintf("Using content overlay: %s", cfg.ContentDir))
			}
		}

		profileModules, profileConfigLayers, profileErr := config.LoadProfileResolved(sys.DotfilesDir, cfg.ContentDir, cfg.Profile)
		if profileErr != nil {
			// Only ErrProfileNotFound on an implicit (config.yml) profile is safe to
			// treat as "no profile → fall back to all modules." Every other error
			// (cycle, malformed YAML, missing parent under `extends:`, IO) is a real
			// failure that must not silently install the whole module set: a user
			// editing config.yml to add a broken `extends:` chain would otherwise get
			// a fleet-wide install with only a Debug-level trace.
			if !explicitProfile && errors.Is(profileErr, config.ErrProfileNotFound) {
				u.Debug(fmt.Sprintf("No profile %q found, using all modules", cfg.Profile))
			} else {
				u.Error(fmt.Sprintf("Profile %q could not be loaded: %v", cfg.Profile, profileErr))
				if errors.Is(profileErr, config.ErrProfileNotFound) && !config.ProfileIsPath(cfg.Profile) {
					u.Info(fmt.Sprintf("Profiles live in %s; --profile also accepts a path to a profile file.",
						filepath.Join(sys.DotfilesDir, "profiles")))
				}
				if errors.Is(profileErr, config.ErrProfileCycle) {
					u.Info("A profile appears in its own `extends:` ancestor chain; fix the cycle in the profile files.")
				}
				return profileErr
			}
		}

		// Determine requested modules: CLI args > profile > all.
		requested := args
		if len(requested) == 0 && profileErr == nil {
			requested = profileModules
		}

		// A full-profile reconcile — no module args, a real resolved profile — is the
		// only shape the observability contract (Phase 7 / D13) publishes metrics for.
		// A targeted `install <mod>` or a no-profile all-modules fallback is not a
		// reconcile and must not overwrite the reconcile's .prom (see the emitter defer).
		metricsFull = len(args) == 0 && profileErr == nil && len(profileModules) > 0

		// Phase 5 (ADR 0028 §D10): layer config values estate-default → baseline →
		// overlays (declared order) → host. estate-default (the config.yml chain) is
		// already in cfg.Modules; apply the profile `config:` layers in resolved
		// extends order, then the host-owned layer last — it wins over everything, and
		// the reconcile only ever reads it. A malformed host file is a hard error
		// (T2 discipline); an absent one is simply no host overrides. Layering always
		// applies, independent of which modules this run installs.
		hostModules, hostErr := config.LoadHostConfig(hostConfig)
		if hostErr != nil {
			u.Error(fmt.Sprintf("Host config %q could not be loaded: %v", hostConfig, hostErr))
			return hostErr
		}
		cfg.Modules = config.ComposeModules(cfg.Modules, profileConfigLayers, hostModules)

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

		// Phase 3: Module discovery and dependency resolution. Discover across the
		// engine's modules and the content overlay's modules/ (content wins on
		// same-name modules). With no content dir this is exactly the engine root.
		roots := module.ModuleRoots(sys.DotfilesDir, cfg.ContentDir)
		allModules, err := module.DiscoverRoots(roots)
		if err != nil {
			return fmt.Errorf("module discovery: %w", err)
		}
		if len(allModules) == 0 {
			u.Warn("No modules found in " + strings.Join(roots, ", "))
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
			u.DrainStdin()

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

		// Prune reconcile (ADR 0028 §D6): only in a full PROFILE reconcile. Gates:
		// no module args (else plan.Modules is a subset), not --update-only, and a
		// profile actually in effect (never prune against the "all modules"
		// fallback). The desired set is the PROFILE's resolved closure — computed
		// fresh from profileModules, NOT from `plan`/`requested`, so an interactive
		// multi-select (or any subset the operator picked this run) can't turn a
		// still-desired module into a prune target. Candidates are computed now so
		// they show in the plan / dry-run; execution happens after a clean install.
		var pruneCandidates []string
		var pruneOptedIn bool
		pruneEligible := len(args) == 0 && !updateOnly && !noPrune && profileErr == nil && len(profileModules) > 0
		if pruneEligible {
			profilePlan, pErr := module.Resolve(allModules, profileModules, sys.OS)
			if pErr != nil {
				// Never prune against an uncertain desired set — disable, don't guess.
				u.Warn(fmt.Sprintf("Prune disabled: could not resolve the profile's desired set: %v", pErr))
				pruneEligible = false
			} else {
				// Desired = resolved profile modules + deps, PLUS OS-skipped desired
				// modules: a module the profile wants but that isn't applicable to this
				// OS is not drift and must never be pruned.
				effectiveNames := make(map[string]bool, len(profilePlan.Modules)+len(profilePlan.Skipped))
				for _, m := range profilePlan.Modules {
					effectiveNames[m.Name] = true
				}
				for _, m := range profilePlan.Skipped {
					effectiveNames[m.Name] = true
				}

				protect, mErr := loadAdditionsManifest(additionsManifest)
				if mErr != nil {
					return mErr // malformed manifest: hard fail, never prune blind
				}
				stateStore := state.NewStore(filepath.Join(sys.DotfilesDir, ".state"))
				pruneCandidates, err = computePruneCandidates(stateStore, effectiveNames, protect)
				if err != nil {
					return fmt.Errorf("computing prune candidates: %w", err)
				}
				pruneOptedIn = hasPruneOptIn(sys.DotfilesDir) || allowPrune

				// Phase 7 (D-d): report real drift for visibility on every host, but
				// "arm" the drift alert only where the host has opted into prune — so
				// migration hosts (additive-only) surface drift_count without tripping
				// ReconcileDrift (P7-3). Captured before the fail-closed flip below so
				// drift is reported honestly even when prune is disabled for safety.
				metricsDrift = len(pruneCandidates)
				metricsArmed = pruneOptedIn

				// Fail CLOSED: if we would actually remove modules but no additions
				// manifest path is configured, the host-local protect list is silently
				// absent (a real hazard under `ssh … bash -lc`, which strips DOTFILES_*
				// env). Refuse to prune rather than delete with nothing protected;
				// install still proceeds.
				if pruneOptedIn && len(pruneCandidates) > 0 && additionsManifest == "" {
					u.Warn("Prune disabled: no additions manifest configured (--additions-manifest / DOTFILES_ADDITIONS_MANIFEST).")
					u.Warn("Refusing to prune with nothing protected. Configure the path (an absent file = empty allowlist) to enable prune.")
					pruneEligible = false
				} else if len(pruneCandidates) > 0 {
					list := strings.Join(pruneCandidates, ", ")
					if pruneOptedIn {
						u.Warn(fmt.Sprintf("Prune: %d module(s) not in the profile will be removed: %s", len(pruneCandidates), list))
					} else {
						u.Warn(fmt.Sprintf("Drift: %d installed module(s) not in the profile: %s", len(pruneCandidates), list))
						u.Info("Not removed — this host has not opted into prune. Pass --allow-prune to remove them")
						u.Info("(recorded per host), or list them under `modules:` in the additions manifest to keep them.")
					}
				}
			}
		}

		if dryRun {
			u.Info("Dry-run mode: no changes will be made")
			return nil
		}

		// Phase 4: Module execution.
		runCfg := &module.RunConfig{
			SysInfo:            sys,
			Config:             cfg,
			UI:                 u,
			Secrets:            provider,
			State:              state.NewStore(filepath.Join(sys.DotfilesDir, ".state")),
			DryRun:             dryRun,
			Unattended:         unattended,
			FailFast:           failFast,
			Verbose:            verbose,
			Force:              force,
			SkipFailed:         skipFailed,
			UpdateOnly:         updateOnly,
			ExplicitModules:    plan.ExplicitlyRequested,
			PromptDependencies: promptDependencies,
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

		// Phase 6: Prune — remove modules absent from the profile. ONLY on a CLEAN
		// reconcile: if any install failed, skip prune AND skip recording the opt-in
		// (a durable destructive policy must not be armed by a failed run, and a
		// partial state must not drive removals).
		var prunedWithErrors int
		if pruneEligible {
			if failed > 0 {
				if pruneOptedIn && len(pruneCandidates) > 0 {
					u.Warn(fmt.Sprintf("Prune skipped: %d module(s) failed — not pruning on a partial reconcile.", failed))
				}
			} else {
				if allowPrune && !hasPruneOptIn(sys.DotfilesDir) {
					if err := recordPruneOptIn(sys.DotfilesDir); err != nil {
						u.Warn(fmt.Sprintf("could not record prune opt-in: %v", err))
					} else {
						u.Info("Recorded prune opt-in for this host; future reconciles prune automatically.")
					}
				}
				if pruneOptedIn && len(pruneCandidates) > 0 {
					stateStore := state.NewStore(filepath.Join(sys.DotfilesDir, ".state"))
					var pruned int
					for _, name := range pruneCandidates {
						ms, gErr := stateStore.Get(name)
						if gErr != nil {
							u.Warn(fmt.Sprintf("Prune: could not read state for %s: %v (skipping)", name, gErr))
							prunedWithErrors++
							continue
						}
						if ms == nil {
							continue // already gone
						}
						u.Info(fmt.Sprintf("Pruning %s (not in profile)...", name))
						if errs := removeModuleForPrune(u, stateStore, ms); len(errs) > 0 {
							// State is preserved (removeModuleForPrune keeps it on error) so
							// the module stays a candidate for the next reconcile.
							prunedWithErrors++
							u.Warn(fmt.Sprintf("Prune of %s incomplete (%d error(s)); state kept for retry", name, len(errs)))
						} else {
							pruned++
							u.Success(fmt.Sprintf("Pruned %s", name))
						}
					}
					if pruned > 0 {
						u.Success(fmt.Sprintf("Pruned %d module(s) not in the profile", pruned))
					}
				}
			}
		}

		if failed > 0 {
			return fmt.Errorf("%d module(s) failed", failed)
		}
		if prunedWithErrors > 0 {
			// Surface prune failures as a non-zero exit so an unattended fleet
			// reconcile does not report success while modules were only half-removed.
			return fmt.Errorf("%d module(s) failed to prune cleanly", prunedWithErrors)
		}
		return nil
	},
}

func init() {
	installCmd.Flags().BoolVar(&failFast, "fail-fast", false, "Stop on first module failure")
	installCmd.Flags().BoolVar(&force, "force", false, "Force reinstall all modules even if up-to-date")
	installCmd.Flags().BoolVar(&skipFailed, "skip-failed", false, "Skip modules that failed previously")
	installCmd.Flags().BoolVar(&updateOnly, "update-only", false, "Only update existing modules, don't install new ones")
	installCmd.Flags().BoolVar(&promptDependencies, "prompt-dependencies", false, "Show prompts for auto-included dependency modules (default: use defaults)")
	installCmd.Flags().StringVar(&profile, "profile", "", "Use a specific profile: a name from profiles/ (e.g. minimal, developer) or a path to a profile file")
	installCmd.Flags().BoolVar(&allowPrune, "allow-prune", false, "Opt this host into prune: remove installed modules absent from the profile, and record the opt-in so future reconciles prune automatically (ADR 0028 §D6)")
	installCmd.Flags().BoolVar(&noPrune, "no-prune", false, "Skip prune for this run even if the host has opted in")
	installCmd.Flags().StringVar(&additionsManifest, "additions-manifest", os.Getenv("DOTFILES_ADDITIONS_MANIFEST"), "Path to the host-owned additions manifest (modules to protect from prune); the estate sets /etc/gnet/local-additions.yml")
	installCmd.Flags().StringVar(&hostConfig, "host-config", os.Getenv("DOTFILES_HOST_CONFIG"), "Path to the host-owned config layer (settings this host overrides/adds, applied last); the estate sets /etc/gnet/local-config.yml (ADR 0028 §D10)")
	installCmd.Flags().StringVar(&metricsTextfile, "metrics-textfile", os.Getenv("DOTFILES_METRICS_TEXTFILE"), "Path to write reconcile metrics as a node_exporter textfile (empty = off); the estate sets /var/lib/node_exporter/textfile/gnet_reconcile.prom (ADR 0028 §D13)")
	rootCmd.AddCommand(installCmd)
}
