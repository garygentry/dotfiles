package dotfiles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/garygentry/dotfiles/internal/config"
	"github.com/garygentry/dotfiles/internal/module"
	"github.com/garygentry/dotfiles/internal/template"
)

// parityTemplate exercises every field that used to diverge between the
// in-process runner and the render-template subcommand: .User, the module's
// config settings (.Module, including a bool to prove type preservation),
// .XDGConfigHome, plus the always-shared .OS/.Arch/.Home/.Env fields.
const parityTemplate = `user={{ .User.email }} gh={{ .User.github_user }}
key_source={{ .Module.key_source }}
enabled={{ if .Module.enabled }}on{{ else }}off{{ end }}
xdg={{ .XDGConfigHome }}
os={{ .OS }} arch={{ .Arch }} home={{ .Home }}
greeting={{ index .Env "DOTFILES_PROMPT_GREETING" }}`

// TestRenderContextParity asserts that a template renders identically whether
// its context is built by the in-process runner (module.NewTemplateContext with
// values from sysinfo) or by the render-template subcommand (newRenderContext,
// which reloads the same layered config and reads DOTFILES_* env vars). The
// overlay case proves the subcommand honors a content overlay via config.Load,
// per watch-out §4 of the phase-4 plan.
func TestRenderContextParity(t *testing.T) {
	const moduleName = "ssh"

	tests := []struct {
		name        string
		baseConfig  string
		overlay     string // content-overlay config.yml; "" means no overlay
		wantContain string // a substring the rendered output must include
	}{
		{
			name: "no overlay, bool false stays bool",
			baseConfig: `user:
  name: Base User
  email: base@example.com
  github_user: baseuser
modules:
  ssh:
    key_source: generate
    enabled: false
`,
			wantContain: "enabled=off",
		},
		{
			name: "content overlay overrides user email and module setting",
			baseConfig: `user:
  name: Base User
  email: base@example.com
  github_user: baseuser
modules:
  ssh:
    key_source: generate
    enabled: true
`,
			overlay: `user:
  email: overlay@example.com
modules:
  ssh:
    key_source: 1password
`,
			wantContain: "user=overlay@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dotfilesDir := t.TempDir()
			writeFile(t, filepath.Join(dotfilesDir, "config.yml"), tt.baseConfig)

			if tt.overlay != "" {
				contentDir := t.TempDir()
				writeFile(t, filepath.Join(contentDir, "config.yml"), tt.overlay)
				t.Setenv("DOTFILES_CONTENT_DIR", contentDir)
			} else {
				// Ensure no stray overlay from the ambient environment.
				t.Setenv("DOTFILES_CONTENT_DIR", "")
			}

			// Shared system-level values. The subcommand reads these from the
			// environment; the runner gets them from sysinfo. We set the env and
			// pass the same literals to the runner side so any divergence in how
			// the subcommand assembles them surfaces as a render mismatch.
			const (
				osName = "linux"
				arch   = "amd64"
				home   = "/home/tester"
				xdg    = "/home/tester/.config"
			)
			t.Setenv("DOTFILES_DIR", dotfilesDir)
			t.Setenv("DOTFILES_OS", osName)
			t.Setenv("DOTFILES_ARCH", arch)
			t.Setenv("DOTFILES_HOME", home)
			t.Setenv("DOTFILES_XDG_CONFIG_HOME", xdg)
			t.Setenv("DOTFILES_MODULE_NAME", moduleName)
			t.Setenv("DOTFILES_PROMPT_GREETING", "hello")

			// In-process (runner) context: built from the loaded config and
			// explicit sysinfo values, mirroring module.buildTemplateContext.
			cfg, err := config.Load(dotfilesDir)
			if err != nil {
				t.Fatalf("config.Load: %v", err)
			}
			runnerCtx := module.NewTemplateContext(module.TemplateInputs{
				Config:        cfg,
				ModuleName:    moduleName,
				OS:            osName,
				Arch:          arch,
				Home:          home,
				DotfilesDir:   dotfilesDir,
				XDGConfigHome: xdg,
				Env:           map[string]string{"DOTFILES_PROMPT_GREETING": "hello"},
			})

			// Subcommand context: reloads config + reads DOTFILES_* env vars.
			subCtx, err := newRenderContext(dotfilesDir, os.Getenv("DOTFILES_MODULE_NAME"))
			if err != nil {
				t.Fatalf("newRenderContext: %v", err)
			}

			runnerOut, err := template.RenderString(parityTemplate, runnerCtx)
			if err != nil {
				t.Fatalf("render (runner): %v", err)
			}
			subOut, err := template.RenderString(parityTemplate, subCtx)
			if err != nil {
				t.Fatalf("render (subcommand): %v", err)
			}

			if runnerOut != subOut {
				t.Errorf("render mismatch between in-process and subcommand paths:\n--- in-process ---\n%s\n--- subcommand ---\n%s", runnerOut, subOut)
			}
			if tt.wantContain != "" && !strings.Contains(runnerOut, tt.wantContain) {
				t.Errorf("rendered output missing %q; got:\n%s", tt.wantContain, runnerOut)
			}
		})
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
