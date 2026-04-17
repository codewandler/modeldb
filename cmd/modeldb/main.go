package main

import (
	"context"
	"fmt"
	"os"

	modeldbcli "github.com/codewandler/modeldb/cli"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cmd := modeldbcli.NewRootCommand(modeldbcli.RootCommandOptions{})
	return cmd.ExecuteContext(ctx)
}
