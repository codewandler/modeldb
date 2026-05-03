package modeldb

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBedrockModelClient struct {
	models   *bedrock.ListFoundationModelsOutput
	profiles *bedrock.ListInferenceProfilesOutput
}

func marshalCatalogForTest(catalog Catalog) ([]byte, error) {
	return json.Marshal(stripArtifactVolatileFields(catalogArtifactFromCatalog(catalog)))
}

func (f fakeBedrockModelClient) ListFoundationModels(context.Context, *bedrock.ListFoundationModelsInput, ...func(*bedrock.Options)) (*bedrock.ListFoundationModelsOutput, error) {
	return f.models, nil
}

func (f fakeBedrockModelClient) ListInferenceProfiles(context.Context, *bedrock.ListInferenceProfilesInput, ...func(*bedrock.Options)) (*bedrock.ListInferenceProfilesOutput, error) {
	return f.profiles, nil
}

func TestBedrockRuntimeSourceFetchesConverseProfiles(t *testing.T) {
	source := BedrockRuntimeSource{
		Region: "us-east-1",
		Client: fakeBedrockModelClient{
			models: &bedrock.ListFoundationModelsOutput{ModelSummaries: []types.FoundationModelSummary{{
				ModelArn:                   aws.String("arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-sonnet-4-6"),
				ModelId:                    aws.String("anthropic.claude-sonnet-4-6"),
				ModelName:                  aws.String("Claude Sonnet 4.6"),
				InputModalities:            []types.ModelModality{types.ModelModalityText},
				OutputModalities:           []types.ModelModality{types.ModelModalityText},
				ResponseStreamingSupported: aws.Bool(true),
			}}},
			profiles: &bedrock.ListInferenceProfilesOutput{InferenceProfileSummaries: []types.InferenceProfileSummary{{
				InferenceProfileId:   aws.String("global.anthropic.claude-sonnet-4-6"),
				InferenceProfileName: aws.String("Global Claude Sonnet 4.6"),
				Status:               types.InferenceProfileStatusActive,
				Type:                 types.InferenceProfileTypeSystemDefined,
				Models: []types.InferenceProfileModel{{
					ModelArn: aws.String("arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-sonnet-4-6"),
				}},
			}, {
				InferenceProfileId:   aws.String("us.anthropic.claude-sonnet-4-6"),
				InferenceProfileName: aws.String("US Claude Sonnet 4.6"),
				Status:               types.InferenceProfileStatusActive,
				Type:                 types.InferenceProfileTypeSystemDefined,
				Models: []types.InferenceProfileModel{{
					ModelArn: aws.String("arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-sonnet-4-6"),
				}},
			}}},
		},
	}

	fragment, err := source.Fetch(context.Background())
	require.NoError(t, err)

	catalog := NewCatalog()
	require.NoError(t, MergeCatalogFragment(&catalog, fragment))
	require.NoError(t, ValidateCatalog(catalog))

	assert.Contains(t, catalog.Services, "bedrock")
	assert.Contains(t, catalog.Runtimes, "bedrock-us")
	ref := OfferingRef{ServiceID: "bedrock", WireModelID: "anthropic.claude-sonnet-4-6"}
	offering, ok := catalog.Offerings[ref]
	require.True(t, ok)
	assert.Equal(t, APITypeBedrockConverse, offering.Exposures[0].APIType)
	assert.True(t, offering.Exposures[0].ExposedCapabilities.ToolUse)
	assert.True(t, offering.Exposures[0].ExposedCapabilities.SupportsReasoning())

	access := catalog.RuntimeAccess[RuntimeAccessKey{RuntimeID: "bedrock-us", ServiceID: "bedrock", WireModelID: "anthropic.claude-sonnet-4-6"}]
	assert.True(t, access.Routable)
	assert.Equal(t, "us.anthropic.claude-sonnet-4-6", access.ResolvedWireID)
}

func TestBedrockRuntimeAccessRoundTripJSON(t *testing.T) {
	catalog := NewCatalog()
	require.NoError(t, MergeCatalogFragment(&catalog, &Fragment{
		Services: []Service{{ID: "bedrock"}},
		Models: []ModelRecord{{
			Key:       ModelKey{Creator: "anthropic", Family: "claude", Series: "sonnet", Version: "4.6"},
			Canonical: true,
		}},
		Offerings: []Offering{{
			ServiceID:   "bedrock",
			WireModelID: "anthropic.claude-sonnet-4-6",
			ModelKey:    ModelKey{Creator: "anthropic", Family: "claude", Series: "sonnet", Version: "4.6"},
			Exposures:   []OfferingExposure{{APIType: APITypeBedrockConverse}},
		}},
		Runtimes: []Runtime{{ID: "bedrock-us", ServiceID: "bedrock"}},
		RuntimeAccess: []RuntimeAccess{{
			RuntimeID:      "bedrock-us",
			Offering:       OfferingRef{ServiceID: "bedrock", WireModelID: "anthropic.claude-sonnet-4-6"},
			Routable:       true,
			ResolvedWireID: "us.anthropic.claude-sonnet-4-6",
		}},
	}))

	data, err := marshalCatalogForTest(catalog)
	require.NoError(t, err)
	loaded, err := LoadJSONBytes(data)
	require.NoError(t, err)

	key := RuntimeAccessKey{RuntimeID: "bedrock-us", ServiceID: "bedrock", WireModelID: "anthropic.claude-sonnet-4-6"}
	require.Contains(t, loaded.RuntimeAccess, key)
	assert.Equal(t, "us.anthropic.claude-sonnet-4-6", loaded.RuntimeAccess[key].ResolvedWireID)
}
