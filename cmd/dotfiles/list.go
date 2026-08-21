package dotfiles

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/garygentry/dotfiles/internal/config"
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

		// Content dir (if any) contributes overlay modules; a failed config load
		// falls back to the engine-only root.
		var contentDir string
		if cfg, cerr := config.Load(sys.DotfilesDir); cerr == nil {
			contentDir = cfg.ContentDir
		} else {
			u.Debug(fmt.Sprintf("Could not load config: %v", cerr))
		}

		roots := module.ModuleRoots(sys.DotfilesDir, contentDir)
		modules, err := module.DiscoverRoots(roots)
		if err != nil {
			return fmt.Errorf("module discovery: %w", err)
		}

		if len(modules) == 0 {
			u.Warn("No modules found in " + strings.Join(roots, ", "))
			return nil
		}

		store := state.NewStore(filepath.Join(sys.DotfilesDir, ".state"))

		// Only show the Source column when the overlay actually contributed a
		// modules root (len(roots) > 1). A content dir that is set but has no
		// modules/ dir adds nothing, so the output stays byte-identical to
		// before the overlay existed.
		showSource := len(roots) > 1

		// Build table data.
		type row struct {
			name, description, os, source, status string
		}

		rows := make([]row, 0, len(modules))
		maxName, maxDesc, maxOS, maxSource := 4, 11, 2, 6 // header widths: Name, Description, OS, Source

		for _, m := range modules {
			desc := m.Description
			if desc == "" {
				desc = "-"
			}

			osStr := "all"
			if len(m.OS) > 0 {
				osStr = strings.Join(m.OS, ",")
			}

			source := m.Source
			if source == "" {
				source = module.SourceBuiltin
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
			if len(source) > maxSource {
				maxSource = len(source)
			}

			rows = append(rows, row{
				name:        m.Name,
				description: desc,
				os:          osStr,
				source:      source,
				status:      status,
			})
		}

		// Cap description width.
		if maxDesc > 40 {
			maxDesc = 40
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\n")
		if showSource {
			fmtStr := fmt.Sprintf("  %%-%ds  %%-%ds  %%-%ds  %%-%ds  %%s\n", maxName, maxDesc, maxOS, maxSource)
			fmt.Fprintf(cmd.OutOrStdout(), fmtStr, "Name", "Description", "OS", "Source", "Status")
			fmt.Fprintf(cmd.OutOrStdout(), "  %s  %s  %s  %s  %s\n",
				strings.Repeat("-", maxName),
				strings.Repeat("-", maxDesc),
				strings.Repeat("-", maxOS),
				strings.Repeat("-", maxSource),
				strings.Repeat("-", 13))
			for _, r := range rows {
				desc := r.description
				if len(desc) > maxDesc {
					desc = desc[:maxDesc-3] + "..."
				}
				fmt.Fprintf(cmd.OutOrStdout(), fmtStr, r.name, desc, r.os, r.source, r.status)
			}
		} else {
			fmtStr := fmt.Sprintf("  %%-%ds  %%-%ds  %%-%ds  %%s\n", maxName, maxDesc, maxOS)
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
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\n")

		_ = u // ui is available for future verbose output
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
