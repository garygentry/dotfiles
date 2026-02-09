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
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Status      string    `json:"status"` // installed, failed, removed
	InstalledAt time.Time `json:"installed_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	OS          string    `json:"os"`
	Error       string    `json:"error,omitempty"` // last error if failed
	Checksum    string    `json:"checksum"`        // for change detection
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
