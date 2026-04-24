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
	key := NormalizeKey(ModelKey{Creator: "openai", Family: "gpt", Version: "5.5"})
	m, ok := c.Models[key]
	require.True(t, ok)
	assert.True(t, m.Capabilities.StructuredOutput)
	assert.True(t, m.Capabilities.Vision)
	if assert.NotNil(t, m.Capabilities.Reasoning) {
		assert.Contains(t, m.Capabilities.Reasoning.Efforts, ReasoningEffortHigh)
		assert.Contains(t, m.Capabilities.Reasoning.Efforts, ReasoningEffortNone)
		assert.Contains(t, m.Capabilities.Reasoning.Efforts, ReasoningEffortXHigh)
	}
	assert.Equal(t, 1050000, m.Limits.ContextWindow)
	assert.Equal(t, 128000, m.Limits.MaxOutput)
	if assert.NotNil(t, m.ReferencePricing) {
		assert.Equal(t, 5.0, m.ReferencePricing.Input)
		assert.Equal(t, 0.5, m.ReferencePricing.CachedInput)
		assert.Equal(t, 30.0, m.ReferencePricing.Output)
		assert.Equal(t, 0.0, m.ReferencePricing.CacheWrite)
	}

	off, ok := c.Offerings[OfferingRef{ServiceID: "openai", WireModelID: "gpt-5.5"}]
	require.True(t, ok)
	exp := off.Exposure(APITypeOpenAIResponses)
	require.NotNil(t, exp)
	require.NotNil(t, exp.ExposedCapabilities)
	if assert.NotNil(t, off.Pricing) {
		assert.Equal(t, 5.0, off.Pricing.Input)
		assert.Equal(t, 0.5, off.Pricing.CachedInput)
		assert.Equal(t, 30.0, off.Pricing.Output)
		assert.Equal(t, 0.0, off.Pricing.CacheWrite)
	}
	assert.True(t, exp.ExposedCapabilities.StructuredOutput)
	assert.True(t, exp.ExposedCapabilities.ToolUse)
	if assert.NotNil(t, exp.ExposedCapabilities.Reasoning) {
		assert.Contains(t, exp.ExposedCapabilities.Reasoning.Efforts, ReasoningEffortHigh)
		assert.Contains(t, exp.ExposedCapabilities.Reasoning.Efforts, ReasoningEffortNone)
		assert.Contains(t, exp.ExposedCapabilities.Reasoning.Efforts, ReasoningEffortXHigh)
	}
	assert.Contains(t, exp.SupportedParameters, ParamTools)
	assert.Contains(t, exp.SupportedParameters, ParamResponseFormat)
	assert.Contains(t, exp.SupportedParameters, ParamThinking)
	assert.Contains(t, exp.SupportedParameters, ParamReasoningEffort)
	assert.Contains(t, exp.SupportedParameters, ParamPromptCacheRetention)
	assert.Contains(t, exp.SupportedParameters, ParamPromptCacheKey)
	if assert.NotNil(t, exp.ExposedCapabilities.Caching) {
		assert.True(t, exp.ExposedCapabilities.Caching.Available)
		assert.Equal(t, CachingModeMixed, exp.ExposedCapabilities.Caching.Mode)
		assert.True(t, exp.ExposedCapabilities.Caching.Configurable)
		assert.True(t, exp.ExposedCapabilities.Caching.PromptCacheRetention)
		assert.True(t, exp.ExposedCapabilities.Caching.PromptCacheKey)
		assert.ElementsMatch(t, []string{"in_memory", "24h"}, exp.ExposedCapabilities.Caching.RetentionValues)
	}
	assert.True(t, exp.SupportsParameterValue(string(ParamReasoningEffort), string(ReasoningEffortLow)))
	assert.True(t, exp.SupportsParameterValue(string(ParamReasoningEffort), string(ReasoningEffortNone)))
	assert.True(t, exp.SupportsParameterValue(string(ParamReasoningEffort), string(ReasoningEffortXHigh)))
}

func TestOpenAIStaticPricingCoverage(t *testing.T) {
	frag, err := NewOpenAIStaticSource().Fetch(context.Background())
	require.NoError(t, err)
	c := NewCatalog()
	require.NoError(t, MergeCatalogFragment(&c, frag))
	require.NoError(t, ValidateCatalog(c))
	priced := []string{
		"gpt-4-turbo",
		"gpt-4.1-mini",
		"gpt-4.1-nano",
		"gpt-4.1",
		"gpt-4",
		"gpt-4o-mini",
		"gpt-4o",
		"gpt-5-chat-latest",
		"gpt-5-codex",
		"gpt-5-mini",
		"gpt-5-nano",
		"gpt-5-pro",
		"gpt-5.1-chat-latest",
		"gpt-5.1-codex-max",
		"gpt-5.1-codex-mini",
		"gpt-5.1-codex",
		"gpt-5.1",
		"gpt-5.2-chat-latest",
		"gpt-5.2-codex",
		"gpt-5.2-pro",
		"gpt-5.2",
		"gpt-5.3-chat-latest",
		"gpt-5.3-codex",
		"gpt-5.4-mini",
		"gpt-5.4-nano",
		"gpt-5.4-pro",
		"gpt-5.4",
		"gpt-5.5-pro",
		"gpt-5.5",
		"gpt-5",
	}
	for _, slug := range priced {
		key, ok := inferOpenAIModelKey(slug)
		require.True(t, ok, slug)
		model, ok := c.Models[NormalizeKey(key)]
		require.True(t, ok, slug)
		if assert.NotNil(t, model.ReferencePricing, slug) {
			assert.Equal(t, 0.0, model.ReferencePricing.CacheWrite, slug)
		}
		off, ok := c.Offerings[OfferingRef{ServiceID: "openai", WireModelID: slug}]
		require.True(t, ok, slug)
		if assert.NotNil(t, off.Pricing, slug) {
			assert.Equal(t, 0.0, off.Pricing.CacheWrite, slug)
		}
	}
}

func TestOpenAIStaticCoreOpenAIPricingExtensions(t *testing.T) {
	frag, err := NewOpenAIStaticSource().Fetch(context.Background())
	require.NoError(t, err)
	c := NewCatalog()
	require.NoError(t, MergeCatalogFragment(&c, frag))
	require.NoError(t, ValidateCatalog(c))
	for _, tc := range []struct {
		slug          string
		input, output float64
	}{
		{slug: "gpt-3.5-turbo", input: 0.5, output: 1.5},
		{slug: "gpt-3.5-turbo-instruct", input: 1.5, output: 2},
		{slug: "o1", input: 15, output: 60},
		{slug: "o1-pro", input: 150, output: 600},
		{slug: "o3", input: 2, output: 8},
		{slug: "o3-mini", input: 1.1, output: 4.4},
		{slug: "o3-pro", input: 20, output: 80},
		{slug: "o4-mini", input: 1.1, output: 4.4},
	} {
		off, ok := c.Offerings[OfferingRef{ServiceID: "openai", WireModelID: tc.slug}]
		require.True(t, ok, tc.slug)
		if assert.NotNil(t, off.Pricing, tc.slug) {
			assert.Equal(t, tc.input, off.Pricing.Input, tc.slug)
			assert.Equal(t, tc.output, off.Pricing.Output, tc.slug)
			assert.Equal(t, 0.0, off.Pricing.CacheWrite, tc.slug)
		}
	}
}

func TestOpenAIStaticLegacyResponsesModelsDoNotAssumePromptCaching(t *testing.T) {
	frag, err := NewOpenAIStaticSource().Fetch(context.Background())
	require.NoError(t, err)
	c := NewCatalog()
	require.NoError(t, MergeCatalogFragment(&c, frag))
	require.NoError(t, ValidateCatalog(c))
	off, ok := c.Offerings[OfferingRef{ServiceID: "openai", WireModelID: "gpt-4-turbo"}]
	require.True(t, ok)
	exp := off.Exposure(APITypeOpenAIResponses)
	require.NotNil(t, exp)
	require.NotNil(t, exp.ExposedCapabilities)
	assert.Nil(t, exp.ExposedCapabilities.Caching)
	assert.NotContains(t, exp.SupportedParameters, ParamPromptCacheRetention)
	assert.NotContains(t, exp.SupportedParameters, ParamPromptCacheKey)
}

func TestOpenAIStaticModelCachingIsCoarse(t *testing.T) {
	frag, err := NewOpenAIStaticSource().Fetch(context.Background())
	require.NoError(t, err)
	c := NewCatalog()
	require.NoError(t, MergeCatalogFragment(&c, frag))
	require.NoError(t, ValidateCatalog(c))
	key := NormalizeKey(ModelKey{Creator: "openai", Family: "gpt", Version: "5.5"})
	m, ok := c.Models[key]
	require.True(t, ok)
	if assert.NotNil(t, m.Capabilities.Caching) {
		assert.True(t, m.Capabilities.Caching.Available)
		assert.Empty(t, m.Capabilities.Caching.Mode)
		assert.False(t, m.Capabilities.Caching.Configurable)
		assert.False(t, m.Capabilities.Caching.PromptCacheRetention)
		assert.False(t, m.Capabilities.Caching.PromptCacheKey)
	}
}
