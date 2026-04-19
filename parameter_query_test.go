package modeldb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOfferingsByServiceAPIAndParameter(t *testing.T) {
	c := NewCatalog()
	key := NormalizeKey(ModelKey{Creator: "openrouter", Family: "gpt", Version: "5"})
	c.Models[key] = ModelRecord{Key: key}
	c.Services["openrouter"] = Service{ID: "openrouter"}
	c.Offerings[OfferingRef{ServiceID: "openrouter", WireModelID: "openai/gpt-5"}] = Offering{ServiceID: "openrouter", WireModelID: "openai/gpt-5", ModelKey: key, Exposures: []OfferingExposure{{APIType: APITypeOpenAIResponses, SupportedParameters: []NormalizedParameter{ParamReasoningEffort}}}}
	items := c.OfferingsByServiceAPIAndParameter("openrouter", APITypeOpenAIResponses, ParamReasoningEffort)
	require.Len(t, items, 1)
	assert.Equal(t, "openai/gpt-5", items[0].WireModelID)
}
