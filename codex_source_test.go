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
	model := c.Models[offeringModelKey(c, "codex", "gpt-5.4")]
	if assert.NotNil(t, model.Capabilities.Caching) {
		assert.True(t, model.Capabilities.Caching.Available)
		assert.Empty(t, model.Capabilities.Caching.Mode)
		assert.False(t, model.Capabilities.Caching.Configurable)
	}
	offering, exposure, ok := c.ResolveOfferingExposure("codex", "gpt-5.4", APITypeOpenAIResponses)
	require.True(t, ok)
	assert.Equal(t, "codex", offering.ServiceID)
	assert.Contains(t, exposure.SupportedParameters, ParamReasoningEffort)
	if assert.NotNil(t, exposure.ExposedCapabilities.Reasoning) {
		assert.Contains(t, exposure.ExposedCapabilities.Reasoning.Efforts, ReasoningEffortNone)
		assert.Contains(t, exposure.ExposedCapabilities.Reasoning.Summaries, ReasoningSummaryAuto)
		assert.True(t, exposure.ExposedCapabilities.Reasoning.VisibleSummary)
	}
	if assert.NotNil(t, exposure.ExposedCapabilities.Caching) {
		assert.True(t, exposure.ExposedCapabilities.Caching.Available)
		assert.Equal(t, CachingModeImplicit, exposure.ExposedCapabilities.Caching.Mode)
		assert.False(t, exposure.ExposedCapabilities.Caching.Configurable)
	}
	assert.NotContains(t, exposure.SupportedParameters, ParamPromptCacheRetention)
	assert.NotContains(t, exposure.SupportedParameters, ParamPromptCacheKey)
}

func TestCodexPricingHydratesFromOpenAIReferencePricing(t *testing.T) {
	c := NewCatalog()
	frag, err := NewCodexSource().Fetch(context.Background())
	require.NoError(t, err)
	require.NoError(t, MergeCatalogFragment(&c, frag))
	staticFrag, err := NewOpenAIStaticSource().Fetch(context.Background())
	require.NoError(t, err)
	require.NoError(t, MergeCatalogFragment(&c, staticFrag))
	require.NoError(t, ValidateCatalog(c))
	offering, _, ok := c.ResolveOfferingExposure("codex", "gpt-5.4", APITypeOpenAIResponses)
	require.True(t, ok)
	if assert.NotNil(t, offering.Pricing) {
		assert.Equal(t, 2.5, offering.Pricing.Input)
		assert.Equal(t, 0.25, offering.Pricing.CachedInput)
		assert.Equal(t, 15.0, offering.Pricing.Output)
		assert.Equal(t, 0.0, offering.Pricing.CacheWrite)
	}
}

func offeringModelKey(c Catalog, serviceID, wireModelID string) ModelKey {
	return c.Offerings[OfferingRef{ServiceID: serviceID, WireModelID: wireModelID}].ModelKey
}
