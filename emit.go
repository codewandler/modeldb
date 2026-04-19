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
	Models    []ModelRecord `json:"models,omitempty"`
	Services  []Service     `json:"services,omitempty"`
	Offerings []Offering    `json:"offerings,omitempty"`
}

func catalogArtifactFromCatalog(c Catalog) catalogArtifact {
	artifact := catalogArtifact{
		Models:    make([]ModelRecord, 0, len(c.Models)),
		Services:  make([]Service, 0, len(c.Services)),
		Offerings: make([]Offering, 0, len(c.Offerings)),
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
	return artifact
}
