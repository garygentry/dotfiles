package dotfiles

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/garygentry/dotfiles/internal/module"
	"github.com/garygentry/dotfiles/internal/state"
	"github.com/garygentry/dotfiles/internal/sysinfo"
	"github.com/garygentry/dotfiles/internal/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available modules and their status",
	RunE: func(cmd *cobra.Command, args []string) error {
		u := ui.New(verbose)

		sys, err := sysinfo.Detect()
		if err != nil {
			return fmt.Errorf("system detection: %w", err)
		}

		modulesDir := filepath.Join(sys.DotfilesDir, "modules")
		modules, err := module.Discover(modulesDir)
		if err != nil {
			return fmt.Errorf("module discovery: %w", err)
		}

		if len(modules) == 0 {
			u.Warn("No modules found in " + modulesDir)
			return nil
		}

		store := state.NewStore(filepath.Join(sys.DotfilesDir, ".state"))

		// Build table data.
		type row struct {
			name, description, os, status string
		}

		rows := make([]row, 0, len(modules))
		maxName, maxDesc, maxOS := 4, 11, 2 // header widths: Name, Description, OS

		for _, m := range modules {
			desc := m.Description
			if desc == "" {
				desc = "-"
			}

			osStr := "all"
			if len(m.OS) > 0 {
				osStr = strings.Join(m.OS, ",")
			}

			status := "not installed"
			ms, _ := store.Get(m.Name)
			if ms != nil {
				status = ms.Status
			}

			if len(m.Name) > maxName {
				maxName = len(m.Name)
			}
			if len(desc) > maxDesc {
				maxDesc = len(desc)
			}
			if len(osStr) > maxOS {
				maxOS = len(osStr)
			}

			rows = append(rows, row{
				name:        m.Name,
				description: desc,
				os:          osStr,
				status:      status,
			})
		}

		// Cap description width.
		if maxDesc > 40 {
			maxDesc = 40
		}

		fmtStr := fmt.Sprintf("  %%-%ds  %%-%ds  %%-%ds  %%s\n", maxName, maxDesc, maxOS)

		fmt.Fprintf(cmd.OutOrStdout(), "\n")
		fmt.Fprintf(cmd.OutOrStdout(), fmtStr, "Name", "Description", "OS", "Status")
		fmt.Fprintf(cmd.OutOrStdout(), "  %s  %s  %s  %s\n",
			strings.Repeat("-", maxName),
			strings.Repeat("-", maxDesc),
			strings.Repeat("-", maxOS),
			strings.Repeat("-", 13))

		for _, r := range rows {
			desc := r.description
			if len(desc) > maxDesc {
				desc = desc[:maxDesc-3] + "..."
			}
			fmt.Fprintf(cmd.OutOrStdout(), fmtStr, r.name, desc, r.os, r.status)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\n")

		_ = u // ui is available for future verbose output
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
