package modeldb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindModels_FiltersByAPIType(t *testing.T) {
	c := NewCatalog()
	key := NormalizeKey(ModelKey{Creator: "anthropic", Family: "claude", Series: "sonnet", Version: "4.5"})
	c.Models[key] = ModelRecord{Key: key, Name: "Claude Sonnet 4.5", Aliases: []string{"sonnet"}}
	c.Services["anthropic"] = Service{ID: "anthropic", Name: "Anthropic"}
	c.Services["openrouter"] = Service{ID: "openrouter", Name: "OpenRouter"}
	c.Offerings[OfferingRef{ServiceID: "anthropic", WireModelID: "claude-sonnet-4-5"}] = Offering{ServiceID: "anthropic", WireModelID: "claude-sonnet-4-5", ModelKey: key, Exposures: []OfferingExposure{{APIType: APITypeAnthropicMessages}}}
	c.Offerings[OfferingRef{ServiceID: "openrouter", WireModelID: "anthropic/claude-sonnet-4.5"}] = Offering{ServiceID: "openrouter", WireModelID: "anthropic/claude-sonnet-4.5", ModelKey: key, Exposures: []OfferingExposure{{APIType: APITypeOpenAIChat}}}

	matches := c.FindModels(ModelSelector{Name: "sonnet", Version: "4.5", APIType: APITypeOpenAIChat})
	require.Len(t, matches, 1)
	require.Len(t, matches[0].Offerings, 1)
	assert.Equal(t, "openrouter", matches[0].Offerings[0].Service.ID)
}
