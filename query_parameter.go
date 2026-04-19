package modeldb

func (c Catalog) OfferingsByServiceAPIAndParameter(serviceID string, apiType APIType, param NormalizedParameter) []Offering {
	serviceID = normalizeKeyPart(serviceID)
	out := make([]Offering, 0)
	for _, offering := range c.Offerings {
		if serviceID != "" && offering.ServiceID != serviceID {
			continue
		}
		exp := offering.Exposure(apiType)
		if exp == nil || !exp.SupportsParameter(param) {
			continue
		}
		out = append(out, offering)
	}
	return out
}
