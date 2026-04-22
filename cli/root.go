package cli

import (
	"context"

	modeldb "github.com/codewandler/modeldb"
	"github.com/spf13/cobra"
)

type RootCommandOptions struct {
	IO                 IO
	DefaultCatalogPath string
	LoadBaseCatalog    func(ctx context.Context) (modeldb.Catalog, error)
}

func NewRootCommand(opts RootCommandOptions) *cobra.Command {
	ioCfg := opts.IO.withDefaults()
	if opts.DefaultCatalogPath == "" {
		opts.DefaultCatalogPath = "catalog.json"
	}
	if opts.LoadBaseCatalog == nil {
		opts.LoadBaseCatalog = func(context.Context) (modeldb.Catalog, error) {
			return modeldb.LoadBuiltIn()
		}
	}

	root := &cobra.Command{
		Use:           "modeldb",
		Short:         "Query and build the model database",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(NewBuildCommand(BuildCommandOptions{IO: ioCfg, DefaultCatalogPath: opts.DefaultCatalogPath}))
	root.AddCommand(NewValidateCommand(ValidateCommandOptions{IO: ioCfg, DefaultCatalogPath: opts.DefaultCatalogPath}))
	root.AddCommand(NewModelsCommand(ModelsCommandOptions{IO: ioCfg, LoadBaseCatalog: opts.LoadBaseCatalog}))
	root.AddCommand(NewSkillCommand(SkillCommandOptions{IO: ioCfg}))
	root.InitDefaultCompletionCmd()
	return root
}
