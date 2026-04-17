package modeldb

import (
	"fmt"
	"sort"
	"strings"
)

type ModelSelector struct {
	ID          string
	Name        string
	Creator     string
	ServiceID   string
	Family      string
	Series      string
	Version     string
	ReleaseDate string
}

type ModelSelection struct {
	Selector  ModelSelector
	Model     ModelRecord
	Offerings []ServiceOffering
}

type ModelMatch struct {
	Model     ModelRecord
	Offerings []ServiceOffering
}

type ServiceOffering struct {
	Service  Service
	Model    ModelRecord
	Offering Offering
}

type ModelSelectorNotFoundError struct {
	Selector ModelSelector
}

func (e *ModelSelectorNotFoundError) Error() string {
	return fmt.Sprintf(
		"no catalog model matches id=%q name=%q creator=%q service=%q family=%q series=%q version=%q release_date=%q",
		e.Selector.ID,
		e.Selector.Name,
		e.Selector.Creator,
		e.Selector.ServiceID,
		e.Selector.Family,
		e.Selector.Series,
		e.Selector.Version,
		e.Selector.ReleaseDate,
	)
}

type AmbiguousModelSelectorError struct {
	Selector   ModelSelector
	Candidates []ModelRecord
}

func (e *AmbiguousModelSelectorError) Error() string {
	return fmt.Sprintf(
		"ambiguous catalog model selector id=%q name=%q creator=%q service=%q family=%q series=%q version=%q release_date=%q (%d candidates)",
		e.Selector.ID,
		e.Selector.Name,
		e.Selector.Creator,
		e.Selector.ServiceID,
		e.Selector.Family,
		e.Selector.Series,
		e.Selector.Version,
		e.Selector.ReleaseDate,
		len(e.Candidates),
	)
}

func ParseModelSelector(name, version string) (ModelSelector, error) {
	selector := normalizeModelSelector(ModelSelector{Name: name, Version: version})
	if selector.Name == "" {
		return ModelSelector{}, fmt.Errorf("model name is required")
	}
	return selector, nil
}

func (c Catalog) FindModels(sel ModelSelector) []ModelMatch {
	sel = normalizeModelSelector(sel)

	matches := make([]ModelMatch, 0)
	for _, model := range c.Models {
		if !matchesModelSelector(model, sel) {
			continue
		}
		offerings := selectOfferingsForModel(c, model, sel)
		if sel.ServiceID != "" && len(offerings) == 0 {
			continue
		}
		matches = append(matches, ModelMatch{Model: model, Offerings: offerings})
	}

	sort.Slice(matches, func(i, j int) bool {
		left := formatModelID(matches[i].Model.Key)
		right := formatModelID(matches[j].Model.Key)
		if left != right {
			return left < right
		}
		return matches[i].Model.Name < matches[j].Model.Name
	})
	return matches
}

func (c Catalog) SelectModel(sel ModelSelector) (ModelRecord, error) {
	sel = normalizeModelSelector(sel)
	if selectorIsZero(sel) {
		return ModelRecord{}, fmt.Errorf("at least one selector field is required")
	}

	matches := c.FindModels(sel)
	switch len(matches) {
	case 0:
		return ModelRecord{}, &ModelSelectorNotFoundError{Selector: sel}
	case 1:
		return matches[0].Model, nil
	default:
		candidates := make([]ModelRecord, 0, len(matches))
		for _, match := range matches {
			candidates = append(candidates, match.Model)
		}
		return ModelRecord{}, &AmbiguousModelSelectorError{Selector: sel, Candidates: candidates}
	}
}

func (c Catalog) SelectOfferingsByModel(sel ModelSelector) (ModelSelection, error) {
	sel = normalizeModelSelector(sel)
	model, err := c.SelectModel(sel)
	if err != nil {
		return ModelSelection{}, err
	}
	return ModelSelection{
		Selector:  sel,
		Model:     model,
		Offerings: selectOfferingsForModel(c, model, sel),
	}, nil
}

func selectOfferingsForModel(c Catalog, model ModelRecord, sel ModelSelector) []ServiceOffering {
	selected := make(map[string]ServiceOffering)
	targetLine := LineKey(model.Key)
	for _, offering := range c.Offerings {
		if LineKey(offering.ModelKey) != targetLine {
			continue
		}
		if sel.ServiceID != "" && offering.ServiceID != sel.ServiceID {
			continue
		}
		service, ok := c.Services[offering.ServiceID]
		if !ok {
			continue
		}
		candidate := ServiceOffering{Service: service, Model: model, Offering: offering}
		current, ok := selected[offering.ServiceID]
		if !ok || preferServiceOffering(sel, candidate, current) {
			selected[offering.ServiceID] = candidate
		}
	}

	services := make([]string, 0, len(selected))
	for serviceID := range selected {
		services = append(services, serviceID)
	}
	sort.Strings(services)

	offerings := make([]ServiceOffering, 0, len(services))
	for _, serviceID := range services {
		offerings = append(offerings, selected[serviceID])
	}
	return offerings
}

func selectorIsZero(sel ModelSelector) bool {
	return sel.ID == "" && sel.Name == "" && sel.Creator == "" && sel.ServiceID == "" && sel.Family == "" && sel.Series == "" && sel.Version == "" && sel.ReleaseDate == ""
}

func normalizeModelSelector(sel ModelSelector) ModelSelector {
	sel.ID = normalizeKeyPart(sel.ID)
	sel.Name = normalizeKeyPart(sel.Name)
	sel.Creator = normalizeKeyPart(sel.Creator)
	sel.ServiceID = normalizeKeyPart(sel.ServiceID)
	sel.Family = normalizeKeyPart(sel.Family)
	sel.Series = normalizeKeyPart(sel.Series)
	sel.Version = normalizeKeyPart(sel.Version)
	sel.ReleaseDate = normalizeDate(sel.ReleaseDate)
	return sel
}

func matchesModelSelector(model ModelRecord, sel ModelSelector) bool {
	key := NormalizeKey(model.Key)
	if sel.ID != "" && formatModelID(key) != sel.ID {
		return false
	}
	if sel.Creator != "" && key.Creator != sel.Creator {
		return false
	}
	if sel.Family != "" && key.Family != sel.Family {
		return false
	}
	if sel.Series != "" && key.Series != sel.Series {
		return false
	}
	if sel.Version != "" && key.Version != sel.Version {
		return false
	}
	if sel.ReleaseDate != "" && key.ReleaseDate != sel.ReleaseDate {
		return false
	}
	if sel.Name == "" {
		return true
	}

	for _, candidate := range candidateModelNames(model) {
		if candidate == sel.Name {
			return true
		}
	}
	return false
}

func candidateModelNames(model ModelRecord) []string {
	key := NormalizeKey(model.Key)
	seen := make(map[string]struct{})
	ordered := make([]string, 0, len(model.Aliases)+6)
	add := func(value string) {
		value = normalizeKeyPart(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		ordered = append(ordered, value)
	}

	add(key.Creator)
	add(key.Family)
	add(key.Series)
	add(key.Variant)
	if key.Family != "" && key.Series != "" {
		add(key.Family + "-" + key.Series)
	}
	if key.Creator != "" && key.Family != "" {
		add(key.Creator + "-" + key.Family)
	}
	if key.Creator != "" && key.Family != "" && key.Series != "" {
		add(key.Creator + "-" + key.Family + "-" + key.Series)
	}
	add(model.Name)
	for _, alias := range model.Aliases {
		add(alias)
	}
	return ordered
}

func preferServiceOffering(sel ModelSelector, candidate, current ServiceOffering) bool {
	candidateHit := selectorMatchesOffering(sel, candidate.Offering)
	currentHit := selectorMatchesOffering(sel, current.Offering)
	if candidateHit != currentHit {
		return candidateHit
	}

	candidateDated := offeringUsesDatedWireID(candidate.Offering)
	currentDated := offeringUsesDatedWireID(current.Offering)
	if candidateDated != currentDated {
		return !candidateDated
	}

	if len(candidate.Offering.Aliases) != len(current.Offering.Aliases) {
		return len(candidate.Offering.Aliases) > len(current.Offering.Aliases)
	}

	return candidate.Offering.WireModelID < current.Offering.WireModelID
}

func selectorMatchesOffering(sel ModelSelector, offering Offering) bool {
	if sel.Name == "" {
		return false
	}
	if normalizeKeyPart(offering.WireModelID) == sel.Name {
		return true
	}
	for _, alias := range offering.Aliases {
		if normalizeKeyPart(alias) == sel.Name {
			return true
		}
	}
	return false
}

func offeringUsesDatedWireID(offering Offering) bool {
	releaseDate := normalizeDate(offering.ModelKey.ReleaseDate)
	if releaseDate == "" {
		return false
	}
	wireID := strings.ToLower(offering.WireModelID)
	compactDate := strings.ReplaceAll(releaseDate, "-", "")
	return strings.Contains(wireID, releaseDate) || strings.Contains(wireID, compactDate)
}

func formatModelID(key ModelKey) string {
	if releaseID := ReleaseID(key); releaseID != "" {
		return releaseID
	}
	return LineID(key)
}
