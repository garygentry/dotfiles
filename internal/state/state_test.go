package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// tempStore creates a Store backed by a temporary directory that is
// automatically cleaned up when the test finishes.
func tempStore(t *testing.T) *Store {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".state")
	return NewStore(dir)
}

func TestNewStoreCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "state")
	_ = NewStore(dir)

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("expected state directory to exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected state path to be a directory")
	}
}

func TestSetThenGet(t *testing.T) {
	store := tempStore(t)

	now := time.Now().Truncate(time.Second)
	original := &ModuleState{
		Name:        "git",
		Version:     "1.0.0",
		Status:      "installed",
		InstalledAt: now,
		OS:          "linux",
		Checksum:    "abc123",
	}

	if err := store.Set(original); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, err := store.Get("git")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil ModuleState")
	}

	if got.Name != original.Name {
		t.Errorf("Name: got %q, want %q", got.Name, original.Name)
	}
	if got.Version != original.Version {
		t.Errorf("Version: got %q, want %q", got.Version, original.Version)
	}
	if got.Status != original.Status {
		t.Errorf("Status: got %q, want %q", got.Status, original.Status)
	}
	if !got.InstalledAt.Equal(original.InstalledAt) {
		t.Errorf("InstalledAt: got %v, want %v", got.InstalledAt, original.InstalledAt)
	}
	if got.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set by Store.Set")
	}
	if got.OS != original.OS {
		t.Errorf("OS: got %q, want %q", got.OS, original.OS)
	}
	if got.Checksum != original.Checksum {
		t.Errorf("Checksum: got %q, want %q", got.Checksum, original.Checksum)
	}
}

func TestSetPreservesErrorField(t *testing.T) {
	store := tempStore(t)

	original := &ModuleState{
		Name:   "broken",
		Status: "failed",
		Error:  "something went wrong",
	}

	if err := store.Set(original); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, err := store.Get("broken")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Error != original.Error {
		t.Errorf("Error: got %q, want %q", got.Error, original.Error)
	}
}

func TestGetAll(t *testing.T) {
	store := tempStore(t)

	modules := []*ModuleState{
		{Name: "git", Version: "1.0.0", Status: "installed"},
		{Name: "vim", Version: "2.0.0", Status: "installed"},
		{Name: "zsh", Version: "3.0.0", Status: "failed", Error: "oops"},
	}

	for _, m := range modules {
		if err := store.Set(m); err != nil {
			t.Fatalf("Set(%s) failed: %v", m.Name, err)
		}
	}

	all, err := store.GetAll()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}

	if len(all) != len(modules) {
		t.Fatalf("GetAll returned %d states, want %d", len(all), len(modules))
	}

	found := make(map[string]bool)
	for _, s := range all {
		found[s.Name] = true
	}
	for _, m := range modules {
		if !found[m.Name] {
			t.Errorf("module %q not found in GetAll results", m.Name)
		}
	}
}

func TestRemove(t *testing.T) {
	store := tempStore(t)

	original := &ModuleState{
		Name:   "git",
		Status: "installed",
	}

	if err := store.Set(original); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	if err := store.Remove("git"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	got, err := store.Get("git")
	if err != nil {
		t.Fatalf("Get after Remove failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil after Remove, got non-nil")
	}
}

func TestRemoveNonExistent(t *testing.T) {
	store := tempStore(t)

	if err := store.Remove("does-not-exist"); err != nil {
		t.Fatalf("Remove non-existent should return nil, got: %v", err)
	}
}

func TestGetNonExistentReturnsNilNil(t *testing.T) {
	store := tempStore(t)

	got, err := store.Get("nonexistent")
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil state, got: %+v", got)
	}
}

func TestRecordOperation(t *testing.T) {
	ms := &ModuleState{
		Name:    "test",
		Version: "1.0.0",
		Status:  "installed",
	}

	op := Operation{
		Type:   "file_deploy",
		Action: "created",
		Path:   "/home/user/.config/test/config.conf",
		Metadata: map[string]string{
			"source": "files/config.conf",
		},
	}

	ms.RecordOperation(op)

	if len(ms.Operations) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ms.Operations))
	}

	recorded := ms.Operations[0]
	if recorded.Type != "file_deploy" {
		t.Errorf("Type = %q, want %q", recorded.Type, "file_deploy")
	}
	if recorded.Action != "created" {
		t.Errorf("Action = %q, want %q", recorded.Action, "created")
	}
	if recorded.Path != op.Path {
		t.Errorf("Path = %q, want %q", recorded.Path, op.Path)
	}
	if recorded.Timestamp.IsZero() {
		t.Error("Timestamp should be set automatically")
	}
}

func TestRollbackInstructions(t *testing.T) {
	ms := &ModuleState{
		Name:    "test",
		Version: "1.0.0",
		Status:  "installed",
	}

	// Add various operations
	ms.RecordOperation(Operation{
		Type:   "dir_create",
		Action: "created",
		Path:   "/home/user/.config/test",
	})

	ms.RecordOperation(Operation{
		Type:   "file_deploy",
		Action: "created",
		Path:   "/home/user/.config/test/config.conf",
	})

	ms.RecordOperation(Operation{
		Type:   "file_deploy",
		Action: "modified",
		Path:   "/home/user/.bashrc",
		Metadata: map[string]string{
			"backup_path": "/home/user/.bashrc.backup",
		},
	})

	ms.RecordOperation(Operation{
		Type:   "package_install",
		Action: "installed",
		Path:   "test-package",
	})

	instructions := ms.RollbackInstructions()

	// Should have 4 instructions (one per operation)
	if len(instructions) != 4 {
		t.Fatalf("expected 4 instructions, got %d", len(instructions))
	}

	// Instructions should be in reverse order
	// Last operation (package) should come first
	if !strings.Contains(instructions[0], "test-package") {
		t.Errorf("first instruction should mention package, got: %s", instructions[0])
	}

	// Check for restore instruction
	foundRestore := false
	for _, inst := range instructions {
		if strings.Contains(inst, "Restore") && strings.Contains(inst, ".bashrc") {
			foundRestore = true
			break
		}
	}
	if !foundRestore {
		t.Error("expected restore instruction for modified .bashrc")
	}
}

func TestCanRollback(t *testing.T) {
	tests := []struct {
		name       string
		operations []Operation
		want       bool
	}{
		{
			name:       "no operations",
			operations: []Operation{},
			want:       false,
		},
		{
			name: "with operations",
			operations: []Operation{
				{Type: "file_deploy", Action: "created", Path: "/test"},
			},
			want: true,
		},
		{
			name: "multiple operations",
			operations: []Operation{
				{Type: "file_deploy", Action: "created", Path: "/test1"},
				{Type: "file_deploy", Action: "created", Path: "/test2"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &ModuleState{
				Name:       "test",
				Operations: tt.operations,
			}

			got := ms.CanRollback()
			if got != tt.want {
				t.Errorf("CanRollback() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOperationsPersistence(t *testing.T) {
	store := tempStore(t)

	ms := &ModuleState{
		Name:        "test",
		Version:     "1.0.0",
		Status:      "installed",
		InstalledAt: time.Now(),
	}

	// Record some operations
	ms.RecordOperation(Operation{
		Type:   "file_deploy",
		Action: "created",
		Path:   "/test/path",
	})

	ms.RecordOperation(Operation{
		Type:   "dir_create",
		Action: "created",
		Path:   "/test/dir",
	})

	// Save to store
	if err := store.Set(ms); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Load from store
	loaded, err := store.Get("test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if loaded == nil {
		t.Fatal("expected state to exist")
	}

	if len(loaded.Operations) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(loaded.Operations))
	}

	// Verify first operation
	if loaded.Operations[0].Type != "file_deploy" {
		t.Errorf("first operation type = %q, want %q", loaded.Operations[0].Type, "file_deploy")
	}

	// Verify second operation
	if loaded.Operations[1].Type != "dir_create" {
		t.Errorf("second operation type = %q, want %q", loaded.Operations[1].Type, "dir_create")
	}
}
