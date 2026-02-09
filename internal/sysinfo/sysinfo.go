package sysinfo

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// SystemInfo holds detected information about the host system.
type SystemInfo struct {
	OS            string // "macos", "ubuntu", "arch", "debian", etc.
	Arch          string // runtime.GOARCH value, e.g. "amd64", "arm64"
	PkgMgr       string // "brew", "apt", "pacman", or ""
	HasSudo       bool   // whether the current user can run sudo without a password
	User          string // current username
	HomeDir       string // user home directory
	DotfilesDir   string // location of dotfiles repository
	IsInteractive bool   // true when stdin is a terminal
}

// Detect gathers system information and returns a populated SystemInfo.
// It is intended to be called once at startup.
func Detect() (*SystemInfo, error) {
	info := &SystemInfo{}

	// --- OS ---
	info.OS = detectOS()

	// --- Arch ---
	info.Arch = runtime.GOARCH

	// --- PkgMgr ---
	info.PkgMgr = detectPkgMgr(info.OS)

	// --- HasSudo ---
	info.HasSudo = detectSudo()

	// --- User ---
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	info.User = u.Username

	// --- HomeDir ---
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	info.HomeDir = home

	// --- DotfilesDir ---
	if dir := os.Getenv("DOTFILES_DIR"); dir != "" {
		info.DotfilesDir = dir
	} else {
		info.DotfilesDir = filepath.Join(home, ".dotfiles")
	}

	// --- IsInteractive ---
	info.IsInteractive = detectInteractive()

	return info, nil
}

// detectOS returns a human-friendly OS identifier.
// On macOS it returns "macos". On Linux it parses /etc/os-release for the
// distribution ID (e.g. "ubuntu", "arch", "debian"). Falls back to
// runtime.GOOS when the distribution cannot be determined.
func detectOS() string {
	if runtime.GOOS == "darwin" {
		return "macos"
	}

	if runtime.GOOS == "linux" {
		if id := parseOSReleaseID("/etc/os-release"); id != "" {
			return id
		}
	}

	return runtime.GOOS
}

// parseOSReleaseID reads the given file looking for a line of the form
// ID=<value> and returns the unquoted value in lowercase.
func parseOSReleaseID(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "ID=") {
			value := strings.TrimPrefix(line, "ID=")
			value = strings.Trim(value, `"'`)
			return strings.ToLower(value)
		}
	}
	return ""
}

// detectPkgMgr returns the expected package manager for the given OS identifier.
func detectPkgMgr(osID string) string {
	switch osID {
	case "macos":
		return "brew"
	case "ubuntu", "debian":
		return "apt"
	case "arch", "manjaro":
		return "pacman"
	default:
		return ""
	}
}

// detectSudo checks whether the current user can invoke sudo without a
// password prompt by running "sudo -n true" with a 2-second timeout.
func detectSudo() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sudo", "-n", "true")
	// Suppress all output.
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// detectInteractive returns true when standard input is connected to a
// terminal (character device).
func detectInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
