package secrets

import "fmt"

// NoopProvider is a Provider that does nothing. It is returned when no
// real secret backend is configured.
type NoopProvider struct{}

// Name returns "none".
func (p *NoopProvider) Name() string {
	return "none"
}

// Available always returns true because the noop provider has no
// external dependencies.
func (p *NoopProvider) Available() bool {
	return true
}

// Authenticate is a no-op and always succeeds.
func (p *NoopProvider) Authenticate() error {
	return nil
}

// IsAuthenticated always returns true because there is nothing to
// authenticate against.
func (p *NoopProvider) IsAuthenticated() bool {
	return true
}

// GetSecret always returns an error because no real secret backend is
// configured.
func (p *NoopProvider) GetSecret(ref string) (string, error) {
	return "", fmt.Errorf("no secret provider configured")
}
