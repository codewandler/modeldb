package modeldb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiniMaxStaticSourceCaching(t *testing.T) {
	frag, err := NewMiniMaxStaticSource().Fetch(context.Background())
	require.NoError(t, err)
	c := NewCatalog()
	require.NoError(t, MergeCatalogFragment(&c, frag))
	require.NoError(t, ValidateCatalog(c))

	key := NormalizeKey(ModelKey{Creator: "minimax", Family: "m2", Series: "standard", Version: "2.7"})
	model, ok := c.Models[key]
	require.True(t, ok)
	if assert.NotNil(t, model.Capabilities.Caching) {
		assert.True(t, model.Capabilities.Caching.Available)
		assert.Empty(t, model.Capabilities.Caching.Mode)
		assert.False(t, model.Capabilities.Caching.Configurable)
	}

	offering, exp, ok := c.ResolveOfferingExposure("minimax", "MiniMax-M2.7", APITypeAnthropicMessages)
	require.True(t, ok)
	assert.Equal(t, key, offering.ModelKey)
	if assert.NotNil(t, exp.ExposedCapabilities) && assert.NotNil(t, exp.ExposedCapabilities.Caching) {
		assert.True(t, exp.ExposedCapabilities.Caching.Available)
		assert.Equal(t, CachingModeExplicit, exp.ExposedCapabilities.Caching.Mode)
		assert.True(t, exp.ExposedCapabilities.Caching.Configurable)
		assert.True(t, exp.ExposedCapabilities.Caching.TopLevelRequestCaching)
		assert.False(t, exp.ExposedCapabilities.Caching.PerMessageCaching)
	}
}
