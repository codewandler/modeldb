package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillCommand_PrintsEmbeddedSkill(t *testing.T) {
	var out bytes.Buffer
	cmd := NewSkillCommand(SkillCommandOptions{IO: IO{Out: &out, Err: &out}})
	cmd.SetArgs(nil)

	require.NoError(t, cmd.Execute())

	got := out.String()
	// Must contain the top-level heading so we know the real file was embedded.
	assert.True(t, strings.HasPrefix(got, "# modeldb CLI Skill"), "output should start with the skill heading")
	// Spot-check a few stable sections.
	assert.Contains(t, got, "## Commands")
	assert.Contains(t, got, "modeldb models")
	assert.Contains(t, got, "modeldb build")
	assert.Contains(t, got, "modeldb validate")
	// Must not be empty.
	assert.Greater(t, len(got), 1000, "embedded skill should have substantial content")
}

func TestSkillCommand_WritesToConfiguredWriter(t *testing.T) {
	var out, errBuf bytes.Buffer
	cmd := NewSkillCommand(SkillCommandOptions{IO: IO{Out: &out, Err: &errBuf}})
	cmd.SetArgs(nil)

	require.NoError(t, cmd.Execute())

	assert.NotEmpty(t, out.String(), "output should be written to the configured Out writer")
	assert.Empty(t, errBuf.String(), "no output should go to Err on success")
}
