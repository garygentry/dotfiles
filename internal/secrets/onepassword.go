package secrets

import (
	"bytes"
	"context"
	"fmt"
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

// Authenticate runs an interactive vault list command so that the 1Password
// CLI can prompt the user for credentials if needed. It uses a 30-second
// timeout to avoid hanging indefinitely.
func (p *OnePasswordProvider) Authenticate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "op", "--account", p.account, "vault", "list")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("1password authentication failed: %s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return nil
}

// IsAuthenticated runs a silent vault list command and returns true when
// it exits successfully, indicating a valid session already exists.
func (p *OnePasswordProvider) IsAuthenticated() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "op", "--account", p.account, "vault", "list")
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run() == nil
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
