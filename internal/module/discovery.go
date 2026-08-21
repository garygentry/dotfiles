package module

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
)

// Discover scans each immediate subdirectory of modulesDir for a module.yml
// file, parses it, and returns the collected modules sorted first by Priority
// (ascending) then by Name (ascending). Subdirectories that do not contain a
// module.yml are silently skipped.
func Discover(modulesDir string) ([]*Module, error) {
	modules, err := scanRoot(modulesDir)
	if err != nil {
		return nil, err
	}
	sortModules(modules)
	return modules, nil
}

// ModuleRoots returns the ordered module roots: the engine's built-in modules
// first, then the content dir's modules/ if a content dir is set and that
// directory exists. Passing the result to DiscoverRoots gives content-wins
// precedence (later roots override earlier ones by module name).
//
// When contentDir is "" (no overlay configured) the result is exactly the
// single engine root, so discovery behaves identically to before the overlay.
func ModuleRoots(dotfilesDir, contentDir string) []string {
	roots := []string{filepath.Join(dotfilesDir, "modules")}
	if contentDir != "" {
		contentModules := filepath.Join(contentDir, "modules")
		if fi, err := os.Stat(contentModules); err == nil && fi.IsDir() {
			roots = append(roots, contentModules)
		}
	}
	return roots
}

// DiscoverRoots discovers modules across an ordered list of roots. Later roots
// override earlier ones by module name (so pass [engineModules, contentModules]
// for content-wins precedence). Missing roots are skipped. A malformed
// module.yml in any root is a hard error naming the file, not a silent skip.
//
// Each returned module's Source records where it came from: SourceBuiltin (only
// in the first root), SourceOverride (a later root shadowed an earlier same-name
// module), or SourceCustom (defined only in a later root). Modules are sorted by
// Priority (ascending) then Name (ascending).
func DiscoverRoots(roots []string) ([]*Module, error) {
	byName := make(map[string]*Module)
	for i, root := range roots {
		mods, err := scanRoot(root)
		if err != nil {
			// A root that does not exist is skipped (an unset/absent content
			// modules dir simply contributes nothing). Any other error is fatal.
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		for _, m := range mods {
			if _, shadowed := byName[m.Name]; shadowed {
				// A later root replaces a same-name module from an earlier one.
				m.Source = SourceOverride
			} else if i == 0 {
				m.Source = SourceBuiltin
			} else {
				m.Source = SourceCustom
			}
			byName[m.Name] = m
		}
	}

	modules := make([]*Module, 0, len(byName))
	for _, m := range byName {
		modules = append(modules, m)
	}
	sortModules(modules)
	return modules, nil
}

// scanRoot reads one modules root: it parses and validates the module.yml in
// each immediate subdirectory and returns the collected modules unsorted. A
// missing root returns the underlying os error (callers decide whether that is
// fatal); a subdirectory without a module.yml is silently skipped; a malformed
// module.yml is a hard error naming the file.
func scanRoot(modulesDir string) ([]*Module, error) {
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
			return nil, fmt.Errorf("parsing %s: %w", ymlPath, err)
		}

		// Warn about invalid modules but continue so a single bad module
		// does not prevent the rest from loading.
		if validErrs := validateModule(m); len(validErrs) > 0 {
			for _, e := range validErrs {
				log.Printf("WARN module %q: %s", m.Name, e)
			}
			// Skip modules with no name — we cannot safely reference them.
			if m.Name == "" {
				continue
			}
		}

		modules = append(modules, m)
	}

	return modules, nil
}

// sortModules orders modules by Priority (ascending) then Name (ascending).
func sortModules(modules []*Module) {
	sort.Slice(modules, func(i, j int) bool {
		if modules[i].Priority != modules[j].Priority {
			return modules[i].Priority < modules[j].Priority
		}
		return modules[i].Name < modules[j].Name
	})
}
