package dotfiles

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/garygentry/dotfiles/internal/config"
	"github.com/garygentry/dotfiles/internal/module"
	"github.com/garygentry/dotfiles/internal/sysinfo"
	"github.com/spf13/cobra"
)

var notesCmd = &cobra.Command{
	Use:   "notes [module]...",
	Short: "Print modules' post-install notes",
	Long: `Notes prints the post-install notes of the named modules, or of every module
that has notes when none is named. Install shows a module's notes on its first
install on a machine (and again when they change); this command shows them any time.

Example:
  dotfiles notes mise yazi`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sys, err := sysinfo.Detect()
		if err != nil {
			return fmt.Errorf("system detection: %w", err)
		}
		var contentDir string
		if cfg, cerr := config.Load(sys.DotfilesDir); cerr == nil {
			contentDir = cfg.ContentDir
		}
		modules, err := module.DiscoverRoots(module.ModuleRoots(sys.DotfilesDir, contentDir))
		if err != nil {
			return fmt.Errorf("module discovery: %w", err)
		}
		return printNotes(cmd.OutOrStdout(), modules, args)
	},
}

func init() {
	rootCmd.AddCommand(notesCmd)
}

// printNotes writes the notes of the named modules (all modules with notes when
// names is empty). An unknown name is an error; a named module without notes
// says so.
func printNotes(w io.Writer, modules []*module.Module, names []string) error {
	byName := make(map[string]*module.Module, len(modules))
	for _, m := range modules {
		byName[m.Name] = m
	}
	if len(names) == 0 {
		for _, m := range modules {
			if len(m.Notes) > 0 {
				names = append(names, m.Name)
			}
		}
		sort.Strings(names)
	}
	var unknown []string
	for i, name := range names {
		m, ok := byName[name]
		if !ok {
			unknown = append(unknown, name)
			continue
		}
		if i > 0 {
			fmt.Fprintln(w)
		}
		if len(m.Notes) == 0 {
			fmt.Fprintf(w, "%s: no notes\n", name)
			continue
		}
		fmt.Fprintf(w, "%s:\n", name)
		for _, note := range m.Notes {
			fmt.Fprintf(w, "  - %s\n", note)
		}
	}
	if len(unknown) > 0 {
		fmt.Fprintf(os.Stderr, "dotfiles notes: unknown module(s): %s\n", strings.Join(unknown, ", "))
		return fmt.Errorf("unknown module(s): %s", strings.Join(unknown, ", "))
	}
	return nil
}
