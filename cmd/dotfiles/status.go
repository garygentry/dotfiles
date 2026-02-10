package dotfiles

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

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
			name, version, status, installedAt, os string
		}

		rows := make([]row, 0, len(states))
		maxName, maxVersion, maxStatus, maxTime := 4, 7, 6, 10 // header widths

		for _, ms := range states {
			timeStr := formatTime(ms.InstalledAt)

			if len(ms.Name) > maxName {
				maxName = len(ms.Name)
			}
			if len(ms.Version) > maxVersion {
				maxVersion = len(ms.Version)
			}
			if len(ms.Status) > maxStatus {
				maxStatus = len(ms.Status)
			}
			if len(timeStr) > maxTime {
				maxTime = len(timeStr)
			}

			rows = append(rows, row{
				name:        ms.Name,
				version:     ms.Version,
				status:      ms.Status,
				installedAt: timeStr,
				os:          ms.OS,
			})
		}

		// Print table
		fmtStr := fmt.Sprintf("  %%-%ds  %%-%ds  %%-%ds  %%-%ds  %%s\n", maxName, maxVersion, maxStatus, maxTime)

		fmt.Fprintf(cmd.OutOrStdout(), fmtStr, "Name", "Version", "Status", "Installed", "OS")
		fmt.Fprintf(cmd.OutOrStdout(), "  %s  %s  %s  %s  %s\n",
			strings.Repeat("-", maxName),
			strings.Repeat("-", maxVersion),
			strings.Repeat("-", maxStatus),
			strings.Repeat("-", maxTime),
			strings.Repeat("-", 10))

		for _, r := range rows {
			fmt.Fprintf(cmd.OutOrStdout(), fmtStr, r.name, r.version, r.status, r.installedAt, r.os)
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

		u.Info(fmt.Sprintf("Total: %d modules (%d installed, %d failed)", len(states), succeeded, failed))

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
