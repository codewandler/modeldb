package cli

import (
	"context"
	"fmt"

	modeldb "github.com/codewandler/modeldb"
	"github.com/spf13/cobra"
)

type BuildCommandOptions struct {
	IO                 IO
	DefaultCatalogPath string
}

func NewBuildCommand(opts BuildCommandOptions) *cobra.Command {
	ioCfg := opts.IO.withDefaults()
	defaultPath := opts.DefaultCatalogPath
	if defaultPath == "" {
		defaultPath = "catalog.json"
	}

	var outPath string
	var anthropicFile string
	var modelsDevFile string
	var codexFile string
	var openAIStaticFile string
	var useFixture bool

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build a catalog snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			sources := modeldb.DefaultBuildSources()
			if anthropicFile != "" {
				for i := range sources {
					if _, ok := sources[i].Source.(modeldb.AnthropicAPISource); ok {
						sources[i].Source = modeldb.NewAnthropicAPISourceFromFile(anthropicFile)
					}
				}
			}
			if modelsDevFile != "" || useFixture {
				for i := range sources {
					if _, ok := sources[i].Source.(modeldb.ModelsDevSource); ok {
						cfg := modeldb.NewModelsDevSource()
						cfg.FilePath = modelsDevFile
						cfg.UseFixture = useFixture
						sources[i].Source = cfg
					}
				}
			}
			if codexFile != "" {
				for i := range sources {
					if _, ok := sources[i].Source.(modeldb.CodexSource); ok {
						sources[i].Source = modeldb.NewCodexSourceFromFile(codexFile)
					}
				}
			}
			if openAIStaticFile != "" {
				for i := range sources {
					if _, ok := sources[i].Source.(modeldb.OpenAIStaticSource); ok {
						sources[i].Source = modeldb.NewOpenAIStaticSourceFromFile(openAIStaticFile)
					}
				}
			}
			builder := modeldb.Builder{Sources: sources}
			built, err := builder.Build(context.Background())
			if err != nil {
				return fmt.Errorf("build catalog: %w", err)
			}
			if err := modeldb.SaveJSON(outPath, built); err != nil {
				return fmt.Errorf("save catalog: %w", err)
			}
			return nil
		},
	}
	cmd.SetOut(ioCfg.Out)
	cmd.SetErr(ioCfg.Err)
	cmd.Flags().StringVar(&outPath, "out", defaultPath, "output catalog JSON path")
	cmd.Flags().StringVar(&anthropicFile, "anthropic-file", "", "optional local Anthropic models payload path")
	cmd.Flags().StringVar(&modelsDevFile, "modelsdev-file", "", "optional local models.dev payload path")
	cmd.Flags().StringVar(&codexFile, "codex-file", "", "optional local codex models payload path")
	cmd.Flags().StringVar(&openAIStaticFile, "openai-static-file", "", "optional local OpenAI static manifest path")
	cmd.Flags().BoolVar(&useFixture, "modelsdev-fixture", false, "use bundled models.dev fixture instead of live fetch")
	return cmd
}
