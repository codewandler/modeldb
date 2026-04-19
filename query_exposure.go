package modeldb

func (c Catalog) OfferingsByServiceAndAPIType(serviceID string, apiType APIType) []Offering {
	serviceID = normalizeKeyPart(serviceID)
	out := make([]Offering, 0)
	for _, offering := range c.Offerings {
		if offering.ServiceID == serviceID && offering.HasExposure(apiType) {
			out = append(out, offering)
		}
	}
	return out
}

func (c Catalog) ExposuresByModel(key ModelKey) []OfferingExposure {
	key = NormalizeKey(key)
	out := make([]OfferingExposure, 0)
	for _, offering := range c.Offerings {
		if offering.ModelKey != key {
			continue
		}
		out = append(out, offering.Exposures...)
	}
	return out
}
