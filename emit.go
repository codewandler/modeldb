package modeldb

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

func SaveJSON(filePath string, c Catalog) error {
	artifact := stripArtifactVolatileFields(catalogArtifactFromCatalog(c))
	data, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0o644)
}

func LoadJSON(filePath string) (Catalog, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return Catalog{}, err
	}
	return LoadJSONBytes(data)
}

func LoadJSONBytes(data []byte) (Catalog, error) {
	var artifact catalogArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return Catalog{}, err
	}
	c := NewCatalog()
	for _, service := range artifact.Services {
		c.Services[normalizeKeyPart(service.ID)] = service
	}
	for _, model := range artifact.Models {
		model.Key = NormalizeKey(model.Key)
		c.Models[model.Key] = model
	}
	for _, offering := range artifact.Offerings {
		offering.ServiceID = normalizeKeyPart(offering.ServiceID)
		offering.ModelKey = NormalizeKey(offering.ModelKey)
		c.Offerings[OfferingRef{ServiceID: offering.ServiceID, WireModelID: offering.WireModelID}] = offering
	}
	for _, runtime := range artifact.Runtimes {
		runtime.ID = normalizeKeyPart(runtime.ID)
		c.Runtimes[runtime.ID] = runtime
	}
	for _, access := range artifact.RuntimeAccess {
		access.RuntimeID = normalizeKeyPart(access.RuntimeID)
		access.Offering.ServiceID = normalizeKeyPart(access.Offering.ServiceID)
		key := RuntimeAccessKey{
			RuntimeID:   access.RuntimeID,
			ServiceID:   access.Offering.ServiceID,
			WireModelID: access.Offering.WireModelID,
		}
		c.RuntimeAccess[key] = access
	}
	for _, acquisition := range artifact.RuntimeAcquisition {
		acquisition.RuntimeID = normalizeKeyPart(acquisition.RuntimeID)
		acquisition.Offering.ServiceID = normalizeKeyPart(acquisition.Offering.ServiceID)
		key := RuntimeAcquisitionKey{
			RuntimeID:   acquisition.RuntimeID,
			ServiceID:   acquisition.Offering.ServiceID,
			WireModelID: acquisition.Offering.WireModelID,
		}
		c.RuntimeAcquisition[key] = acquisition
	}
	if err := ValidateCatalog(c); err != nil {
		return Catalog{}, fmt.Errorf("validate catalog: %w", err)
	}
	return c, nil
}

func stripArtifactVolatileFields(artifact catalogArtifact) catalogArtifact {
	for i := range artifact.Models {
		artifact.Models[i].Provenance = stripProvenanceTimestamps(artifact.Models[i].Provenance)
	}
	for i := range artifact.Services {
		artifact.Services[i].Provenance = stripProvenanceTimestamps(artifact.Services[i].Provenance)
	}
	for i := range artifact.Offerings {
		artifact.Offerings[i].Provenance = stripProvenanceTimestamps(artifact.Offerings[i].Provenance)
		for j := range artifact.Offerings[i].Exposures {
			artifact.Offerings[i].Exposures[j].Provenance = stripProvenanceTimestamps(artifact.Offerings[i].Exposures[j].Provenance)
		}
	}
	for i := range artifact.Runtimes {
		artifact.Runtimes[i].Provenance = stripProvenanceTimestamps(artifact.Runtimes[i].Provenance)
	}
	for i := range artifact.RuntimeAccess {
		artifact.RuntimeAccess[i].Provenance = stripProvenanceTimestamps(artifact.RuntimeAccess[i].Provenance)
	}
	for i := range artifact.RuntimeAcquisition {
		artifact.RuntimeAcquisition[i].Provenance = stripProvenanceTimestamps(artifact.RuntimeAcquisition[i].Provenance)
	}
	return artifact
}

func stripProvenanceTimestamps(items []Provenance) []Provenance {
	if len(items) == 0 {
		return nil
	}
	out := make([]Provenance, len(items))
	copy(out, items)
	for i := range out {
		out[i].ObservedAt = time.Time{}
	}
	return out
}

type catalogArtifact struct {
	Models             []ModelRecord        `json:"models,omitempty"`
	Services           []Service            `json:"services,omitempty"`
	Offerings          []Offering           `json:"offerings,omitempty"`
	Runtimes           []Runtime            `json:"runtimes,omitempty"`
	RuntimeAccess      []RuntimeAccess      `json:"runtime_access,omitempty"`
	RuntimeAcquisition []RuntimeAcquisition `json:"runtime_acquisition,omitempty"`
}

func catalogArtifactFromCatalog(c Catalog) catalogArtifact {
	artifact := catalogArtifact{
		Models:             make([]ModelRecord, 0, len(c.Models)),
		Services:           make([]Service, 0, len(c.Services)),
		Offerings:          make([]Offering, 0, len(c.Offerings)),
		Runtimes:           make([]Runtime, 0, len(c.Runtimes)),
		RuntimeAccess:      make([]RuntimeAccess, 0, len(c.RuntimeAccess)),
		RuntimeAcquisition: make([]RuntimeAcquisition, 0, len(c.RuntimeAcquisition)),
	}
	for _, model := range c.Models {
		artifact.Models = append(artifact.Models, model)
	}
	for _, service := range c.Services {
		artifact.Services = append(artifact.Services, service)
	}
	for _, offering := range c.Offerings {
		artifact.Offerings = append(artifact.Offerings, offering)
	}
	for _, runtime := range c.Runtimes {
		artifact.Runtimes = append(artifact.Runtimes, runtime)
	}
	for _, access := range c.RuntimeAccess {
		artifact.RuntimeAccess = append(artifact.RuntimeAccess, access)
	}
	for _, acquisition := range c.RuntimeAcquisition {
		artifact.RuntimeAcquisition = append(artifact.RuntimeAcquisition, acquisition)
	}
	sort.Slice(artifact.Models, func(i, j int) bool {
		return modelID(artifact.Models[i].Key) < modelID(artifact.Models[j].Key)
	})
	sort.Slice(artifact.Services, func(i, j int) bool {
		return artifact.Services[i].ID < artifact.Services[j].ID
	})
	sort.Slice(artifact.Offerings, func(i, j int) bool {
		if artifact.Offerings[i].ServiceID != artifact.Offerings[j].ServiceID {
			return artifact.Offerings[i].ServiceID < artifact.Offerings[j].ServiceID
		}
		return artifact.Offerings[i].WireModelID < artifact.Offerings[j].WireModelID
	})
	sort.Slice(artifact.Runtimes, func(i, j int) bool {
		return artifact.Runtimes[i].ID < artifact.Runtimes[j].ID
	})
	sort.Slice(artifact.RuntimeAccess, func(i, j int) bool {
		left := artifact.RuntimeAccess[i]
		right := artifact.RuntimeAccess[j]
		if left.RuntimeID != right.RuntimeID {
			return left.RuntimeID < right.RuntimeID
		}
		if left.Offering.ServiceID != right.Offering.ServiceID {
			return left.Offering.ServiceID < right.Offering.ServiceID
		}
		return left.Offering.WireModelID < right.Offering.WireModelID
	})
	sort.Slice(artifact.RuntimeAcquisition, func(i, j int) bool {
		left := artifact.RuntimeAcquisition[i]
		right := artifact.RuntimeAcquisition[j]
		if left.RuntimeID != right.RuntimeID {
			return left.RuntimeID < right.RuntimeID
		}
		if left.Offering.ServiceID != right.Offering.ServiceID {
			return left.Offering.ServiceID < right.Offering.ServiceID
		}
		return left.Offering.WireModelID < right.Offering.WireModelID
	})
	return artifact
}

func FilterCatalogByPricingStatus(c Catalog, excludeStatus ...string) Catalog {
	out := NewCatalog()
	excluded := map[string]bool{}
	for _, status := range excludeStatus {
		excluded[status] = true
	}
	for k, v := range c.Services {
		out.Services[k] = v
	}
	for k, v := range c.Runtimes {
		out.Runtimes[k] = v
	}
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
		if excluded[status] {
			continue
		}
		out.Offerings[ref] = offering
		out.Models[offering.ModelKey] = c.Models[offering.ModelKey]
	}
	for key, access := range c.RuntimeAccess {
		if _, ok := out.Offerings[access.Offering]; ok {
			out.RuntimeAccess[key] = access
		}
	}
	for key, acquisition := range c.RuntimeAcquisition {
		if _, ok := out.Offerings[acquisition.Offering]; ok {
			out.RuntimeAcquisition[key] = acquisition
		}
	}
	return out
}
