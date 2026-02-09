package module

import (
	"os"
	"path/filepath"
	"sort"
)

// Discover scans each immediate subdirectory of modulesDir for a module.yml
// file, parses it, and returns the collected modules sorted first by Priority
// (ascending) then by Name (ascending). Subdirectories that do not contain a
// module.yml are silently skipped.
func Discover(modulesDir string) ([]*Module, error) {
	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		return nil, err
	}

	var modules []*Module
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		ymlPath := filepath.Join(modulesDir, entry.Name(), "module.yml")
		if _, err := os.Stat(ymlPath); os.IsNotExist(err) {
			continue
		}

		m, err := ParseModuleYAML(ymlPath)
		if err != nil {
			return nil, err
		}

		modules = append(modules, m)
	}

	sort.Slice(modules, func(i, j int) bool {
		if modules[i].Priority != modules[j].Priority {
			return modules[i].Priority < modules[j].Priority
		}
		return modules[i].Name < modules[j].Name
	})

	return modules, nil
}
