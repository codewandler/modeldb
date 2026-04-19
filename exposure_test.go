package modeldb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOfferingExposureHelpers(t *testing.T) {
	o := Offering{Exposures: []OfferingExposure{{APIType: APITypeOpenAIChat, SupportedParameters: []NormalizedParameter{ParamTools}, ParameterValues: map[string][]string{"reasoning_effort": {"low", "high"}}}}}
	require.True(t, o.HasExposure(APITypeOpenAIChat))
	exp := o.Exposure(APITypeOpenAIChat)
	require.NotNil(t, exp)
	assert.True(t, exp.SupportsParameter("tools"))
	assert.True(t, exp.SupportsParameterValue("reasoning_effort", "low"))
	assert.False(t, exp.SupportsParameterValue("reasoning_effort", "medium"))
}

func TestValidateCatalogRejectsDuplicateExposureAPIType(t *testing.T) {
	c := NewCatalog()
	key := NormalizeKey(ModelKey{Creator: "openai", Family: "gpt", Version: "5"})
	c.Models[key] = ModelRecord{Key: key}
	c.Services["openai"] = Service{ID: "openai"}
	c.Offerings[OfferingRef{ServiceID: "openai", WireModelID: "gpt-5"}] = Offering{
		ServiceID:   "openai",
		WireModelID: "gpt-5",
		ModelKey:    key,
		Exposures: []OfferingExposure{{APIType: APITypeDefault}, {APIType: APITypeDefault}},
	}
	assert.Error(t, ValidateCatalog(c))
}
