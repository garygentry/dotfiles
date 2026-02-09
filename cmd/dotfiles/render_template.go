package dotfiles

import (
	"os"
	"strings"

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
		ctx := &template.Context{
			OS:          os.Getenv("DOTFILES_OS"),
			Arch:        os.Getenv("DOTFILES_ARCH"),
			Home:        os.Getenv("DOTFILES_HOME"),
			DotfilesDir: os.Getenv("DOTFILES_DIR"),
			Module:      make(map[string]any),
		}

		// Set the module name if provided via environment.
		if name := os.Getenv("DOTFILES_MODULE_NAME"); name != "" {
			ctx.Module["name"] = name
		}

		// Collect all DOTFILES_PROMPT_* environment variables into the
		// Module map with lowercased keys (prefix stripped).
		const promptPrefix = "DOTFILES_PROMPT_"
		for _, entry := range os.Environ() {
			key, value, ok := strings.Cut(entry, "=")
			if !ok {
				continue
			}
			if strings.HasPrefix(key, promptPrefix) {
				mapKey := strings.ToLower(strings.TrimPrefix(key, promptPrefix))
				ctx.Module[mapKey] = value
			}
		}

		return template.RenderToFile(renderSrc, renderDest, ctx)
	},
}

func init() {
	renderTemplateCmd.Flags().StringVar(&renderSrc, "src", "", "Source template file path")
	renderTemplateCmd.Flags().StringVar(&renderDest, "dest", "", "Destination file path")
	renderTemplateCmd.Flags().StringVar(&renderModule, "module", "", "Module directory")
	_ = renderTemplateCmd.MarkFlagRequired("src")
	_ = renderTemplateCmd.MarkFlagRequired("dest")
	rootCmd.AddCommand(renderTemplateCmd)
}
