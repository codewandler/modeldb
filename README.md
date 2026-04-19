# modeldb

`modeldb` is a standalone Go package and CLI for working with a typed model
catalog across providers, brokers, and runtimes.

It gives you a consistent graph for questions like:

- Which services expose Claude Sonnet 4.5?
- What is the canonical identity of `openrouter/anthropic/claude-sonnet-4.5`?
- Which local runtime can route a given model right now?
- What factual aliases exist without mixing in app-specific shortcuts like
  `default` or `fast`?

The module ships with a prebuilt `catalog.json` snapshot and a `modeldb` CLI for
refreshing or inspecting that snapshot.

## Install

```bash
go get github.com/codewandler/modeldb
```

## What You Can Do

- load a built-in cross-provider model catalog
- select one logical model and find its offerings across services
- inspect service-scoped or runtime-scoped views
- merge external fragments onto the built-in graph
- build deterministic snapshots from live or fixture-backed upstream sources

## Quick Start

### Load the built-in catalog

```go
package main

import (
	"fmt"
	"log"

	"github.com/codewandler/modeldb"
)

func main() {
	catalog, err := modeldb.LoadBuiltIn()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("models=%d services=%d offerings=%d\n", len(catalog.Models), len(catalog.Services), len(catalog.Offerings))
}
```

### Resolve one logical model across providers

This is the package-level equivalent of querying:

```bash
modeldb models --name sonnet --version 4.5
```

```go
package main

import (
	"fmt"
	"log"

	"github.com/codewandler/modeldb"
)

func main() {
	catalog, err := modeldb.LoadBuiltIn()
	if err != nil {
		log.Fatal(err)
	}

	selector, err := modeldb.ParseModelSelector("sonnet", "4.5")
	if err != nil {
		log.Fatal(err)
	}

	selection, err := catalog.SelectOfferingsByModel(selector)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(modeldb.ReleaseID(selection.Model.Key))
	for _, item := range selection.Offerings {
		fmt.Printf("%s -> %s\n", item.Service.ID, item.Offering.WireModelID)
	}
}
```

Typical output:

```text
anthropic/claude/sonnet/4.5@2025-09-29
anthropic -> claude-sonnet-4-5-20250929
bedrock -> anthropic.claude-sonnet-4-5-20250929-v1:0
openrouter -> anthropic/claude-sonnet-4.5
```

### Filter by API surface or normalized parameter

```bash
modeldb models --name sonnet --version 4.5 --api-type openai-messages --offerings
modeldb models --creator openai --parameter reasoning_effort --offerings
```

Use `--api-type` and `--parameter` together when you want to ask invocation-
surface questions rather than purely logical-model questions.

## Data Model Layers

`modeldb` now distinguishes four layers:

- `ModelRecord`: canonical logical model identity and normalized capabilities
- `Offering`: a service-specific listing for a wire model ID
- `OfferingExposure`: one provider-native API surface for invoking an offering
- unified modeldb request shape: an internal normalization layer, not a provider exposure

`OfferingExposure` is the main place where invocation semantics live. Each
exposure is scoped to exactly one API type and carries:

- exposed capabilities
- normalized supported parameters
- wire parameter mappings
- valid parameter values

That means capabilities exist at two levels:

- `ModelRecord.Capabilities` describes what the logical model supports canonically
- `Offering.Exposures[*].ExposedCapabilities` describes what a concrete service/API surface exposes

Runtime invocation should target an exposure, not just an offering.

Reasoning is structured rather than boolean-only. The catalog now records, when
known:

- supported effort values (`none`, `low`, `medium`, `high`, `max`, `xhigh`)
- supported summary values (`none`, `auto`, `concise`, `detailed`)
- supported modes (`enabled`, `off`, `adaptive`, `interleaved`)
- visible-summary support

Parameters are normalized per exposure and remain API-type-specific. A
normalized parameter is only valid for the exposure that declares it.

### Browse one service view

```go
package main

import (
	"fmt"
	"log"

	"github.com/codewandler/modeldb"
)

func main() {
	catalog, err := modeldb.LoadBuiltIn()
	if err != nil {
		log.Fatal(err)
	}

	view := modeldb.ServiceView(catalog, "anthropic", modeldb.ViewOptions{})
	item, ok := view.Resolve("sonnet")
	if !ok {
		log.Fatal("sonnet not found")
	}

	fmt.Println(item.Offering.WireModelID)
	fmt.Println(modeldb.ReleaseID(item.Model.Key))
}
```

### Add runtime visibility on top of the built-in snapshot

```go
package main

import (
	"context"
	"log"

	"github.com/codewandler/modeldb"
)

func main() {
	base, err := modeldb.LoadBuiltIn()
	if err != nil {
		log.Fatal(err)
	}

	source := modeldb.NewOllamaRuntimeSource()
	resolved, err := modeldb.ResolveCatalog(context.Background(), base, modeldb.RegisteredSource{
		Stage:     modeldb.StageRuntime,
		Authority: modeldb.AuthorityLocal,
		Source:    source,
	})
	if err != nil {
		log.Fatal(err)
	}

	view := modeldb.RuntimeView(resolved, "ollama-local", modeldb.ViewOptions{VisibleOnly: true})
	_ = view.List()
}
```

## CLI

Validate a snapshot:

```bash
modeldb validate --in catalog.json
```

List matching logical models from the built-in catalog:

```bash
modeldb models --name sonnet --version 4.5
```

Search loosely across canonical IDs, names, aliases, services, and offering wire IDs:

```bash
modeldb models --query gpt-5.4
```

The default `models` listing only shows logical models that currently have at
least one offering in the catalog.

Expand offerings for each matching model:

```bash
modeldb models --name sonnet --version 4.5 --offerings
```

Filter to one service. `--service` implies `--offerings`:

```bash
modeldb models --service openrouter --name sonnet
```

Require exactly one logical model match and show text details:

```bash
modeldb models --id anthropic/claude/sonnet/4.5@2025-09-29 --select --details
```

Emit structured JSON. JSON always implies full detail output:

```bash
modeldb models --name sonnet --version 4.5 --json
```

Rebuild the snapshot from fixture-backed sources:

```bash
modeldb build \
	--out catalog.json \
	--anthropic-file internal/source/anthropic/testdata/models.json \
	--modelsdev-file internal/source/modelsdev/testdata/api.json
```

## Core Concepts

### `ModelKey`

The canonical identity of a model line or release.

Fields:

- `Creator`
- `Family`
- `Series`
- `Version`
- `Variant`
- `ReleaseDate`

Examples:

- line identity: `anthropic/claude/sonnet/4.6`
- release identity: `anthropic/claude/sonnet/4.5@2025-09-29`

### `ModelRecord`

Canonical or provisional metadata for a model identity.

Includes:

- factual aliases
- capabilities
- modalities
- limits
- reference pricing
- provenance

### `Offering`

A service-specific wire model ID mapped onto a canonical `ModelKey`.

Examples:

- `anthropic -> claude-sonnet-4-5-20250929`
- `openrouter -> anthropic/claude-sonnet-4.5`
- `bedrock -> anthropic.claude-sonnet-4-5-20250929-v1:0`

Each offering may expose one or more provider-native API surfaces via
`Offering.Exposures`. For example, two offerings may share a logical model and
wire identity but differ in API type, supported parameters, wire mappings,
valid parameter values, and exposed capabilities.

### `View`

`View` is the main end-user query API.

It provides:

- service-scoped browsing
- runtime-scoped browsing
- alias resolution
- filtering and ordering

In general, prefer `ServiceView(...)` and `RuntimeView(...)` over directly
walking the underlying maps.

## Alias Philosophy

The base graph stores factual aliases only.

Allowed:

- creator aliases
- service aliases
- runtime-discovered aliases

Deliberately excluded from the base graph:

- `default`
- `fast`
- `powerful`

Those intent aliases belong in consuming applications, overlays, or provider
policy layers, not in the shared model database.

## Build Model

Snapshot generation is creator-first.

- creator/direct sources define root `ModelRecord`s
- broker/platform sources add offerings and enrich existing roots
- offerings carry provider-native API exposures under `Offering.Exposures`
- capability-rich enrichment may come from creator-native docs or checked-in
  source fixtures when a live inventory endpoint is too thin
- when a broker/platform fragment points at a line that already has a creator
  root, the builder rebinds that fragment onto the creator-owned key
- when no creator root exists yet, the fragment may still create a provisional
  non-canonical root so coverage is not lost

This keeps cross-service offerings attached to one logical model instead of
letting each broker invent its own competing root entry.

## Repository Layout

- root package `modeldb`: public types, views, builders, selectors, sources
- `cli`: reusable Cobra command builders for host applications and the `modeldb` binary
- `cmd/modeldb`: thin executable wrapper around the reusable CLI package
- `internal/source/...`: upstream-specific fetchers, schemas, and fixtures

## Source Notes

Current source quality differs by upstream:

- Anthropic: creator-native API source with structured capability data
- OpenAI: live `/v1/models` inventory plus docs-backed fixture enrichment for
  richer capability data
- Codex: fixture-backed source with rich reasoning and parameter metadata
- OpenRouter: broker-native exposure metadata with normalized parameters and
  mappings
- MiniMax: currently still weaker on per-exposure capability completeness than
  the other major sources above; treat it as partially modeled until a richer
  source is added

## Roadmap

Planned CLI work that is intentionally deferred from the current Cobra migration:

- add `modeldb services`
- add `modeldb offerings`
- support consumer-provided overlay aliases and resolved-catalog sources in the reusable `models` Cobra command
- expand shell completions to narrow results based on already-selected flags

## Development

Regenerate the embedded snapshot:

```bash
go generate ./...
```

Run the test suite:

```bash
go test ./...
```
