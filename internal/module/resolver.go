package module

import (
	"fmt"
	"sort"
	"strings"
)

// ExecutionPlan holds the result of dependency resolution.
type ExecutionPlan struct {
	// Modules contains the resolved modules in dependency-safe execution order.
	Modules []*Module
	// Skipped contains modules that were excluded because they do not support
	// the target operating system.
	Skipped []*Module
}

// Resolve takes all known modules, a list of requested module names, and the
// target OS name. It returns an ExecutionPlan with modules ordered so that
// every module appears after all of its dependencies.
//
// Behaviour:
//  1. Build a name-to-module lookup map from allModules.
//  2. If requested is empty, treat every module name as requested.
//  3. Expand requested modules to include all transitive dependencies.
//  4. Filter out modules that do not support osName (placed in Skipped).
//  5. Order remaining modules via Kahn's algorithm (BFS topological sort).
//  6. Within the same topological level, sort by Priority ascending, then Name.
//  7. Detect cycles: if unprocessed edges remain after BFS, report the cycle path.
//  8. Detect missing dependencies: if a dependency name is not in allModules,
//     return a descriptive error before starting the sort.
func Resolve(allModules []*Module, requested []string, osName string) (*ExecutionPlan, error) {
	// Step 1: Build name -> module map.
	moduleMap := make(map[string]*Module, len(allModules))
	for _, m := range allModules {
		moduleMap[m.Name] = m
	}

	// Step 2: Default to all module names when requested is empty.
	if len(requested) == 0 {
		requested = make([]string, 0, len(allModules))
		for _, m := range allModules {
			requested = append(requested, m.Name)
		}
	}

	// Step 3: Expand requested set to include transitive dependencies and
	// detect missing dependencies along the way.
	needed, err := expandDependencies(requested, moduleMap)
	if err != nil {
		return nil, err
	}

	// Step 4: Partition into compatible and skipped.
	compatible := make(map[string]*Module, len(needed))
	var skipped []*Module
	for name := range needed {
		m := moduleMap[name]
		if m.SupportsOS(osName) {
			compatible[name] = m
		} else {
			skipped = append(skipped, m)
		}
	}
	// Sort skipped for deterministic output.
	sort.Slice(skipped, func(i, j int) bool {
		return skipped[i].Name < skipped[j].Name
	})

	// Steps 5-7: Kahn's algorithm on the compatible set.
	ordered, err := topoSort(compatible)
	if err != nil {
		return nil, err
	}

	return &ExecutionPlan{
		Modules: ordered,
		Skipped: skipped,
	}, nil
}

// expandDependencies performs a BFS walk from the requested module names,
// collecting every transitive dependency. It returns an error if any
// dependency references a module that does not exist in moduleMap.
func expandDependencies(requested []string, moduleMap map[string]*Module) (map[string]bool, error) {
	needed := make(map[string]bool, len(requested))
	queue := make([]string, 0, len(requested))

	// Seed the queue with requested names (validate they exist).
	for _, name := range requested {
		if _, ok := moduleMap[name]; !ok {
			return nil, fmt.Errorf("requested module %q not found in available modules", name)
		}
		if !needed[name] {
			needed[name] = true
			queue = append(queue, name)
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		m := moduleMap[current]
		for _, dep := range m.Dependencies {
			if _, ok := moduleMap[dep]; !ok {
				return nil, fmt.Errorf("module %q depends on %q, which does not exist", current, dep)
			}
			if !needed[dep] {
				needed[dep] = true
				queue = append(queue, dep)
			}
		}
	}

	return needed, nil
}

// topoSort performs Kahn's algorithm on the compatible module set. Within each
// topological level the modules are sorted by Priority (ascending) then Name
// (ascending). It returns an error describing the cycle path if one is detected.
func topoSort(compatible map[string]*Module) ([]*Module, error) {
	// Build adjacency list and in-degree counts restricted to compatible set.
	inDegree := make(map[string]int, len(compatible))
	// dependents maps a module name to the list of modules that depend on it
	// (i.e., the "reverse" adjacency list: dep -> []dependent).
	dependents := make(map[string][]string, len(compatible))

	for name := range compatible {
		if _, exists := inDegree[name]; !exists {
			inDegree[name] = 0
		}
		m := compatible[name]
		for _, dep := range m.Dependencies {
			if _, ok := compatible[dep]; !ok {
				// Dependency was filtered out (e.g., OS-incompatible) or not
				// in the compatible set; skip the edge.
				continue
			}
			inDegree[name]++
			dependents[dep] = append(dependents[dep], name)
		}
	}

	// Collect initial zero-in-degree nodes (the first "level").
	var queue []*Module
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, compatible[name])
		}
	}
	sortModuleSlice(queue)

	var ordered []*Module

	for len(queue) > 0 {
		// Process the entire current level.
		level := queue
		queue = nil

		var nextLevel []*Module
		for _, m := range level {
			ordered = append(ordered, m)
			for _, dependent := range dependents[m.Name] {
				inDegree[dependent]--
				if inDegree[dependent] == 0 {
					nextLevel = append(nextLevel, compatible[dependent])
				}
			}
		}
		sortModuleSlice(nextLevel)
		queue = nextLevel
	}

	// Cycle detection: if we haven't placed every node, a cycle exists.
	if len(ordered) != len(compatible) {
		cyclePath := detectCyclePath(compatible, inDegree)
		return nil, fmt.Errorf("dependency cycle detected: %s", cyclePath)
	}

	return ordered, nil
}

// sortModuleSlice sorts a slice of modules by Priority ascending, then Name
// ascending for deterministic ordering within the same topological level.
func sortModuleSlice(modules []*Module) {
	sort.Slice(modules, func(i, j int) bool {
		if modules[i].Priority != modules[j].Priority {
			return modules[i].Priority < modules[j].Priority
		}
		return modules[i].Name < modules[j].Name
	})
}

// detectCyclePath walks the remaining unprocessed nodes (those with inDegree > 0)
// and returns a human-readable cycle path string such as "a -> b -> c -> a".
func detectCyclePath(compatible map[string]*Module, inDegree map[string]int) string {
	// Collect all nodes still in the graph (in-degree > 0).
	remaining := make(map[string]bool)
	for name, deg := range inDegree {
		if deg > 0 {
			remaining[name] = true
		}
	}

	if len(remaining) == 0 {
		return "unknown cycle"
	}

	// Pick the lexicographically first remaining node as starting point for
	// deterministic output.
	var start string
	for name := range remaining {
		if start == "" || name < start {
			start = name
		}
	}

	// Walk edges that stay within the remaining set to trace the cycle.
	visited := make(map[string]bool)
	var path []string
	current := start
	for {
		if visited[current] {
			// Find where the cycle begins in path and trim prefix.
			for i, name := range path {
				if name == current {
					path = path[i:]
					break
				}
			}
			path = append(path, current)
			return strings.Join(path, " -> ")
		}
		visited[current] = true
		path = append(path, current)

		// Follow the first dependency edge that leads to another remaining node.
		m := compatible[current]
		found := false
		// Sort dependencies for deterministic walk.
		deps := make([]string, len(m.Dependencies))
		copy(deps, m.Dependencies)
		sort.Strings(deps)
		for _, dep := range deps {
			if remaining[dep] {
				current = dep
				found = true
				break
			}
		}
		if !found {
			// Should not happen if cycle truly exists, but guard against it.
			path = append(path, "???")
			return strings.Join(path, " -> ")
		}
	}
}
