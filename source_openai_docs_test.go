package modeldb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIDocsSourceFetch(t *testing.T) {
	frag, err := NewOpenAIDocsSource().Fetch(context.Background())
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
	}
