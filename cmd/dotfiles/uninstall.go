package dotfiles

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/garygentry/dotfiles/internal/state"
	"github.com/garygentry/dotfiles/internal/sysinfo"
	"github.com/garygentry/dotfiles/internal/ui"
	"github.com/spf13/cobra"
)

var (
	uninstallForce bool
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <module>...",
	Short: "Uninstall modules and remove their files",
	Long: `Uninstall reverses the installation of modules by removing files
and restoring backups based on recorded operations. This command reads
the operation history from the module state and undoes each action.

Example:
  dotfiles uninstall git
  dotfiles uninstall git zsh --dry-run
  dotfiles uninstall tmux --force`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		u := ui.New(verbose)

		sys, err := sysinfo.Detect()
		if err != nil {
			return fmt.Errorf("system detection: %w", err)
		}

		store := state.NewStore(filepath.Join(sys.DotfilesDir, ".state"))

		for _, moduleName := range args {
			if err := uninstallModule(u, store, moduleName); err != nil {
				u.Error(fmt.Sprintf("Failed to uninstall %s: %v", moduleName, err))
				if !uninstallForce {
					return err
				}
			}
		}

		return nil
	},
}

func init() {
	uninstallCmd.Flags().BoolVar(&uninstallForce, "force", false, "Continue uninstalling even if errors occur")
	rootCmd.AddCommand(uninstallCmd)
}

func uninstallModule(u *ui.UI, store *state.Store, moduleName string) error {
	u.Info(fmt.Sprintf("Uninstalling %s...", moduleName))

	// Load module state
	ms, err := store.Get(moduleName)
	if err != nil {
		return fmt.Errorf("reading state: %w", err)
	}

	if ms == nil {
		u.Warn(fmt.Sprintf("Module %s is not installed", moduleName))
		return nil
	}

	if !ms.CanRollback() {
		u.Warn(fmt.Sprintf("Module %s has no recorded operations to rollback", moduleName))
		u.Info("This module may have been installed before operation recording was implemented")

		// Ask for confirmation to remove state anyway
		if !dryRun && !uninstallForce {
			confirm, err := u.PromptConfirm("Remove module state anyway?", false)
			if err != nil || !confirm {
				return fmt.Errorf("uninstall cancelled")
			}
		}

		// Remove state
		if !dryRun {
			if err := store.Remove(moduleName); err != nil {
				return fmt.Errorf("removing state: %w", err)
			}
			u.Success(fmt.Sprintf("Removed %s from state", moduleName))
		}
		return nil
	}

	// Show rollback plan
	instructions := ms.RollbackInstructions()
	u.Info(fmt.Sprintf("Rollback plan (%d operations):", len(instructions)))
	for i, inst := range instructions {
		u.Info(fmt.Sprintf("  %d. %s", i+1, inst))
	}

	if dryRun {
		u.Info("[dry-run] Would uninstall module and execute rollback operations")
		return nil
	}

	// Ask for confirmation
	if !uninstallForce {
		confirm, err := u.PromptConfirm(fmt.Sprintf("Proceed with uninstall of %s?", moduleName), false)
		if err != nil || !confirm {
			return fmt.Errorf("uninstall cancelled")
		}
	}

	// Execute rollback operations in reverse order
	var errors []string
	for i := len(ms.Operations) - 1; i >= 0; i-- {
		op := ms.Operations[i]
		if err := rollbackOperation(u, op); err != nil {
			errMsg := fmt.Sprintf("operation %d failed: %v", i, err)
			errors = append(errors, errMsg)
			u.Warn(errMsg)
			if !uninstallForce {
				return fmt.Errorf("rollback failed: %w", err)
			}
		}
	}

	// Remove from state
	if err := store.Remove(moduleName); err != nil {
		return fmt.Errorf("removing state: %w", err)
	}

	if len(errors) > 0 {
		u.Warn(fmt.Sprintf("Uninstalled %s with %d errors", moduleName, len(errors)))
	} else {
		u.Success(fmt.Sprintf("Uninstalled %s successfully", moduleName))
	}

	return nil
}

func rollbackOperation(u *ui.UI, op state.Operation) error {
	switch op.Type {
	case "file_deploy":
		return rollbackFileDeploy(u, op)

	case "dir_create":
		return rollbackDirCreate(u, op)

	case "script_run":
		u.Debug(fmt.Sprintf("Script was executed: %s (no automatic rollback)", op.Path))
		return nil

	case "package_install":
		u.Debug(fmt.Sprintf("Package was installed: %s (manual removal may be needed)", op.Path))
		return nil

	default:
		u.Debug(fmt.Sprintf("Unknown operation type: %s", op.Type))
		return nil
	}
}

func rollbackFileDeploy(u *ui.UI, op state.Operation) error {
	switch op.Action {
	case "created", "symlinked":
		// Remove the file/symlink
		u.Debug(fmt.Sprintf("Removing: %s", op.Path))
		if err := os.Remove(op.Path); err != nil {
			if os.IsNotExist(err) {
				u.Debug(fmt.Sprintf("File already removed: %s", op.Path))
				return nil
			}
			return fmt.Errorf("removing %s: %w", op.Path, err)
		}
		return nil

	case "modified":
		// Check if there's a backup
		if backup := op.Metadata["backup_path"]; backup != "" {
			u.Debug(fmt.Sprintf("Restoring: %s from %s", op.Path, backup))
			// Read backup
			data, err := os.ReadFile(backup)
			if err != nil {
				return fmt.Errorf("reading backup %s: %w", backup, err)
			}
			// Write to original location
			if err := os.WriteFile(op.Path, data, 0o644); err != nil {
				return fmt.Errorf("restoring %s: %w", op.Path, err)
			}
			// Remove backup
			os.Remove(backup)
			return nil
		}
		u.Warn(fmt.Sprintf("File was modified but no backup available: %s", op.Path))
		return nil

	default:
		u.Debug(fmt.Sprintf("Unknown file action: %s for %s", op.Action, op.Path))
		return nil
	}
}

func rollbackDirCreate(u *ui.UI, op state.Operation) error {
	// Only remove if directory is empty
	u.Debug(fmt.Sprintf("Checking if directory is empty: %s", op.Path))

	entries, err := os.ReadDir(op.Path)
	if err != nil {
		if os.IsNotExist(err) {
			u.Debug(fmt.Sprintf("Directory already removed: %s", op.Path))
			return nil
		}
		return fmt.Errorf("reading directory %s: %w", op.Path, err)
	}

	if len(entries) == 0 {
		u.Debug(fmt.Sprintf("Removing empty directory: %s", op.Path))
		if err := os.Remove(op.Path); err != nil {
			return fmt.Errorf("removing directory %s: %w", op.Path, err)
		}
	} else {
		u.Debug(fmt.Sprintf("Directory not empty, keeping: %s (%d files)", op.Path, len(entries)))
	}

	return nil
}
