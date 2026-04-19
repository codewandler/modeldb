package modeldb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExposureByRef(t *testing.T) {
	c := NewCatalog()
	key := NormalizeKey(ModelKey{Creator: "openai", Family: "gpt", Version: "5"})
	c.Models[key] = ModelRecord{Key: key}
	c.Services["openai"] = Service{ID: "openai"}
	c.Offerings[OfferingRef{ServiceID: "openai", WireModelID: "gpt-5"}] = Offering{ServiceID: "openai", WireModelID: "gpt-5", ModelKey: key, Exposures: []OfferingExposure{{APIType: APITypeOpenAIResponses}}}
	offering, exposure, ok := c.ExposureByRef(ExposureRef{ServiceID: "openai", WireModelID: "gpt-5", APIType: APITypeOpenAIResponses})
	require.True(t, ok)
	assert.Equal(t, "gpt-5", offering.WireModelID)
	assert.Equal(t, APITypeOpenAIResponses, exposure.APIType)
}
