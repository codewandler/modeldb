package modeldb

import (
	"context"
	"time"
)

const bedrockServiceSourceID = "bedrock-service"

type BedrockServiceSource struct{}

func NewBedrockServiceSource() BedrockServiceSource { return BedrockServiceSource{} }

func (BedrockServiceSource) ID() string { return bedrockServiceSourceID }

func (BedrockServiceSource) Fetch(context.Context) (*Fragment, error) {
	observedAt := time.Time{}
	return &Fragment{Services: []Service{{
		ID:       "bedrock",
		Name:     "Amazon Bedrock",
		Kind:     ServiceKindPlatform,
		Operator: "aws",
		EnvVars:  []string{"AWS_PROFILE", "AWS_REGION", "AWS_DEFAULT_REGION"},
		DocsURL:  "https://docs.aws.amazon.com/bedrock/latest/userguide/models-supported.html",
		Provenance: []Provenance{{
			SourceID:   bedrockServiceSourceID,
			Authority:  string(AuthorityTrusted),
			ObservedAt: observedAt,
		}},
	}}}, nil
}
