package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version will be set at build time using -ldflags
	Version = "dev"
)

type VersionCommandOptions struct {
	IO IO
}

func NewVersionCommand(opts VersionCommandOptions) *cobra.Command {
	ioCfg := opts.IO.withDefaults()

	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(ioCfg.Out, "modeldb version %s\n", Version)
			return nil
		},
	}
}
