package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenderString_SimpleVariable(t *testing.T) {
	ctx := &Context{
		User: map[string]string{"name": "Alice", "email": "alice@example.com"},
		OS:   "linux",
		Arch: "amd64",
		Home: "/home/alice",
	}

	tests := []struct {
		name     string
		tmpl     string
		expected string
	}{
		{
			name:     "user field",
			tmpl:     `Hello, {{ .User.name }}!`,
			expected: "Hello, Alice!",
		},
		{
			name:     "os field",
			tmpl:     `OS={{ .OS }}`,
			expected: "OS=linux",
		},
		{
			name:     "arch field",
			tmpl:     `Arch={{ .Arch }}`,
			expected: "Arch=amd64",
		},
		{
			name:     "home field",
			tmpl:     `Home={{ .Home }}`,
			expected: "Home=/home/alice",
		},
		{
			name:     "multiple fields",
			tmpl:     `{{ .User.name }} on {{ .OS }}/{{ .Arch }}`,
			expected: "Alice on linux/amd64",
		},
		{
			name:     "dotfiles dir",
			tmpl:     `Dir={{ .DotfilesDir }}`,
			expected: "Dir=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderString(tt.tmpl, ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRenderString_CustomFunctions(t *testing.T) {
	// Set an environment variable for the env function test.
	t.Setenv("DOTFILES_TEST_VAR", "from_env")

	ctx := &Context{
		User: map[string]string{"name": "bob"},
		Env:  map[string]string{"SHELL": "/bin/zsh"},
	}

	tests := []struct {
		name     string
		tmpl     string
		expected string
	}{
		{
			name:     "default with value present",
			tmpl:     `{{ default .User.name "fallback" }}`,
			expected: "bob",
		},
		{
			name:     "default with empty value",
			tmpl:     `{{ default "" "fallback" }}`,
			expected: "fallback",
		},
		{
			name:     "default all empty",
			tmpl:     `{{ default "" "" }}`,
			expected: "",
		},
		{
			name:     "upper",
			tmpl:     `{{ upper .User.name }}`,
			expected: "BOB",
		},
		{
			name:     "lower",
			tmpl:     `{{ lower "HELLO" }}`,
			expected: "hello",
		},
		{
			name:     "env",
			tmpl:     `{{ env "DOTFILES_TEST_VAR" }}`,
			expected: "from_env",
		},
		{
			name:     "env missing variable",
			tmpl:     `[{{ env "DOTFILES_NONEXISTENT_VAR_12345" }}]`,
			expected: "[]",
		},
		{
			name:     "join",
			tmpl:     `{{ join .List ", " }}`,
			expected: "", // join requires a slice; tested separately below
		},
		{
			name:     "trimSpace",
			tmpl:     `[{{ trimSpace "  hello  " }}]`,
			expected: "[hello]",
		},
		{
			name:     "contains true",
			tmpl:     `{{ contains "foobar" "bar" }}`,
			expected: "true",
		},
		{
			name:     "contains false",
			tmpl:     `{{ contains "foobar" "baz" }}`,
			expected: "false",
		},
	}

	for _, tt := range tests {
		if tt.name == "join" {
			continue // tested separately
		}
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderString(tt.tmpl, ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRenderString_JoinFunction(t *testing.T) {
	// join requires a []string, which we can supply through Module.
	ctx := &Context{
		Module: map[string]any{
			"tags": []string{"go", "rust", "python"},
		},
	}

	tmpl := `{{ join .Module.tags ", " }}`
	result, err := RenderString(tmpl, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "go, rust, python"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRender_FromFile(t *testing.T) {
	// Create a temporary .tpl file.
	dir := t.TempDir()
	tplPath := filepath.Join(dir, "greeting.tpl")

	content := `Hello, {{ .User.name }}! Welcome to {{ .OS }}.`
	if err := os.WriteFile(tplPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}

	ctx := &Context{
		User: map[string]string{"name": "Charlie"},
		OS:   "darwin",
	}

	result, err := Render(tplPath, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Hello, Charlie! Welcome to darwin."
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRender_FileNotFound(t *testing.T) {
	ctx := &Context{}
	_, err := Render("/nonexistent/path/template.tpl", ctx)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestRenderToFile_CreatesOutput(t *testing.T) {
	dir := t.TempDir()

	// Create a source template with specific permissions.
	tplPath := filepath.Join(dir, "config.tpl")
	tplContent := `# Config for {{ .User.name }}
home = "{{ .Home }}"
os = "{{ .OS }}"
`
	if err := os.WriteFile(tplPath, []byte(tplContent), 0o755); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}

	// Destination is in a nested directory that does not yet exist.
	destPath := filepath.Join(dir, "output", "nested", "config")

	ctx := &Context{
		User: map[string]string{"name": "Dana"},
		Home: "/home/dana",
		OS:   "linux",
	}

	if err := RenderToFile(tplPath, destPath, ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the output file exists and has the expected content.
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	expected := `# Config for Dana
home = "/home/dana"
os = "linux"
`
	if string(data) != expected {
		t.Errorf("got:\n%s\nwant:\n%s", string(data), expected)
	}

	// Verify permissions are preserved from the source template.
	info, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("failed to stat output file: %v", err)
	}

	// The source was written with 0755; check the dest matches.
	gotPerm := info.Mode().Perm()
	wantPerm := os.FileMode(0o755)
	if gotPerm != wantPerm {
		t.Errorf("permissions: got %o, want %o", gotPerm, wantPerm)
	}
}

func TestRenderToFile_TemplateError(t *testing.T) {
	dir := t.TempDir()

	// Write a template with invalid syntax.
	tplPath := filepath.Join(dir, "bad.tpl")
	if err := os.WriteFile(tplPath, []byte(`{{ .Invalid`), 0o644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}

	destPath := filepath.Join(dir, "output.txt")
	ctx := &Context{}

	err := RenderToFile(tplPath, destPath, ctx)
	if err == nil {
		t.Fatal("expected error for invalid template, got nil")
	}

	// Verify the destination file was not created.
	if _, statErr := os.Stat(destPath); !os.IsNotExist(statErr) {
		t.Error("destination file should not exist after template error")
	}
}

func TestRenderString_SecretsAndEnv(t *testing.T) {
	ctx := &Context{
		Secrets: map[string]string{"api_key": "sk-12345"},
		Env:     map[string]string{"EDITOR": "vim"},
	}

	tmpl := `key={{ .Secrets.api_key }} editor={{ .Env.EDITOR }}`
	result, err := RenderString(tmpl, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "key=sk-12345 editor=vim"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRenderString_ModuleData(t *testing.T) {
	ctx := &Context{
		Module: map[string]any{
			"theme":   "gruvbox",
			"plugins": []string{"vim-go", "fzf"},
		},
	}

	tmpl := `theme={{ .Module.theme }}`
	result, err := RenderString(tmpl, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "theme=gruvbox"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}
