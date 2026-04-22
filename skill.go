package modeldb

import "embed"

// skillFS embeds the agent skill document so it can be printed by the CLI
// without any external files. The all: prefix is required because .agents is a
// hidden directory and would otherwise be silently excluded by the embed tool.
//
//go:embed all:.agents/skills/modeldb
var skillFS embed.FS

// BuiltInSkillContent returns the raw bytes of the embedded SKILL.md.
func BuiltInSkillContent() ([]byte, error) {
	return skillFS.ReadFile(".agents/skills/modeldb/SKILL.md")
}
