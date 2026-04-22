package modeldb

import (
	"context"
	"time"
)

const kimiStaticSourceID = "kimi-static"

type KimiStaticSource struct{}

func NewKimiStaticSource() KimiStaticSource { return KimiStaticSource{} }

func (KimiStaticSource) ID() string { return kimiStaticSourceID }

func (KimiStaticSource) Fetch(context.Context) (*Fragment, error) {
	observedAt := time.Time{}

	service := Service{
		ID:       "kimi",
		Name:     "Kimi",
		Kind:     ServiceKindDirect,
		Operator: "moonshot",
		APIURL:   "https://api.moonshot.ai/v1",
		DocsURL:  "https://platform.kimi.ai/docs/api/overview",
		EnvVars:  []string{"MOONSHOT_API_KEY"},
		Provenance: []Provenance{{
			SourceID:   kimiStaticSourceID,
			Authority:  string(AuthorityCanonical),
			ObservedAt: observedAt,
		}},
	}

	k26Key := NormalizeKey(ModelKey{Creator: "moonshot", Family: "kimi", Version: "2.6"})

	k26Model := ModelRecord{
		Key:     k26Key,
		Name:    "Kimi K2.6",
		Aliases: []string{"kimi-k2-6"},
		Canonical: true,
		Capabilities: Capabilities{
			Reasoning:   &ReasoningCapability{Available: true, Modes: []ReasoningMode{ReasoningModeEnabled, ReasoningModeOff}},
			ToolUse:     true,
			Vision:      true,
			Streaming:   true,
			Temperature: true,
			Caching:     &CachingCapability{Available: true, Mode: CachingModeImplicit},
		},
		Limits:           Limits{ContextWindow: 262144, MaxOutput: 32768},
		InputModalities:  []string{"text", "image", "video"},
		OutputModalities: []string{"text"},
		ReferencePricing: &Pricing{Input: 0.95, Output: 4.00, CachedInput: 0.16},
		Provenance: []Provenance{{
			SourceID:   kimiStaticSourceID,
			Authority:  string(AuthorityCanonical),
			ObservedAt: observedAt,
			RawID:      "kimi-k2.6",
		}},
	}

	k26Offering := Offering{
		ServiceID:   service.ID,
		WireModelID: "kimi-k2.6",
		ModelKey:    k26Key,
		Aliases:     []string{"kimi-k2-6"},
		Exposures: []OfferingExposure{
			{
				APIType: APITypeOpenAIMessages,
				ExposedCapabilities: &Capabilities{
					Reasoning:   &ReasoningCapability{Available: true, Modes: []ReasoningMode{ReasoningModeEnabled, ReasoningModeOff}},
					ToolUse:     true,
					Vision:      true,
					Streaming:   true,
					Temperature: true,
					Caching:     &CachingCapability{Available: true, Mode: CachingModeImplicit},
				},
				SupportedParameters: []NormalizedParameter{
					ParamMessages,
					ParamTools,
					ParamToolChoice,
					ParamTemperature,
				},
				Provenance: []Provenance{{
					SourceID:   kimiStaticSourceID,
					Authority:  string(AuthorityCanonical),
					ObservedAt: observedAt,
					RawID:      "kimi-k2.6",
				}},
			},
		},
		Pricing: &Pricing{Input: 0.95, Output: 4.00, CachedInput: 0.16},
		Provenance: []Provenance{{
			SourceID:   kimiStaticSourceID,
			Authority:  string(AuthorityCanonical),
			ObservedAt: observedAt,
			RawID:      "kimi-k2.6",
		}},
	}

	return &Fragment{
		Services: []Service{service},
		Models:   []ModelRecord{k26Model},
		Offerings: []Offering{k26Offering},
	}, nil
}
