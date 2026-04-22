package cli

import (
	"fmt"

	modeldb "github.com/codewandler/modeldb"
	"github.com/spf13/cobra"
)

// SkillCommandOptions holds IO configuration for the skill command.
type SkillCommandOptions struct {
	IO IO
}

// NewSkillCommand returns a cobra.Command that prints the embedded SKILL.md to
// stdout. The content is baked into the binary at build time via go:embed so
// the command works without any external files.
func NewSkillCommand(opts SkillCommandOptions) *cobra.Command {
	ioCfg := opts.IO.withDefaults()
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Print the modeldb CLI skill reference (embedded SKILL.md)",
		Long: `Print the modeldb CLI skill reference that is embedded in this binary.

The skill document describes all commands, flags, concepts, and common
workflows for end-users and agents consuming the modeldb CLI.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := modeldb.BuiltInSkillContent()
			if err != nil {
				return fmt.Errorf("read embedded skill: %w", err)
			}
			_, err = fmt.Fprint(ioCfg.Out, string(content))
			return err
		},
	}
	cmd.SetOut(ioCfg.Out)
	cmd.SetErr(ioCfg.Err)
	return cmd
}
