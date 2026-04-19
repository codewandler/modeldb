package modeldb

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const codexSourceID = "codex-api"

type CodexSource struct {
	FilePath string
}

func NewCodexSource() CodexSource                    { return CodexSource{FilePath: DefaultCodexFixturePath()} }
func NewCodexSourceFromFile(path string) CodexSource { return CodexSource{FilePath: path} }
func DefaultCodexFixturePath() string {
	return filepath.Join("internal", "source", "codex", "testdata", "models.json")
}
func (CodexSource) ID() string { return codexSourceID }

type codexReasoningLevel struct {
	Effort string `json:"effort"`
}
type codexModelEntry struct {
	Slug                     string                `json:"slug"`
	DisplayName              string                `json:"display_name"`
	Description              string                `json:"description"`
	DefaultReasoningLevel    string                `json:"default_reasoning_level"`
	SupportedReasoningLevels []codexReasoningLevel `json:"supported_reasoning_levels"`
	SupportedInAPI           bool                  `json:"supported_in_api"`
	SupportVerbosity         bool                  `json:"support_verbosity"`
	DefaultVerbosity         string                `json:"default_verbosity"`
	SupportsReasoningSummary bool                  `json:"supports_reasoning_summaries"`
	DefaultReasoningSummary  string                `json:"default_reasoning_summary"`
	ContextWindow            int                   `json:"context_window"`
	InputModalities          []string              `json:"input_modalities"`
	OutputModalities         []string              `json:"output_modalities"`
	LastUpdated              string                `json:"last_updated"`
	Deprecated               bool                  `json:"deprecated"`
	SupportsParallelTools    bool                  `json:"supports_parallel_tool_calls"`
}

type codexPayload struct {
	Models []codexModelEntry `json:"models"`
}

func (s CodexSource) Fetch(context.Context) (*Fragment, error) {
	if s.FilePath == "" {
		return nil, fmt.Errorf("codex source: missing fixture path")
	}
	data, err := os.ReadFile(s.FilePath)
	if err != nil {
		return nil, err
	}
	var payload codexPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	observedAt := time.Time{}
	frag := &Fragment{Services: []Service{{ID: "codex", Name: "Codex", Kind: ServiceKindDirect, Operator: "openai", DocsURL: "https://chatgpt.com/codex", Provenance: []Provenance{{SourceID: codexSourceID, Authority: string(AuthorityTrusted), ObservedAt: observedAt}}}}}
	for _, item := range payload.Models {
		if !item.SupportedInAPI {
			continue
		}
		key, ok := inferOpenAIModelKey(item.Slug)
		if !ok {
			continue
		}
		modelCaps := coarseCachingCapabilities(capabilitiesFromCodexModel(item), true)
		exposureCaps := capabilitiesFromCodexModel(item)
		frag.Models = append(frag.Models, ModelRecord{Key: key, Name: item.DisplayName, Description: item.Description, Canonical: false, Capabilities: modelCaps, Limits: Limits{ContextWindow: item.ContextWindow}, InputModalities: normalizeStrings(item.InputModalities), OutputModalities: normalizeStrings(item.OutputModalities), LastUpdated: normalizeDate(item.LastUpdated), Deprecated: item.Deprecated, Provenance: []Provenance{{SourceID: codexSourceID, Authority: string(AuthorityTrusted), ObservedAt: observedAt, RawID: item.Slug}}})
		exp := OfferingExposure{APIType: APITypeOpenAIResponses, ExposedCapabilities: capabilitiesPtr(exposureCaps), SupportedParameters: codexSupportedParameters(item), ParameterMappings: codexParameterMappings(item), ParameterValues: codexParameterValues(item), Provenance: []Provenance{{SourceID: codexSourceID, Authority: string(AuthorityTrusted), ObservedAt: observedAt, RawID: item.Slug}}}
		frag.Offerings = append(frag.Offerings, Offering{ServiceID: "codex", WireModelID: item.Slug, ModelKey: key, Exposures: []OfferingExposure{exp}, Provenance: []Provenance{{SourceID: codexSourceID, Authority: string(AuthorityTrusted), ObservedAt: observedAt, RawID: item.Slug}}})
	}
	sort.Slice(frag.Offerings, func(i, j int) bool { return frag.Offerings[i].WireModelID < frag.Offerings[j].WireModelID })
	return frag, nil
}

func coarseCachingCapabilities(caps Capabilities, available bool) Capabilities {
	if !available {
		return caps
	}
	caps.Caching = &CachingCapability{Available: true}
	return caps
}

func capabilitiesFromCodexModel(item codexModelEntry) Capabilities {
	caps := Capabilities{ToolUse: true, ParallelToolCalls: item.SupportsParallelTools, StructuredOutput: true, Streaming: true, Temperature: true, Vision: containsString(item.InputModalities, "image"), Caching: &CachingCapability{Available: true, Mode: CachingModeImplicit}}
	efforts := make([]ReasoningEffortLevel, 0, len(item.SupportedReasoningLevels)+1)
	if !strings.Contains(strings.ToLower(item.Slug), "mini") {
		efforts = append(efforts, ReasoningEffortNone)
	}
	for _, e := range item.SupportedReasoningLevels {
		switch strings.ToLower(strings.TrimSpace(e.Effort)) {
		case "low":
			efforts = append(efforts, ReasoningEffortLow)
		case "medium":
			efforts = append(efforts, ReasoningEffortMedium)
		case "high":
			efforts = append(efforts, ReasoningEffortHigh)
		case "max":
			efforts = append(efforts, ReasoningEffortMax)
		case "xhigh":
			efforts = append(efforts, ReasoningEffortXHigh)
		}
	}
	if len(efforts) > 0 || item.SupportsReasoningSummary {
		caps.Reasoning = &ReasoningCapability{Available: true, Efforts: dedupeEfforts(efforts), Summaries: codexSummaryValues(item.SupportsReasoningSummary), Modes: []ReasoningMode{ReasoningModeEnabled, ReasoningModeOff}, VisibleSummary: item.SupportsReasoningSummary}
	}
	return caps
}

func codexSummaryValues(enabled bool) []ReasoningSummaryValue {
	if !enabled {
		return nil
	}
	return []ReasoningSummaryValue{ReasoningSummaryNone, ReasoningSummaryAuto, ReasoningSummaryConcise, ReasoningSummaryDetailed}
}

func dedupeEfforts(in []ReasoningEffortLevel) []ReasoningEffortLevel {
	seen := map[ReasoningEffortLevel]bool{}
	out := make([]ReasoningEffortLevel, 0, len(in))
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func codexSupportedParameters(item codexModelEntry) []NormalizedParameter {
	params := []NormalizedParameter{ParamResponseFormat, ParamTools, ParamTemperature}
	if item.SupportsParallelTools {
		params = append(params, ParamParallelTools)
	}
	if len(item.SupportedReasoningLevels) > 0 {
		params = append(params, ParamThinking, ParamReasoningEffort)
	}
	if item.SupportsReasoningSummary {
		params = append(params, ParamReasoningSummary)
	}
	return normalizeNormalizedParameters(params)
}

func codexParameterMappings(item codexModelEntry) []ParameterMapping {
	m := []ParameterMapping{{Normalized: ParamResponseFormat, WireName: "response_format"}, {Normalized: ParamTools, WireName: "tools"}, {Normalized: ParamTemperature, WireName: "temperature"}}
	if item.SupportsParallelTools {
		m = append(m, ParameterMapping{Normalized: ParamParallelTools, WireName: "parallel_tool_calls"})
	}
	if len(item.SupportedReasoningLevels) > 0 {
		m = append(m, ParameterMapping{Normalized: ParamThinking, WireName: "reasoning"}, ParameterMapping{Normalized: ParamReasoningEffort, WireName: "reasoning.effort"})
	}
	if item.SupportsReasoningSummary {
		m = append(m, ParameterMapping{Normalized: ParamReasoningSummary, WireName: "reasoning.summary"})
	}
	return m
}

func codexParameterValues(item codexModelEntry) map[string][]string {
	values := map[string][]string{}
	efforts := make([]string, 0, len(item.SupportedReasoningLevels)+1)
	if !strings.Contains(strings.ToLower(item.Slug), "mini") {
		efforts = append(efforts, string(ReasoningEffortNone))
	}
	for _, e := range item.SupportedReasoningLevels {
		s := strings.ToLower(strings.TrimSpace(e.Effort))
		switch s {
		case "low", "medium", "high", "max", "xhigh":
			efforts = append(efforts, s)
		}
	}
	if len(efforts) > 0 {
		values[string(ParamReasoningEffort)] = efforts
	}
	if item.SupportsReasoningSummary {
		values[string(ParamReasoningSummary)] = []string{string(ReasoningSummaryAuto), string(ReasoningSummaryConcise), string(ReasoningSummaryDetailed)}
	}
	if item.SupportVerbosity {
		values["verbosity"] = []string{"low", "medium", "high"}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}
