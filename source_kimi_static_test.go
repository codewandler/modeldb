package modeldb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKimiStaticSourceFetch(t *testing.T) {
	frag, err := NewKimiStaticSource().Fetch(context.Background())
	require.NoError(t, err)
	require.NotNil(t, frag)

	// Verify service is present
	require.Len(t, frag.Services, 1)
	assert.Equal(t, "kimi", frag.Services[0].ID)
	assert.Equal(t, "moonshot", frag.Services[0].Operator)

	// Verify K2.6 model record
	k26Key := NormalizeKey(ModelKey{Creator: "moonshot", Family: "kimi", Version: "2.6"})
	var found bool
	for _, m := range frag.Models {
		if m.Key == k26Key {
			found = true
			assert.Equal(t, "Kimi K2.6", m.Name)
			assert.True(t, m.Capabilities.Vision)
			assert.True(t, m.Capabilities.ToolUse)
			assert.True(t, m.Capabilities.Streaming)
			require.NotNil(t, m.Capabilities.Reasoning)
			assert.True(t, m.Capabilities.Reasoning.Available)
			require.NotNil(t, m.Capabilities.Caching)
			assert.True(t, m.Capabilities.Caching.Available)
			assert.Equal(t, CachingModeImplicit, m.Capabilities.Caching.Mode)
			assert.Equal(t, 262144, m.Limits.ContextWindow)
			assert.Contains(t, m.InputModalities, "image")
			assert.Contains(t, m.InputModalities, "video")
			require.NotNil(t, m.ReferencePricing)
			assert.Equal(t, 0.95, m.ReferencePricing.Input)
			assert.Equal(t, 4.00, m.ReferencePricing.Output)
			assert.Equal(t, 0.16, m.ReferencePricing.CachedInput)
		}
	}
	assert.True(t, found, "K2.6 model record not found")

	// Verify K2.6 offering
	var offeringFound bool
	for _, o := range frag.Offerings {
		if o.WireModelID == "kimi-k2.6" {
			offeringFound = true
			assert.Equal(t, "kimi", o.ServiceID)
			assert.Equal(t, k26Key, o.ModelKey)
		}
	}
	assert.True(t, offeringFound, "kimi-k2.6 offering not found")
}

func TestKimiStaticSourceCatalogMerge(t *testing.T) {
	frag, err := NewKimiStaticSource().Fetch(context.Background())
	require.NoError(t, err)
	c := NewCatalog()
	require.NoError(t, MergeCatalogFragment(&c, frag))
	require.NoError(t, ValidateCatalog(c))

	k26Key := NormalizeKey(ModelKey{Creator: "moonshot", Family: "kimi", Version: "2.6"})
	model, ok := c.Models[k26Key]
	require.True(t, ok, "K2.6 model not in catalog")
	assert.Equal(t, "Kimi K2.6", model.Name)

	offering, ok := c.Offerings[OfferingRef{ServiceID: "kimi", WireModelID: "kimi-k2.6"}]
	require.True(t, ok, "kimi-k2.6 offering not in catalog")
	assert.Equal(t, k26Key, offering.ModelKey)

	exp := offering.Exposure(APITypeOpenAIMessages)
	require.NotNil(t, exp.ExposedCapabilities)
	assert.True(t, exp.ExposedCapabilities.Vision)
	assert.True(t, exp.ExposedCapabilities.ToolUse)
}
