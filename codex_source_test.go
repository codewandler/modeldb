package modeldb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodexStaticSourceFetch(t *testing.T) {
	frag, err := NewCodexSource().Fetch(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, frag.Offerings)
	c := NewCatalog()
	require.NoError(t, MergeCatalogFragment(&c, frag))
	require.NoError(t, ValidateCatalog(c))
	_, ok := c.Services["codex"]
	require.True(t, ok)
	offering, exposure, ok := c.ResolveOfferingExposure("codex", "gpt-5.4", APITypeOpenAIResponses)
	require.True(t, ok)
	assert.Equal(t, "codex", offering.ServiceID)
	assert.Contains(t, exposure.SupportedParameters, ParamReasoningEffort)
	if assert.NotNil(t, exposure.ExposedCapabilities.Reasoning) {
		assert.Contains(t, exposure.ExposedCapabilities.Reasoning.Efforts, ReasoningEffortNone)
		assert.Contains(t, exposure.ExposedCapabilities.Reasoning.Summaries, ReasoningSummaryAuto)
		assert.True(t, exposure.ExposedCapabilities.Reasoning.VisibleSummary)
	}
}
