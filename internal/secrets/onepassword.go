package secrets

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// OnePasswordProvider implements Provider using the 1Password CLI ("op").
type OnePasswordProvider struct {
	account string
}

// Name returns "1password".
func (p *OnePasswordProvider) Name() string {
	return "1password"
}

// Available reports whether the "op" CLI is installed on the system.
func (p *OnePasswordProvider) Available() bool {
	_, err := exec.LookPath("op")
	return err == nil
}

// Authenticate runs an interactive sign-in flow so the user can authenticate
// with 1Password. Stdin, stdout, and stderr are connected to the terminal so
// the op CLI can prompt for credentials. Uses a 60-second timeout.
func (p *OnePasswordProvider) Authenticate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "op", "signin", "--account", p.account)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("1password sign-in failed: %w", err)
	}
	return nil
}

// IsAuthenticated performs a non-interactive check for a valid 1Password
// session. It first verifies that at least one account is configured (via
// "op account list") and then checks for an active session (via "op whoami").
// Neither command triggers interactive prompts.
func (p *OnePasswordProvider) IsAuthenticated() bool {
	// Step 1: Check if any accounts are configured. This never prompts.
	ctx1, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel1()

	listCmd := exec.CommandContext(ctx1, "op", "account", "list")
	listCmd.Stdin = nil
	listOut, err := listCmd.Output()
	if err != nil || len(strings.TrimSpace(string(listOut))) == 0 {
		return false
	}

	// Step 2: Check for an active session. This fails silently if no
	// session exists and never triggers interactive prompts.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	whoamiCmd := exec.CommandContext(ctx2, "op", "whoami", "--account", p.account)
	whoamiCmd.Stdin = nil
	whoamiCmd.Stdout = nil
	whoamiCmd.Stderr = nil

	return whoamiCmd.Run() == nil
}

// GetSecret resolves a 1Password secret reference (e.g.
// "op://vault/item/field") and returns the secret value.
func (p *OnePasswordProvider) GetSecret(ref string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "op", "--account", p.account, "read", ref)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("1password read failed: %s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}
