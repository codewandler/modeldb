package modeldb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

const defaultOpenRouterBaseURL = "https://openrouter.ai/api"

type OpenRouterSource struct {
	APIKey  string
	BaseURL string
	Client  *http.Client
}

func NewOpenRouterSource(apiKey string) OpenRouterSource {
	return OpenRouterSource{APIKey: apiKey, BaseURL: defaultOpenRouterBaseURL, Client: http.DefaultClient}
}

func NewOpenRouterSourceFromEnv() OpenRouterSource {
	return NewOpenRouterSource(os.Getenv("OPENROUTER_API_KEY"))
}

func (OpenRouterSource) ID() string { return "openrouter-api" }

func (s OpenRouterSource) Fetch(ctx context.Context) (*Fragment, error) {
	if s.APIKey == "" {
		return nil, fmt.Errorf("openrouter source: missing API key")
	}
	baseURL := s.BaseURL
	if baseURL == "" {
		baseURL = defaultOpenRouterBaseURL
	}
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openrouter source: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openrouter source: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Data []struct {
			ID            string `json:"id"`
			CanonicalSlug string `json:"canonical_slug"`
			Name          string `json:"name"`
			Created       int64  `json:"created,omitempty"`
			ContextLength int    `json:"context_length,omitempty"`
			Architecture  struct {
				Modality         string   `json:"modality"`
				InputModalities  []string `json:"input_modalities"`
				OutputModalities []string `json:"output_modalities"`
				InstructType     string   `json:"instruct_type"`
				Tokenizer        string   `json:"tokenizer"`
			} `json:"architecture"`
			Description     string `json:"description"`
			KnowledgeCutoff string `json:"knowledge_cutoff"`
			ExpirationDate  string `json:"expiration_date"`
			Pricing         struct {
				Prompt            string `json:"prompt"`
				Completion        string `json:"completion"`
				InputCacheRead    string `json:"input_cache_read"`
				InputCacheWrite   string `json:"input_cache_write"`
				InternalReasoning string `json:"internal_reasoning"`
				Image             string `json:"image"`
				ImageToken        string `json:"image_token"`
				ImageOutput       string `json:"image_output"`
				Audio             string `json:"audio"`
				AudioOutput       string `json:"audio_output"`
				Request           string `json:"request"`
				WebSearch         string `json:"web_search"`
			} `json:"pricing"`
			TopProvider struct {
				ContextLength       int  `json:"context_length"`
				MaxCompletionTokens int  `json:"max_completion_tokens"`
				IsModerated         bool `json:"is_moderated"`
			} `json:"top_provider"`
			SupportedParameters []string `json:"supported_parameters"`
			DefaultParameters   *struct {
				Temperature       *float64 `json:"temperature"`
				TopP              *float64 `json:"top_p"`
				TopK              *int     `json:"top_k"`
				FrequencyPenalty  *float64 `json:"frequency_penalty"`
				PresencePenalty   *float64 `json:"presence_penalty"`
				RepetitionPenalty *float64 `json:"repetition_penalty"`
			} `json:"default_parameters"`
			PerRequestLimits *struct {
				PromptTokens     float64 `json:"prompt_tokens"`
				CompletionTokens float64 `json:"completion_tokens"`
			} `json:"per_request_limits"`
			Links struct {
				Details string `json:"details"`
			} `json:"links"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	observedAt := time.Time{}
	fragment := &Fragment{Services: []Service{{
		ID:       "openrouter",
		Name:     "OpenRouter",
		Kind:     ServiceKindBroker,
		Operator: "openrouter",
		Provenance: []Provenance{{
			SourceID:   s.ID(),
			Authority:  string(AuthorityTrusted),
			ObservedAt: observedAt,
		}},
	}}}

	for _, item := range payload.Data {
		key, ok := inferOpenRouterModelKey(item.ID)
		if !ok {
			continue
		}
		fragment.Models = append(fragment.Models, ModelRecord{
			Key:              key,
			Name:             item.Name,
			Description:      item.Description,
			Canonical:        false,
			Capabilities:     capabilitiesFromOpenRouter(item.SupportedParameters, item.Architecture.InputModalities, item.Pricing.InputCacheRead != ""),
			InputModalities:  normalizeStrings(item.Architecture.InputModalities),
			OutputModalities: normalizeStrings(item.Architecture.OutputModalities),
			KnowledgeCutoff:  normalizeDate(item.KnowledgeCutoff),
			ExpirationDate:   normalizeDate(item.ExpirationDate),
			InstructType:     item.Architecture.InstructType,
			Tokenizer:        item.Architecture.Tokenizer,
			Modality:         item.Architecture.Modality,
			Provenance: []Provenance{{
				SourceID:   s.ID(),
				Authority:  string(AuthorityTrusted),
				ObservedAt: observedAt,
				RawID:      item.ID,
			}},
		})
		caps := capabilitiesFromOpenRouter(item.SupportedParameters, item.Architecture.InputModalities, item.Pricing.InputCacheRead != "")
		pricing, pricingStatus := pricingFromOpenRouter(
			item.Pricing.Prompt,
			item.Pricing.Completion,
			item.Pricing.InputCacheRead,
			item.Pricing.InputCacheWrite,
			item.Pricing.InternalReasoning,
			item.Pricing.Image,
			item.Pricing.ImageToken,
			item.Pricing.ImageOutput,
			item.Pricing.Audio,
			item.Pricing.AudioOutput,
			item.Pricing.Request,
			item.Pricing.WebSearch,
		)
		offering := Offering{
			ServiceID:     "openrouter",
			WireModelID:   item.ID,
			ModelKey:      key,
			Exposures: openRouterExposures(
				s.ID(),
				observedAt,
				item.ID,
				caps,
				item.SupportedParameters,
				item.DefaultParameters,
			),
			Pricing:       pricing,
			PricingStatus: pricingStatus,
			LimitsOverride:   limitsPtr(item.TopProvider.ContextLength, item.TopProvider.MaxCompletionTokens),
			PerRequestLimits: convertPerRequestLimits(item.PerRequestLimits),
			IsModerated:      item.TopProvider.IsModerated,
			Provenance: []Provenance{{
				SourceID:   s.ID(),
				Authority:  string(AuthorityTrusted),
				ObservedAt: observedAt,
				RawID:      item.ID,
			}},
		}
		if item.CanonicalSlug != "" && item.CanonicalSlug != item.ID {
			offering.Aliases = []string{item.CanonicalSlug}
		}
		fragment.Offerings = append(fragment.Offerings, offering)
	}

	return fragment, nil
}

func capabilitiesFromOpenRouter(params []string, inputModalities []string, hasCacheReadPricing bool) Capabilities {
	return Capabilities{
		Reasoning:         reasoningFromOpenRouter(params),
		ToolUse:           containsString(params, "tools") || containsString(params, "tool_choice"),
		ParallelToolCalls: containsString(params, "parallel_tool_calls"),
		StructuredOutput:  containsString(params, "response_format"),
		StructuredOutputs: containsString(params, "structured_outputs"),
		Vision:            containsString(inputModalities, "image") || containsString(inputModalities, "video"),
		Streaming:         true,
		Caching:           hasCacheReadPricing,
		Temperature:       containsString(params, "temperature"),
		Logprobs:          containsString(params, "logprobs"),
		Seed:              containsString(params, "seed"),
		WebSearch:         containsString(params, "web_search_options"),
	}
}

func reasoningFromOpenRouter(params []string) *ReasoningCapability {
	hasReasoning := containsString(params, "reasoning") || containsString(params, "include_reasoning") || containsString(params, "reasoning_effort")
	if !hasReasoning {
		return nil
	}
	r := &ReasoningCapability{Available: true, Modes: []ReasoningMode{ReasoningModeEnabled, ReasoningModeOff}}
	if containsString(params, "reasoning_effort") {
		r.Efforts = []ReasoningEffortLevel{ReasoningEffortLow, ReasoningEffortMedium, ReasoningEffortHigh}
	}
	return r
}

func pricingFromOpenRouter(prompt, completion, cacheRead, cacheWrite, reasoning, image, imageToken, imageOutput, audio, audioOutput, request, webSearch string) (*Pricing, string) {
	p := &Pricing{}
	fields := []string{prompt, completion, cacheRead, cacheWrite, reasoning, image, imageToken, imageOutput, audio, audioOutput, request, webSearch}
	sawField := false
	for _, field := range fields {
		if field != "" {
			sawField = true
			break
		}
	}
	if v, ok := parsePerTokenDollars(prompt); ok {
		p.Input = v
	}
	if v, ok := parsePerTokenDollars(completion); ok {
		p.Output = v
	}
	if v, ok := parsePerTokenDollars(cacheRead); ok {
		p.CachedInput = v
	}
	if v, ok := parsePerTokenDollars(cacheWrite); ok {
		p.CacheWrite = v
	}
	if v, ok := parsePerTokenDollars(reasoning); ok {
		p.Reasoning = v
	}
	if v, ok := parsePerTokenDollars(image); ok {
		p.Image = v
	}
	if v, ok := parsePerTokenDollars(imageToken); ok {
		p.ImageToken = v
	}
	if v, ok := parsePerTokenDollars(imageOutput); ok {
		p.ImageOutput = v
	}
	if v, ok := parsePerTokenDollars(audio); ok {
		p.Audio = v
	}
	if v, ok := parsePerTokenDollars(audioOutput); ok {
		p.AudioOutput = v
	}
	if v, ok := parsePerTokenDollars(request); ok {
		p.Request = v
	}
	if v, ok := parsePerTokenDollars(webSearch); ok {
		p.WebSearch = v
	}
	if !sawField {
		return nil, "unknown"
	}
	if pricingIsFree(p) {
		return p, "free"
	}
	return p, "known"
}

func parsePerTokenDollars(raw string) (float64, bool) {
	if raw == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	return v * 1_000_000, true
}

func convertDefaultParameters(dp *struct {
	Temperature       *float64 `json:"temperature"`
	TopP              *float64 `json:"top_p"`
	TopK              *int     `json:"top_k"`
	FrequencyPenalty  *float64 `json:"frequency_penalty"`
	PresencePenalty   *float64 `json:"presence_penalty"`
	RepetitionPenalty *float64 `json:"repetition_penalty"`
}) *DefaultParameters {
	if dp == nil {
		return nil
	}
	return &DefaultParameters{
		Temperature:       dp.Temperature,
		TopP:              dp.TopP,
		TopK:              dp.TopK,
		FrequencyPenalty:  dp.FrequencyPenalty,
		PresencePenalty:   dp.PresencePenalty,
		RepetitionPenalty: dp.RepetitionPenalty,
	}
}

func convertPerRequestLimits(prl *struct {
	PromptTokens     float64 `json:"prompt_tokens"`
	CompletionTokens float64 `json:"completion_tokens"`
}) *PerRequestLimits {
	if prl == nil {
		return nil
	}
	return &PerRequestLimits{
		PromptTokens:     prl.PromptTokens,
		CompletionTokens: prl.CompletionTokens,
	}
}

func parameterValuesFromOpenRouter(params []string) map[string][]string {
	values := map[string][]string{}
	if containsString(params, "reasoning_effort") {
		values["reasoning_effort"] = []string{string(ReasoningEffortMinimal), string(ReasoningEffortLow), string(ReasoningEffortMedium), string(ReasoningEffortHigh)}
	}
	if containsString(params, "reasoning_summary") {
		values["reasoning_summary"] = []string{string(ReasoningSummaryAuto), string(ReasoningSummaryConcise), string(ReasoningSummaryDetailed)}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func normalizedParametersFromOpenRouter(params []string) []NormalizedParameter {
	out := make([]NormalizedParameter, 0)
	if containsString(params, "tools") {
		out = append(out, ParamTools)
	}
	if containsString(params, "tool_choice") {
		out = append(out, ParamToolChoice)
	}
	if containsString(params, "temperature") {
		out = append(out, ParamTemperature)
	}
	if containsString(params, "response_format") {
		out = append(out, ParamResponseFormat)
	}
	if containsString(params, "reasoning_effort") {
		out = append(out, ParamReasoningEffort)
	}
	if containsString(params, "reasoning") || containsString(params, "include_reasoning") {
		out = append(out, ParamThinking)
	}
	if containsString(params, "parallel_tool_calls") {
		out = append(out, ParamParallelTools)
	}
	if containsString(params, "logprobs") {
		out = append(out, ParamLogprobs)
	}
	if containsString(params, "seed") {
		out = append(out, ParamSeed)
	}
	if containsString(params, "web_search_options") {
		out = append(out, ParamWebSearch)
	}
	if containsString(params, "reasoning_summary") {
		out = append(out, ParamReasoningSummary)
	}
	return normalizeNormalizedParameters(out)
}

func parameterMappingsFromOpenRouter(params []string) []ParameterMapping {
	out := make([]ParameterMapping, 0)
	if containsString(params, "tools") {
		out = append(out, ParameterMapping{Normalized: ParamTools, WireName: "tools"})
	}
	if containsString(params, "tool_choice") {
		out = append(out, ParameterMapping{Normalized: ParamToolChoice, WireName: "tool_choice"})
	}
	if containsString(params, "temperature") {
		out = append(out, ParameterMapping{Normalized: ParamTemperature, WireName: "temperature"})
	}
	if containsString(params, "response_format") {
		out = append(out, ParameterMapping{Normalized: ParamResponseFormat, WireName: "response_format"})
	}
	if containsString(params, "reasoning_effort") {
		out = append(out, ParameterMapping{Normalized: ParamReasoningEffort, WireName: "reasoning_effort"})
	}
	if containsString(params, "reasoning") {
		out = append(out, ParameterMapping{Normalized: ParamThinking, WireName: "reasoning"})
	}
	if containsString(params, "include_reasoning") {
		out = append(out, ParameterMapping{Normalized: ParamThinking, WireName: "include_reasoning"})
	}
	if containsString(params, "parallel_tool_calls") {
		out = append(out, ParameterMapping{Normalized: ParamParallelTools, WireName: "parallel_tool_calls"})
	}
	if containsString(params, "logprobs") {
		out = append(out, ParameterMapping{Normalized: ParamLogprobs, WireName: "logprobs"})
	}
	if containsString(params, "seed") {
		out = append(out, ParameterMapping{Normalized: ParamSeed, WireName: "seed"})
	}
	if containsString(params, "web_search_options") {
		out = append(out, ParameterMapping{Normalized: ParamWebSearch, WireName: "web_search_options"})
	}
	if containsString(params, "reasoning_summary") {
		out = append(out, ParameterMapping{Normalized: ParamReasoningSummary, WireName: "reasoning_summary"})
	}
	return out
}

func openRouterExposures(sourceID string, observedAt time.Time, rawID string, caps Capabilities, supportedParams []string, defaults *struct {
	Temperature       *float64 `json:"temperature"`
	TopP              *float64 `json:"top_p"`
	TopK              *int     `json:"top_k"`
	FrequencyPenalty  *float64 `json:"frequency_penalty"`
	PresencePenalty   *float64 `json:"presence_penalty"`
	RepetitionPenalty *float64 `json:"repetition_penalty"`
}) []OfferingExposure {
	params := normalizedParametersFromOpenRouter(supportedParams)
	mappings := parameterMappingsFromOpenRouter(supportedParams)
	values := parameterValuesFromOpenRouter(supportedParams)
	def := convertDefaultParameters(defaults)
	prov := []Provenance{{SourceID: sourceID, Authority: string(AuthorityTrusted), ObservedAt: observedAt, RawID: rawID}}
	responsesParams := mergeNormalizedParameters(params, []NormalizedParameter{ParamReasoningEffort, ParamReasoningSummary})
	responsesMappings := mergeParameterMappings(mappings, []ParameterMapping{
		{Normalized: ParamReasoningEffort, WireName: "reasoning.effort"},
		{Normalized: ParamReasoningSummary, WireName: "reasoning.summary"},
	})
	responsesValues := values
	if responsesValues == nil {
		responsesValues = map[string][]string{}
	}
	if _, ok := responsesValues[string(ParamReasoningEffort)]; !ok {
		responsesValues[string(ParamReasoningEffort)] = []string{string(ReasoningEffortMinimal), string(ReasoningEffortLow), string(ReasoningEffortMedium), string(ReasoningEffortHigh)}
	}
	if _, ok := responsesValues[string(ParamReasoningSummary)]; !ok {
		responsesValues[string(ParamReasoningSummary)] = []string{string(ReasoningSummaryAuto), string(ReasoningSummaryConcise), string(ReasoningSummaryDetailed)}
	}
	return []OfferingExposure{
		{APIType: APITypeOpenAIResponses, ExposedCapabilities: capabilitiesPtr(caps), SupportedParameters: responsesParams, ParameterMappings: responsesMappings, ParameterValues: responsesValues, DefaultParameters: def, Provenance: prov},
		{APIType: APITypeOpenAIMessages, ExposedCapabilities: capabilitiesPtr(caps), SupportedParameters: params, ParameterMappings: mappings, ParameterValues: values, DefaultParameters: def, Provenance: prov},
	}
}
