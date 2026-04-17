package cli

import (
	"fmt"

	modeldb "github.com/codewandler/modeldb"
	"github.com/spf13/cobra"
)

type ValidateCommandOptions struct {
	IO                 IO
	DefaultCatalogPath string
}

func NewValidateCommand(opts ValidateCommandOptions) *cobra.Command {
	ioCfg := opts.IO.withDefaults()
	defaultPath := opts.DefaultCatalogPath
	if defaultPath == "" {
		defaultPath = "catalog.json"
	}

	var inPath string
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a catalog snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			catalog, err := modeldb.LoadJSON(inPath)
			if err != nil {
				return fmt.Errorf("load catalog: %w", err)
			}
			if err := modeldb.ValidateCatalog(catalog); err != nil {
				return fmt.Errorf("validate catalog: %w", err)
			}
			return nil
		},
	}
	cmd.SetOut(ioCfg.Out)
	cmd.SetErr(ioCfg.Err)
	cmd.Flags().StringVar(&inPath, "in", defaultPath, "catalog JSON path")
	return cmd
}
