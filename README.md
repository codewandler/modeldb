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

This is the package-level equivalent of:

```bash
modeldb model show --name sonnet --version 4.5
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
go run ./cmd/modeldb validate --in catalog.json
```

Inspect services or offerings:

```bash
go run ./cmd/modeldb inspect --in catalog.json
go run ./cmd/modeldb inspect --in catalog.json --service anthropic
```

Resolve a logical model across providers:

```bash
go run ./cmd/modeldb model show --in catalog.json --name sonnet --version 4.5
```

Rebuild the snapshot from fixture-backed sources:

```bash
go run ./cmd/modeldb build \
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
- when a broker/platform fragment points at a line that already has a creator
  root, the builder rebinds that fragment onto the creator-owned key
- when no creator root exists yet, the fragment may still create a provisional
  non-canonical root so coverage is not lost

This keeps cross-service offerings attached to one logical model instead of
letting each broker invent its own competing root entry.

## Repository Layout

- root package `modeldb`: public types, views, builders, selectors, sources
- `cmd/modeldb`: CLI for building, validating, and querying snapshots
- `internal/source/...`: upstream-specific fetchers, schemas, and fixtures

## Development

Regenerate the embedded snapshot:

```bash
go generate ./...
```

Run the test suite:

```bash
go test ./...
```
