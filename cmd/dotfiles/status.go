package dotfiles

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/garygentry/dotfiles/internal/config"
	"github.com/garygentry/dotfiles/internal/module"
	"github.com/garygentry/dotfiles/internal/state"
	"github.com/garygentry/dotfiles/internal/sysinfo"
	"github.com/garygentry/dotfiles/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of installed modules",
	Long: `Status displays information about currently installed modules including
version, installation time, and status. This helps you understand what's
installed on your system and when it was set up.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		u := ui.New(verbose)

		sys, err := sysinfo.Detect()
		if err != nil {
			return fmt.Errorf("system detection: %w", err)
		}

		cfg, err := config.Load(sys.DotfilesDir)
		if err != nil {
			u.Debug(fmt.Sprintf("Could not load config: %v", err))
			cfg = &config.Config{} // Use empty config if loading fails
		}

		// Discover available modules
		modulesDir := filepath.Join(sys.DotfilesDir, "modules")
		allModules, err := module.Discover(modulesDir)
		if err != nil {
			u.Debug(fmt.Sprintf("Module discovery failed: %v", err))
			allModules = nil
		}

		// Build module lookup map
		modulesByName := make(map[string]*module.Module)
		for _, mod := range allModules {
			modulesByName[mod.Name] = mod
		}

		store := state.NewStore(filepath.Join(sys.DotfilesDir, ".state"))

		// Get all module states
		states, err := store.GetAll()
		if err != nil {
			return fmt.Errorf("reading state: %w", err)
		}

		if len(states) == 0 {
			u.Info("No modules installed yet")
			u.Info("Run 'dotfiles install' to get started")
			return nil
		}

		// Print header
		fmt.Fprintf(cmd.OutOrStdout(), "\n")
		u.Info(fmt.Sprintf("System: %s/%s", sys.OS, sys.Arch))
		u.Info(fmt.Sprintf("Dotfiles directory: %s", sys.DotfilesDir))
		fmt.Fprintf(cmd.OutOrStdout(), "\n")

		// Build table data
		type row struct {
			name, version, status, updateStatus, installedAt, os string
		}

		rows := make([]row, 0, len(states))
		maxName, maxVersion, maxStatus, maxUpdate, maxTime := 4, 7, 6, 6, 10 // header widths
		var needsUpdate, userModified int

		for _, ms := range states {
			timeStr := formatTime(ms.InstalledAt)
			updateStatus := "✓"

			// Check if module needs update
			if mod, exists := modulesByName[ms.Name]; exists && ms.Status == "installed" {
				// Check version
				if mod.Version != ms.Version {
					updateStatus = "• version"
					needsUpdate++
				} else {
					// Check module checksum
					currentChecksum, _ := module.ComputeModuleChecksum(mod)
					if ms.Checksum != "" && currentChecksum != "" && currentChecksum != ms.Checksum {
						updateStatus = "• changed"
						needsUpdate++
					} else {
						// Check config hash
						currentConfigHash := module.ComputeConfigHash(mod, cfg)
						if ms.ConfigHash != "" && currentConfigHash != ms.ConfigHash {
							updateStatus = "• config"
							needsUpdate++
						} else {
							// Check for user modifications
							for _, fs := range ms.FileStates {
								if fs.UserModified {
									updateStatus = "⚠ modified"
									userModified++
									break
								}
							}
						}
					}
				}
			} else if ms.Status == "failed" {
				updateStatus = "! failed"
			}

			if len(ms.Name) > maxName {
				maxName = len(ms.Name)
			}
			if len(ms.Version) > maxVersion {
				maxVersion = len(ms.Version)
			}
			if len(ms.Status) > maxStatus {
				maxStatus = len(ms.Status)
			}
			if len(updateStatus) > maxUpdate {
				maxUpdate = len(updateStatus)
			}
			if len(timeStr) > maxTime {
				maxTime = len(timeStr)
			}

			rows = append(rows, row{
				name:         ms.Name,
				version:      ms.Version,
				status:       ms.Status,
				updateStatus: updateStatus,
				installedAt:  timeStr,
				os:           ms.OS,
			})
		}

		// Print table
		fmtStr := fmt.Sprintf("  %%-%ds  %%-%ds  %%-%ds  %%-%ds  %%-%ds  %%s\n", maxName, maxVersion, maxStatus, maxUpdate, maxTime)

		fmt.Fprintf(cmd.OutOrStdout(), fmtStr, "Name", "Version", "Status", "Update", "Installed", "OS")
		fmt.Fprintf(cmd.OutOrStdout(), "  %s  %s  %s  %s  %s  %s\n",
			strings.Repeat("-", maxName),
			strings.Repeat("-", maxVersion),
			strings.Repeat("-", maxStatus),
			strings.Repeat("-", maxUpdate),
			strings.Repeat("-", maxTime),
			strings.Repeat("-", 10))

		for _, r := range rows {
			fmt.Fprintf(cmd.OutOrStdout(), fmtStr, r.name, r.version, r.status, r.updateStatus, r.installedAt, r.os)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\n")

		// Summary
		var succeeded, failed int
		for _, ms := range states {
			if ms.Status == "installed" {
				succeeded++
			} else if ms.Status == "failed" {
				failed++
			}
		}

		summaryParts := []string{fmt.Sprintf("%d installed", succeeded)}
		if failed > 0 {
			summaryParts = append(summaryParts, fmt.Sprintf("%d failed", failed))
		}
		if needsUpdate > 0 {
			summaryParts = append(summaryParts, fmt.Sprintf("%d need update", needsUpdate))
		}
		if userModified > 0 {
			summaryParts = append(summaryParts, fmt.Sprintf("%d user modified", userModified))
		}

		u.Info(fmt.Sprintf("Total: %d modules (%s)", len(states), strings.Join(summaryParts, ", ")))

		// Legend
		fmt.Fprintf(cmd.OutOrStdout(), "\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  Update status:  ✓ up-to-date  • needs update  ⚠ user modified  ! failed\n")

		// Show any failed modules with error details
		if failed > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "\n")
			u.Warn("Failed modules:")
			for _, ms := range states {
				if ms.Status == "failed" && ms.Error != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "  • %s: %s\n", ms.Name, ms.Error)
				}
			}
		}

		// Helpful hints
		if needsUpdate > 0 || failed > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "\n")
			if needsUpdate > 0 {
				u.Info("Run 'dotfiles install' to update out-of-date modules")
			}
			if failed > 0 {
				u.Info("Run 'dotfiles install --force <module>' to retry failed modules")
				u.Info("Or use 'dotfiles install --skip-failed' to skip them")
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

// formatTime formats a timestamp in a human-readable way.
// Shows relative time for recent timestamps, absolute date for older ones.
func formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("2006-01-02")
	}
}
