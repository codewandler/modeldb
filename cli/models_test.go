package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	modeldb "github.com/codewandler/modeldb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelsCommand_ServiceImpliesOfferings(t *testing.T) {
	var out bytes.Buffer
	cmd := NewModelsCommand(ModelsCommandOptions{
		IO: IO{Out: &out, Err: &out},
		LoadBaseCatalog: func(context.Context) (modeldb.Catalog, error) {
			return testCatalog(), nil
		},
	})
	cmd.SetArgs([]string{"--service", "openrouter", "--name", "sonnet", "--version", "4.5"})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "WIRE MODEL ID")
	assert.Contains(t, out.String(), "openrouter")
	assert.Contains(t, out.String(), "anthropic/claude-sonnet-4.5")
}

func TestModelsCommand_NoFlagsListsAllModels(t *testing.T) {
	var out bytes.Buffer
	cmd := NewModelsCommand(ModelsCommandOptions{
		IO: IO{Out: &out, Err: &out},
		LoadBaseCatalog: func(context.Context) (modeldb.Catalog, error) {
			return testCatalog(), nil
		},
	})
	cmd.SetArgs(nil)

	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "MODEL")
	assert.Contains(t, out.String(), "anthropic/claude/sonnet/4.5@2025-09-29")
	assert.Contains(t, out.String(), "anthropic, openrouter")
	assert.NotContains(t, out.String(), "No models found.")
}

func TestModelsCommand_QuerySearchesAcrossModelIDs(t *testing.T) {
	var out bytes.Buffer
	cmd := NewModelsCommand(ModelsCommandOptions{
		IO: IO{Out: &out, Err: &out},
		LoadBaseCatalog: func(context.Context) (modeldb.Catalog, error) {
			return testCatalog(), nil
		},
	})
	cmd.SetArgs([]string{"--query", "gpt-5.4"})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "openai/gpt/5.4")
	assert.NotContains(t, out.String(), "No models found.")
}

func TestModelsCommand_QuerySearchesOfferingWireIDs(t *testing.T) {
	var out bytes.Buffer
	cmd := NewModelsCommand(ModelsCommandOptions{
		IO: IO{Out: &out, Err: &out},
		LoadBaseCatalog: func(context.Context) (modeldb.Catalog, error) {
			return testCatalog(), nil
		},
	})
	cmd.SetArgs([]string{"--query", "claude-sonnet-4.5"})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "anthropic/claude/sonnet/4.5@2025-09-29")
}

func TestModelsCommand_HidesModelsWithoutOfferings(t *testing.T) {
	var out bytes.Buffer
	cmd := NewModelsCommand(ModelsCommandOptions{
		IO: IO{Out: &out, Err: &out},
		LoadBaseCatalog: func(context.Context) (modeldb.Catalog, error) {
			c := testCatalog()
			orphan := modeldb.NormalizeKey(modeldb.ModelKey{Creator: "openrouter", Family: "sherlock", Variant: "dash-alpha", ReleaseDate: "2025-11-15"})
			c.Models[orphan] = modeldb.ModelRecord{Key: orphan, Name: "Sherlock Dash Alpha"}
			return c, nil
		},
	})
	cmd.SetArgs([]string{"--query", "sherlock"})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "No models found.")
	assert.NotContains(t, out.String(), "openrouter/sherlock/dash-alpha@2025-11-15")
}

func TestModelsCommand_SelectRequiresSingleMatch(t *testing.T) {
	var out bytes.Buffer
	cmd := NewModelsCommand(ModelsCommandOptions{
		IO: IO{Out: &out, Err: &out},
		LoadBaseCatalog: func(context.Context) (modeldb.Catalog, error) {
			return testCatalogWithAmbiguity(), nil
		},
	})
	cmd.SetArgs([]string{"--name", "sonnet", "--version", "4.6", "--select"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous catalog model selector")
}

func TestModelsCommand_JSONImpliesDetailedRecords(t *testing.T) {
	var out bytes.Buffer
	cmd := NewModelsCommand(ModelsCommandOptions{
		IO: IO{Out: &out, Err: &out},
		LoadBaseCatalog: func(context.Context) (modeldb.Catalog, error) {
			return testCatalog(), nil
		},
	})
	cmd.SetArgs([]string{"--name", "sonnet", "--version", "4.5", "--json"})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "\"model\"")
	assert.Contains(t, out.String(), "\"offerings\"")
	assert.Contains(t, out.String(), "claude-sonnet-4-5-20250929")
}

func TestModelsCommand_DetailsShowsVariant(t *testing.T) {
	var out bytes.Buffer
	cmd := NewModelsCommand(ModelsCommandOptions{
		IO: IO{Out: &out, Err: &out},
		LoadBaseCatalog: func(context.Context) (modeldb.Catalog, error) {
			return testCatalog(), nil
		},
	})
	cmd.SetArgs([]string{"--id", "openai/gpt/realtime/mini", "--details"})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "variant: mini")
}

func TestModelsCommand_OfferingsShowsAPITypes(t *testing.T) {
	var out bytes.Buffer
	cmd := NewModelsCommand(ModelsCommandOptions{
		IO: IO{Out: &out, Err: &out},
		LoadBaseCatalog: func(context.Context) (modeldb.Catalog, error) {
			return testCatalog(), nil
		},
	})
	cmd.SetArgs([]string{"--offerings", "--name", "sonnet", "--version", "4.5"})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "API TYPES")
	assert.Contains(t, out.String(), "openai-responses")
}

func TestModelsCommand_APITypeFiltersOfferings(t *testing.T) {
	var out bytes.Buffer
	cmd := NewModelsCommand(ModelsCommandOptions{
		IO: IO{Out: &out, Err: &out},
		LoadBaseCatalog: func(context.Context) (modeldb.Catalog, error) {
			return testCatalog(), nil
		},
	})
	cmd.SetArgs([]string{"--api-type", "openai-responses", "--offerings", "--name", "sonnet", "--version", "4.5"})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "openrouter")
	assert.NotContains(t, out.String(), "  anthropic  ")
}

func TestModelsCommand_ParameterFiltersOfferings(t *testing.T) {
	var out bytes.Buffer
	cmd := NewModelsCommand(ModelsCommandOptions{
		IO:              IO{Out: &out, Err: &out},
		LoadBaseCatalog: func(context.Context) (modeldb.Catalog, error) { return testCatalog(), nil },
	})
	cmd.SetArgs([]string{"--parameter", "reasoning_effort", "--offerings", "--name", "sonnet", "--version", "4.5"})
	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "openrouter")
}

func TestModelsCommand_IDCompletion(t *testing.T) {
	values := completeModelIDs(testCatalog())
	joined := strings.Join(values, "\n")
	assert.Contains(t, joined, "anthropic/claude/sonnet/4.5@2025-09-29")
}

func testCatalog() modeldb.Catalog {
	c := modeldb.NewCatalog()
	sonnet := modeldb.NormalizeKey(modeldb.ModelKey{Creator: "anthropic", Family: "claude", Series: "sonnet", Version: "4.5", ReleaseDate: "2025-09-29"})
	gpt := modeldb.NormalizeKey(modeldb.ModelKey{Creator: "openai", Family: "gpt", Version: "5.4"})
	realtimeMini := modeldb.NormalizeKey(modeldb.ModelKey{Creator: "openai", Family: "gpt", Version: "realtime", Variant: "mini"})
	c.Models[sonnet] = modeldb.ModelRecord{Key: sonnet, Name: "Claude Sonnet 4.5", Aliases: []string{"sonnet", "claude-sonnet-4-5"}, Limits: modeldb.Limits{ContextWindow: 1000000, MaxOutput: 64000}, Capabilities: modeldb.Capabilities{Reasoning: &modeldb.ReasoningCapability{Available: true}, StructuredOutput: true}, ReferencePricing: &modeldb.Pricing{Input: 3.0, Output: 15.0}}
	c.Models[gpt] = modeldb.ModelRecord{Key: gpt, Name: "GPT-5.4", Aliases: []string{"gpt-5.4"}, Limits: modeldb.Limits{ContextWindow: 400000, MaxOutput: 128000}}
	c.Models[realtimeMini] = modeldb.ModelRecord{Key: realtimeMini, Name: "GPT Realtime Mini"}
	c.Services["anthropic"] = modeldb.Service{ID: "anthropic", Name: "Anthropic"}
	c.Services["openrouter"] = modeldb.Service{ID: "openrouter", Name: "OpenRouter"}
	c.Services["openai"] = modeldb.Service{ID: "openai", Name: "OpenAI"}
	c.Offerings[modeldb.OfferingRef{ServiceID: "anthropic", WireModelID: "claude-sonnet-4-5-20250929"}] = modeldb.Offering{ServiceID: "anthropic", WireModelID: "claude-sonnet-4-5-20250929", ModelKey: sonnet, Exposures: []modeldb.OfferingExposure{{APIType: modeldb.APITypeAnthropicMessages}}}
	c.Offerings[modeldb.OfferingRef{ServiceID: "openrouter", WireModelID: "anthropic/claude-sonnet-4.5"}] = modeldb.Offering{ServiceID: "openrouter", WireModelID: "anthropic/claude-sonnet-4.5", ModelKey: sonnet, Exposures: []modeldb.OfferingExposure{{APIType: modeldb.APITypeOpenAIResponses, SupportedParameters: []modeldb.NormalizedParameter{modeldb.ParamTools, modeldb.ParamReasoningEffort}, ParameterMappings: []modeldb.ParameterMapping{{Normalized: modeldb.ParamTools, WireName: "tools"}}}}}
	c.Offerings[modeldb.OfferingRef{ServiceID: "openai", WireModelID: "gpt-5.4"}] = modeldb.Offering{ServiceID: "openai", WireModelID: "gpt-5.4", ModelKey: gpt, Exposures: []modeldb.OfferingExposure{{APIType: modeldb.APITypeDefault}}}
	c.Offerings[modeldb.OfferingRef{ServiceID: "openai", WireModelID: "gpt-realtime-mini"}] = modeldb.Offering{ServiceID: "openai", WireModelID: "gpt-realtime-mini", ModelKey: realtimeMini, Exposures: []modeldb.OfferingExposure{{APIType: modeldb.APITypeDefault}}}
	return c
}

func testCatalogWithAmbiguity() modeldb.Catalog {
	c := modeldb.NewCatalog()
	line := modeldb.NormalizeKey(modeldb.ModelKey{Creator: "anthropic", Family: "claude", Series: "sonnet", Version: "4.6"})
	release := modeldb.NormalizeKey(modeldb.ModelKey{Creator: "anthropic", Family: "claude", Series: "sonnet", Version: "4.6", ReleaseDate: "2026-02-17"})
	c.Models[line] = modeldb.ModelRecord{Key: line, Name: "Claude Sonnet 4.6", Aliases: []string{"sonnet"}}
	c.Models[release] = modeldb.ModelRecord{Key: release, Name: "Claude Sonnet 4.6 (2026-02-17)", Aliases: []string{"sonnet"}}
	c.Services["anthropic"] = modeldb.Service{ID: "anthropic", Name: "Anthropic"}
	c.Services["openrouter"] = modeldb.Service{ID: "openrouter", Name: "OpenRouter"}
	c.Offerings[modeldb.OfferingRef{ServiceID: "anthropic", WireModelID: "claude-sonnet-4-6"}] = modeldb.Offering{ServiceID: "anthropic", WireModelID: "claude-sonnet-4-6", ModelKey: line}
	c.Offerings[modeldb.OfferingRef{ServiceID: "openrouter", WireModelID: "anthropic/claude-sonnet-4.6"}] = modeldb.Offering{ServiceID: "openrouter", WireModelID: "anthropic/claude-sonnet-4.6", ModelKey: release}
	return c
}
