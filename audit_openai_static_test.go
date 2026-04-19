package modeldb

import (
	"context"
	"testing"
)

func TestOpenAIStaticAuditAgainstOpenRouter(t *testing.T) {
	staticFrag, err := NewOpenAIStaticSource().Fetch(context.Background())
	if err != nil {
		t.Fatalf("fetch openai static: %v", err)
	}
	staticCatalog := NewCatalog()
	if err := MergeCatalogFragment(&staticCatalog, staticFrag); err != nil {
		t.Fatalf("merge openai static: %v", err)
	}
	if err := ValidateCatalog(staticCatalog); err != nil {
		t.Fatalf("validate openai static: %v", err)
	}

	orSrc := NewOpenRouterSourceFromEnv()
	if orSrc.APIKey == "" {
		t.Skip("requires OPENROUTER_API_KEY for openai static cross-check")
	}
	orFrag, err := orSrc.Fetch(context.Background())
	if err != nil {
		t.Fatalf("fetch openrouter: %v", err)
	}
	orCatalog := NewCatalog()
	if err := MergeCatalogFragment(&orCatalog, orFrag); err != nil {
		t.Fatalf("merge openrouter: %v", err)
	}
	if err := ValidateCatalog(orCatalog); err != nil {
		t.Fatalf("validate openrouter: %v", err)
	}

	seen := 0
	for _, o := range orCatalog.Offerings {
		if o.ServiceID != "openrouter" || o.ModelKey.Creator != "openai" {
			continue
		}
		seen++
		staticModel, ok := staticCatalog.Models[o.ModelKey]
		if !ok {
			t.Logf("WARN openai-static missing model for openrouter offering %s -> %s", o.WireModelID, modelID(o.ModelKey))
			continue
		}
		orExp := o.Exposure(APITypeOpenAIChat)
		staticOff, ok := staticCatalog.Offerings[OfferingRef{ServiceID: "openai", WireModelID: canonicalOpenAIStaticSlug(o.WireModelID)}]
		if !ok {
			// fallback to normalized key rendered by openai inventory id expectation is not directly derivable from OR wire IDs
			for _, candidate := range staticCatalog.Offerings {
				if candidate.ServiceID == "openai" && candidate.ModelKey == o.ModelKey {
					staticOff = candidate
					ok = true
					break
				}
			}
		}
		if !ok {
			t.Logf("WARN openai-static missing offering for key %s (openrouter %s)", modelID(o.ModelKey), o.WireModelID)
			continue
		}
		staticExp := staticOff.Exposure(APITypeOpenAIResponses)
		if staticExp == nil {
			t.Logf("WARN openai-static missing openai-responses exposure for %s", modelID(o.ModelKey))
			continue
		}
		if orExp != nil {
			if orExp.SupportsParameter(ParamReasoningEffort) && !staticExp.SupportsParameter(ParamReasoningEffort) {
				t.Logf("WARN static missing reasoning_effort for %s; openrouter exposes it", modelID(o.ModelKey))
			}
			if orExp.SupportsParameter(ParamTools) && !staticExp.SupportsParameter(ParamTools) {
				t.Logf("WARN static missing tools for %s; openrouter exposes it", modelID(o.ModelKey))
			}
			if orExp.SupportsParameter(ParamResponseFormat) && !staticExp.SupportsParameter(ParamResponseFormat) {
				t.Logf("WARN static missing response_format for %s; openrouter exposes it", modelID(o.ModelKey))
			}
			if orExp.ExposedCapabilities != nil && orExp.ExposedCapabilities.Vision && !staticModel.Capabilities.Vision {
				t.Logf("WARN static marks non-vision for %s; openrouter suggests vision", modelID(o.ModelKey))
			}
			if orExp.ExposedCapabilities != nil && orExp.ExposedCapabilities.Reasoning != nil && orExp.ExposedCapabilities.Reasoning.Available {
				if staticExp.ExposedCapabilities == nil || staticExp.ExposedCapabilities.Reasoning == nil || !staticExp.ExposedCapabilities.Reasoning.Available {
					t.Logf("WARN static missing reasoning exposure for %s; openrouter suggests reasoning", modelID(o.ModelKey))
				}
				for _, effort := range orExp.ExposedCapabilities.Reasoning.Efforts {
					if staticExp.ExposedCapabilities == nil || !staticExp.ExposedCapabilities.SupportsReasoningEffort(effort) {
						t.Logf("WARN static missing reasoning effort %q for %s; openrouter exposes it", effort, modelID(o.ModelKey))
					}
				}
			}
		}
	}
	if seen == 0 {
		t.Log("WARN no OpenRouter OpenAI overlap found for audit")
	}
}

func canonicalOpenAIStaticSlug(wireID string) string {
	if _, ok := inferOpenAIModelKey(wireID); ok {
		return wireID
	}
	const prefix = "openai/"
	if len(wireID) > len(prefix) && wireID[:len(prefix)] == prefix {
		return wireID[len(prefix):]
	}
	return wireID
}
