package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ErrProfileNotFound reports that a profile file does not exist. Callers use it to
// distinguish "you asked for a profile that isn't there" from "the profile is broken".
var ErrProfileNotFound = errors.New("profile not found")

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
	Profile     string                    `yaml:"profile"`
	DotfilesDir string                    `yaml:"-"`
	ContentDir  string                    `yaml:"-"` // resolved user overlay dir, "" if none
	Secrets     SecretsConfig             `yaml:"secrets"`
	User        UserConfig                `yaml:"user"`
	Modules     map[string]map[string]any `yaml:"modules"`
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

	// Overlay an optional user content directory's config.yml on top of the
	// base. When no content dir is present this is a no-op and the engine
	// behaves exactly as it does with only the committed config.yml.
	cfg.ContentDir = ResolveContentDir()
	if cfg.ContentDir != "" {
		overlayPath := filepath.Join(cfg.ContentDir, "config.yml")
		if overlay, oerr := loadOverlay(overlayPath); oerr != nil {
			return nil, oerr
		} else if overlay != nil {
			mergeConfig(cfg, overlay)
		}
	}

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

// ResolveContentDir returns the optional user content directory that overlays the
// generic repo, or "" when none is configured. It is opt-in via the
// DOTFILES_CONTENT_DIR environment variable (a leading ~ is expanded); a later
// --content-root flag can override it by setting the same variable.
//
// Opt-in is deliberate: auto-detecting a conventional path would make every
// process (tests, CI, any machine that happens to have such a directory) pick it
// up as global state. Requiring an explicit pointer keeps behavior predictable
// and existing single-repo installs completely unaffected.
func ResolveContentDir() string {
	if v := os.Getenv("DOTFILES_CONTENT_DIR"); v != "" {
		return expandHomePath(v)
	}
	return ""
}

// expandHomePath expands a leading ~ to the user's home directory.
func expandHomePath(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(path, "~"), "/"))
		}
	}
	return path
}

// loadOverlay reads and parses an overlay config.yml. A missing file is not an
// error (returns nil, nil) — the overlay is simply absent.
func loadOverlay(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading overlay config %s: %w", path, err)
	}
	var overlay Config
	if err := yaml.Unmarshal(data, &overlay); err != nil {
		return nil, fmt.Errorf("parsing overlay config %s: %w", path, err)
	}
	return &overlay, nil
}

// mergeConfig applies an overlay onto base in place. Scalars override only when
// the overlay sets a non-empty value, so an overlay that omits a field leaves the
// base value intact. Module settings merge per key, so an overlay can change a
// single modules.<name>.<key> without redefining the whole module's settings.
// DotfilesDir/ContentDir are engine-resolved and never taken from an overlay.
func mergeConfig(base, overlay *Config) {
	if overlay.Profile != "" {
		base.Profile = overlay.Profile
	}
	if overlay.Secrets.Provider != "" {
		base.Secrets.Provider = overlay.Secrets.Provider
	}
	if overlay.Secrets.Account != "" {
		base.Secrets.Account = overlay.Secrets.Account
	}
	if overlay.User.Name != "" {
		base.User.Name = overlay.User.Name
	}
	if overlay.User.Email != "" {
		base.User.Email = overlay.User.Email
	}
	if overlay.User.GithubUser != "" {
		base.User.GithubUser = overlay.User.GithubUser
	}
	if base.Modules == nil {
		base.Modules = make(map[string]map[string]any)
	}
	for mod, settings := range overlay.Modules {
		if base.Modules[mod] == nil {
			base.Modules[mod] = make(map[string]any)
		}
		for k, v := range settings {
			base.Modules[mod][k] = v
		}
	}
}

// ProfileIsPath reports whether a profile argument should be read as a literal file
// path rather than as the name of a profile in <dotfilesDir>/profiles.
//
// An argument is a path when it contains a separator or carries a YAML extension.
// Bare names such as "minimal" keep their existing meaning, so nothing that worked
// before changes. This lets a project outside the dotfiles repo keep its own profile
// alongside its own code and hand it to the CLI directly.
func ProfileIsPath(name string) bool {
	if name == "" {
		return false
	}
	if strings.ContainsRune(name, '/') || strings.ContainsRune(name, os.PathSeparator) {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yml" || ext == ".yaml"
}

// ResolveProfilePath returns the file that a profile argument refers to: the argument
// itself when it is a path (with a leading ~ expanded and relative paths taken from the
// working directory), otherwise <dotfilesDir>/profiles/<name>.yml.
func ResolveProfilePath(dotfilesDir, name string) string {
	if !ProfileIsPath(name) {
		return filepath.Join(dotfilesDir, "profiles", name+".yml")
	}

	path := name
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(path, "~"), "/"))
		}
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return path
}

// LoadProfile reads a profile and returns the module names defined under its "modules"
// key. The name is resolved by ResolveProfilePath, so it may be either a bare profile
// name or a path to a profile file anywhere on disk.
//
// A missing file yields an error wrapping ErrProfileNotFound.
func LoadProfile(dotfilesDir, name string) ([]string, error) {
	profilePath := ResolveProfilePath(dotfilesDir, name)

	data, err := os.ReadFile(profilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrProfileNotFound, profilePath)
		}
		return nil, fmt.Errorf("reading profile file %s: %w", profilePath, err)
	}

	var pf profileFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("parsing profile file %s: %w", profilePath, err)
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
