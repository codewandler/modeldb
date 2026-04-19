package modeldb

import (
	"fmt"
	"sort"
)

func ValidateCatalog(c Catalog) error {
	for ref, offering := range c.Offerings {
		if _, ok := c.Services[offering.ServiceID]; !ok {
			return fmt.Errorf("offering %s/%s references unknown service %q", ref.ServiceID, ref.WireModelID, offering.ServiceID)
		}
		if _, ok := c.Models[offering.ModelKey]; !ok {
			return fmt.Errorf("offering %s/%s references unknown model %q", ref.ServiceID, ref.WireModelID, modelID(offering.ModelKey))
		}
		if offering.PricingStatus != "" && offering.PricingStatus != "known" && offering.PricingStatus != "free" && offering.PricingStatus != "unknown" {
			return fmt.Errorf("offering %s/%s has invalid pricing_status %q", ref.ServiceID, ref.WireModelID, offering.PricingStatus)
		}
		if offering.PricingStatus == "free" && offering.Pricing == nil {
			return fmt.Errorf("offering %s/%s marked free without explicit zero pricing", ref.ServiceID, ref.WireModelID)
		}
		if offering.PricingStatus == "free" && !pricingIsFree(offering.Pricing) {
			return fmt.Errorf("offering %s/%s marked free but has non-zero pricing", ref.ServiceID, ref.WireModelID)
		}
		seen := map[APIType]bool{}
		for _, exposure := range offering.Exposures {
			if exposure.APIType == "" {
				return fmt.Errorf("offering %s/%s has exposure with empty api_type", ref.ServiceID, ref.WireModelID)
			}
			if seen[exposure.APIType] {
				return fmt.Errorf("offering %s/%s has duplicate exposure api_type %q", ref.ServiceID, ref.WireModelID, exposure.APIType)
			}
			seen[exposure.APIType] = true
			for _, param := range exposure.SupportedParameters {
				if !validateNormalizedParameter(param) {
					return fmt.Errorf("offering %s/%s exposure %s has invalid normalized parameter %q", ref.ServiceID, ref.WireModelID, exposure.APIType, param)
				}
			}
			for _, mapping := range exposure.ParameterMappings {
				if !validateNormalizedParameter(mapping.Normalized) {
					return fmt.Errorf("offering %s/%s exposure %s has invalid parameter mapping %q", ref.ServiceID, ref.WireModelID, exposure.APIType, mapping.Normalized)
				}
			}
			if exposure.ExposedCapabilities != nil {
				if err := validateCapabilities(*exposure.ExposedCapabilities, "offering "+ref.ServiceID+"/"+ref.WireModelID+" exposure "+string(exposure.APIType)); err != nil {
					return err
				}
			}
		}
	}
	for key, model := range c.Models {
		if err := validateCapabilities(model.Capabilities, "model "+modelID(key)); err != nil {
			return err
		}
	}
	return nil
}

func ValidateResolvedCatalog(c ResolvedCatalog) error {
	if err := ValidateCatalog(c.Catalog); err != nil {
		return err
	}
	for id, runtime := range c.Runtimes {
		if _, ok := c.Services[runtime.ServiceID]; !ok {
			return fmt.Errorf("runtime %q references unknown service %q", id, runtime.ServiceID)
		}
	}
	for key, access := range c.RuntimeAccess {
		if _, ok := c.Runtimes[access.RuntimeID]; !ok {
			return fmt.Errorf("runtime access for %q references unknown runtime %q", key.WireModelID, access.RuntimeID)
		}
		if _, ok := c.Offerings[access.Offering]; !ok {
			return fmt.Errorf("runtime access for %q references unknown offering %q/%q", access.RuntimeID, access.Offering.ServiceID, access.Offering.WireModelID)
		}
	}
	for key, acquisition := range c.RuntimeAcquisition {
		if _, ok := c.Runtimes[acquisition.RuntimeID]; !ok {
			return fmt.Errorf("runtime acquisition for %q references unknown runtime %q", key.WireModelID, acquisition.RuntimeID)
		}
		if _, ok := c.Offerings[acquisition.Offering]; !ok {
			return fmt.Errorf("runtime acquisition for %q references unknown offering %q/%q", acquisition.RuntimeID, acquisition.Offering.ServiceID, acquisition.Offering.WireModelID)
		}
	}
	return nil
}

func validateCapabilities(c Capabilities, id string) error {
	if c.Reasoning == nil {
		return nil
	}
	for _, effort := range c.Reasoning.Efforts {
		switch effort {
		case ReasoningEffortMinimal, ReasoningEffortNone, ReasoningEffortLow, ReasoningEffortMedium, ReasoningEffortHigh, ReasoningEffortMax, ReasoningEffortXHigh:
		default:
			return fmt.Errorf("%s has invalid reasoning effort %q", id, effort)
		}
	}
	for _, summary := range c.Reasoning.Summaries {
		switch summary {
		case ReasoningSummaryNone, ReasoningSummaryAuto, ReasoningSummaryConcise, ReasoningSummaryDetailed:
		default:
			return fmt.Errorf("%s has invalid reasoning summary %q", id, summary)
		}
	}
	for _, mode := range c.Reasoning.Modes {
		switch mode {
		case ReasoningModeEnabled, ReasoningModeAdaptive, ReasoningModeInterleaved, ReasoningModeOff:
		default:
			return fmt.Errorf("%s has invalid reasoning mode %q", id, mode)
		}
	}
	return nil
}

func validateNormalizedParameter(p NormalizedParameter) bool {
	switch p {
	case ParamMessages, ParamThinking, ParamThinkingMode, ParamReasoningEffort, ParamReasoningSummary, ParamResponseFormat, ParamTools, ParamToolChoice, ParamTemperature, ParamSeed, ParamLogprobs, ParamParallelTools, ParamWebSearch:
		return true
	default:
		return false
	}
}


type PricingReport struct {
	Unknown []string
	Free    []string
	Known   []string
}

func AuditPricing(c Catalog) PricingReport {
	report := PricingReport{}
	for ref, offering := range c.Offerings {
		status := offering.PricingStatus
		if status == "" {
			if offering.Pricing == nil {
				status = "unknown"
			} else if pricingIsFree(offering.Pricing) {
				status = "free"
			} else {
				status = "known"
			}
		}
		id := ref.ServiceID + "/" + ref.WireModelID
		switch status {
		case "free":
			report.Free = append(report.Free, id)
		case "known":
			report.Known = append(report.Known, id)
		default:
			report.Unknown = append(report.Unknown, id)
		}
	}
	sort.Strings(report.Unknown)
	sort.Strings(report.Free)
	sort.Strings(report.Known)
	return report
}
