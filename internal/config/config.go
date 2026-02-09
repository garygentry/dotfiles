package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SecretsConfig holds secrets provider settings.
type SecretsConfig struct {
	Provider string `yaml:"provider"`
	Account  string `yaml:"account"`
}

// UserConfig holds user identity settings.
type UserConfig struct {
	Name       string `yaml:"name"`
	Email      string `yaml:"email"`
	GithubUser string `yaml:"github_user"`
}

// Config is the top-level dotfiles configuration.
type Config struct {
	Profile     string                       `yaml:"profile"`
	DotfilesDir string                       `yaml:"-"`
	Secrets     SecretsConfig                `yaml:"secrets"`
	User        UserConfig                   `yaml:"user"`
	Modules     map[string]map[string]any    `yaml:"modules"`
}

// profileFile represents the YAML structure of a profile file.
type profileFile struct {
	Modules []string `yaml:"modules"`
}

// Load reads config.yml from dotfilesDir, applies defaults, then applies
// environment variable overrides.
func Load(dotfilesDir string) (*Config, error) {
	cfg := &Config{
		Profile:     "developer",
		DotfilesDir: dotfilesDir,
		Modules:     make(map[string]map[string]any),
	}

	configPath := filepath.Join(dotfilesDir, "config.yml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Ensure DotfilesDir is always the passed-in value, not from YAML.
	cfg.DotfilesDir = dotfilesDir

	// Re-apply default profile if YAML left it empty.
	if cfg.Profile == "" {
		cfg.Profile = "developer"
	}

	// Environment variable overrides.
	if v := os.Getenv("DOTFILES_PROFILE"); v != "" {
		cfg.Profile = v
	}
	if v := os.Getenv("DOTFILES_SECRETS_PROVIDER"); v != "" {
		cfg.Secrets.Provider = v
	}

	return cfg, nil
}

// LoadProfile reads profiles/<name>.yml from dotfilesDir and returns the list
// of module names defined under the "modules" key.
func LoadProfile(dotfilesDir, name string) ([]string, error) {
	profilePath := filepath.Join(dotfilesDir, "profiles", name+".yml")
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("reading profile file: %w", err)
	}

	var pf profileFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("parsing profile file: %w", err)
	}

	return pf.Modules, nil
}

// GetModuleSetting returns the value associated with key inside the named
// module's settings map. The second return value indicates whether the key
// was found.
func (c *Config) GetModuleSetting(moduleName, key string) (any, bool) {
	mod, ok := c.Modules[moduleName]
	if !ok {
		return nil, false
	}
	val, ok := mod[key]
	return val, ok
}
