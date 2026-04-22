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
	var openRouterFile string
	var useFixture bool
	var failOnUnknownPricing bool
	var excludeUnknownPricing bool

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
			if openRouterFile != "" {
				replaced := false
				for i := range sources {
					if _, ok := sources[i].Source.(modeldb.OpenRouterSource); ok {
						sources[i].Source = modeldb.NewOpenRouterSourceFromFile(openRouterFile)
						replaced = true
					}
				}
				if !replaced {
					sources = append(sources, modeldb.RegisteredSource{Stage: modeldb.StageBuild, Authority: modeldb.AuthorityTrusted, Source: modeldb.NewOpenRouterSourceFromFile(openRouterFile)})
				}
			}
			builder := modeldb.Builder{Sources: sources}
			built, err := builder.Build(context.Background())
			if err != nil {
				return fmt.Errorf("build catalog: %w", err)
			}
			report := modeldb.AuditPricing(built)
			if len(report.Unknown) > 0 {
				fmt.Fprintf(ioCfg.Err, "WARN pricing unknown for %d offerings\n", len(report.Unknown))
				for _, id := range report.Unknown {
					fmt.Fprintf(ioCfg.Err, "WARN missing pricing: %s\n", id)
				}
			}
			if failOnUnknownPricing && len(report.Unknown) > 0 {
				return fmt.Errorf("build catalog: unknown pricing for %d offerings", len(report.Unknown))
			}
			if excludeUnknownPricing {
				built = modeldb.FilterCatalogByPricingStatus(built, "unknown")
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
	cmd.Flags().StringVar(&openRouterFile, "openrouter-file", "", "optional local OpenRouter models payload path")
	cmd.Flags().BoolVar(&useFixture, "modelsdev-fixture", false, "use bundled models.dev fixture instead of live fetch")
	cmd.Flags().BoolVar(&failOnUnknownPricing, "fail-on-unknown-pricing", false, "fail build when any offering has unknown pricing")
	cmd.Flags().BoolVar(&excludeUnknownPricing, "exclude-unknown-pricing", false, "exclude offerings with unknown pricing from output")
	return cmd
}
