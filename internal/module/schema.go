package module

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Module represents a single dotfiles module defined by a module.yml file.
type Module struct {
	Name         string      `yaml:"name"`
	Description  string      `yaml:"description"`
	Version      string      `yaml:"version"`
	Priority     int         `yaml:"priority"`
	Dependencies []string    `yaml:"dependencies"`
	OS           []string    `yaml:"os"`
	Requires     []string    `yaml:"requires"`
	Files        []FileEntry `yaml:"files"`
	Prompts      []Prompt    `yaml:"prompts"`
	Tags         []string    `yaml:"tags"`
	Timeout      string      `yaml:"timeout"` // e.g., "10m", parsed via time.ParseDuration
	Notes        []string    `yaml:"notes"`   // Post-install messages displayed after run
	Dir          string      `yaml:"-"`
}

// FileEntry describes a single file to deploy as part of a module.
type FileEntry struct {
	Source string `yaml:"source"`
	Dest   string `yaml:"dest"`
	Type   string `yaml:"type"` // symlink, copy, or template
}

// Prompt describes an interactive prompt to present during module installation.
type Prompt struct {
	Key      string   `yaml:"key"`
	Message  string   `yaml:"message"`
	Default  string   `yaml:"default"`
	Type     string   `yaml:"type"`      // input, confirm, or choice
	Options  []string `yaml:"options"`
	ShowWhen string   `yaml:"show_when"` // always, explicit_install, or interactive (default: explicit_install)
}

// ParseModuleYAML reads a module.yml file at the given path and returns the
// parsed Module. If the Name field is empty after parsing, it is set to the
// base name of the directory containing the file. The Dir field is set to the
// directory containing the file. Priority defaults to 50 when not specified
// (i.e. when the YAML value is zero).
func ParseModuleYAML(path string) (*Module, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	m := &Module{
		Priority: 50,
	}
	if err := yaml.Unmarshal(data, m); err != nil {
		return nil, err
	}

	dir := filepath.Dir(path)
	m.Dir = dir

	if m.Name == "" {
		m.Name = filepath.Base(dir)
	}

	return m, nil
}

// SupportsOS reports whether the module supports the given operating system.
// If the module's OS list is empty the module is considered to support all
// operating systems and the method returns true.
func (m *Module) SupportsOS(os string) bool {
	if len(m.OS) == 0 {
		return true
	}
	for _, supported := range m.OS {
		if supported == os {
			return true
		}
	}
	return false
}
