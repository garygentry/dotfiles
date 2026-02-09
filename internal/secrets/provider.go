package secrets

import "strings"

// Provider is the interface that all secret backends must implement.
type Provider interface {
	// Name returns the human-readable name of the provider.
	Name() string

	// Available reports whether the provider's prerequisites (e.g. CLI
	// tools) are installed on the current system.
	Available() bool

	// Authenticate performs an interactive authentication flow so that
	// subsequent calls to GetSecret can succeed.
	Authenticate() error

	// IsAuthenticated reports whether the provider already has a valid
	// session and can serve secrets without re-authenticating.
	IsAuthenticated() bool

	// GetSecret resolves ref (provider-specific secret reference) and
	// returns the secret value.
	GetSecret(ref string) (string, error)
}

// NewProvider returns a Provider implementation that matches name.
// For "1password" / "onepassword" it returns a OnePasswordProvider
// configured with account. For anything else (including an empty
// string) it returns a NoopProvider.
func NewProvider(name string, account string) Provider {
	switch strings.ToLower(name) {
	case "1password", "onepassword":
		return &OnePasswordProvider{account: account}
	default:
		return &NoopProvider{}
	}
}
