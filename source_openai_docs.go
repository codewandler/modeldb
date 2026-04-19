package modeldb

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const openAIDocsSourceID = "openai-docs"

type OpenAIDocsSource struct {
	Dir string
}

func NewOpenAIDocsSource() OpenAIDocsSource { return OpenAIDocsSource{Dir: DefaultOpenAIDocsFixtureDir()} }
func NewOpenAIDocsSourceFromDir(dir string) OpenAIDocsSource { return OpenAIDocsSource{Dir: dir} }
func DefaultOpenAIDocsFixtureDir() string { return filepath.Join("internal", "source", "openai", "testdata", "models") }
func (OpenAIDocsSource) ID() string { return openAIDocsSourceID }

type openAIDocsModel struct {
	Slug             string `json:"slug"`
	ContextWindow    int    `json:"context_window"`
	MaxOutput        int    `json:"max_output"`
	StructuredOutput bool   `json:"structured_output"`
	ReasoningEffort  bool   `json:"reasoning_effort"`
	Vision           bool   `json:"vision"`
}

func (s OpenAIDocsSource) Fetch(context.Context) (*Fragment, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil, err
	}
	observedAt := time.Now().UTC()
	frag := &Fragment{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.Dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var item openAIDocsModel
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		canonicalSlug := canonicalOpenAIDocsSlug(item.Slug)
		key, ok := inferOpenAIModelKey(canonicalSlug)
		if !ok {
			continue
		}
		caps := Capabilities{StructuredOutput: item.StructuredOutput, Vision: item.Vision, Streaming: true, ToolUse: true}
		if item.ReasoningEffort {
			caps.Reasoning = &ReasoningCapability{Available: true, Efforts: []ReasoningEffortLevel{ReasoningEffortLow, ReasoningEffortMedium, ReasoningEffortHigh}, Modes: []ReasoningMode{ReasoningModeEnabled, ReasoningModeOff}}
		}
		frag.Models = append(frag.Models, ModelRecord{
			Key:          key,
			Canonical:    false,
			Capabilities: caps,
			Limits:       Limits{ContextWindow: item.ContextWindow, MaxOutput: item.MaxOutput},
			Provenance:   []Provenance{{SourceID: openAIDocsSourceID, Authority: string(AuthorityTrusted), ObservedAt: observedAt, RawID: canonicalSlug}},
		})
	}
	sort.Slice(frag.Models, func(i, j int) bool { return modelID(frag.Models[i].Key) < modelID(frag.Models[j].Key) })
	return frag, nil
}


func canonicalOpenAIDocsSlug(slug string) string {
	slug = strings.TrimSpace(slug)
	if key, ok := inferOpenAIModelKey(slug); ok {
		return slug
	} else {
		_ = key
	}
	parts := strings.Split(slug, "-")
	if len(parts) > 2 {
		last := parts[len(parts)-1]
		if len(last) == 10 && last[4] == '-' && last[7] == '-' {
			candidate := strings.Join(parts[:len(parts)-1], "-")
			if _, ok := inferOpenAIModelKey(candidate); ok {
				return candidate
			}
		}
	}
	return slug
}
