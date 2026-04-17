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
	for _, item := range flat {
		if len(item.ModelID) > maxID {
			maxID = len(item.ModelID)
		}
		if len(item.ServiceID) > maxService {
			maxService = len(item.ServiceID)
		}
	}
	fmt.Fprintf(out, "%-*s  %-*s  %s\n", maxID, "MODEL", maxService, "SERVICE", "WIRE MODEL ID")
	for _, item := range flat {
		fmt.Fprintf(out, "%-*s  %-*s  %s\n", maxID, item.ModelID, maxService, item.ServiceID, item.WireModelID)
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
			fmt.Fprintf(out, "  capabilities: %s\n", strings.Join(caps, ", "))
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
}

func flattenOfferings(matches []modeldb.ModelMatch) []offeringRow {
	rows := make([]offeringRow, 0)
	for _, match := range matches {
		id := modelID(match.Model)
		for _, offering := range match.Offerings {
			rows = append(rows, offeringRow{ModelID: id, ServiceID: offering.Service.ID, WireModelID: offering.Offering.WireModelID})
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
	all := []capability{
		{"reasoning", caps.Reasoning},
		{"reasoning_effort", caps.ReasoningEffort},
		{"tool_use", caps.ToolUse},
		{"parallel_tool_calls", caps.ParallelToolCalls},
		{"structured_output", caps.StructuredOutput || caps.StructuredOutputs},
		{"vision", caps.Vision},
		{"streaming", caps.Streaming},
		{"caching", caps.Caching},
		{"interleaved_thinking", caps.InterleavedThinking},
		{"adaptive_thinking", caps.AdaptiveThinking},
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
