package secrets

import "testing"

func TestNewProvider_OnePassword(t *testing.T) {
	p := NewProvider("1password", "my.1password.com")

	op, ok := p.(*OnePasswordProvider)
	if !ok {
		t.Fatalf("NewProvider(\"1password\", ...) returned %T, want *OnePasswordProvider", p)
	}
	if op.Name() != "1password" {
		t.Errorf("Name() = %q, want %q", op.Name(), "1password")
	}
}

func TestNewProvider_OnePassword_Alias(t *testing.T) {
	p := NewProvider("onepassword", "my.1password.com")

	if _, ok := p.(*OnePasswordProvider); !ok {
		t.Fatalf("NewProvider(\"onepassword\", ...) returned %T, want *OnePasswordProvider", p)
	}
}

func TestNewProvider_Empty(t *testing.T) {
	p := NewProvider("", "")

	if _, ok := p.(*NoopProvider); !ok {
		t.Fatalf("NewProvider(\"\", \"\") returned %T, want *NoopProvider", p)
	}
}

func TestNewProvider_Unknown(t *testing.T) {
	p := NewProvider("vault", "vault.example.com")

	if _, ok := p.(*NoopProvider); !ok {
		t.Fatalf("NewProvider(\"vault\", ...) returned %T, want *NoopProvider", p)
	}
}

func TestNoopProvider_GetSecret_ReturnsError(t *testing.T) {
	p := &NoopProvider{}

	_, err := p.GetSecret("op://vault/item/field")
	if err == nil {
		t.Fatal("GetSecret() returned nil error, want error")
	}
	if err.Error() != "no secret provider configured" {
		t.Errorf("GetSecret() error = %q, want %q", err.Error(), "no secret provider configured")
	}
}

func TestNoopProvider_Available(t *testing.T) {
	p := &NoopProvider{}

	if !p.Available() {
		t.Error("Available() = false, want true")
	}
}

func TestNoopProvider_IsAuthenticated(t *testing.T) {
	p := &NoopProvider{}

	if !p.IsAuthenticated() {
		t.Error("IsAuthenticated() = false, want true")
	}
}

func TestNoopProvider_Authenticate(t *testing.T) {
	p := &NoopProvider{}

	if err := p.Authenticate(); err != nil {
		t.Errorf("Authenticate() error = %v, want nil", err)
	}
}

func TestNoopProvider_Name(t *testing.T) {
	p := &NoopProvider{}

	if p.Name() != "none" {
		t.Errorf("Name() = %q, want %q", p.Name(), "none")
	}
}
