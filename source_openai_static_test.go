package modeldb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIStaticSourceFetch(t *testing.T) {
	frag, err := NewOpenAIStaticSource().Fetch(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, frag.Models)
	c := NewCatalog()
	require.NoError(t, MergeCatalogFragment(&c, frag))
	require.NoError(t, ValidateCatalog(c))
	key := NormalizeKey(ModelKey{Creator: "openai", Family: "gpt", Version: "5.2"})
	m, ok := c.Models[key]
	require.True(t, ok)
	assert.True(t, m.Capabilities.StructuredOutput)
	assert.True(t, m.Capabilities.Vision)
	if assert.NotNil(t, m.Capabilities.Reasoning) {
		assert.Contains(t, m.Capabilities.Reasoning.Efforts, ReasoningEffortHigh)
	}
	assert.Equal(t, 400000, m.Limits.ContextWindow)
	assert.Equal(t, 128000, m.Limits.MaxOutput)

	off, ok := c.Offerings[OfferingRef{ServiceID: "openai", WireModelID: "gpt-5.2"}]
	require.True(t, ok)
	exp := off.Exposure(APITypeOpenAIResponses)
	require.NotNil(t, exp)
	require.NotNil(t, exp.ExposedCapabilities)
	assert.True(t, exp.ExposedCapabilities.StructuredOutput)
	assert.True(t, exp.ExposedCapabilities.ToolUse)
	if assert.NotNil(t, exp.ExposedCapabilities.Reasoning) {
		assert.Contains(t, exp.ExposedCapabilities.Reasoning.Efforts, ReasoningEffortHigh)
	}
	assert.Contains(t, exp.SupportedParameters, ParamTools)
	assert.Contains(t, exp.SupportedParameters, ParamResponseFormat)
	assert.Contains(t, exp.SupportedParameters, ParamThinking)
	assert.Contains(t, exp.SupportedParameters, ParamReasoningEffort)
	assert.True(t, exp.SupportsParameterValue(string(ParamReasoningEffort), string(ReasoningEffortLow)))
	assert.True(t, exp.SupportsParameterValue(string(ParamReasoningEffort), string(ReasoningEffortNone)))
}
