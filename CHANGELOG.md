# Changelog

## Unreleased

## v0.11.2 - 2026-04-19

### Changed

- Catalog JSON emission now strips volatile provenance observation timestamps so regenerating `catalog.json` does not churn on `observed_at` values alone.
- OpenRouter offerings now expose `openai-responses` and `openai-messages` API surfaces instead of the previous single `openai-chat` exposure.
- README examples and OpenAI static audit coverage were updated to use `openai-messages` for OpenRouter-facing message-surface queries.

### Model Changes

- `catalog.json` was regenerated with stable provenance timestamps removed from snapshot churn.
- OpenRouter offerings across creators now carry dual translated broker exposures: `openai-responses` and `openai-messages`.

## v0.11.1 - 2026-04-19

### Changed

- Anthropic Claude Opus 4.7 now models adaptive-only reasoning semantics.
- Anthropic Claude Opus 4.7 now exposes the `xhigh` reasoning effort tier.
- Anthropic Claude Opus 4.7 now carries `default_display=omitted` reasoning metadata.
- Anthropic Claude Opus 4.6 and Claude Sonnet 4.6 now carry `default_display=summarized` reasoning metadata.
- Reasoning capability merge now preserves summary/display metadata that was previously dropped during catalog merges.

### Model Changes

- `catalog.json` was regenerated for updated Anthropic Claude 4.x reasoning metadata.
- Anthropic Opus 4.7 offerings now expose `thinking.mode=[adaptive,off]` and `reasoning_effort` including `xhigh`.
- Regenerated catalog provenance timestamps and preserved reasoning summary fields may produce broader catalog diff churn than the Anthropic-only metadata changes.

## v0.11.0 - 2026-04-19

### Added

- Added `openai-static`, a curated static OpenAI enrichment source layered on
  top of live OpenAI inventory from `/v1/models`.
- Added reusable OpenAI static manifest profiles, family defaults, and
  per-model / per-exposure overrides.
- Added an OpenRouter-backed audit test for `openai-static` to flag likely
  drift and missing high-confidence text-model capabilities.
- Added maintainer documentation for OpenAI static enrichment scope and update
  policy under `internal/source/openai/README.md`.

### Changed

- Replaced the previous docs-derived OpenAI enrichment flow with a checked-in
  static manifest at `internal/source/openai/testdata/static.json`.
- OpenAI placeholder `default` exposures are no longer emitted; OpenAI
  exposures are now only emitted when statically curated.
- Narrowed OpenAI static enrichment scope to maintained text models; audio,
  realtime, image, transcription, TTS, moderation, and other non-text-focused
  models now remain inventory-only unless explicitly curated later.
- Renamed the build override flag from `--openai-docs-dir` to
  `--openai-static-file`.

### Removed

- Removed the old `openai-docs` source implementation and its per-model docs
  fixture workflow.

### Model Changes

- `catalog.json` was regenerated with `openai-static` as the OpenAI enrichment
  source.
- OpenAI text-model offerings now carry curated `openai-responses` exposures
  with normalized parameters, mappings, and parameter values where maintained.
- Non-text OpenAI inventory remains present in the catalog, but no longer
  receives broad static exposure enrichment by default.

## v0.10.0 - 2026-04-18

### Added

- Added first-class per-offering per-API `OfferingExposure` modeling.
- Added normalized exposure parameter metadata, including supported normalized
  parameters, wire mappings, and valid parameter values.
- Added exact exposure resolution helpers for service + wire model + API type.
- Added a first-class `codex` source in modeldb, backed by a checked-in fixture.
- Added an OpenAI static enrichment source for richer model capabilities.
- Added structured `ReasoningCapability` metadata with explicit effort,
  summary, mode, and visible-summary support.
- Added CLI filtering by `--api-type` and `--parameter`.

### Changed

- Replaced flat offering API metadata with `Offering.Exposures`.
- Replaced boolean-only reasoning capability flags with structured reasoning
  metadata.
- OpenAI capability enrichment now combines inventory from `/v1/models` with
  static manifest enrichment for richer model facts.
- `modeldb models --details` and `--offerings` now expose API-type-specific
  offering surface details.

### Removed

- Removed the derived `ReasoningCapability.Toggle` field from the catalog model.

### Model Changes

- `catalog.json` was regenerated for the new exposure-oriented schema.
- Anthropic, OpenRouter, Codex, and OpenAI models now carry richer capability
  information where upstream data is available.
- OpenAI model capability coverage is materially improved via static
  enrichment, while OpenAI offering exposures remain conservative unless
  creator-native surface metadata is known.

## v0.9.0 - 2026-04-18

### Added

- Added broad `--query` / `-q` search to `modeldb models` for exploratory lookup
  across canonical model IDs, names, aliases, services, and offering wire IDs.

### Changed

- `modeldb models --details` now prints the model `variant` field when present.
- The default `models` command output remains offering-backed and now treats
  orphan model records more consistently across normal, query, and detail modes.

### Removed

- Removed the `package` field from `Service` in the public API and generated
  snapshot.
- Stopped carrying NPM package metadata through `models.dev` service enrichment.

### Fixed

- Queries such as `modeldb models --query gpt-5.4` now work as a broad search
  mode instead of requiring structured selector flags.
- `modeldb models --query sherlock` no longer prints orphan models with empty
  service columns.

### Model Changes

- `catalog.json` was regenerated without service `package` fields.
- The regenerated snapshot also dropped the `meta-llama/llama-guard-4-12b:free`
  OpenRouter offering and now keeps the canonical `Meta: Llama Guard 4 12B`
  entry without the free-tier suffix.

## v0.8.0 - 2026-04-18

### Added

- Added a reusable public `cli` package for embedding `modeldb` Cobra commands in
  other applications.
- Added a new `modeldb models` command as the primary query interface.
- Added `--offerings`, `--details`, `--select`, and `--query` support to the
  `models` command.
- Added shell completion for `--id`, `--service`, `--creator`, `--family`,
  `--series`, `--version`, and `--release-date`.
- Added a roadmap section to the README to track deferred CLI commands and
  future overlay-oriented integration work.

### Changed

- Replaced the hand-rolled CLI dispatcher with Cobra.
- Made `cmd/modeldb` a thin executable wrapper around the reusable CLI package.
- Expanded `ModelSelector` with `ID`, `Creator`, `ServiceID`, `Family`, and
  `Series` fields.
- Added `FindModels(...)` as the listing-oriented public API and kept
  `SelectModel(...)` as the strict single-match API built on top of it.
- Updated the README CLI examples to use `modeldb ...` directly instead of
  `go run ./cmd/modeldb ...`.

### Removed

- Removed the old `inspect` command.
- Removed the old `model show` command in favor of `models`.

### Fixed

- `modeldb models` with no filters now lists all logical models that have at
  least one offering instead of returning an empty result.
- The default `models` output now hides orphan model records that have no
  attached offerings, avoiding blank service columns.
- Added broad `--query` search so exploratory lookups like `modeldb models --query gpt-5.4`
  work without needing exact structured filters.

## v0.7.0 - 2026-04-17

### Changed

- Initialized `github.com/codewandler/modeldb` as a standalone Go module.
- Renamed the root package from `catalog` to `modeldb`.
- Rewrote the README around end-user workflows, package usage, and CLI examples.
- Added repository-specific `AGENTS.md` guidance for future work.

## v0.6.0 - 2026-04-17

### Changed

- Builder snapshot generation is now creator-first.
- Anthropic creator models now come from the live `/v1/models` schema, with a
  checked-in fixture used for deterministic offline snapshot builds.
- Creator/direct sources define root `ModelRecord`s.
- Broker/platform sources such as OpenRouter and `models.dev` now rebind onto
  creator-owned roots when a matching creator model line already exists.
- Broker/platform fragments still create provisional non-canonical roots when no
  creator root exists, preserving fallback coverage.

### Fixed

- Cross-service offerings no longer leave duplicate root models behind when a
  creator source already provides the canonical model.
- `modeldb model show --name sonnet --version 4.5` now resolves cleanly to the
  Anthropic canonical Claude Sonnet 4.5 release instead of surfacing a
  line-vs-release ambiguity caused by broker-derived root records.

### Model Changes

- `catalog.json` was regenerated with creator-root rebinding enabled.
- Anthropic root models now reflect the real `/v1/models` payload, including the
  new `claude-opus-4-7` entry and current creator capability flags.
- Root model count dropped from `583` to `526` because duplicate broker-derived
  roots were collapsed onto creator-owned canonical roots.
- Anthropic-backed offerings across `anthropic`, `bedrock`, and `openrouter`
  now consistently attach to the same creator root for a given Claude model line.
- A large set of duplicate OpenAI release/enrichment roots were removed, with
  metadata merged into the surviving creator-owned roots.
- The regenerated snapshot also picked up normal upstream drift from live source
  refreshes, including changes in pricing, limits, supported parameters, and
  metadata such as `knowledge_cutoff` and `last_updated`.

## v0.5.0 - 2026-04-17

### Added

- Added catalog-side model selection for `modeldb model show` with line-level,
  cross-service offering lookup and ambiguity reporting.

### Changed

- Reworked snapshot building to merge creator roots first and rebind broker and
  platform fragments onto canonical creator models.

### Model Changes

- Duplicate broker-derived roots were collapsed onto creator-owned canonical
  roots in the generated snapshot.

## v0.4.0 - 2026-04-16

### Changed

- Added MiniMax as a built-in source.
- Moved pricing authority further into the catalog snapshot and model graph.

## v0.3.0 - 2026-04-16

### Changed

- Clarified source layout and alias policy.
- Tightened the distinction between factual catalog aliases and consumer policy
  aliases.

## v0.2.0 - 2026-04-16

### Changed

- Finished the `models.dev` source migration and folded it into the standalone
  catalog pipeline.

## v0.1.0 - 2026-04-16

### Added

- Initial standalone model catalog package, snapshot, builder, and CLI.
