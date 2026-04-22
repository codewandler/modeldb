package modeldb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type cachingGolden struct {
	ServiceID           string                `json:"service_id"`
	WireModelID         string                `json:"wire_model_id"`
	APIType             APIType               `json:"api_type"`
	ModelCaching        *CachingCapability    `json:"model_caching,omitempty"`
	ExposureCaching     *CachingCapability    `json:"exposure_caching,omitempty"`
	SupportedParameters []NormalizedParameter `json:"supported_parameters,omitempty"`
	ParameterMappings   []ParameterMapping    `json:"parameter_mappings,omitempty"`
	ParameterValues     map[string][]string   `json:"parameter_values,omitempty"`
}

func TestCachingGoldenSnapshots(t *testing.T) {
	openAIFrag, err := NewOpenAIStaticSource().Fetch(context.Background())
	require.NoError(t, err)
	codexFrag, err := NewCodexSource().Fetch(context.Background())
	require.NoError(t, err)
	anthropicFrag, err := NewAnthropicAPISourceFromFile(DefaultAnthropicFixturePath()).Fetch(context.Background())
	require.NoError(t, err)
	minimaxFrag, err := NewMiniMaxStaticSource().Fetch(context.Background())
	require.NoError(t, err)
	kimiFrag, err := NewKimiStaticSource().Fetch(context.Background())
	require.NoError(t, err)
	openrouterFrag, err := openRouterGoldenFragment(t)
	require.NoError(t, err)

	c := NewCatalog()
	for _, frag := range []*Fragment{openAIFrag, codexFrag, anthropicFrag, minimaxFrag, kimiFrag, openrouterFrag} {
		require.NoError(t, MergeCatalogFragment(&c, frag))
	}
	require.NoError(t, ValidateCatalog(c))

	cases := []struct {
		name string
		path string
		data cachingGolden
	}{
		{name: "openai", path: filepath.Join("testdata", "caching", "openai_gpt_5_2.json"), data: goldenFromCatalog(c, "openai", "gpt-5.2", APITypeOpenAIResponses)},
		{name: "codex", path: filepath.Join("testdata", "caching", "codex_gpt_5_4.json"), data: goldenFromCatalog(c, "codex", "gpt-5.4", APITypeOpenAIResponses)},
		{name: "anthropic", path: filepath.Join("testdata", "caching", "anthropic_claude_sonnet_4_6.json"), data: goldenFromCatalog(c, "anthropic", "claude-sonnet-4-6", APITypeAnthropicMessages)},
		{name: "minimax", path: filepath.Join("testdata", "caching", "minimax_m2_7.json"), data: goldenFromCatalog(c, "minimax", "MiniMax-M2.7", APITypeAnthropicMessages)},
		{name: "kimi", path: filepath.Join("testdata", "caching", "kimi_k2_6.json"), data: goldenFromCatalog(c, "kimi", "kimi-k2.6", APITypeOpenAIMessages)},
		{name: "openrouter", path: filepath.Join("testdata", "caching", "openrouter_claude_sonnet_4_6.json"), data: goldenFromCatalog(c, "openrouter", "anthropic/claude-sonnet-4-6", APITypeOpenAIResponses)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.MarshalIndent(tc.data, "", "  ")
			require.NoError(t, err)
			expected, err := os.ReadFile(tc.path)
			require.NoError(t, err)
			require.JSONEq(t, string(expected), string(b))
		})
	}
}

func goldenFromCatalog(c Catalog, serviceID, wireModelID string, apiType APIType) cachingGolden {
	off := c.Offerings[OfferingRef{ServiceID: serviceID, WireModelID: wireModelID}]
	exp := off.Exposure(apiType)
	model := c.Models[off.ModelKey]
	return cachingGolden{
		ServiceID:           serviceID,
		WireModelID:         wireModelID,
		APIType:             apiType,
		ModelCaching:        model.Capabilities.Caching,
		ExposureCaching:     exp.ExposedCapabilities.Caching,
		SupportedParameters: exp.SupportedParameters,
		ParameterMappings:   exp.ParameterMappings,
		ParameterValues:     exp.ParameterValues,
	}
}

func openRouterGoldenFragment(t *testing.T) (*Fragment, error) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"anthropic/claude-sonnet-4-6","canonical_slug":"claude-sonnet-4-6","name":"Claude Sonnet 4.6","architecture":{"modality":"text->text","input_modalities":["text","image"],"output_modalities":["text"],"instruct_type":"claude","tokenizer":"Claude"},"pricing":{"prompt":"0.000003","completion":"0.000015","input_cache_read":"0.0000003"},"top_provider":{"context_length":200000,"max_completion_tokens":32000,"is_moderated":true},"supported_parameters":["tools","tool_choice","temperature","response_format","reasoning","logprobs"],"description":"Claude Sonnet 4.6 model"}]}`))
	}))
	defer server.Close()
	source := NewOpenRouterSource("test-key")
	source.BaseURL = server.URL
	source.Client = server.Client()
	return source.Fetch(context.Background())
}
