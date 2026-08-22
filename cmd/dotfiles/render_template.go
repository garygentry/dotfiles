package dotfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/garygentry/dotfiles/internal/config"
	"github.com/garygentry/dotfiles/internal/module"
	"github.com/garygentry/dotfiles/internal/template"
	"github.com/spf13/cobra"
)

var (
	renderSrc    string
	renderDest   string
	renderModule string
)

var renderTemplateCmd = &cobra.Command{
	Use:    "render-template",
	Short:  "Render a template file to a destination path",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		moduleName := pickModuleName(renderModule, os.Getenv("DOTFILES_MODULE_NAME"))
		ctx, err := newRenderContext(resolveDotfilesDir(), moduleName)
		if err != nil {
			return err
		}
		return template.RenderToFile(renderSrc, renderDest, ctx)
	},
}

// pickModuleName resolves which module's config.yml settings populate .Module.
// An explicit --module flag wins; otherwise it falls back to DOTFILES_MODULE_NAME,
// which the in-process runner exports to every script (so the render_template
// shell helper needs no flag).
func pickModuleName(flag, env string) string {
	if flag != "" {
		return flag
	}
	return env
}

// resolveDotfilesDir determines the dotfiles repository root the same way the
// other shell-invoked subcommands do: prefer the DOTFILES_DIR the Go runner
// exports to every script, falling back to ~/.dotfiles.
func resolveDotfilesDir() string {
	if dir := os.Getenv("DOTFILES_DIR"); dir != "" {
		return dir
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".dotfiles")
	}
	return ".dotfiles"
}

// newRenderContext builds the template context for the render-template
// subcommand. It reuses config.Load — the SAME layered load the in-process
// runner uses (base config.yml + optional content overlay via
// DOTFILES_CONTENT_DIR) — and NewTemplateContext, so a template rendered from a
// shell script's render_template call is identical to one rendered in-process.
//
// Before this, the subcommand populated only .OS/.Arch/.Home/.Env and stuffed
// the module name plus prompt answers into .Module, leaving .User, the module's
// config settings, and .XDGConfigHome empty — so those fields silently rendered
// blank from a script. Loading the config here fixes that divergence.
func newRenderContext(dotfilesDir, moduleName string) (*template.Context, error) {
	cfg, err := config.Load(dotfilesDir)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	return module.NewTemplateContext(module.TemplateInputs{
		Config:        cfg,
		ModuleName:    moduleName,
		OS:            os.Getenv("DOTFILES_OS"),
		Arch:          os.Getenv("DOTFILES_ARCH"),
		Home:          os.Getenv("DOTFILES_HOME"),
		DotfilesDir:   dotfilesDir,
		XDGConfigHome: os.Getenv("DOTFILES_XDG_CONFIG_HOME"),
		Env:           collectDotfilesEnv(),
	}), nil
}

// collectDotfilesEnv gathers every DOTFILES_* environment variable into a map,
// mirroring the .Env the in-process runner passes (the runner exports exactly
// these vars to each script before invoking render_template).
func collectDotfilesEnv() map[string]string {
	const dotfilesPrefix = "DOTFILES_"
	env := make(map[string]string)
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if strings.HasPrefix(key, dotfilesPrefix) {
			env[key] = value
		}
	}
	return env
}

func init() {
	renderTemplateCmd.Flags().StringVar(&renderSrc, "src", "", "Source template file path")
	renderTemplateCmd.Flags().StringVar(&renderDest, "dest", "", "Destination file path")
	renderTemplateCmd.Flags().StringVar(&renderModule, "module", "", "Module name whose config.yml settings populate .Module (overrides DOTFILES_MODULE_NAME)")
	_ = renderTemplateCmd.MarkFlagRequired("src")
	_ = renderTemplateCmd.MarkFlagRequired("dest")
	rootCmd.AddCommand(renderTemplateCmd)
}
