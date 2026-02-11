package module

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/garygentry/dotfiles/internal/config"
	"github.com/garygentry/dotfiles/internal/sysinfo"
)

func TestCreateBackup(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	dotfilesDir := filepath.Join(tmpDir, "dotfiles")

	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dotfilesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a test file
	testFile := filepath.Join(homeDir, ".testrc")
	testContent := "user modified content\n"
	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &RunConfig{
		SysInfo: &sysinfo.SystemInfo{
			HomeDir:     homeDir,
			DotfilesDir: dotfilesDir,
		},
		Config: &config.Config{},
		UI:     &mockUI{},
		DryRun: false,
	}

	// Create backup
	err := createBackup(testFile, cfg, "testmodule")
	if err != nil {
		t.Fatalf("createBackup failed: %v", err)
	}

	// Verify backup was created
	backupRoot := filepath.Join(dotfilesDir, ".backups")
	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		t.Fatalf("reading backup dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no backup timestamp directory created")
	}

	// Find the backup file
	timestampDir := filepath.Join(backupRoot, entries[0].Name())
	backupFile := filepath.Join(timestampDir, ".testrc")
	backupContent, err := os.ReadFile(backupFile)
	if err != nil {
		t.Fatalf("reading backup file: %v", err)
	}

	// Verify content matches
	if string(backupContent) != testContent {
		t.Errorf("backup content doesn't match: got %q, want %q", string(backupContent), testContent)
	}

	// Verify metadata was created
	metaPath := backupFile + ".meta.json"
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("reading metadata: %v", err)
	}

	var meta BackupMetadata
	if err := json.Unmarshal(metaData, &meta); err != nil {
		t.Fatalf("parsing metadata: %v", err)
	}

	// Verify metadata fields
	if meta.OriginalPath != testFile {
		t.Errorf("wrong original path: got %q, want %q", meta.OriginalPath, testFile)
	}
	if meta.Module != "testmodule" {
		t.Errorf("wrong module: got %q, want %q", meta.Module, "testmodule")
	}
	if meta.ContentHash == "" {
		t.Error("content hash is empty")
	}
	if meta.BackupTime.IsZero() {
		t.Error("backup time is zero")
	}
}

func TestCreateBackupNonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &RunConfig{
		SysInfo: &sysinfo.SystemInfo{
			HomeDir:     tmpDir,
			DotfilesDir: tmpDir,
		},
		Config: &config.Config{},
		UI:     &mockUI{},
		DryRun: false,
	}

	// Backup non-existent file should succeed (no-op)
	err := createBackup(filepath.Join(tmpDir, "nonexistent"), cfg, "test")
	if err != nil {
		t.Errorf("createBackup failed for non-existent file: %v", err)
	}
}

func TestCreateBackupDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, ".testrc")
	if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &RunConfig{
		SysInfo: &sysinfo.SystemInfo{
			HomeDir:     tmpDir,
			DotfilesDir: tmpDir,
		},
		Config: &config.Config{},
		UI:     &mockUI{},
		DryRun: true,
	}

	// Dry run should not create backup
	err := createBackup(testFile, cfg, "test")
	if err != nil {
		t.Fatalf("createBackup failed: %v", err)
	}

	// Verify no backup directory was created
	backupRoot := filepath.Join(tmpDir, ".backups")
	if _, err := os.Stat(backupRoot); !os.IsNotExist(err) {
		t.Error("backup directory was created in dry-run mode")
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	content := "test content\n"
	if err := os.WriteFile(src, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	// Copy file
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Verify content
	dstContent, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading dest: %v", err)
	}
	if string(dstContent) != content {
		t.Errorf("content doesn't match: got %q, want %q", string(dstContent), content)
	}

	// Verify permissions preserved
	srcInfo, _ := os.Stat(src)
	dstInfo, _ := os.Stat(dst)
	if srcInfo.Mode() != dstInfo.Mode() {
		t.Errorf("permissions not preserved: src=%v, dst=%v", srcInfo.Mode(), dstInfo.Mode())
	}
}

// mockUI implements RunnerUI for testing
type mockUI struct {
	messages []string
}

func (m *mockUI) Info(msg string)                   { m.messages = append(m.messages, "INFO: "+msg) }
func (m *mockUI) Warn(msg string)                   { m.messages = append(m.messages, "WARN: "+msg) }
func (m *mockUI) Error(msg string)                  { m.messages = append(m.messages, "ERROR: "+msg) }
func (m *mockUI) Success(msg string)                { m.messages = append(m.messages, "SUCCESS: "+msg) }
func (m *mockUI) Debug(msg string)                  { m.messages = append(m.messages, "DEBUG: "+msg) }
func (m *mockUI) PromptInput(msg, def string) (string, error) { return def, nil }
func (m *mockUI) PromptConfirm(msg string, def bool) (bool, error) { return def, nil }
func (m *mockUI) PromptChoice(msg string, opts []string) (string, error) {
	if len(opts) > 0 {
		return opts[0], nil
	}
	return "", nil
}
func (m *mockUI) StartSpinner(msg string) any                      { return nil }
func (m *mockUI) StopSpinnerSuccess(s any, msg string)             {}
func (m *mockUI) StopSpinnerFail(s any, msg string)                {}
func (m *mockUI) StopSpinnerSkip(s any, msg string)                {}
func (m *mockUI) PromptMultiSelect(msg string, opts []MultiSelectOption, pre []string) ([]string, error) {
	return pre, nil
}
