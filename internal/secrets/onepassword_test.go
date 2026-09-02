package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeOpOnPath drops a fake "op" executable early on PATH so IsAuthenticated can
// be exercised without a real 1Password CLI or network. The script body decides
// exit codes per subcommand.
func fakeOpOnPath(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "op"), []byte(script), 0o755); err != nil {
		t.Fatalf("write fake op: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// A working service-account token authenticates via "op whoami" even though the
// account list is empty for it, so IsAuthenticated must report true (and callers
// must not fall through to an interactive sign-in prompt).
func TestOnePassword_IsAuthenticated_ServiceAccountValid(t *testing.T) {
	// whoami succeeds; account list is empty (as it is for a service account).
	fakeOpOnPath(t, "#!/bin/sh\ncase \"$1\" in\n  whoami) exit 0 ;;\n  *) exit 0 ;;\nesac\n")
	t.Setenv("OP_SERVICE_ACCOUNT_TOKEN", "ops_faketoken")

	p := &OnePasswordProvider{account: "my.1password.com"}
	if !p.IsAuthenticated() {
		t.Error("IsAuthenticated() = false with a valid service-account token, want true")
	}
}

// A service-account token that "op whoami" rejects is not authenticated.
func TestOnePassword_IsAuthenticated_ServiceAccountInvalid(t *testing.T) {
	fakeOpOnPath(t, "#!/bin/sh\nexit 1\n")
	t.Setenv("OP_SERVICE_ACCOUNT_TOKEN", "ops_faketoken")

	p := &OnePasswordProvider{account: "my.1password.com"}
	if p.IsAuthenticated() {
		t.Error("IsAuthenticated() = true with a rejected service-account token, want false")
	}
}

// Without a service-account token, an empty account list means not authenticated
// (the token branch must not swallow the normal path).
func TestOnePassword_IsAuthenticated_NoTokenNoAccounts(t *testing.T) {
	fakeOpOnPath(t, "#!/bin/sh\ncase \"$1\" in\n  account) ;;\n  *) exit 1 ;;\nesac\n")
	t.Setenv("OP_SERVICE_ACCOUNT_TOKEN", "")

	p := &OnePasswordProvider{account: "my.1password.com"}
	if p.IsAuthenticated() {
		t.Error("IsAuthenticated() = true with no token and no accounts, want false")
	}
}
