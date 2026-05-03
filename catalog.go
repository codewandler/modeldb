package modeldb

type Catalog struct {
	Models             map[ModelKey]ModelRecord                     `json:"-"`
	Services           map[string]Service                           `json:"-"`
	Offerings          map[OfferingRef]Offering                     `json:"-"`
	Runtimes           map[string]Runtime                           `json:"-"`
	RuntimeAccess      map[RuntimeAccessKey]RuntimeAccess           `json:"-"`
	RuntimeAcquisition map[RuntimeAcquisitionKey]RuntimeAcquisition `json:"-"`
}

type RuntimeAccessKey struct {
	RuntimeID   string
	ServiceID   string
	WireModelID string
}

type RuntimeAcquisitionKey struct {
	RuntimeID   string
	ServiceID   string
	WireModelID string
}

type ResolvedCatalog struct {
	Catalog
}

func NewCatalog() Catalog {
	return Catalog{
		Models:             make(map[ModelKey]ModelRecord),
		Services:           make(map[string]Service),
		Offerings:          make(map[OfferingRef]Offering),
		Runtimes:           make(map[string]Runtime),
		RuntimeAccess:      make(map[RuntimeAccessKey]RuntimeAccess),
		RuntimeAcquisition: make(map[RuntimeAcquisitionKey]RuntimeAcquisition),
	}
}

func NewResolvedCatalog(base Catalog) ResolvedCatalog {
	out := ResolvedCatalog{
		Catalog: Catalog{
			Models:             make(map[ModelKey]ModelRecord, len(base.Models)),
			Services:           make(map[string]Service, len(base.Services)),
			Offerings:          make(map[OfferingRef]Offering, len(base.Offerings)),
			Runtimes:           make(map[string]Runtime, len(base.Runtimes)),
			RuntimeAccess:      make(map[RuntimeAccessKey]RuntimeAccess, len(base.RuntimeAccess)),
			RuntimeAcquisition: make(map[RuntimeAcquisitionKey]RuntimeAcquisition, len(base.RuntimeAcquisition)),
		},
	}
	for k, v := range base.Models {
		out.Models[k] = v
	}
	for k, v := range base.Services {
		out.Services[k] = v
	}
	for k, v := range base.Offerings {
		out.Offerings[k] = v
	}
	for k, v := range base.Runtimes {
		out.Runtimes[k] = v
	}
	for k, v := range base.RuntimeAccess {
		out.RuntimeAccess[k] = v
	}
	for k, v := range base.RuntimeAcquisition {
		out.RuntimeAcquisition[k] = v
	}
	return out
}
