package modeldb

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoadJSONRoundTrip(t *testing.T) {
	timepoint := time.Unix(123, 0).UTC()
	key := NormalizeKey(ModelKey{Creator: "anthropic", Family: "claude", Series: "sonnet", Version: "4.6"})
	c := NewCatalog()
	c.Services["anthropic"] = Service{ID: "anthropic", Name: "Anthropic", Provenance: []Provenance{{SourceID: "test", ObservedAt: timepoint}}}
	c.Models[key] = ModelRecord{Key: key, Name: "Claude Sonnet 4.6", Canonical: true, Provenance: []Provenance{{SourceID: "test", ObservedAt: timepoint}}}
	c.Offerings[OfferingRef{ServiceID: "anthropic", WireModelID: "claude-sonnet-4-6"}] = Offering{
		ServiceID:   "anthropic",
		WireModelID: "claude-sonnet-4-6",
		ModelKey:    key,
		Provenance:  []Provenance{{SourceID: "test", ObservedAt: timepoint}},
	}

	path := filepath.Join(t.TempDir(), "catalog.json")
	require.NoError(t, SaveJSON(path, c))

	loaded, err := LoadJSON(path)
	require.NoError(t, err)
	expected := c
	expected.Services["anthropic"] = Service{ID: "anthropic", Name: "Anthropic", Provenance: []Provenance{{SourceID: "test"}}}
	expected.Models[key] = ModelRecord{Key: key, Name: "Claude Sonnet 4.6", Canonical: true, Provenance: []Provenance{{SourceID: "test"}}}
	expected.Offerings[OfferingRef{ServiceID: "anthropic", WireModelID: "claude-sonnet-4-6"}] = Offering{
		ServiceID:   "anthropic",
		WireModelID: "claude-sonnet-4-6",
		ModelKey:    key,
		Provenance:  []Provenance{{SourceID: "test"}},
	}
	assert.Equal(t, expected, loaded)
	assert.NoError(t, ValidateCatalog(loaded))
}


func TestFilterCatalogByPricingStatus(t *testing.T) {
	c := NewCatalog()
	key := NormalizeKey(ModelKey{Creator: "openai", Family: "gpt", Version: "5"})
	c.Services["openai"] = Service{ID: "openai"}
	c.Models[key] = ModelRecord{Key: key}
	c.Offerings[OfferingRef{ServiceID: "openai", WireModelID: "known"}] = Offering{ServiceID: "openai", WireModelID: "known", ModelKey: key, Pricing: &Pricing{Input: 1, CacheWrite: 0}, PricingStatus: "known"}
	c.Offerings[OfferingRef{ServiceID: "openai", WireModelID: "free"}] = Offering{ServiceID: "openai", WireModelID: "free", ModelKey: key, Pricing: &Pricing{CacheWrite: 0}, PricingStatus: "free"}
	c.Offerings[OfferingRef{ServiceID: "openai", WireModelID: "unknown"}] = Offering{ServiceID: "openai", WireModelID: "unknown", ModelKey: key, PricingStatus: "unknown"}
	filtered := FilterCatalogByPricingStatus(c, "unknown")
	assert.Len(t, filtered.Offerings, 2)
	_, ok := filtered.Offerings[OfferingRef{ServiceID: "openai", WireModelID: "unknown"}]
	assert.False(t, ok)
}
