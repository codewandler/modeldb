package modeldb

type mergeRole string

const (
	mergeRoleCreatorRoot       mergeRole = "creator_root"
	mergeRoleOfferingEnriching mergeRole = "offering_enriching"
)

func sourceMergeRole(registered RegisteredSource) mergeRole {
	switch registered.Source.ID() {
	case anthropicSourceID, minimaxSourceID, "openai-api":
		return mergeRoleCreatorRoot
	case modelsDevSourceID, "openrouter-api", openAIStaticSourceID:
		return mergeRoleOfferingEnriching
	default:
		return mergeRoleCreatorRoot
	}
}
