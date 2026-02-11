package module

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/garygentry/dotfiles/internal/config"
)

// ComputeModuleChecksum calculates a SHA256 hash of the module's definition
// and implementation files. This hash changes whenever the module.yml file,
// install script, verify script, or any OS-specific scripts are modified.
//
// Files included in the hash (if they exist):
//   - module.yml
//   - install.sh
//   - verify.sh
//   - os/<os_name>.sh (for all detected OS files)
//
// This enables detection of module updates that require re-running installation.
func ComputeModuleChecksum(mod *Module) (string, error) {
	h := sha256.New()

	// Collect all files that define this module's behavior
	filesToHash := []string{
		filepath.Join(mod.Dir, "module.yml"),
		filepath.Join(mod.Dir, "install.sh"),
		filepath.Join(mod.Dir, "verify.sh"),
	}

	// Add OS-specific scripts
	osDir := filepath.Join(mod.Dir, "os")
	if osEntries, err := os.ReadDir(osDir); err == nil {
		for _, entry := range osEntries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sh" {
				filesToHash = append(filesToHash, filepath.Join(osDir, entry.Name()))
			}
		}
	}

	// Sort for deterministic ordering
	sort.Strings(filesToHash)

	// Hash each file's content
	for _, fpath := range filesToHash {
		if err := hashFileInto(h, fpath); err != nil {
			// Skip files that don't exist (optional scripts)
			if !os.IsNotExist(err) {
				return "", fmt.Errorf("hashing %s: %w", fpath, err)
			}
		}
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// ComputeConfigHash calculates a SHA256 hash of the user configuration
// settings that affect this module. This includes the User section and
// any module-specific settings from the Modules map.
//
// The hash changes when:
//   - User name, email, or github_user changes
//   - Module-specific config values change (from config.modules.<name>)
//
// This enables re-running modules when their configuration changes,
// even if the module definition itself hasn't changed.
func ComputeConfigHash(mod *Module, cfg *config.Config) string {
	h := sha256.New()

	// Hash user config (affects templates, prompts, etc.)
	userJSON, _ := json.Marshal(cfg.User)
	h.Write(userJSON)

	// Hash module-specific config if it exists
	if modCfg, exists := cfg.Modules[mod.Name]; exists {
		// Sort keys for deterministic hashing
		keys := make([]string, 0, len(modCfg))
		for k := range modCfg {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// Hash key-value pairs in sorted order
		for _, k := range keys {
			h.Write([]byte(k))
			valueJSON, _ := json.Marshal(modCfg[k])
			h.Write(valueJSON)
		}
	}

	return hex.EncodeToString(h.Sum(nil))
}

// ComputeFileHash calculates the SHA256 hash of a file's content.
// Returns an error if the file cannot be read.
func ComputeFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// hashFileInto reads a file and writes its content to the provided hash.
// Used as a helper for ComputeModuleChecksum.
func hashFileInto(h io.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Include filename in hash for uniqueness
	h.Write([]byte(filepath.Base(path)))
	h.Write([]byte{0}) // separator

	_, err = io.Copy(h, f)
	return err
}
