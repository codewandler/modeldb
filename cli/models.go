package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	modeldb "github.com/codewandler/modeldb"
	"github.com/spf13/cobra"
)

type ModelsCommandOptions struct {
	IO              IO
	LoadBaseCatalog func(ctx context.Context) (modeldb.Catalog, error)
}

type modelsFlags struct {
	id          string
	query       string
	name        string
	creator     string
	service     string
	apiType     string
	parameter   string
	family      string
	series      string
	version     string
	releaseDate string
	offerings   bool
	details     bool
	selectOne   bool
	json        bool
}

type jsonModelMatch struct {
	ID        string                `json:"id"`
	Model     modeldb.ModelRecord   `json:"model"`
	Services  []string              `json:"services"`
	Offerings []jsonServiceOffering `json:"offerings"`
}

type jsonServiceOffering struct {
	Service  modeldb.Service  `json:"service"`
	Offering modeldb.Offering `json:"offering"`
}

func NewModelsCommand(opts ModelsCommandOptions) *cobra.Command {
	ioCfg := opts.IO.withDefaults()
	if opts.LoadBaseCatalog == nil {
		opts.LoadBaseCatalog = func(context.Context) (modeldb.Catalog, error) {
			return modeldb.LoadBuiltIn()
		}
	}

	var flags modelsFlags
	cmd := &cobra.Command{
		Use:   "models",
		Short: "List logical models from the built-in catalog",
		RunE: func(cmd *cobra.Command, args []string) error {
			catalog, err := opts.LoadBaseCatalog(cmd.Context())
			if err != nil {
				return fmt.Errorf("load built-in catalog: %w", err)
			}
			selector := modeldb.ModelSelector{
				ID:          flags.id,
				Name:        flags.name,
				Creator:     flags.creator,
				ServiceID:   flags.service,
				APIType:     modeldb.APIType(flags.apiType),
				Parameter:   modeldb.NormalizedParameter(flags.parameter),
				Family:      flags.family,
				Series:      flags.series,
				Version:     flags.version,
				ReleaseDate: flags.releaseDate,
			}
			if flags.service != "" {
				flags.offerings = true
			}
			matches := catalog.FindModels(selector)
			matches = filterMatchesByQuery(matches, flags.query)
			matches = filterMatchesWithOfferings(matches)
			if flags.selectOne {
				if len(matches) == 0 {
					return &modeldb.ModelSelectorNotFoundError{Selector: selector}
				}
				if len(matches) > 1 {
					candidates := make([]modeldb.ModelRecord, 0, len(matches))
					for _, match := range matches {
						candidates = append(candidates, match.Model)
					}
					return &modeldb.AmbiguousModelSelectorError{Selector: selector, Candidates: candidates}
				}
			}
			if flags.json {
				return printModelsJSON(ioCfg.Out, matches)
			}
			if flags.details {
				printModelsDetails(ioCfg.Out, matches, flags.offerings)
				return nil
			}
			if flags.offerings {
				printModelsOfferings(ioCfg.Out, matches)
				return nil
			}
			printModelsSummary(ioCfg.Out, matches)
			return nil
		},
	}
	cmd.SetOut(ioCfg.Out)
	cmd.SetErr(ioCfg.Err)
	cmd.Flags().StringVar(&flags.id, "id", "", "Exact canonical model ID")
	cmd.Flags().StringVarP(&flags.query, "query", "q", "", "Substring search across model IDs, names, aliases, services, and offering wire IDs")
	cmd.Flags().StringVar(&flags.name, "name", "", "Logical model name or alias")
	cmd.Flags().StringVar(&flags.creator, "creator", "", "Filter by creator")
	cmd.Flags().StringVar(&flags.service, "service", "", "Filter by service and expand offerings")
	cmd.Flags().StringVar(&flags.apiType, "api-type", "", "Filter offerings/exposures by API type")
	cmd.Flags().StringVar(&flags.parameter, "parameter", "", "Filter offerings/exposures by normalized parameter")
	cmd.Flags().StringVar(&flags.family, "family", "", "Filter by model family")
	cmd.Flags().StringVar(&flags.series, "series", "", "Filter by model series")
	cmd.Flags().StringVar(&flags.version, "version", "", "Filter by model version")
	cmd.Flags().StringVar(&flags.releaseDate, "release-date", "", "Filter by model release date")
	cmd.Flags().BoolVar(&flags.offerings, "offerings", false, "Expand matching models to per-service offerings")
	cmd.Flags().BoolVar(&flags.details, "details", false, "Show richer model details in text output")
	cmd.Flags().BoolVar(&flags.selectOne, "select", false, "Require exactly one logical model match")
	cmd.Flags().BoolVar(&flags.json, "json", false, "Emit machine-readable JSON output")
	registerModelsCompletions(cmd, opts.LoadBaseCatalog)
	return cmd
}

func filterMatchesByQuery(matches []modeldb.ModelMatch, query string) []modeldb.ModelMatch {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return matches
	}
	filtered := make([]modeldb.ModelMatch, 0, len(matches))
	for _, match := range matches {
		if matchModelQuery(match, query) {
			filtered = append(filtered, match)
		}
	}
	return filtered
}

func filterMatchesWithOfferings(matches []modeldb.ModelMatch) []modeldb.ModelMatch {
	filtered := make([]modeldb.ModelMatch, 0, len(matches))
	for _, match := range matches {
		if len(match.Offerings) == 0 {
			continue
		}
		filtered = append(filtered, match)
	}
	return filtered
}

func matchModelQuery(match modeldb.ModelMatch, query string) bool {
	key := modeldb.NormalizeKey(match.Model.Key)
	search := []string{
		modelID(match.Model),
		match.Model.Name,
		key.Creator,
		key.Family,
		key.Series,
		key.Version,
		key.ReleaseDate,
	}
	for _, alias := range match.Model.Aliases {
		search = append(search, alias)
	}
	for _, offering := range match.Offerings {
		search = append(search, offering.Service.ID, offering.Service.Name, offering.Offering.WireModelID)
		for _, exposure := range offering.Offering.Exposures {
			search = append(search, string(exposure.APIType))
			for _, p := range exposure.SupportedParameters {
				search = append(search, string(p))
			}
			for _, m := range exposure.ParameterMappings {
				search = append(search, m.WireName)
			}
		}
		for _, alias := range offering.Offering.Aliases {
			search = append(search, alias)
		}
	}
	for _, candidate := range search {
		if strings.Contains(strings.ToLower(candidate), query) {
			return true
		}
	}
	return false
}

func registerModelsCompletions(cmd *cobra.Command, load func(context.Context) (modeldb.Catalog, error)) {
	completion := func(values func(modeldb.Catalog) []string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			catalog, err := load(cmd.Context())
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return values(catalog), cobra.ShellCompDirectiveNoFileComp
		}
	}
	_ = cmd.RegisterFlagCompletionFunc("id", completion(completeModelIDs))
	_ = cmd.RegisterFlagCompletionFunc("service", completion(completeServiceIDs))
	_ = cmd.RegisterFlagCompletionFunc("api-type", completion(completeAPITypes))
	_ = cmd.RegisterFlagCompletionFunc("parameter", completion(completeNormalizedParameters))
	_ = cmd.RegisterFlagCompletionFunc("creator", completion(completeCreators))
	_ = cmd.RegisterFlagCompletionFunc("family", completion(completeFamilies))
	_ = cmd.RegisterFlagCompletionFunc("series", completion(completeSeries))
	_ = cmd.RegisterFlagCompletionFunc("version", completion(completeVersions))
	_ = cmd.RegisterFlagCompletionFunc("release-date", completion(completeReleaseDates))
}

func printModelsSummary(out io.Writer, matches []modeldb.ModelMatch) {
	if len(matches) == 0 {
		fmt.Fprintln(out, "No models found.")
		return
	}
	rows := make([][]string, 0, len(matches))
	maxID := len("MODEL")
	for _, match := range matches {
		id := modelID(match.Model)
		if len(id) > maxID {
			maxID = len(id)
		}
		rows = append(rows, []string{id, strings.Join(serviceIDs(match.Offerings), ", ")})
	}
	fmt.Fprintf(out, "%-*s  %s\n", maxID, "MODEL", "SERVICES")
	for _, row := range rows {
		fmt.Fprintf(out, "%-*s  %s\n", maxID, row[0], row[1])
	}
}

func printModelsOfferings(out io.Writer, matches []modeldb.ModelMatch) {
	flat := flattenOfferings(matches)
	if len(flat) == 0 {
		fmt.Fprintln(out, "No models found.")
		return
	}
	maxID := len("MODEL")
	maxService := len("SERVICE")
	maxAPI := len("API TYPES")
	for _, item := range flat {
		if len(item.ModelID) > maxID {
			maxID = len(item.ModelID)
		}
		if len(item.ServiceID) > maxService {
			maxService = len(item.ServiceID)
		}
		if len(item.APITypes) > maxAPI {
			maxAPI = len(item.APITypes)
		}
	}
	fmt.Fprintf(out, "%-*s  %-*s  %-*s  %s\n", maxID, "MODEL", maxService, "SERVICE", maxAPI, "API TYPES", "WIRE MODEL ID")
	for _, item := range flat {
		fmt.Fprintf(out, "%-*s  %-*s  %-*s  %s\n", maxID, item.ModelID, maxService, item.ServiceID, maxAPI, item.APITypes, item.WireModelID)
	}
}

func printModelsDetails(out io.Writer, matches []modeldb.ModelMatch, includeOfferings bool) {
	if len(matches) == 0 {
		fmt.Fprintln(out, "No models found.")
		return
	}
	for i, match := range matches {
		if i > 0 {
			fmt.Fprintln(out)
		}
		model := match.Model
		key := modeldb.NormalizeKey(model.Key)
		fmt.Fprintf(out, "%s\n", modelID(model))
		if model.Name != "" {
			fmt.Fprintf(out, "  name: %s\n", model.Name)
		}
		fmt.Fprintf(out, "  creator: %s\n", key.Creator)
		if key.Family != "" {
			fmt.Fprintf(out, "  family: %s\n", key.Family)
		}
		if key.Series != "" {
			fmt.Fprintf(out, "  series: %s\n", key.Series)
		}
		if key.Variant != "" {
			fmt.Fprintf(out, "  variant: %s\n", key.Variant)
		}
		if key.Version != "" {
			fmt.Fprintf(out, "  version: %s\n", key.Version)
		}
		if key.ReleaseDate != "" {
			fmt.Fprintf(out, "  release_date: %s\n", key.ReleaseDate)
		}
		if len(model.Aliases) > 0 {
			fmt.Fprintf(out, "  aliases: %s\n", strings.Join(model.Aliases, ", "))
		}
		fmt.Fprintf(out, "  services: %s\n", strings.Join(serviceIDs(match.Offerings), ", "))
		if model.Limits.ContextWindow > 0 || model.Limits.MaxOutput > 0 {
			fmt.Fprintf(out, "  limits: context_window=%d max_output=%d\n", model.Limits.ContextWindow, model.Limits.MaxOutput)
		}
		if caps := capabilityNames(model.Capabilities); len(caps) > 0 {
			fmt.Fprintf(out, "  capabilities: %s\n", capabilitySummary(model.Capabilities))
		}
		if caching := formatCachingCapability(model.Capabilities.Caching); caching != "" {
			fmt.Fprintf(out, "  caching: %s\n", caching)
		}
		if model.ReferencePricing != nil {
			fmt.Fprintf(out, "  pricing: input=%s output=%s cached_input=%s cache_write=%s\n",
				formatPrice(model.ReferencePricing.Input),
				formatPrice(model.ReferencePricing.Output),
				formatPrice(model.ReferencePricing.CachedInput),
				formatPrice(model.ReferencePricing.CacheWrite),
			)
		}
		if includeOfferings && len(match.Offerings) > 0 {
			fmt.Fprintln(out, "  offerings:")
			for _, offering := range match.Offerings {
				fmt.Fprintf(out, "    - %s -> %s\n", offering.Service.ID, offering.Offering.WireModelID)
				for _, exposure := range offering.Offering.Exposures {
					fmt.Fprintf(out, "      api_type: %s\n", exposure.APIType)
					if exposure.ExposedCapabilities != nil {
						if caps := capabilityNames(*exposure.ExposedCapabilities); len(caps) > 0 {
							fmt.Fprintf(out, "      capabilities: %s\n", capabilitySummary(*exposure.ExposedCapabilities))
						}
						if caching := formatCachingCapability(exposure.ExposedCapabilities.Caching); caching != "" {
							fmt.Fprintf(out, "      caching: %s\n", caching)
						}
					}
					if len(exposure.SupportedParameters) > 0 {
						fmt.Fprintf(out, "      supported_parameters: %s\n", joinNormalizedParameters(exposure.SupportedParameters))
					}
					if len(exposure.ParameterMappings) > 0 {
						for _, m := range exposure.ParameterMappings {
							fmt.Fprintf(out, "      parameter_mapping: %s -> %s\n", m.Normalized, m.WireName)
						}
					}
					if len(exposure.ParameterValues) > 0 {
						keys := make([]string, 0, len(exposure.ParameterValues))
						for k := range exposure.ParameterValues {
							keys = append(keys, k)
						}
						sort.Strings(keys)
						for _, k := range keys {
							fmt.Fprintf(out, "      parameter_values[%s]: %s\n", k, strings.Join(exposure.ParameterValues[k], ", "))
						}
					}
				}
			}
		}
	}
}

func printModelsJSON(out io.Writer, matches []modeldb.ModelMatch) error {
	records := make([]jsonModelMatch, 0, len(matches))
	for _, match := range matches {
		record := jsonModelMatch{
			ID:        modelID(match.Model),
			Model:     match.Model,
			Services:  serviceIDs(match.Offerings),
			Offerings: make([]jsonServiceOffering, 0, len(match.Offerings)),
		}
		for _, offering := range match.Offerings {
			record.Offerings = append(record.Offerings, jsonServiceOffering{Service: offering.Service, Offering: offering.Offering})
		}
		records = append(records, record)
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(records)
}

type offeringRow struct {
	ModelID     string
	ServiceID   string
	WireModelID string
	APITypes    string
}

func flattenOfferings(matches []modeldb.ModelMatch) []offeringRow {
	rows := make([]offeringRow, 0)
	for _, match := range matches {
		id := modelID(match.Model)
		for _, offering := range match.Offerings {
			apiTypes := make([]string, 0, len(offering.Offering.Exposures))
			for _, exposure := range offering.Offering.Exposures {
				apiTypes = append(apiTypes, string(exposure.APIType))
			}
			sort.Strings(apiTypes)
			rows = append(rows, offeringRow{ModelID: id, ServiceID: offering.Service.ID, WireModelID: offering.Offering.WireModelID, APITypes: strings.Join(apiTypes, ",")})
		}
	}
	return rows
}

func serviceIDs(offerings []modeldb.ServiceOffering) []string {
	out := make([]string, 0, len(offerings))
	for _, offering := range offerings {
		out = append(out, offering.Service.ID)
	}
	return out
}

func modelID(model modeldb.ModelRecord) string {
	if releaseID := modeldb.ReleaseID(model.Key); releaseID != "" {
		return releaseID
	}
	return modeldb.LineID(model.Key)
}

func capabilityNames(caps modeldb.Capabilities) []string {
	type capability struct {
		name string
		on   bool
	}
	reasoning := caps.Reasoning
	all := []capability{
		{"reasoning", reasoning != nil && reasoning.Available},
		{"reasoning_effort", reasoning != nil && len(reasoning.Efforts) > 0},
		{"tool_use", caps.ToolUse},
		{"parallel_tool_calls", caps.ParallelToolCalls},
		{"structured_output", caps.StructuredOutput || caps.StructuredOutputs},
		{"vision", caps.Vision},
		{"streaming", caps.Streaming},
		{"caching", caps.Caching != nil && caps.Caching.Available},
		{"interleaved_thinking", reasoning != nil && reasoning.Interleaved},
		{"adaptive_thinking", reasoning != nil && reasoning.Adaptive},
		{"temperature", caps.Temperature},
		{"logprobs", caps.Logprobs},
		{"seed", caps.Seed},
		{"web_search", caps.WebSearch},
	}
	out := make([]string, 0, len(all))
	for _, item := range all {
		if item.on {
			out = append(out, item.name)
		}
	}
	return out
}

func capabilitySummary(caps modeldb.Capabilities) string {
	names := capabilityNames(caps)
	if len(names) == 0 {
		return ""
	}
	parts := []string{strings.Join(names, ", ")}
	if caps.Reasoning != nil {
		if caps.Reasoning.VisibleSummary {
			parts = append(parts, "visible_summary=true")
		}
		if len(caps.Reasoning.Efforts) > 0 {
			efforts := make([]string, 0, len(caps.Reasoning.Efforts))
			for _, effort := range caps.Reasoning.Efforts {
				efforts = append(efforts, string(effort))
			}
			parts = append(parts, "efforts=["+strings.Join(efforts, ",")+"]")
		}
		if len(caps.Reasoning.Summaries) > 0 {
			summaries := make([]string, 0, len(caps.Reasoning.Summaries))
			for _, s := range caps.Reasoning.Summaries {
				summaries = append(summaries, string(s))
			}
			parts = append(parts, "summaries=["+strings.Join(summaries, ",")+"]")
		}
		if len(caps.Reasoning.Modes) > 0 {
			modes := make([]string, 0, len(caps.Reasoning.Modes))
			for _, mode := range caps.Reasoning.Modes {
				modes = append(modes, string(mode))
			}
			parts = append(parts, "modes=["+strings.Join(modes, ",")+"]")
		}
		if caps.Reasoning.AdaptiveOnly {
			parts = append(parts, "adaptive_only=true")
		}
		if caps.Reasoning.DefaultDisplay != "" {
			parts = append(parts, "default_display="+caps.Reasoning.DefaultDisplay)
		}
	}
	return strings.Join(parts, "; ")
}

func formatCachingCapability(c *modeldb.CachingCapability) string {
	if c == nil || !c.Available {
		return ""
	}
	parts := []string{"available=true"}
	if c.Mode != "" {
		parts = append(parts, "mode="+string(c.Mode))
	}
	if c.Configurable {
		parts = append(parts, "configurable=true")
	}
	if c.PromptCacheRetention {
		parts = append(parts, "prompt_cache_retention=true")
	}
	if c.PromptCacheKey {
		parts = append(parts, "prompt_cache_key=true")
	}
	if len(c.RetentionValues) > 0 {
		parts = append(parts, "retention_values=["+strings.Join(c.RetentionValues, ",")+"]")
	}
	if c.TopLevelRequestCaching {
		parts = append(parts, "top_level_request_caching=true")
	}
	if c.PerMessageCaching {
		parts = append(parts, "per_message_caching=true")
	}
	if len(c.CacheControlTypes) > 0 {
		parts = append(parts, "cache_control_types=["+strings.Join(c.CacheControlTypes, ",")+"]")
	}
	return strings.Join(parts, "; ")
}

func formatPrice(v float64) string {
	if v == 0 {
		return "0"
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func completeModelIDs(c modeldb.Catalog) []string {
	ids := make([]string, 0, len(c.Models))
	for _, model := range c.Models {
		ids = append(ids, fmt.Sprintf("%s\t%s", modelID(model), model.Name))
	}
	sort.Strings(ids)
	return ids
}

func completeServiceIDs(c modeldb.Catalog) []string {
	values := make([]string, 0, len(c.Services))
	for id, service := range c.Services {
		if service.Name != "" {
			values = append(values, fmt.Sprintf("%s\t%s", id, service.Name))
			continue
		}
		values = append(values, id)
	}
	sort.Strings(values)
	return values
}

func completeCreators(c modeldb.Catalog) []string {
	return completeKeyPart(c, func(key modeldb.ModelKey) string { return modeldb.NormalizeKey(key).Creator })
}

func completeFamilies(c modeldb.Catalog) []string {
	return completeKeyPart(c, func(key modeldb.ModelKey) string { return modeldb.NormalizeKey(key).Family })
}

func completeSeries(c modeldb.Catalog) []string {
	return completeKeyPart(c, func(key modeldb.ModelKey) string { return modeldb.NormalizeKey(key).Series })
}

func completeVersions(c modeldb.Catalog) []string {
	return completeKeyPart(c, func(key modeldb.ModelKey) string { return modeldb.NormalizeKey(key).Version })
}

func completeReleaseDates(c modeldb.Catalog) []string {
	return completeKeyPart(c, func(key modeldb.ModelKey) string { return modeldb.NormalizeKey(key).ReleaseDate })
}

func completeKeyPart(c modeldb.Catalog, pick func(modeldb.ModelKey) string) []string {
	seen := make(map[string]struct{})
	values := make([]string, 0)
	for key := range c.Models {
		value := pick(key)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func completeAPITypes(c modeldb.Catalog) []string {
	seen := map[string]bool{}
	for _, offering := range c.Offerings {
		for _, exposure := range offering.Exposures {
			if exposure.APIType != "" {
				seen[string(exposure.APIType)] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for apiType := range seen {
		out = append(out, apiType)
	}
	sort.Strings(out)
	return out
}

func joinNormalizedParameters(values []modeldb.NormalizedParameter) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, string(v))
	}
	return strings.Join(parts, ", ")
}

func completeNormalizedParameters(c modeldb.Catalog) []string {
	seen := map[string]bool{}
	for _, offering := range c.Offerings {
		for _, exposure := range offering.Exposures {
			for _, p := range exposure.SupportedParameters {
				seen[string(p)] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
