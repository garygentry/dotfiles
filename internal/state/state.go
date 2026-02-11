package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ModuleState represents the persisted state of a single dotfiles module.
type ModuleState struct {
	Name        string      `json:"name"`
	Version     string      `json:"version"`
	Status      string      `json:"status"` // installed, failed, removed
	InstalledAt time.Time   `json:"installed_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	OS          string      `json:"os"`
	Error       string      `json:"error,omitempty"`     // last error if failed
	Checksum    string      `json:"checksum,omitempty"`  // SHA256 of module.yml + scripts
	ConfigHash  string      `json:"config_hash,omitempty"` // Hash of user config for this module
	FileStates  []FileState `json:"file_states,omitempty"` // Per-file deployment tracking
	Operations  []Operation `json:"operations,omitempty"` // rollback metadata
}

// FileState tracks the deployment state of an individual file.
// This enables file-level idempotence by detecting changes to source files,
// user modifications to deployed files, and skipping unnecessary redeployments.
type FileState struct {
	Source       string    `json:"source"`        // Relative path in module dir (e.g., "files/.gitconfig")
	Dest         string    `json:"dest"`          // Absolute destination path (e.g., "/home/user/.gitconfig")
	Type         string    `json:"type"`          // "symlink", "copy", or "template"
	DeployedAt   time.Time `json:"deployed_at"`   // When this file was last deployed
	SourceHash   string    `json:"source_hash"`   // SHA256 of source file at deploy time
	DeployedHash string    `json:"deployed_hash"` // SHA256 of deployed content at deploy time
	UserModified bool      `json:"user_modified"` // True if user changed dest after deployment
	LastChecked  time.Time `json:"last_checked"`  // Last time we verified this file's state
}

// Operation represents a single action taken during module installation.
// Operations are recorded to enable rollback/uninstall functionality.
type Operation struct {
	Type      string            `json:"type"`      // file_deploy, dir_create, script_run, package_install
	Action    string            `json:"action"`    // created, modified, backed_up, symlinked, executed
	Path      string            `json:"path"`      // file path, package name, or script path
	Timestamp time.Time         `json:"timestamp"` // when operation was performed
	Metadata  map[string]string `json:"metadata,omitempty"` // additional context (backup_path, original_content, etc.)
}

// Store manages reading and writing module state files.
// Each module's state is stored as an individual JSON file inside Dir.
type Store struct {
	Dir string // path to state directory, default ~/.dotfiles/.state/
}

// NewStore creates a new Store rooted at dir.
// If dir is empty the default ~/.dotfiles/.state/ is used.
// The directory is created (along with parents) if it does not already exist.
func NewStore(dir string) *Store {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		dir = filepath.Join(home, ".dotfiles", ".state")
	}

	// Best-effort directory creation; errors will surface on first read/write.
	_ = os.MkdirAll(dir, 0o755)

	return &Store{Dir: dir}
}

// stateFilePath returns the full path for a module's state file.
func (s *Store) stateFilePath(name string) string {
	return filepath.Join(s.Dir, name+".json")
}

// Get reads the state for the named module.
// If the state file does not exist it returns (nil, nil).
func (s *Store) Get(name string) (*ModuleState, error) {
	data, err := os.ReadFile(s.stateFilePath(name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var ms ModuleState
	if err := json.Unmarshal(data, &ms); err != nil {
		return nil, err
	}
	return &ms, nil
}

// Set writes the module state to disk. UpdatedAt is always set to the
// current time before persisting.
func (s *Store) Set(state *ModuleState) error {
	state.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.stateFilePath(state.Name), data, 0o644)
}

// GetAll reads every JSON state file in the store directory and returns
// the collected module states.
func (s *Store) GetAll() ([]*ModuleState, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var states []*ModuleState
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(s.Dir, entry.Name()))
		if err != nil {
			return nil, err
		}

		var ms ModuleState
		if err := json.Unmarshal(data, &ms); err != nil {
			return nil, err
		}
		states = append(states, &ms)
	}

	return states, nil
}

// Remove deletes the state file for the named module.
// It returns nil if the file does not exist.
func (s *Store) Remove(name string) error {
	err := os.Remove(s.stateFilePath(name))
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

// RecordOperation adds an operation to the module state's operation history.
// The timestamp is automatically set to the current time.
func (ms *ModuleState) RecordOperation(op Operation) {
	op.Timestamp = time.Now()
	ms.Operations = append(ms.Operations, op)
}

// RollbackInstructions returns a list of human-readable instructions
// for rolling back the module installation, in reverse chronological order.
func (ms *ModuleState) RollbackInstructions() []string {
	var instructions []string

	// Process operations in reverse order (most recent first)
	for i := len(ms.Operations) - 1; i >= 0; i-- {
		op := ms.Operations[i]

		switch op.Type {
		case "file_deploy":
			switch op.Action {
			case "created", "symlinked":
				instructions = append(instructions, "Remove: "+op.Path)
			case "modified":
				if backup := op.Metadata["backup_path"]; backup != "" {
					instructions = append(instructions, "Restore: "+op.Path+" from "+backup)
				} else {
					instructions = append(instructions, "File was modified: "+op.Path+" (no backup available)")
				}
			}

		case "dir_create":
			if op.Action == "created" {
				instructions = append(instructions, "Remove directory: "+op.Path)
			}

		case "package_install":
			instructions = append(instructions, "Consider removing package: "+op.Path)

		case "script_run":
			instructions = append(instructions, "Script was executed: "+op.Path+" (manual cleanup may be needed)")
		}
	}

	return instructions
}

// CanRollback returns true if the module has recorded operations that can be rolled back.
func (ms *ModuleState) CanRollback() bool {
	return len(ms.Operations) > 0
}

// NeedsMigration returns true if this state was written by an older version
// and needs migration to the current schema (e.g., missing FileStates).
func (ms *ModuleState) NeedsMigration() bool {
	// If module is marked as installed but has no FileStates, it needs migration
	return ms.Status == "installed" && len(ms.FileStates) == 0
}

// MigrateFileStatesFromOperations attempts to reconstruct FileStates from
// recorded operations for modules installed before file tracking was added.
// This is best-effort - we can extract deployment information from operations
// but can't recover exact hashes without reading the files again.
func (ms *ModuleState) MigrateFileStatesFromOperations() {
	if !ms.NeedsMigration() {
		return
	}

	// Build FileStates from file_deploy operations
	seenFiles := make(map[string]bool)
	for _, op := range ms.Operations {
		if op.Type != "file_deploy" {
			continue
		}

		// Skip duplicates (if file was deployed multiple times)
		if seenFiles[op.Path] {
			continue
		}
		seenFiles[op.Path] = true

		fs := FileState{
			Source:       op.Metadata["source"],
			Dest:         op.Path,
			Type:         op.Metadata["type"],
			DeployedAt:   op.Timestamp,
			SourceHash:   op.Metadata["source_hash"], // May be empty for old operations
			DeployedHash: "",                          // Unknown for old installations
			UserModified: false,                       // Assume not modified
			LastChecked:  time.Now(),
		}

		ms.FileStates = append(ms.FileStates, fs)
	}
}
