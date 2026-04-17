# AGENTS

## Scope

This repository is the standalone model database extracted from `llm/catalog`.

It owns:

- canonical model identity and release keys
- service offerings and pricing references
- runtime visibility and acquisition overlays
- snapshot build, merge, validation, and CLI inspection

It does not own provider clients or app-specific routing policy.

## Package Boundaries

- root package `modeldb`: public graph types, selectors, views, builders, and source adapters
- `cmd/modeldb`: CLI for building and inspecting snapshots
- `internal/source/...`: upstream-specific transport, fixtures, and wire schemas

Keep the public API in the root package free of dependencies on `llm` or provider packages.

## Alias Rules

- The base graph stores factual aliases only.
- Do not add intent aliases like `default`, `fast`, or `powerful` to catalog data.
- Intent aliases belong in consuming applications or provider policy layers.

## Build Rules

- Creator/direct sources define root `ModelRecord`s.
- Broker/platform sources should add offerings and enrich creator roots.
- Prefer rebinding broker/platform fragments onto creator-owned keys instead of minting competing roots.
- Provisional roots are only acceptable when no creator root exists.

## Testing

- `go test ./...` must pass.
- Snapshot generation should remain deterministic with fixture-backed inputs.
- Prefer fixture-backed tests for upstream API sources.

## Docs

- Keep `README.md` end-user centric.
- Update `CHANGELOG.md` for user-visible snapshot/build behavior changes.
- If snapshot contents materially change, document them under a `Model Changes` section.
