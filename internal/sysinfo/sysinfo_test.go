package sysinfo

import (
	"os"
	"runtime"
	"testing"
)

func TestDetectReturnsNonNil(t *testing.T) {
	info, err := Detect()
	if err != nil {
		t.Fatalf("Detect() returned unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("Detect() returned nil SystemInfo")
	}
}

func TestDetectFieldsNonEmpty(t *testing.T) {
	info, err := Detect()
	if err != nil {
		t.Fatalf("Detect() returned unexpected error: %v", err)
	}

	if info.OS == "" {
		t.Error("OS is empty")
	}
	if info.Arch == "" {
		t.Error("Arch is empty")
	}
	if info.User == "" {
		t.Error("User is empty")
	}
	if info.HomeDir == "" {
		t.Error("HomeDir is empty")
	}
}

func TestDetectArch(t *testing.T) {
	info, err := Detect()
	if err != nil {
		t.Fatalf("Detect() returned unexpected error: %v", err)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("Arch = %q; want %q", info.Arch, runtime.GOARCH)
	}
}

func TestDetectPkgMgrMatchesOS(t *testing.T) {
	info, err := Detect()
	if err != nil {
		t.Fatalf("Detect() returned unexpected error: %v", err)
	}

	expected := detectPkgMgr(info.OS)
	if info.PkgMgr != expected {
		t.Errorf("PkgMgr = %q; want %q for OS %q", info.PkgMgr, expected, info.OS)
	}
}

func TestDetectPkgMgrValues(t *testing.T) {
	tests := []struct {
		osID   string
		expect string
	}{
		{"macos", "brew"},
		{"ubuntu", "apt"},
		{"debian", "apt"},
		{"arch", "pacman"},
		{"manjaro", "pacman"},
		{"fedora", ""},
		{"unknown", ""},
	}

	for _, tc := range tests {
		t.Run(tc.osID, func(t *testing.T) {
			got := detectPkgMgr(tc.osID)
			if got != tc.expect {
				t.Errorf("detectPkgMgr(%q) = %q; want %q", tc.osID, got, tc.expect)
			}
		})
	}
}

func TestDotfilesDirDefault(t *testing.T) {
	// Ensure DOTFILES_DIR is unset for this test.
	orig := os.Getenv("DOTFILES_DIR")
	os.Unsetenv("DOTFILES_DIR")
	t.Cleanup(func() {
		if orig != "" {
			os.Setenv("DOTFILES_DIR", orig)
		}
	})

	info, err := Detect()
	if err != nil {
		t.Fatalf("Detect() returned unexpected error: %v", err)
	}

	want := info.HomeDir + "/.dotfiles"
	if info.DotfilesDir != want {
		t.Errorf("DotfilesDir = %q; want %q", info.DotfilesDir, want)
	}
}

func TestDotfilesDirEnvOverride(t *testing.T) {
	customDir := "/tmp/my-dotfiles"
	orig := os.Getenv("DOTFILES_DIR")
	os.Setenv("DOTFILES_DIR", customDir)
	t.Cleanup(func() {
		if orig != "" {
			os.Setenv("DOTFILES_DIR", orig)
		} else {
			os.Unsetenv("DOTFILES_DIR")
		}
	})

	info, err := Detect()
	if err != nil {
		t.Fatalf("Detect() returned unexpected error: %v", err)
	}

	if info.DotfilesDir != customDir {
		t.Errorf("DotfilesDir = %q; want %q", info.DotfilesDir, customDir)
	}
}

func TestDetectOSDarwin(t *testing.T) {
	// Only meaningful on macOS, but the unit function can be tested anywhere.
	if runtime.GOOS != "darwin" {
		t.Skip("skipping macOS-specific test")
	}
	info, err := Detect()
	if err != nil {
		t.Fatalf("Detect() returned unexpected error: %v", err)
	}
	if info.OS != "macos" {
		t.Errorf("OS = %q; want %q on darwin", info.OS, "macos")
	}
}

func TestParseOSReleaseID(t *testing.T) {
	// Create a temporary os-release file to test the parser.
	content := `NAME="Ubuntu"
VERSION="22.04.3 LTS (Jammy Jellyfish)"
ID=ubuntu
ID_LIKE=debian
`
	tmp, err := os.CreateTemp("", "os-release-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmp.Close()

	got := parseOSReleaseID(tmp.Name())
	if got != "ubuntu" {
		t.Errorf("parseOSReleaseID() = %q; want %q", got, "ubuntu")
	}
}

func TestParseOSReleaseIDQuoted(t *testing.T) {
	content := `ID="arch"
`
	tmp, err := os.CreateTemp("", "os-release-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmp.Close()

	got := parseOSReleaseID(tmp.Name())
	if got != "arch" {
		t.Errorf("parseOSReleaseID() = %q; want %q", got, "arch")
	}
}

func TestParseOSReleaseIDMissing(t *testing.T) {
	got := parseOSReleaseID("/nonexistent/path/os-release")
	if got != "" {
		t.Errorf("parseOSReleaseID() = %q; want empty string for missing file", got)
	}
}
