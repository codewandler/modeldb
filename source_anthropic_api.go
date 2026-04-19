package modeldb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	anthropicSourceID          = "anthropic-api"
	defaultAnthropicBaseURL    = "https://api.anthropic.com"
	defaultAnthropicAPIVersion = "2023-06-01"
)

type AnthropicAPISource struct {
	APIKey   string
	BaseURL  string
	FilePath string
	Client   *http.Client
}

func NewAnthropicAPISource(apiKey string) AnthropicAPISource {
	return AnthropicAPISource{APIKey: apiKey, BaseURL: defaultAnthropicBaseURL, Client: http.DefaultClient}
}

func NewAnthropicAPISourceFromEnv() AnthropicAPISource {
	return NewAnthropicAPISource(os.Getenv("ANTHROPIC_API_KEY"))
}

func NewAnthropicAPISourceFromFile(path string) AnthropicAPISource {
	source := NewAnthropicAPISource("")
	source.FilePath = path
	return source
}

func DefaultAnthropicFixturePath() string {
	return filepath.Join("internal", "source", "anthropic", "testdata", "models.json")
}

func (AnthropicAPISource) ID() string { return anthropicSourceID }

func (s AnthropicAPISource) Fetch(ctx context.Context) (*Fragment, error) {
	payload, err := s.loadPayload(ctx)
	if err != nil {
		return nil, err
	}

	observedAt := time.Now().UTC()
	service := Service{
		ID:       "anthropic",
		Name:     "Anthropic",
		Kind:     ServiceKindDirect,
		Operator: "anthropic",
		Provenance: []Provenance{{
			SourceID:   anthropicSourceID,
			Authority:  string(AuthorityCanonical),
			ObservedAt: observedAt,
		}},
	}

	fragment := &Fragment{Services: []Service{service}}
	seriesLatest := latestAnthropicModelBySeries(payload.Data)

	for _, item := range payload.Data {
		key, ok := inferAnthropicModelKey(item.ID)
		if !ok {
			continue
		}
		key = NormalizeKey(key)

		model := ModelRecord{
			Key:              key,
			Name:             item.DisplayName,
			Aliases:          anthropicModelAliases(item, key, seriesLatest),
			Canonical:        true,
			Capabilities:     capabilitiesFromAnthropicAPI(item),
			Limits:           Limits{ContextWindow: item.MaxInputTokens, MaxOutput: item.MaxTokens},
			InputModalities:  anthropicInputModalities(item),
			OutputModalities: []string{"text"},
			LastUpdated:      anthropicCreatedDate(item.CreatedAt),
			ReferencePricing: anthropicPricing(item.ID, key),
			Provenance: []Provenance{{
				SourceID:   anthropicSourceID,
				Authority:  string(AuthorityCanonical),
				ObservedAt: observedAt,
				RawID:      item.ID,
			}},
		}
		fragment.Models = append(fragment.Models, model)
		caps := capabilitiesFromAnthropicAPI(item)
		fragment.Offerings = append(fragment.Offerings, Offering{
			ServiceID:   service.ID,
			WireModelID: item.ID,
			ModelKey:    key,
			Exposures: []OfferingExposure{{
				APIType:             APITypeAnthropicMessages,
				ExposedCapabilities: capabilitiesPtr(caps),
				SupportedParameters: anthropicSupportedParameters(item),
				ParameterMappings:   anthropicParameterMappings(item),
				ParameterValues:     anthropicParameterValues(item),
				Provenance: []Provenance{{
					SourceID:   anthropicSourceID,
					Authority:  string(AuthorityCanonical),
					ObservedAt: observedAt,
					RawID:      item.ID,
				}},
			}},
			Pricing: anthropicPricing(item.ID, key),
			Provenance: []Provenance{{
				SourceID:   anthropicSourceID,
				Authority:  string(AuthorityCanonical),
				ObservedAt: observedAt,
				RawID:      item.ID,
			}},
		})
	}

	return fragment, nil
}

type anthropicModelsPayload struct {
	Data []anthropicModelEntry `json:"data"`
}

type anthropicModelEntry struct {
	Type           string `json:"type"`
	ID             string `json:"id"`
	DisplayName    string `json:"display_name"`
	CreatedAt      string `json:"created_at"`
	MaxInputTokens int    `json:"max_input_tokens"`
	MaxTokens      int    `json:"max_tokens"`
	Capabilities   struct {
		Batch struct {
			Supported bool `json:"supported"`
		} `json:"batch"`
		Citations struct {
			Supported bool `json:"supported"`
		} `json:"citations"`
		CodeExecution struct {
			Supported bool `json:"supported"`
		} `json:"code_execution"`
		ContextManagement struct {
			Supported bool `json:"supported"`
		} `json:"context_management"`
		Effort struct {
			Supported bool `json:"supported"`
			Low struct {
				Supported bool `json:"supported"`
			} `json:"low"`
			Medium struct {
				Supported bool `json:"supported"`
			} `json:"medium"`
			High struct {
				Supported bool `json:"supported"`
			} `json:"high"`
			Max struct {
				Supported bool `json:"supported"`
			} `json:"max"`
		} `json:"effort"`
		ImageInput struct {
			Supported bool `json:"supported"`
		} `json:"image_input"`
		PDFInput struct {
			Supported bool `json:"supported"`
		} `json:"pdf_input"`
		StructuredOutputs struct {
			Supported bool `json:"supported"`
		} `json:"structured_outputs"`
		Thinking struct {
			Supported bool `json:"supported"`
			Types     struct {
				Enabled struct {
					Supported bool `json:"supported"`
				} `json:"enabled"`
				Adaptive struct {
					Supported bool `json:"supported"`
				} `json:"adaptive"`
			} `json:"types"`
		} `json:"thinking"`
	} `json:"capabilities"`
}

func (s AnthropicAPISource) loadPayload(ctx context.Context) (anthropicModelsPayload, error) {
	if s.FilePath != "" {
		data, err := os.ReadFile(s.FilePath)
		if err != nil {
			return anthropicModelsPayload{}, fmt.Errorf("anthropic source: read file: %w", err)
		}
		var payload anthropicModelsPayload
		if err := json.Unmarshal(data, &payload); err != nil {
			return anthropicModelsPayload{}, fmt.Errorf("anthropic source: decode file: %w", err)
		}
		return payload, nil
	}

	if s.APIKey == "" {
		return anthropicModelsPayload{}, fmt.Errorf("anthropic source: missing API key")
	}
	baseURL := s.BaseURL
	if baseURL == "" {
		baseURL = defaultAnthropicBaseURL
	}
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models", nil)
	if err != nil {
		return anthropicModelsPayload{}, err
	}
	req.Header.Set("x-api-key", s.APIKey)
	req.Header.Set("anthropic-version", defaultAnthropicAPIVersion)

	resp, err := client.Do(req)
	if err != nil {
		return anthropicModelsPayload{}, fmt.Errorf("anthropic source: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return anthropicModelsPayload{}, fmt.Errorf("anthropic source: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var payload anthropicModelsPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return anthropicModelsPayload{}, err
	}
	return payload, nil
}

func capabilitiesFromAnthropicAPI(item anthropicModelEntry) Capabilities {
	return Capabilities{
		Reasoning:         reasoningFromAnthropicAPI(item),
		ToolUse:           true,
		StructuredOutput:  item.Capabilities.StructuredOutputs.Supported,
		Vision:            item.Capabilities.ImageInput.Supported,
		Streaming:         true,
		Caching:           item.Capabilities.ContextManagement.Supported,
		Temperature:       true,
	}
}

func reasoningFromAnthropicAPI(item anthropicModelEntry) *ReasoningCapability {
	if !item.Capabilities.Thinking.Supported && !item.Capabilities.Effort.Supported {
		return nil
	}
	r := &ReasoningCapability{
		Available: item.Capabilities.Thinking.Supported,
				Adaptive:  item.Capabilities.Thinking.Types.Adaptive.Supported,
	}
	if item.Capabilities.Thinking.Types.Enabled.Supported {
		r.Modes = append(r.Modes, ReasoningModeEnabled, ReasoningModeOff)
	}
	if item.Capabilities.Thinking.Types.Adaptive.Supported {
		r.Modes = append(r.Modes, ReasoningModeAdaptive)
	}
	if item.Capabilities.Effort.Low.Supported {
		r.Efforts = append(r.Efforts, ReasoningEffortLow)
	}
	if item.Capabilities.Effort.Medium.Supported {
		r.Efforts = append(r.Efforts, ReasoningEffortMedium)
	}
	if item.Capabilities.Effort.High.Supported {
		r.Efforts = append(r.Efforts, ReasoningEffortHigh)
	}
	if item.Capabilities.Effort.Max.Supported {
		r.Efforts = append(r.Efforts, ReasoningEffortMax)
	}
	return r
}

func anthropicInputModalities(item anthropicModelEntry) []string {
	modalities := []string{"text"}
	if item.Capabilities.ImageInput.Supported {
		modalities = append(modalities, "image")
	}
	if item.Capabilities.PDFInput.Supported {
		modalities = append(modalities, "pdf")
	}
	return modalities
}

func anthropicCreatedDate(v string) string {
	if len(v) >= 10 {
		return normalizeDate(v[:10])
	}
	return normalizeDate(v)
}

func anthropicModelUsesDatedRelease(id string) bool {
	parts := strings.Split(strings.TrimSpace(id), "-")
	if len(parts) == 0 {
		return false
	}
	last := parts[len(parts)-1]
	return len(last) == 8 && isDigits(last)
}

func anthropicModelAliases(item anthropicModelEntry, key ModelKey, latest map[string]anthropicModelEntry) []string {
	aliases := []string{item.ID}
	if anthropicModelUsesDatedRelease(item.ID) {
		if undated := anthropicUndatedAlias(item.ID); undated != "" {
			aliases = append(aliases, undated)
		}
	}
	if latestItem, ok := latest[key.Series]; ok && latestItem.ID == item.ID && key.Series != "" {
		aliases = append(aliases, key.Series)
	}
	return normalizeStrings(aliases)
}

func anthropicUndatedAlias(id string) string {
	parts := strings.Split(strings.TrimSpace(id), "-")
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	if len(last) == 8 && isDigits(last) {
		return strings.Join(parts[:len(parts)-1], "-")
	}
	return ""
}

func latestAnthropicModelBySeries(models []anthropicModelEntry) map[string]anthropicModelEntry {
	type candidate struct {
		key   ModelKey
		entry anthropicModelEntry
	}
	bySeries := make(map[string]candidate)
	for _, item := range models {
		key, ok := inferAnthropicModelKey(item.ID)
		if !ok || key.Series == "" {
			continue
		}
		current, ok := bySeries[key.Series]
		if !ok || anthropicCandidateLess(current.key, key) {
			bySeries[key.Series] = candidate{key: key, entry: item}
		}
	}
	out := make(map[string]anthropicModelEntry, len(bySeries))
	for series, item := range bySeries {
		out[series] = item.entry
	}
	return out
}

func anthropicCandidateLess(left, right ModelKey) bool {
	left = NormalizeKey(left)
	right = NormalizeKey(right)
	if left.Version != right.Version {
		return versionLess(left.Version, right.Version)
	}
	if left.ReleaseDate != right.ReleaseDate {
		return left.ReleaseDate < right.ReleaseDate
	}
	return modelID(left) < modelID(right)
}

func versionLess(left, right string) bool {
	leftParts := strings.Split(left, ".")
	rightParts := strings.Split(right, ".")
	n := len(leftParts)
	if len(rightParts) > n {
		n = len(rightParts)
	}
	for i := 0; i < n; i++ {
		li := 0
		ri := 0
		if i < len(leftParts) {
			li, _ = strconvAtoi(leftParts[i])
		}
		if i < len(rightParts) {
			ri, _ = strconvAtoi(rightParts[i])
		}
		if li != ri {
			return li < ri
		}
	}
	return false
}

func strconvAtoi(raw string) (int, bool) {
	if raw == "" || !isDigits(raw) {
		return 0, false
	}
	v := 0
	for i := 0; i < len(raw); i++ {
		v = v*10 + int(raw[i]-'0')
	}
	return v, true
}

func anthropicPricing(id string, key ModelKey) *Pricing {
	type price struct {
		input       float64
		output      float64
		cachedInput float64
		cacheWrite  float64
	}
	exact := map[string]price{
		"claude-opus-4-7":            {input: 5.0, output: 25.0, cachedInput: 0.50, cacheWrite: 6.25},
		"claude-opus-4-6":            {input: 5.0, output: 25.0, cachedInput: 0.50, cacheWrite: 6.25},
		"claude-sonnet-4-6":          {input: 3.0, output: 15.0, cachedInput: 0.30, cacheWrite: 3.75},
		"claude-opus-4-5-20251101":   {input: 5.0, output: 25.0, cachedInput: 0.50, cacheWrite: 6.25},
		"claude-haiku-4-5-20251001":  {input: 1.0, output: 5.0, cachedInput: 0.10, cacheWrite: 1.25},
		"claude-sonnet-4-5-20250929": {input: 3.0, output: 15.0, cachedInput: 0.30, cacheWrite: 3.75},
		"claude-opus-4-1-20250805":   {input: 15.0, output: 75.0, cachedInput: 1.50, cacheWrite: 18.75},
		"claude-opus-4-20250514":     {input: 15.0, output: 75.0, cachedInput: 1.50, cacheWrite: 18.75},
		"claude-sonnet-4-20250514":   {input: 3.0, output: 15.0, cachedInput: 0.30, cacheWrite: 3.75},
	}
	if p, ok := exact[id]; ok {
		return &Pricing{Input: p.input, Output: p.output, CachedInput: p.cachedInput, CacheWrite: p.cacheWrite}
	}
	line := LineID(key)
	lineFallback := map[string]price{
		"anthropic/claude/opus/4.7":   {input: 5.0, output: 25.0, cachedInput: 0.50, cacheWrite: 6.25},
		"anthropic/claude/opus/4.6":   {input: 5.0, output: 25.0, cachedInput: 0.50, cacheWrite: 6.25},
		"anthropic/claude/sonnet/4.6": {input: 3.0, output: 15.0, cachedInput: 0.30, cacheWrite: 3.75},
		"anthropic/claude/opus/4.5":   {input: 5.0, output: 25.0, cachedInput: 0.50, cacheWrite: 6.25},
		"anthropic/claude/haiku/4.5":  {input: 1.0, output: 5.0, cachedInput: 0.10, cacheWrite: 1.25},
		"anthropic/claude/sonnet/4.5": {input: 3.0, output: 15.0, cachedInput: 0.30, cacheWrite: 3.75},
		"anthropic/claude/opus/4.1":   {input: 15.0, output: 75.0, cachedInput: 1.50, cacheWrite: 18.75},
		"anthropic/claude/opus/4.0":   {input: 15.0, output: 75.0, cachedInput: 1.50, cacheWrite: 18.75},
		"anthropic/claude/sonnet/4.0": {input: 3.0, output: 15.0, cachedInput: 0.30, cacheWrite: 3.75},
	}
	if p, ok := lineFallback[line]; ok {
		return &Pricing{Input: p.input, Output: p.output, CachedInput: p.cachedInput, CacheWrite: p.cacheWrite}
	}
	return nil
}

func anthropicSupportedParameters(item anthropicModelEntry) []NormalizedParameter {
	params := []NormalizedParameter{ParamMessages}
	if item.Capabilities.Thinking.Supported {
		params = append(params, ParamThinking)
		if item.Capabilities.Thinking.Types.Enabled.Supported || item.Capabilities.Thinking.Types.Adaptive.Supported {
			params = append(params, ParamThinkingMode)
		}
	}
	if item.Capabilities.Effort.Supported {
		params = append(params, ParamReasoningEffort)
	}
	if item.Capabilities.StructuredOutputs.Supported {
		params = append(params, ParamResponseFormat)
	}
	return normalizeNormalizedParameters(params)
}

func anthropicParameterValues(item anthropicModelEntry) map[string][]string {
	values := map[string][]string{}
	if item.Capabilities.Effort.Supported {
		efforts := make([]string, 0, 4)
		if item.Capabilities.Effort.Low.Supported {
			efforts = append(efforts, string(ReasoningEffortLow))
		}
		if item.Capabilities.Effort.Medium.Supported {
			efforts = append(efforts, string(ReasoningEffortMedium))
		}
		if item.Capabilities.Effort.High.Supported {
			efforts = append(efforts, string(ReasoningEffortHigh))
		}
		if item.Capabilities.Effort.Max.Supported {
			efforts = append(efforts, string(ReasoningEffortMax))
		}
		if len(efforts) > 0 {
			values["reasoning_effort"] = efforts
		}
	}
	if item.Capabilities.Thinking.Supported {
		modes := make([]string, 0, 2)
		if item.Capabilities.Thinking.Types.Enabled.Supported {
			modes = append(modes, string(ReasoningModeEnabled))
		}
		if item.Capabilities.Thinking.Types.Adaptive.Supported {
			modes = append(modes, string(ReasoningModeAdaptive))
		}
		if len(modes) > 0 {
			values["thinking.mode"] = modes
		}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}


func anthropicParameterMappings(item anthropicModelEntry) []ParameterMapping {
	mappings := []ParameterMapping{{Normalized: ParamMessages, WireName: "messages"}}
	if item.Capabilities.Thinking.Supported {
		mappings = append(mappings, ParameterMapping{Normalized: ParamThinking, WireName: "thinking"})
		if item.Capabilities.Thinking.Types.Enabled.Supported || item.Capabilities.Thinking.Types.Adaptive.Supported {
			mappings = append(mappings, ParameterMapping{Normalized: ParamThinkingMode, WireName: "thinking.type"})
		}
	}
	if item.Capabilities.Effort.Supported {
		mappings = append(mappings, ParameterMapping{Normalized: ParamReasoningEffort, WireName: "reasoning_effort"})
	}
	if item.Capabilities.StructuredOutputs.Supported {
		mappings = append(mappings, ParameterMapping{Normalized: ParamResponseFormat, WireName: "response_format"})
	}
	return mappings
}
