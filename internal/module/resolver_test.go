package module

import (
	"strings"
	"testing"
)

// helper to extract ordered module names from an execution plan.
func moduleNames(modules []*Module) []string {
	names := make([]string, len(modules))
	for i, m := range modules {
		names[i] = m.Name
	}
	return names
}

func TestResolve_LinearChain(t *testing.T) {
	// a depends on b, b depends on c  =>  execution order: c, b, a
	modules := []*Module{
		{Name: "a", Priority: 30, Dependencies: []string{"b"}},
		{Name: "b", Priority: 20, Dependencies: []string{"c"}},
		{Name: "c", Priority: 10},
	}

	plan, err := Resolve(modules, []string{"a"}, "macos")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := moduleNames(plan.Modules)
	expected := []string{"c", "b", "a"}
	if len(got) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("position %d: expected %q, got %q", i, expected[i], got[i])
		}
	}
	if len(plan.Skipped) != 0 {
		t.Errorf("expected no skipped modules, got %v", moduleNames(plan.Skipped))
	}
}

func TestResolve_DiamondDependency(t *testing.T) {
	// d depends on b and c; both b and c depend on a
	//      d
	//     / \
	//    b   c
	//     \ /
	//      a
	modules := []*Module{
		{Name: "a", Priority: 10},
		{Name: "b", Priority: 20, Dependencies: []string{"a"}},
		{Name: "c", Priority: 20, Dependencies: []string{"a"}},
		{Name: "d", Priority: 30, Dependencies: []string{"b", "c"}},
	}

	plan, err := Resolve(modules, []string{"d"}, "linux")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := moduleNames(plan.Modules)

	// a must come first (only level-0 node).
	if got[0] != "a" {
		t.Errorf("expected first module to be 'a', got %q", got[0])
	}
	// d must come last.
	if got[len(got)-1] != "d" {
		t.Errorf("expected last module to be 'd', got %q", got[len(got)-1])
	}
	// b and c are at the same level; same priority so sorted alphabetically.
	if len(got) != 4 {
		t.Fatalf("expected 4 modules, got %d: %v", len(got), got)
	}
	if got[1] != "b" || got[2] != "c" {
		t.Errorf("expected [b, c] at positions 1-2, got [%s, %s]", got[1], got[2])
	}
}

func TestResolve_CycleDetection(t *testing.T) {
	// a -> b -> c -> a (cycle)
	modules := []*Module{
		{Name: "a", Priority: 10, Dependencies: []string{"b"}},
		{Name: "b", Priority: 20, Dependencies: []string{"c"}},
		{Name: "c", Priority: 30, Dependencies: []string{"a"}},
	}

	_, err := Resolve(modules, []string{"a"}, "linux")
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("expected error to mention 'cycle', got: %v", err)
	}
	// The error should contain the cycle path with arrows.
	if !strings.Contains(err.Error(), "->") {
		t.Errorf("expected error to contain cycle path with '->': %v", err)
	}
}

func TestResolve_MissingDependency(t *testing.T) {
	modules := []*Module{
		{Name: "a", Priority: 10, Dependencies: []string{"nonexistent"}},
	}

	_, err := Resolve(modules, []string{"a"}, "linux")
	if err == nil {
		t.Fatal("expected missing dependency error, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected error to mention 'nonexistent', got: %v", err)
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected error to say 'does not exist', got: %v", err)
	}
}

func TestResolve_MissingRequestedModule(t *testing.T) {
	modules := []*Module{
		{Name: "a", Priority: 10},
	}

	_, err := Resolve(modules, []string{"ghost"}, "linux")
	if err == nil {
		t.Fatal("expected error for missing requested module, got nil")
	}
	if !strings.Contains(err.Error(), "ghost") {
		t.Errorf("expected error to mention 'ghost', got: %v", err)
	}
}

func TestResolve_OSFiltering(t *testing.T) {
	modules := []*Module{
		{Name: "base", Priority: 10},
		{Name: "mac-only", Priority: 20, OS: []string{"macos"}, Dependencies: []string{"base"}},
		{Name: "linux-only", Priority: 20, OS: []string{"linux"}, Dependencies: []string{"base"}},
		{Name: "universal", Priority: 30},
	}

	plan, err := Resolve(modules, nil, "linux")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := moduleNames(plan.Modules)
	skippedNames := moduleNames(plan.Skipped)

	// mac-only should be in Skipped.
	foundSkipped := false
	for _, name := range skippedNames {
		if name == "mac-only" {
			foundSkipped = true
		}
	}
	if !foundSkipped {
		t.Errorf("expected 'mac-only' in Skipped, got: %v", skippedNames)
	}

	// mac-only should NOT be in Modules.
	for _, name := range got {
		if name == "mac-only" {
			t.Errorf("'mac-only' should not be in Modules for linux, got: %v", got)
		}
	}

	// linux-only, base, and universal should be in Modules.
	expectedModules := map[string]bool{"base": false, "linux-only": false, "universal": false}
	for _, name := range got {
		expectedModules[name] = true
	}
	for name, found := range expectedModules {
		if !found {
			t.Errorf("expected %q in Modules, got: %v", name, got)
		}
	}
}

func TestResolve_EmptyRequestedResolvesAll(t *testing.T) {
	modules := []*Module{
		{Name: "alpha", Priority: 10},
		{Name: "beta", Priority: 20},
		{Name: "gamma", Priority: 30},
	}

	plan, err := Resolve(modules, nil, "linux")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := moduleNames(plan.Modules)
	if len(got) != 3 {
		t.Fatalf("expected 3 modules, got %d: %v", len(got), got)
	}
}

func TestResolve_EmptyRequestedSliceResolvesAll(t *testing.T) {
	modules := []*Module{
		{Name: "alpha", Priority: 10},
		{Name: "beta", Priority: 20},
		{Name: "gamma", Priority: 30},
	}

	plan, err := Resolve(modules, []string{}, "linux")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := moduleNames(plan.Modules)
	if len(got) != 3 {
		t.Fatalf("expected 3 modules, got %d: %v", len(got), got)
	}
}

func TestResolve_PriorityOrderingWithinLevel(t *testing.T) {
	// All modules are independent (no dependencies), so they form a single
	// topological level. They should be sorted by Priority, then by Name.
	modules := []*Module{
		{Name: "zsh", Priority: 40},
		{Name: "git", Priority: 30},
		{Name: "ssh", Priority: 20},
		{Name: "neovim", Priority: 50},
		{Name: "1password", Priority: 10},
	}

	plan, err := Resolve(modules, nil, "linux")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := moduleNames(plan.Modules)
	expected := []string{"1password", "ssh", "git", "zsh", "neovim"}
	if len(got) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("position %d: expected %q, got %q", i, expected[i], got[i])
		}
	}
}

func TestResolve_PriorityThenNameWithinLevel(t *testing.T) {
	// Two modules share the same priority and have no dependencies.
	// They should be ordered alphabetically by name.
	modules := []*Module{
		{Name: "bravo", Priority: 10},
		{Name: "alpha", Priority: 10},
		{Name: "charlie", Priority: 10},
	}

	plan, err := Resolve(modules, nil, "linux")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := moduleNames(plan.Modules)
	expected := []string{"alpha", "bravo", "charlie"}
	if len(got) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("position %d: expected %q, got %q", i, expected[i], got[i])
		}
	}
}

func TestResolve_TransitiveDependenciesAutoIncluded(t *testing.T) {
	// Requesting only "top" should auto-include "mid" and "base".
	modules := []*Module{
		{Name: "base", Priority: 10},
		{Name: "mid", Priority: 20, Dependencies: []string{"base"}},
		{Name: "top", Priority: 30, Dependencies: []string{"mid"}},
		{Name: "unrelated", Priority: 40},
	}

	plan, err := Resolve(modules, []string{"top"}, "linux")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := moduleNames(plan.Modules)
	// Should include base, mid, top but NOT unrelated.
	if len(got) != 3 {
		t.Fatalf("expected 3 modules, got %d: %v", len(got), got)
	}
	expected := []string{"base", "mid", "top"}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("position %d: expected %q, got %q", i, expected[i], got[i])
		}
	}
}

func TestResolve_NoModules(t *testing.T) {
	plan, err := Resolve(nil, nil, "linux")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.Modules) != 0 {
		t.Errorf("expected empty plan, got %v", moduleNames(plan.Modules))
	}
	if len(plan.Skipped) != 0 {
		t.Errorf("expected no skipped, got %v", moduleNames(plan.Skipped))
	}
}
