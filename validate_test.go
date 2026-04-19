package modeldb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCatalogMissingService(t *testing.T) {
	key := NormalizeKey(ModelKey{Creator: "anthropic", Family: "claude", Series: "sonnet", Version: "4.6"})
	c := NewCatalog()
	c.Models[key] = ModelRecord{Key: key}
	c.Offerings[OfferingRef{ServiceID: "anthropic", WireModelID: "claude-sonnet-4-6"}] = Offering{
		ServiceID:   "anthropic",
		WireModelID: "claude-sonnet-4-6",
		ModelKey:    key,
	}

	err := ValidateCatalog(c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown service")
}

func TestValidateCatalogMissingModel(t *testing.T) {
	c := NewCatalog()
	c.Services["anthropic"] = Service{ID: "anthropic"}
	c.Offerings[OfferingRef{ServiceID: "anthropic", WireModelID: "claude-sonnet-4-6"}] = Offering{
		ServiceID:   "anthropic",
		WireModelID: "claude-sonnet-4-6",
		ModelKey:    NormalizeKey(ModelKey{Creator: "anthropic", Family: "claude", Series: "sonnet", Version: "4.6"}),
	}

	err := ValidateCatalog(c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown model")
}

func TestValidateResolvedCatalogMissingRuntimeOffering(t *testing.T) {
	key := NormalizeKey(ModelKey{Creator: "anthropic", Family: "claude", Series: "sonnet", Version: "4.6"})
	base := NewCatalog()
	base.Services["anthropic"] = Service{ID: "anthropic"}
	base.Models[key] = ModelRecord{Key: key}
	resolved := NewResolvedCatalog(base)
	resolved.Runtimes["local"] = Runtime{ID: "local", ServiceID: "anthropic"}
	resolved.RuntimeAccess[RuntimeAccessKey{RuntimeID: "local", ServiceID: "anthropic", WireModelID: "claude-sonnet-4-6"}] = RuntimeAccess{
		RuntimeID: "local",
		Offering:  OfferingRef{ServiceID: "anthropic", WireModelID: "claude-sonnet-4-6"},
		Routable:  true,
	}

	err := ValidateResolvedCatalog(resolved)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown offering")
}


func TestValidateCatalogRejectsFreeWithoutExplicitPricing(t *testing.T) {
	c := NewCatalog()
	key := NormalizeKey(ModelKey{Creator: "openai", Family: "gpt", Version: "5"})
	c.Models[key] = ModelRecord{Key: key}
	c.Services["openai"] = Service{ID: "openai"}
	c.Offerings[OfferingRef{ServiceID: "openai", WireModelID: "gpt-5"}] = Offering{ServiceID: "openai", WireModelID: "gpt-5", ModelKey: key, PricingStatus: "free"}
	err := ValidateCatalog(c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marked free without explicit zero pricing")
}

func TestAuditPricingClassifiesStatuses(t *testing.T) {
	c := NewCatalog()
	key := NormalizeKey(ModelKey{Creator: "openai", Family: "gpt", Version: "5"})
	c.Models[key] = ModelRecord{Key: key}
	c.Services["openai"] = Service{ID: "openai"}
	c.Offerings[OfferingRef{ServiceID: "openai", WireModelID: "known"}] = Offering{ServiceID: "openai", WireModelID: "known", ModelKey: key, Pricing: &Pricing{Input: 1, CacheWrite: 0}, PricingStatus: "known"}
	c.Offerings[OfferingRef{ServiceID: "openai", WireModelID: "free"}] = Offering{ServiceID: "openai", WireModelID: "free", ModelKey: key, Pricing: &Pricing{CacheWrite: 0}, PricingStatus: "free"}
	c.Offerings[OfferingRef{ServiceID: "openai", WireModelID: "unknown"}] = Offering{ServiceID: "openai", WireModelID: "unknown", ModelKey: key, PricingStatus: "unknown"}
	report := AuditPricing(c)
	assert.Len(t, report.Known, 1)
	assert.Len(t, report.Free, 1)
	assert.Len(t, report.Unknown, 1)
}


func TestAuditPricingIgnoresNonTextOfferings(t *testing.T) {
	c := NewCatalog()
	textKey := NormalizeKey(ModelKey{Creator: "openai", Family: "gpt", Version: "5"})
	audioKey := NormalizeKey(ModelKey{Creator: "openai", Family: "gpt", Version: "audio", Variant: "1.5"})
	c.Services["openai"] = Service{ID: "openai"}
	c.Models[textKey] = ModelRecord{Key: textKey, InputModalities: []string{"text"}, OutputModalities: []string{"text"}}
	c.Models[audioKey] = ModelRecord{Key: audioKey, InputModalities: []string{"audio"}, OutputModalities: []string{"text"}}
	c.Offerings[OfferingRef{ServiceID: "openai", WireModelID: "gpt-5"}] = Offering{ServiceID: "openai", WireModelID: "gpt-5", ModelKey: textKey, PricingStatus: "unknown"}
	c.Offerings[OfferingRef{ServiceID: "openai", WireModelID: "gpt-audio-1.5"}] = Offering{ServiceID: "openai", WireModelID: "gpt-audio-1.5", ModelKey: audioKey, PricingStatus: "unknown"}
	report := AuditPricing(c)
	assert.Equal(t, []string{"openai/gpt-5"}, report.Unknown)
}
