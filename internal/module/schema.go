package module

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

var moduleNameRE = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

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

// PromptDependency describes a conditional dependency on a prior prompt answer.
// A prompt with DependsOn set is only shown (and only uses its own default)
// when the referenced prompt was answered with the specified value.
type PromptDependency struct {
	Key   string `yaml:"key"`   // key of the prompt this depends on
	Value string `yaml:"value"` // required answer value
}

// Prompt describes an interactive prompt to present during module installation.
type Prompt struct {
	Key       string            `yaml:"key"`
	Message   string            `yaml:"message"`
	Default   string            `yaml:"default"`
	Type      string            `yaml:"type"`      // input, confirm, or choice
	Options   []string          `yaml:"options"`
	ShowWhen  string            `yaml:"show_when"`  // always, explicit_install, or interactive (default: explicit_install)
	DependsOn *PromptDependency `yaml:"depends_on"` // only show when another prompt equals a value
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

// validateModule checks field values of a parsed Module and returns a slice of
// human-readable error strings. An empty slice means the module is valid.
func validateModule(m *Module) []string {
	var errs []string

	// name
	if m.Name == "" {
		errs = append(errs, "name is required")
	} else if !moduleNameRE.MatchString(m.Name) {
		errs = append(errs, fmt.Sprintf("name %q must match ^[a-z0-9]+(-[a-z0-9]+)*$", m.Name))
	}

	// description
	if m.Description == "" {
		errs = append(errs, "description is required")
	}

	// version
	if m.Version == "" {
		errs = append(errs, "version is required")
	}

	// files[].type
	validFileTypes := map[string]bool{"symlink": true, "copy": true, "template": true}
	for i, f := range m.Files {
		if f.Type != "" && !validFileTypes[f.Type] {
			errs = append(errs, fmt.Sprintf("files[%d].type %q is not valid (must be symlink, copy, or template)", i, f.Type))
		}
	}

	// os[] values
	validOS := map[string]bool{"macos": true, "ubuntu": true, "arch": true}
	for i, o := range m.OS {
		if !validOS[o] {
			errs = append(errs, fmt.Sprintf("os[%d] %q is not valid (must be macos, ubuntu, or arch)", i, o))
		}
	}

	// timeout
	if m.Timeout != "" {
		if _, err := time.ParseDuration(m.Timeout); err != nil {
			errs = append(errs, fmt.Sprintf("timeout %q is not a valid duration: %v", m.Timeout, err))
		}
	}

	// prompts
	promptKeys := make(map[string]bool)
	for _, p := range m.Prompts {
		if p.Key != "" {
			promptKeys[p.Key] = true
		}
	}

	validPromptTypes := map[string]bool{"input": true, "confirm": true, "choice": true}
	validShowWhen := map[string]bool{"always": true, "explicit_install": true, "interactive": true}

	for i, p := range m.Prompts {
		if p.Type != "" && !validPromptTypes[p.Type] {
			errs = append(errs, fmt.Sprintf("prompts[%d].type %q is not valid (must be input, confirm, or choice)", i, p.Type))
		}
		if p.Type == "choice" && len(p.Options) == 0 {
			errs = append(errs, fmt.Sprintf("prompts[%d].options must be non-empty for type \"choice\"", i))
		}
		if p.ShowWhen != "" && !validShowWhen[p.ShowWhen] {
			errs = append(errs, fmt.Sprintf("prompts[%d].show_when %q is not valid (must be always, explicit_install, or interactive)", i, p.ShowWhen))
		}
		if p.DependsOn != nil && p.DependsOn.Key != "" {
			if !promptKeys[p.DependsOn.Key] {
				errs = append(errs, fmt.Sprintf("prompts[%d].depends_on.key %q does not reference an existing prompt key", i, p.DependsOn.Key))
			}
		}
	}

	return errs
}

// Validate checks a Module's field values and returns a slice of error strings.
// An empty slice means the module is valid. This is the exported equivalent of
// the internal validateModule function, for use by external packages.
func Validate(m *Module) []string {
	return validateModule(m)
}

// ValidateFiles checks that each files[].source path exists relative to
// the module directory. Returns a slice of error strings for missing files.
// This is only called from the validate subcommand (not discovery).
func ValidateFiles(m *Module) []string {
	var errs []string
	for i, f := range m.Files {
		fullPath := filepath.Join(m.Dir, f.Source)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("files[%d].source %q does not exist", i, f.Source))
		}
	}
	return errs
}
