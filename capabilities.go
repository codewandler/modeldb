package modeldb

func capabilitiesPtr(c Capabilities) *Capabilities {
	return &c
}

func (c Capabilities) SupportsReasoning() bool {
	return c.Reasoning != nil && c.Reasoning.Available
}

func (c Capabilities) SupportsReasoningEffort(level ReasoningEffortLevel) bool {
	if c.Reasoning == nil {
		return false
	}
	for _, effort := range c.Reasoning.Efforts {
		if effort == level {
			return true
		}
	}
	return false
}

func (o Offering) Exposure(apiType APIType) *OfferingExposure {
	for i := range o.Exposures {
		if o.Exposures[i].APIType == apiType {
			return &o.Exposures[i]
		}
	}
	return nil
}

func (c Capabilities) SupportsReasoningMode(mode ReasoningMode) bool {
	if c.Reasoning == nil {
		return false
	}
	for _, m := range c.Reasoning.Modes {
		if m == mode {
			return true
		}
	}
	return false
}

func (o Offering) HasExposure(apiType APIType) bool {
	return o.Exposure(apiType) != nil
}

func (e OfferingExposure) SupportsParameter(name NormalizedParameter) bool {
	for _, p := range e.SupportedParameters {
		if p == name {
			return true
		}
	}
	return false
}

func (e OfferingExposure) SupportsParameterValue(name, value string) bool {
	values, ok := e.ParameterValues[name]
	if !ok {
		return false
	}
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func (o Offering) ExposureRef(apiType APIType) ExposureRef {
	return ExposureRef{ServiceID: o.ServiceID, WireModelID: o.WireModelID, APIType: apiType}
}
