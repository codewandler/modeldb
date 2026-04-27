package modeldb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodexStaticSourceFetch(t *testing.T) {
	frag, err := NewCodexSourceFromFile(DefaultCodexFixturePath()).Fetch(context.Background())
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

func TestCodexSourceFetchesLiveModels(t *testing.T) {
	var sawAuth bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "0.124.0", r.URL.Query().Get("client_version"))
		sawAuth = r.Header.Get("Authorization") == "Bearer test-token"
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{
			"slug":"gpt-5.5",
			"display_name":"GPT-5.5",
			"description":"Latest frontier agentic coding model.",
			"default_reasoning_level":"medium",
			"supported_reasoning_levels":[{"effort":"low"},{"effort":"medium"},{"effort":"high"},{"effort":"xhigh"}],
			"supported_in_api":true,
			"support_verbosity":true,
			"supports_reasoning_summaries":true,
			"context_window":272000,
			"input_modalities":["text","image"],
			"supports_parallel_tool_calls":true
		}]}`))
	}))
	defer server.Close()

	source := CodexSource{
		AccessToken:   "test-token",
		ModelsURL:     server.URL,
		ClientVersion: "0.124.0",
		Client:        server.Client(),
	}
	frag, err := source.Fetch(context.Background())
	require.NoError(t, err)
	require.True(t, sawAuth)
	offering := frag.Offerings[0]
	assert.Equal(t, "gpt-5.5", offering.WireModelID)
	assert.Equal(t, "codex", offering.ServiceID)
}

func TestCodexPricingHydratesFromOpenAIReferencePricing(t *testing.T) {
	c := NewCatalog()
	frag, err := NewCodexSourceFromFile(DefaultCodexFixturePath()).Fetch(context.Background())
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
