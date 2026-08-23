package dotfiles

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/garygentry/dotfiles/internal/config"
	"github.com/garygentry/dotfiles/internal/module"
	"github.com/garygentry/dotfiles/internal/sysinfo"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	validateJSON   bool
	validateStrict bool
)

type validateResult struct {
	Module string   `json:"module"`
	Errors []string `json:"errors"`
}

var validateCmd = &cobra.Command{
	Use:   "validate [modules...]",
	Short: "Validate module YAML schemas",
	Long: `Validate checks every module's module.yml for schema errors such as
invalid field values, missing required fields, and broken references.

Without arguments all modules are validated. Specific modules can be named.

Exit code is 0 when all modules pass, 1 when any module fails.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sys, err := sysinfo.Detect()
		if err != nil {
			return fmt.Errorf("system detection: %w", err)
		}

		// Content dir (if any) contributes overlay modules to validate; a failed
		// config load falls back to the engine-only root.
		var contentDir string
		if cfg, cerr := config.Load(sys.DotfilesDir); cerr == nil {
			contentDir = cfg.ContentDir
		}

		roots := module.ModuleRoots(sys.DotfilesDir, contentDir)
		allModules, strictErrs, err := loadModulesForValidate(roots, validateStrict)
		if err != nil {
			return fmt.Errorf("module discovery: %w", err)
		}

		// Filter to requested modules if args provided.
		modules := allModules
		if len(args) > 0 {
			requested := make(map[string]bool, len(args))
			for _, a := range args {
				requested[a] = true
			}
			modules = modules[:0]
			for _, m := range allModules {
				if requested[m.Name] {
					modules = append(modules, m)
				}
			}
		}

		if len(modules) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No modules found.")
			return nil
		}

		var results []validateResult
		failCount := 0

		for _, m := range modules {
			errs := module.Validate(m)
			errs = append(errs, module.ValidateFiles(m)...)
			errs = append(errs, module.ValidateSymlinkTargets(m)...)
			if se := strictErrs[m.Name]; se != "" {
				errs = append(errs, "unknown YAML key: "+se)
			}

			if len(errs) > 0 {
				failCount++
			}
			results = append(results, validateResult{Module: m.Name, Errors: errs})
		}

		if validateJSON {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(results)
		}

		// Human-readable output.
		for _, r := range results {
			if len(r.Errors) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "✓ %s\n", r.Module)
			} else {
				for _, e := range r.Errors {
					fmt.Fprintf(cmd.OutOrStdout(), "✗ %s  — %s\n", r.Module, e)
				}
			}
		}

		if failCount > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "\n%d module(s) failed validation.\n", failCount)
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	validateCmd.Flags().BoolVar(&validateJSON, "json", false, "Output results as JSON")
	validateCmd.Flags().BoolVar(&validateStrict, "strict", false, "Reject unknown YAML keys")
	rootCmd.AddCommand(validateCmd)
}

// loadModulesForValidate reads all modules across the ordered roots, applying
// content-wins precedence (a later root's same-name module replaces an earlier
// one, so validation checks the module that would actually be used). Missing
// roots are skipped. When strict is true each module.yml is additionally
// decoded with KnownFields(true); any unknown-key errors are returned in the
// strictErrs map keyed by module name.
func loadModulesForValidate(roots []string, strict bool) ([]*module.Module, map[string]string, error) {
	byName := make(map[string]*module.Module)
	strictErrs := make(map[string]string)

	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, nil, err
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			ymlPath := filepath.Join(root, entry.Name(), "module.yml")
			if _, err := os.Stat(ymlPath); os.IsNotExist(err) {
				continue
			}

			m, err := module.ParseModuleYAML(ymlPath)
			if err != nil {
				return nil, nil, fmt.Errorf("parsing %s: %w", ymlPath, err)
			}

			// Content-wins: a later root replaces the earlier module and its
			// strict result, so a stale engine strictErr never lingers.
			delete(strictErrs, m.Name)
			if strict {
				data, err := os.ReadFile(ymlPath)
				if err != nil {
					return nil, nil, err
				}
				dec := yaml.NewDecoder(bytes.NewReader(data))
				dec.KnownFields(true)
				var raw map[string]interface{}
				if err := dec.Decode(&raw); err != nil {
					strictErrs[m.Name] = err.Error()
				}
			}

			byName[m.Name] = m
		}
	}

	modules := make([]*module.Module, 0, len(byName))
	for _, m := range byName {
		modules = append(modules, m)
	}
	sort.Slice(modules, func(i, j int) bool { return modules[i].Name < modules[j].Name })

	return modules, strictErrs, nil
}
