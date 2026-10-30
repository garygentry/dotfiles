package dotfiles

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// get-secret runs inside module scripts and the root command silences errors,
// so a failure must explain itself on stderr. Here 1Password is configured and
// `op` exists but has no session: the message must say to sign in.
func TestGetSecretReportsNotSignedInOnStderr(t *testing.T) {
	dotfilesDir := t.TempDir()
	cfg := "secrets:\n  provider: 1password\n  account: my.1password.com\n"
	if err := os.WriteFile(filepath.Join(dotfilesDir, "config.yml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	fakeOp := "#!/bin/sh\ncase \"$1\" in account) echo my.1password.com;; *) exit 1;; esac\n"
	if err := os.WriteFile(filepath.Join(binDir, "op"), []byte(fakeOp), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTFILES_DIR", dotfilesDir)
	t.Setenv("DOTFILES_CONTENT_DIR", "")
	t.Setenv("OP_SERVICE_ACCOUNT_TOKEN", "")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stderr := os.Stderr
	os.Stderr = w
	secretRef = "op://vault/item/field"
	runErr := getSecretCmd.RunE(getSecretCmd, nil)
	os.Stderr = stderr
	w.Close()
	out, _ := io.ReadAll(r)

	if runErr == nil {
		t.Fatal("expected an error without a 1Password session")
	}
	if !strings.Contains(string(out), "not signed in") || !strings.Contains(string(out), "op signin") {
		t.Errorf("stderr = %q, want a sign-in hint", out)
	}
}
