package modeldb

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const openAIStaticSourceID = "openai-static"

type OpenAIStaticSource struct{ File string }

func NewOpenAIStaticSource() OpenAIStaticSource {
	return OpenAIStaticSource{File: DefaultOpenAIStaticManifestPath()}
}
func NewOpenAIStaticSourceFromFile(path string) OpenAIStaticSource {
	return OpenAIStaticSource{File: path}
}
func DefaultOpenAIStaticManifestPath() string {
	return filepath.Join("internal", "source", "openai", "testdata", "static.json")
}
func (OpenAIStaticSource) ID() string { return openAIStaticSourceID }

type openAIStaticManifest struct {
	Profiles map[string]openAIStaticExposureTemplate `json:"profiles"`
	Families map[string]openAIStaticFamilyTemplate   `json:"families"`
	Models   []openAIStaticModelEntry                `json:"models"`
}

type openAIStaticExposureTemplate struct {
	APIType             APIType               `json:"api_type"`
	Capabilities        *Capabilities         `json:"capabilities,omitempty"`
	SupportedParameters []NormalizedParameter `json:"supported_parameters,omitempty"`
	ParameterMappings   []ParameterMapping    `json:"parameter_mappings,omitempty"`
	ParameterValues     map[string][]string   `json:"parameter_values,omitempty"`
}

type openAIStaticFamilyTemplate struct {
	ModelDefaults    *openAIStaticModelTemplate             `json:"model_defaults,omitempty"`
	ExposureDefaults map[string]openAIStaticExposureBinding `json:"exposure_defaults,omitempty"`
}

type openAIStaticExposureBinding struct {
	Profiles  []string                   `json:"profiles,omitempty"`
	Overrides *openAIStaticExposurePatch `json:"overrides,omitempty"`
}

type openAIStaticModelEntry struct {
	Slug              string                               `json:"slug"`
	Family            string                               `json:"family,omitempty"`
	ModelOverrides    *openAIStaticModelTemplate           `json:"model_overrides,omitempty"`
	ExposureOverrides map[string]openAIStaticExposurePatch `json:"exposure_overrides,omitempty"`
}

type openAIStaticModelTemplate struct {
	Capabilities     *Capabilities `json:"capabilities,omitempty"`
	Limits           *Limits       `json:"limits,omitempty"`
	Pricing          *Pricing      `json:"pricing,omitempty"`
	InputModalities  []string      `json:"input_modalities,omitempty"`
	OutputModalities []string      `json:"output_modalities,omitempty"`
}

type openAIStaticExposurePatch struct {
	ReplaceProfiles     bool                  `json:"replace_profiles,omitempty"`
	Profiles            []string              `json:"profiles,omitempty"`
	Capabilities        *Capabilities         `json:"capabilities,omitempty"`
	SupportedParameters []NormalizedParameter `json:"supported_parameters,omitempty"`
	ParameterMappings   []ParameterMapping    `json:"parameter_mappings,omitempty"`
	ParameterValues     map[string][]string   `json:"parameter_values,omitempty"`
}

func (s OpenAIStaticSource) Fetch(context.Context) (*Fragment, error) {
	data, err := os.ReadFile(s.File)
	if err != nil {
		return nil, err
	}
	var manifest openAIStaticManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	observedAt := time.Time{}
	frag := &Fragment{Services: []Service{{
		ID:         "openai",
		Name:       "OpenAI",
		Kind:       ServiceKindDirect,
		Operator:   "openai",
		Provenance: []Provenance{{SourceID: openAIStaticSourceID, Authority: string(AuthorityTrusted), ObservedAt: observedAt}},
	}}}
	for _, item := range manifest.Models {
		key, ok := inferOpenAIModelKey(item.Slug)
		if !ok {
			continue
		}
		resolvedModel, resolvedExposures := resolveOpenAIStaticModel(manifest, item)
		if openAIHasDocumentedPromptCaching(item) {
			resolvedModel.capabilities = mergeCapabilities(resolvedModel.capabilities, Capabilities{Caching: &CachingCapability{Available: true}})
		}
		frag.Models = append(frag.Models, ModelRecord{
			Key:              key,
			Canonical:        false,
			Capabilities:     resolvedModel.capabilities,
			Limits:           resolvedModel.limits,
			ReferencePricing: resolvedModel.pricing,
			InputModalities:  resolvedModel.inputModalities,
			OutputModalities: resolvedModel.outputModalities,
			Provenance:       []Provenance{{SourceID: openAIStaticSourceID, Authority: string(AuthorityTrusted), ObservedAt: observedAt, RawID: item.Slug}},
		})
		offExposures := make([]OfferingExposure, 0, len(resolvedExposures))
		for _, exp := range resolvedExposures {
			exp = applyOpenAICachingDefaults(item, exp)
			offExposures = append(offExposures, OfferingExposure{
				APIType:             exp.APIType,
				ExposedCapabilities: capabilitiesPtr(exp.Capabilities),
				SupportedParameters: exp.SupportedParameters,
				ParameterMappings:   exp.ParameterMappings,
				ParameterValues:     exp.ParameterValues,
				Provenance:          []Provenance{{SourceID: openAIStaticSourceID, Authority: string(AuthorityTrusted), ObservedAt: observedAt, RawID: item.Slug}},
			})
		}
		sort.Slice(offExposures, func(i, j int) bool { return offExposures[i].APIType < offExposures[j].APIType })
		pricingStatus := "unknown"
		if resolvedModel.pricing != nil {
			if pricingIsFree(resolvedModel.pricing) {
				pricingStatus = "free"
			} else {
				pricingStatus = "known"
			}
		}
		frag.Offerings = append(frag.Offerings, Offering{
			ServiceID:     "openai",
			WireModelID:   item.Slug,
			ModelKey:      key,
			Exposures:     offExposures,
			Pricing:       resolvedModel.pricing,
			PricingStatus: pricingStatus,
			Provenance:    []Provenance{{SourceID: openAIStaticSourceID, Authority: string(AuthorityTrusted), ObservedAt: observedAt, RawID: item.Slug}},
		})
	}
	sort.Slice(frag.Models, func(i, j int) bool { return modelID(frag.Models[i].Key) < modelID(frag.Models[j].Key) })
	return frag, nil
}

type resolvedOpenAIStaticModel struct {
	capabilities     Capabilities
	limits           Limits
	pricing          *Pricing
	inputModalities  []string
	outputModalities []string
}

type resolvedOpenAIStaticExposure struct {
	APIType             APIType
	Capabilities        Capabilities
	SupportedParameters []NormalizedParameter
	ParameterMappings   []ParameterMapping
	ParameterValues     map[string][]string
}

func resolveOpenAIStaticModel(manifest openAIStaticManifest, item openAIStaticModelEntry) (resolvedOpenAIStaticModel, []resolvedOpenAIStaticExposure) {
	var resolved resolvedOpenAIStaticModel
	byAPI := map[APIType]resolvedOpenAIStaticExposure{}
	if fam, ok := manifest.Families[item.Family]; ok {
		applyOpenAIStaticModelTemplate(&resolved, fam.ModelDefaults)
		for apiName, binding := range fam.ExposureDefaults {
			api := APIType(apiName)
			exp := byAPI[api]
			exp.APIType = api
			applyOpenAIStaticExposureBinding(&exp, manifest, binding)
			byAPI[api] = exp
		}
	}
	applyOpenAIStaticModelTemplate(&resolved, item.ModelOverrides)
	for apiName, patch := range item.ExposureOverrides {
		api := APIType(apiName)
		exp := byAPI[api]
		exp.APIType = api
		applyOpenAIStaticExposurePatch(&exp, manifest, &patch)
		byAPI[api] = exp
	}
	out := make([]resolvedOpenAIStaticExposure, 0, len(byAPI))
	for _, v := range byAPI {
		v.SupportedParameters = normalizeNormalizedParameters(v.SupportedParameters)
		out = append(out, v)
	}
	return resolved, out
}

func applyOpenAIStaticModelTemplate(dst *resolvedOpenAIStaticModel, tpl *openAIStaticModelTemplate) {
	if tpl == nil {
		return
	}
	if tpl.Capabilities != nil {
		dst.capabilities = mergeCapabilities(dst.capabilities, *tpl.Capabilities)
	}
	if tpl.Limits != nil {
		if tpl.Limits.ContextWindow != 0 {
			dst.limits.ContextWindow = tpl.Limits.ContextWindow
		}
		if tpl.Limits.MaxOutput != 0 {
			dst.limits.MaxOutput = tpl.Limits.MaxOutput
		}
	}
	if tpl.Pricing != nil {
		pricing := *tpl.Pricing
		dst.pricing = &pricing
	}
	if tpl.InputModalities != nil {
		dst.inputModalities = normalizeStrings(tpl.InputModalities)
	}
	if tpl.OutputModalities != nil {
		dst.outputModalities = normalizeStrings(tpl.OutputModalities)
	}
}

func applyOpenAIStaticExposureBinding(dst *resolvedOpenAIStaticExposure, manifest openAIStaticManifest, binding openAIStaticExposureBinding) {
	for _, profileID := range binding.Profiles {
		applyOpenAIStaticExposureTemplate(dst, manifest.Profiles[profileID])
	}
	applyOpenAIStaticExposurePatch(dst, manifest, binding.Overrides)
}

func applyOpenAIStaticExposurePatch(dst *resolvedOpenAIStaticExposure, manifest openAIStaticManifest, patch *openAIStaticExposurePatch) {
	if patch == nil {
		return
	}
	if patch.ReplaceProfiles {
		dst.Capabilities = Capabilities{}
		dst.SupportedParameters = nil
		dst.ParameterMappings = nil
		dst.ParameterValues = nil
	}
	for _, profileID := range patch.Profiles {
		applyOpenAIStaticExposureTemplate(dst, manifest.Profiles[profileID])
	}
	if patch.Capabilities != nil {
		dst.Capabilities = mergeCapabilities(dst.Capabilities, *patch.Capabilities)
	}
	if patch.SupportedParameters != nil {
		dst.SupportedParameters = mergeNormalizedParameters(dst.SupportedParameters, patch.SupportedParameters)
	}
	if patch.ParameterMappings != nil {
		dst.ParameterMappings = mergeParameterMappings(dst.ParameterMappings, patch.ParameterMappings)
	}
	if patch.ParameterValues != nil {
		if dst.ParameterValues == nil {
			dst.ParameterValues = map[string][]string{}
		}
		for k, v := range patch.ParameterValues {
			dst.ParameterValues[k] = normalizeStrings(v)
		}
	}
}

func applyOpenAICachingDefaults(item openAIStaticModelEntry, exp resolvedOpenAIStaticExposure) resolvedOpenAIStaticExposure {
	if exp.APIType != APITypeOpenAIResponses || !openAIHasDocumentedPromptCaching(item) {
		return exp
	}
	exp.Capabilities = mergeCapabilities(exp.Capabilities, Capabilities{Caching: &CachingCapability{
		Available:            true,
		Mode:                 CachingModeMixed,
		Configurable:         true,
		PromptCacheRetention: true,
		PromptCacheKey:       true,
		RetentionValues:      []string{"in_memory", "24h"},
	}})
	exp.SupportedParameters = mergeNormalizedParameters(exp.SupportedParameters, []NormalizedParameter{ParamPromptCacheRetention, ParamPromptCacheKey})
	exp.ParameterMappings = mergeParameterMappings(exp.ParameterMappings, []ParameterMapping{{Normalized: ParamPromptCacheRetention, WireName: "prompt_cache_retention"}, {Normalized: ParamPromptCacheKey, WireName: "prompt_cache_key"}})
	if exp.ParameterValues == nil {
		exp.ParameterValues = map[string][]string{}
	}
	exp.ParameterValues[string(ParamPromptCacheRetention)] = mergeStringSlices(exp.ParameterValues[string(ParamPromptCacheRetention)], []string{"in_memory", "24h"})
	return exp
}

func openAIHasDocumentedPromptCaching(item openAIStaticModelEntry) bool {
	family := item.Family
	switch {
	case family == "gpt-4.1-reasoning":
		return true
	case family == "gpt-4o-reasoning":
		return true
	case family == "o1-reasoning":
		return true
	case family == "o3plus-reasoning":
		return true
	case family == "gpt-5-pre-5.1-reasoning":
		return true
	case family == "gpt-5-5.1-reasoning":
		return true
	case family == "gpt-5-5.2plus-reasoning":
		return true
	case family == "gpt-5-5.2plus-codex-backed":
		return true
	default:
		return false
	}
}

func applyOpenAIStaticExposureTemplate(dst *resolvedOpenAIStaticExposure, tpl openAIStaticExposureTemplate) {
	if tpl.APIType != "" {
		dst.APIType = tpl.APIType
	}
	if tpl.Capabilities != nil {
		dst.Capabilities = mergeCapabilities(dst.Capabilities, *tpl.Capabilities)
	}
	if tpl.SupportedParameters != nil {
		dst.SupportedParameters = mergeNormalizedParameters(dst.SupportedParameters, tpl.SupportedParameters)
	}
	if tpl.ParameterMappings != nil {
		dst.ParameterMappings = mergeParameterMappings(dst.ParameterMappings, tpl.ParameterMappings)
	}
	if tpl.ParameterValues != nil {
		if dst.ParameterValues == nil {
			dst.ParameterValues = map[string][]string{}
		}
		for k, v := range tpl.ParameterValues {
			if _, ok := dst.ParameterValues[k]; !ok {
				dst.ParameterValues[k] = normalizeStrings(v)
			}
		}
	}
}
