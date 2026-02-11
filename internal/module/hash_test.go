package module

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/garygentry/dotfiles/internal/config"
)

func TestComputeFileHash(t *testing.T) {
	// Create a temporary file with known content
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := "test content\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Compute hash
	hash1, err := ComputeFileHash(testFile)
	if err != nil {
		t.Fatalf("ComputeFileHash failed: %v", err)
	}

	// Should be non-empty
	if hash1 == "" {
		t.Fatal("hash is empty")
	}

	// Should be deterministic (same content = same hash)
	hash2, err := ComputeFileHash(testFile)
	if err != nil {
		t.Fatalf("ComputeFileHash failed on second call: %v", err)
	}
	if hash1 != hash2 {
		t.Errorf("hash not deterministic: %s != %s", hash1, hash2)
	}

	// Should detect content changes
	if err := os.WriteFile(testFile, []byte(content+"modified"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash3, err := ComputeFileHash(testFile)
	if err != nil {
		t.Fatalf("ComputeFileHash failed after modification: %v", err)
	}
	if hash1 == hash3 {
		t.Error("hash didn't change after file modification")
	}

	// Should error on non-existent file
	_, err = ComputeFileHash(filepath.Join(tmpDir, "nonexistent"))
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestComputeModuleChecksum(t *testing.T) {
	tmpDir := t.TempDir()
	modDir := filepath.Join(tmpDir, "testmod")
	if err := os.Mkdir(modDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create minimal module structure
	moduleYML := `name: testmod
version: "1.0.0"
description: test module`

	if err := os.WriteFile(filepath.Join(modDir, "module.yml"), []byte(moduleYML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "install.sh"), []byte("#!/bin/bash\necho install"), 0o755); err != nil {
		t.Fatal(err)
	}

	mod := &Module{
		Name:    "testmod",
		Version: "1.0.0",
		Dir:     modDir,
	}

	// Compute initial checksum
	checksum1, err := ComputeModuleChecksum(mod)
	if err != nil {
		t.Fatalf("ComputeModuleChecksum failed: %v", err)
	}
	if checksum1 == "" {
		t.Fatal("checksum is empty")
	}

	// Should be deterministic
	checksum2, err := ComputeModuleChecksum(mod)
	if err != nil {
		t.Fatalf("ComputeModuleChecksum failed on second call: %v", err)
	}
	if checksum1 != checksum2 {
		t.Errorf("checksum not deterministic: %s != %s", checksum1, checksum2)
	}

	// Should detect module.yml changes
	moduleYML2 := moduleYML + "\npriority: 100"
	if err := os.WriteFile(filepath.Join(modDir, "module.yml"), []byte(moduleYML2), 0o644); err != nil {
		t.Fatal(err)
	}
	checksum3, err := ComputeModuleChecksum(mod)
	if err != nil {
		t.Fatalf("ComputeModuleChecksum failed after module.yml change: %v", err)
	}
	if checksum1 == checksum3 {
		t.Error("checksum didn't change after module.yml modification")
	}

	// Should detect install.sh changes
	if err := os.WriteFile(filepath.Join(modDir, "install.sh"), []byte("#!/bin/bash\necho modified"), 0o755); err != nil {
		t.Fatal(err)
	}
	checksum4, err := ComputeModuleChecksum(mod)
	if err != nil {
		t.Fatalf("ComputeModuleChecksum failed after install.sh change: %v", err)
	}
	if checksum3 == checksum4 {
		t.Error("checksum didn't change after install.sh modification")
	}

	// Should include OS-specific scripts
	osDir := filepath.Join(modDir, "os")
	if err := os.Mkdir(osDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(osDir, "linux.sh"), []byte("echo linux"), 0o755); err != nil {
		t.Fatal(err)
	}
	checksum5, err := ComputeModuleChecksum(mod)
	if err != nil {
		t.Fatalf("ComputeModuleChecksum failed after adding OS script: %v", err)
	}
	if checksum4 == checksum5 {
		t.Error("checksum didn't change after adding OS-specific script")
	}
}

func TestComputeConfigHash(t *testing.T) {
	mod := &Module{
		Name: "testmod",
	}

	cfg1 := &config.Config{
		User: config.UserConfig{
			Name:  "Test User",
			Email: "test@example.com",
		},
		Modules: map[string]map[string]any{
			"testmod": {
				"option1": "value1",
				"option2": 42,
			},
		},
	}

	// Compute initial hash
	hash1 := ComputeConfigHash(mod, cfg1)
	if hash1 == "" {
		t.Fatal("hash is empty")
	}

	// Should be deterministic
	hash2 := ComputeConfigHash(mod, cfg1)
	if hash1 != hash2 {
		t.Errorf("hash not deterministic: %s != %s", hash1, hash2)
	}

	// Should detect user config changes
	cfg2 := &config.Config{
		User: config.UserConfig{
			Name:  "Different User",
			Email: "test@example.com",
		},
		Modules: cfg1.Modules,
	}
	hash3 := ComputeConfigHash(mod, cfg2)
	if hash1 == hash3 {
		t.Error("hash didn't change after user.name modification")
	}

	// Should detect module-specific config changes
	cfg3 := &config.Config{
		User: cfg1.User,
		Modules: map[string]map[string]any{
			"testmod": {
				"option1": "modified",
				"option2": 42,
			},
		},
	}
	hash4 := ComputeConfigHash(mod, cfg3)
	if hash1 == hash4 {
		t.Error("hash didn't change after module config modification")
	}

	// Should be consistent regardless of map iteration order
	// (Go randomizes map iteration, but our hash should be deterministic)
	cfg4 := &config.Config{
		User: cfg1.User,
		Modules: map[string]map[string]any{
			"testmod": {
				"option2": 42,
				"option1": "value1",
			},
		},
	}
	hash5 := ComputeConfigHash(mod, cfg4)
	if hash1 != hash5 {
		t.Error("hash changed with different key order (should be deterministic)")
	}

	// Should not be affected by other modules' config
	cfg5 := &config.Config{
		User: cfg1.User,
		Modules: map[string]map[string]any{
			"testmod": {
				"option1": "value1",
				"option2": 42,
			},
			"othermod": {
				"something": "else",
			},
		},
	}
	hash6 := ComputeConfigHash(mod, cfg5)
	if hash1 != hash6 {
		t.Error("hash changed when other module config added")
	}
}
