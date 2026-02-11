package module

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// BackupMetadata contains information about a backed-up file.
type BackupMetadata struct {
	OriginalPath string    `json:"original_path"` // Full path to the original file
	BackupTime   time.Time `json:"backup_time"`   // When the backup was created
	ContentHash  string    `json:"content_hash"`  // SHA256 of the backed-up content
	Reason       string    `json:"reason"`        // Why this backup was created
	Module       string    `json:"module"`        // Module that triggered the backup
}

// createBackup backs up a file before overwriting it with new content.
// Backups are stored in ~/.dotfiles/.backups/<timestamp>/<relative-path>
// with metadata stored alongside in .meta.json files.
//
// This protects user modifications from being lost when modules are updated.
// Users can manually restore files from backups if needed.
func createBackup(filePath string, cfg *RunConfig, moduleName string) error {
	if cfg.DryRun {
		cfg.UI.Debug(fmt.Sprintf("[dry-run] Would backup: %s", filePath))
		return nil
	}

	// Verify source file exists
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, no need to backup
			return nil
		}
		return fmt.Errorf("checking file: %w", err)
	}

	// Compute hash of file being backed up
	contentHash, err := ComputeFileHash(filePath)
	if err != nil {
		return fmt.Errorf("computing hash: %w", err)
	}

	// Create backup with timestamp
	backupRoot := filepath.Join(cfg.SysInfo.DotfilesDir, ".backups")
	timestamp := time.Now().Format("20060102-150405")

	// Preserve directory structure: .backups/<timestamp>/<relative-path>
	relPath, err := filepath.Rel(cfg.SysInfo.HomeDir, filePath)
	if err != nil {
		// If file is outside home dir, just use the filename
		relPath = filepath.Base(filePath)
	}
	backupPath := filepath.Join(backupRoot, timestamp, relPath)

	// Create backup directory
	backupDir := filepath.Dir(backupPath)
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return fmt.Errorf("creating backup directory: %w", err)
	}

	// Copy file to backup location
	if err := copyFile(filePath, backupPath); err != nil {
		return fmt.Errorf("copying to backup: %w", err)
	}

	// Write metadata alongside backup
	meta := BackupMetadata{
		OriginalPath: filePath,
		BackupTime:   time.Now(),
		ContentHash:  contentHash,
		Reason:       "user-modified file overwritten by module update",
		Module:       moduleName,
	}

	metaPath := backupPath + ".meta.json"
	metaData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	if err := os.WriteFile(metaPath, metaData, 0o644); err != nil {
		return fmt.Errorf("writing metadata: %w", err)
	}

	cfg.UI.Warn(fmt.Sprintf("⚠ Backed up user-modified file: %s → %s", filePath, backupPath))
	return nil
}

// copyFile copies the contents of src to dst, creating dst if it doesn't exist.
// File permissions are preserved from the source file.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Get source file info for permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Sync()
}
