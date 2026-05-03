package modeldb

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/bedrock/types"
)

const bedrockRuntimeSourceID = "bedrock-runtime"

type bedrockModelClient interface {
	ListFoundationModels(context.Context, *bedrock.ListFoundationModelsInput, ...func(*bedrock.Options)) (*bedrock.ListFoundationModelsOutput, error)
	ListInferenceProfiles(context.Context, *bedrock.ListInferenceProfilesInput, ...func(*bedrock.Options)) (*bedrock.ListInferenceProfilesOutput, error)
}

type BedrockRuntimeSource struct {
	Region    string
	Profile   string
	RuntimeID string
	Client    bedrockModelClient
}

func NewBedrockRuntimeSource(region string) BedrockRuntimeSource {
	return BedrockRuntimeSource{Region: region}
}

func NewBedrockRuntimeSourceFromEnv() BedrockRuntimeSource {
	return BedrockRuntimeSource{
		Region:  firstNonEmpty(os.Getenv("AWS_REGION"), os.Getenv("AWS_DEFAULT_REGION")),
		Profile: os.Getenv("AWS_PROFILE"),
	}
}

func (BedrockRuntimeSource) ID() string { return bedrockRuntimeSourceID }

func (s BedrockRuntimeSource) Fetch(ctx context.Context) (*Fragment, error) {
	region := firstNonEmpty(s.Region, os.Getenv("AWS_REGION"), os.Getenv("AWS_DEFAULT_REGION"), "us-east-1")
	client := s.Client
	if client == nil {
		opts := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(region)}
		if s.Profile != "" {
			opts = append(opts, awsconfig.WithSharedConfigProfile(s.Profile))
		}
		cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("bedrock runtime source: load AWS config: %w", err)
		}
		client = bedrock.NewFromConfig(cfg)
	}

	models, err := client.ListFoundationModels(ctx, &bedrock.ListFoundationModelsInput{})
	if err != nil {
		return nil, fmt.Errorf("bedrock runtime source: list foundation models: %w", err)
	}
	profiles, err := listBedrockInferenceProfiles(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("bedrock runtime source: list inference profiles: %w", err)
	}

	observedAt := time.Time{}
	runtimePrefix := bedrockRuntimePrefix(region)
	runtimeID := s.RuntimeID
	if runtimeID == "" {
		runtimeID = "bedrock-" + runtimePrefix
	}
	fragment := &Fragment{
		Services: []Service{bedrockService(observedAt, s.ID())},
		Runtimes: []Runtime{{
			ID:        runtimeID,
			ServiceID: "bedrock",
			Name:      "Amazon Bedrock " + region,
			Region:    region,
			Profile:   s.Profile,
			Provenance: []Provenance{{
				SourceID:   s.ID(),
				Authority:  string(AuthorityTrusted),
				ObservedAt: observedAt,
				RawID:      region,
			}},
		}},
	}

	seenOfferings := map[string]struct{}{}
	for _, item := range models.ModelSummaries {
		modelID := aws.ToString(item.ModelId)
		if modelID == "" {
			modelID = bedrockIDFromARN(aws.ToString(item.ModelArn))
		}
		if !bedrockConverseCandidate(modelID) {
			continue
		}
		appendBedrockOffering(fragment, s.ID(), observedAt, item, modelID)
		seenOfferings[modelID] = struct{}{}
	}

	type profileAccess struct {
		profileID string
		name      string
		rank      int
	}
	profileByModel := map[string]profileAccess{}
	for _, profile := range profiles {
		if profile.Status != "" && profile.Status != types.InferenceProfileStatusActive {
			continue
		}
		if profile.Type != "" && profile.Type != types.InferenceProfileTypeSystemDefined {
			continue
		}
		profileID := aws.ToString(profile.InferenceProfileId)
		if profileID == "" {
			profileID = bedrockIDFromARN(aws.ToString(profile.InferenceProfileArn))
		}
		if profileID == "" {
			continue
		}
		modelID := bedrockModelIDFromProfile(profile)
		if modelID == "" {
			modelID = bedrockStripProfilePrefix(profileID)
		}
		if !bedrockConverseCandidate(modelID) {
			continue
		}
		rank := bedrockProfileRank(profileID, runtimePrefix)
		if rank == 0 {
			continue
		}
		existing := profileByModel[modelID]
		if existing.profileID != "" && existing.rank >= rank {
			continue
		}
		profileByModel[modelID] = profileAccess{
			profileID: profileID,
			name:      aws.ToString(profile.InferenceProfileName),
			rank:      rank,
		}
	}

	for modelID, profile := range profileByModel {
		if _, ok := seenOfferings[modelID]; !ok {
			appendBedrockProfileOffering(fragment, s.ID(), observedAt, modelID, profile.name)
			seenOfferings[modelID] = struct{}{}
		}
		fragment.RuntimeAccess = append(fragment.RuntimeAccess, RuntimeAccess{
			RuntimeID:      runtimeID,
			Offering:       OfferingRef{ServiceID: "bedrock", WireModelID: modelID},
			Routable:       true,
			ResolvedWireID: profile.profileID,
			Provenance: []Provenance{{
				SourceID:   s.ID(),
				Authority:  string(AuthorityTrusted),
				ObservedAt: observedAt,
				RawID:      profile.profileID,
			}},
		})
	}

	return fragment, nil
}

func listBedrockInferenceProfiles(ctx context.Context, client bedrockModelClient) ([]types.InferenceProfileSummary, error) {
	var out []types.InferenceProfileSummary
	var nextToken *string
	for {
		resp, err := client.ListInferenceProfiles(ctx, &bedrock.ListInferenceProfilesInput{
			TypeEquals: types.InferenceProfileTypeSystemDefined,
			NextToken:  nextToken,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, resp.InferenceProfileSummaries...)
		if aws.ToString(resp.NextToken) == "" {
			return out, nil
		}
		nextToken = resp.NextToken
	}
}

func bedrockService(observedAt time.Time, sourceID string) Service {
	return Service{
		ID:       "bedrock",
		Name:     "Amazon Bedrock",
		Kind:     ServiceKindPlatform,
		Operator: "aws",
		EnvVars:  []string{"AWS_PROFILE", "AWS_REGION", "AWS_DEFAULT_REGION"},
		DocsURL:  "https://docs.aws.amazon.com/bedrock/latest/userguide/models-supported.html",
		Provenance: []Provenance{{
			SourceID:   sourceID,
			Authority:  string(AuthorityTrusted),
			ObservedAt: observedAt,
		}},
	}
}

func appendBedrockOffering(fragment *Fragment, sourceID string, observedAt time.Time, item types.FoundationModelSummary, modelID string) {
	if modelID == "" {
		return
	}
	key, ok := inferBedrockModelKey(modelID)
	if !ok {
		return
	}
	caps := bedrockConverseCapabilities(modelID, aws.ToBool(item.ResponseStreamingSupported))
	fragment.Models = append(fragment.Models, ModelRecord{
		Key:              key,
		Name:             firstNonEmpty(aws.ToString(item.ModelName), bedrockModelDisplayName(modelID)),
		Canonical:        false,
		Capabilities:     caps,
		InputModalities:  bedrockModalities(item.InputModalities),
		OutputModalities: bedrockModalities(item.OutputModalities),
		Provenance:       bedrockProvenance(sourceID, observedAt, modelID),
	})
	fragment.Offerings = append(fragment.Offerings, Offering{
		ServiceID:     "bedrock",
		WireModelID:   modelID,
		ModelKey:      key,
		Exposures:     bedrockConverseExposures(sourceID, observedAt, modelID, caps),
		PricingStatus: "unknown",
		Provenance:    bedrockProvenance(sourceID, observedAt, modelID),
	})
}

func appendBedrockProfileOffering(fragment *Fragment, sourceID string, observedAt time.Time, modelID string, name string) {
	key, ok := inferBedrockModelKey(modelID)
	if !ok {
		return
	}
	caps := bedrockConverseCapabilities(modelID, true)
	fragment.Models = append(fragment.Models, ModelRecord{
		Key:              key,
		Name:             firstNonEmpty(name, bedrockModelDisplayName(modelID)),
		Canonical:        false,
		Capabilities:     caps,
		InputModalities:  []string{"text"},
		OutputModalities: []string{"text"},
		Provenance:       bedrockProvenance(sourceID, observedAt, modelID),
	})
	fragment.Offerings = append(fragment.Offerings, Offering{
		ServiceID:     "bedrock",
		WireModelID:   modelID,
		ModelKey:      key,
		Exposures:     bedrockConverseExposures(sourceID, observedAt, modelID, caps),
		PricingStatus: "unknown",
		Provenance:    bedrockProvenance(sourceID, observedAt, modelID),
	})
}

func bedrockConverseExposures(sourceID string, observedAt time.Time, modelID string, caps Capabilities) []OfferingExposure {
	return []OfferingExposure{{
		APIType:             APITypeBedrockConverse,
		ExposedCapabilities: capabilitiesPtr(caps),
		SupportedParameters: []NormalizedParameter{ParamMessages, ParamTools, ParamToolChoice, ParamTemperature, ParamThinking},
		ParameterMappings: []ParameterMapping{
			{Normalized: ParamMessages, WireName: "messages"},
			{Normalized: ParamTools, WireName: "toolConfig.tools"},
			{Normalized: ParamToolChoice, WireName: "toolConfig.toolChoice"},
			{Normalized: ParamTemperature, WireName: "inferenceConfig.temperature"},
			{Normalized: ParamThinking, WireName: "additionalModelRequestFields.reasoning_config"},
		},
		ParameterValues: map[string][]string{
			string(ParamThinkingMode): {string(ReasoningModeEnabled), string(ReasoningModeOff)},
		},
		Provenance: bedrockProvenance(sourceID, observedAt, modelID),
	}}
}

func bedrockConverseCapabilities(modelID string, streaming bool) Capabilities {
	caps := Capabilities{
		Streaming:   streaming,
		Temperature: true,
	}
	if strings.HasPrefix(bedrockStripProfilePrefix(modelID), "anthropic.claude-") {
		caps.ToolUse = true
		caps.Reasoning = &ReasoningCapability{
			Available:   true,
			Modes:       []ReasoningMode{ReasoningModeEnabled, ReasoningModeOff},
			Efforts:     []ReasoningEffortLevel{ReasoningEffortLow, ReasoningEffortMedium, ReasoningEffortHigh, ReasoningEffortMax},
			Interleaved: true,
		}
	}
	return caps
}

func bedrockConverseCandidate(modelID string) bool {
	return strings.HasPrefix(bedrockStripProfilePrefix(modelID), "anthropic.claude-")
}

func bedrockModelIDFromProfile(profile types.InferenceProfileSummary) string {
	for _, model := range profile.Models {
		if id := bedrockIDFromARN(aws.ToString(model.ModelArn)); id != "" {
			return id
		}
	}
	return ""
}

func bedrockIDFromARN(arn string) string {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return ""
	}
	if i := strings.LastIndex(arn, "/"); i >= 0 && i+1 < len(arn) {
		return arn[i+1:]
	}
	parts := strings.Split(arn, ":")
	return parts[len(parts)-1]
}

func bedrockStripProfilePrefix(modelID string) string {
	for _, prefix := range []string{"global.", "us.", "eu.", "apac."} {
		modelID = strings.TrimPrefix(modelID, prefix)
	}
	return modelID
}

func bedrockRuntimePrefix(region string) string {
	switch {
	case strings.HasPrefix(region, "us-"):
		return "us"
	case strings.HasPrefix(region, "eu-"):
		return "eu"
	case strings.HasPrefix(region, "ap-"):
		return "apac"
	default:
		return "global"
	}
}

func bedrockProfileRank(profileID string, runtimePrefix string) int {
	switch {
	case strings.HasPrefix(profileID, runtimePrefix+"."):
		return 3
	case strings.HasPrefix(profileID, "global."):
		return 2
	case !hasBedrockProfilePrefix(profileID):
		return 1
	default:
		return 0
	}
}

func hasBedrockProfilePrefix(profileID string) bool {
	for _, prefix := range []string{"global.", "us.", "eu.", "apac."} {
		if strings.HasPrefix(profileID, prefix) {
			return true
		}
	}
	return false
}

func bedrockModalities(values []types.ModelModality) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, strings.ToLower(string(value)))
	}
	return normalizeStrings(out)
}

func bedrockModelDisplayName(modelID string) string {
	id := strings.TrimPrefix(bedrockStripProfilePrefix(modelID), "anthropic.")
	id = strings.TrimSuffix(strings.TrimSuffix(id, "-v1:0"), "-v1")
	parts := strings.Split(id, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func bedrockProvenance(sourceID string, observedAt time.Time, rawID string) []Provenance {
	return []Provenance{{
		SourceID:   sourceID,
		Authority:  string(AuthorityTrusted),
		ObservedAt: observedAt,
		RawID:      rawID,
	}}
}
