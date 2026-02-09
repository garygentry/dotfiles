package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Context holds all data available to templates during rendering.
type Context struct {
	// User contains user-defined key/value pairs from the profile configuration.
	User map[string]string

	// OS is the operating system identifier (e.g., "linux", "darwin").
	OS string

	// Arch is the architecture identifier (e.g., "amd64", "arm64").
	Arch string

	// Home is the path to the user's home directory.
	Home string

	// DotfilesDir is the root path of the dotfiles repository.
	DotfilesDir string

	// Module holds module-specific settings passed during rendering.
	Module map[string]any

	// Secrets contains sensitive key/value pairs (e.g., tokens, passwords).
	Secrets map[string]string

	// Env contains environment variable overrides available to templates.
	Env map[string]string
}

// funcMap returns the custom template functions available in all templates.
func funcMap() template.FuncMap {
	return template.FuncMap{
		// env returns the value of the environment variable named by the key.
		"env": os.Getenv,

		// default returns the first non-empty string among its arguments.
		// Usage: {{ default .Value "fallback" }}
		"default": func(values ...string) string {
			for _, v := range values {
				if v != "" {
					return v
				}
			}
			return ""
		},

		// upper converts a string to uppercase.
		"upper": strings.ToUpper,

		// lower converts a string to lowercase.
		"lower": strings.ToLower,

		// contains reports whether substr is within s.
		"contains": strings.Contains,

		// join concatenates the elements of a string slice with the given separator.
		"join": strings.Join,

		// trimSpace removes leading and trailing whitespace.
		"trimSpace": strings.TrimSpace,
	}
}

// Render reads the template file at templatePath, renders it using the
// provided Context, and returns the resulting string. Template files use
// the .tpl extension by convention.
func Render(templatePath string, ctx *Context) (string, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("reading template %s: %w", templatePath, err)
	}

	name := filepath.Base(templatePath)
	return renderTemplate(name, string(content), ctx)
}

// RenderString renders the given template string using the provided Context
// and returns the resulting string.
func RenderString(templateStr string, ctx *Context) (string, error) {
	return renderTemplate("string", templateStr, ctx)
}

// RenderToFile renders the template at templatePath using ctx and writes the
// output to destPath. Parent directories for destPath are created as needed.
// If the source template file has specific permissions, those permissions are
// preserved on the destination file.
func RenderToFile(templatePath, destPath string, ctx *Context) error {
	rendered, err := Render(templatePath, ctx)
	if err != nil {
		return err
	}

	// Create parent directories for the destination file.
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", destDir, err)
	}

	// Attempt to preserve the source file's permissions.
	perm := os.FileMode(0o644)
	if info, err := os.Stat(templatePath); err == nil {
		perm = info.Mode().Perm()
	}

	if err := os.WriteFile(destPath, []byte(rendered), perm); err != nil {
		return fmt.Errorf("writing file %s: %w", destPath, err)
	}

	return nil
}

// renderTemplate parses and executes a named template string with the given context.
func renderTemplate(name, text string, ctx *Context) (string, error) {
	tmpl, err := template.New(name).Funcs(funcMap()).Parse(text)
	if err != nil {
		return "", fmt.Errorf("parsing template %s: %w", name, err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("executing template %s: %w", name, err)
	}

	return buf.String(), nil
}
