# Changelog

## Unreleased

## v0.12.0 - 2025-04-23

### Added

- Added `modeldb version` command to display the embedded build version information.
- Added `Taskfile.yaml` for automated development tasks: `install`, `build`, `test`, `clean`, and `dev` workflows.
- Build and install tasks now automatically embed version information from git tags (with `git describe`) or use "dev" as fallback.

### Changed

- Updated `AGENTS.md` with a new Development section documenting the Taskfile-based development workflow.
- CLI root command now includes the new `version` subcommand alongside existing `build`, `validate`, `models`, and `skill` commands.

## v0.11.9 - 2026-04-22

### Added

- Added `modeldb skill` command that prints the embedded CLI skill reference
  (`SKILL.md`) to stdout. The document is baked into the binary at build time
  via `go:embed` so the command works without any external files.
- Added `.agents/skills/modeldb/SKILL.md` — a structured skill document
  covering all commands, flags, concepts, workflows, output formats, and error
  handling for end-users and agents consuming the `modeldb` CLI.

## v0.11.8 - 2026-04-19

### Changed

- Caching capabilities are now structured metadata instead of a single boolean, with authoritative exposure-level modeling for `explicit`, `implicit`, and `mixed` caching semantics.
- OpenAI prompt-caching support is now modeled more selectively instead of being applied blanket-wide to every Responses exposure.
- Canonical model-level caching is now kept coarse while exposure-level caching retains detailed configuration semantics.
- Anthropic caching parameter modeling now distinguishes top-level and block-level cache control surfaces.
- OpenRouter caching remains conservative and is no longer inferred from cached-input pricing alone.
- CLI model detail output now shows richer caching metadata instead of only a coarse capability label.
- Added provider snapshot tests covering caching semantics for OpenAI, Codex, Anthropic, MiniMax, and OpenRouter.
- Removed legacy boolean JSON compatibility for `capabilities.caching`; catalogs must now use the structured caching object form.

### Model Changes

- `catalog.json` was regenerated after the structured caching migration and provider-specific caching refinements.
- OpenAI, Codex, Anthropic, MiniMax, and OpenRouter offerings now expose more precise caching semantics and parameter surfaces.

## v0.11.7 - 2026-04-19

### Changed

- Pricing validation is now scoped to regular text offerings so strict pricing checks ignore realtime, audio, image, transcription, moderation, and search-specialized variants.
- OpenAI static pricing coverage was extended further across dated text-model aliases and additional text-focused GPT/o-series offerings.
- Strict pricing builds now succeed for the regular text-model catalog while preserving warnings for specialized offerings outside the text-model scope.

### Model Changes

-  was regenerated after additional OpenAI text-model pricing enrichment and text-only pricing audit refinement.
- Regular text OpenAI offerings now carry more complete pricing coverage across both base and dated aliases.

## v0.11.6 - 2026-04-19

### Changed

- OpenAI static pricing coverage was expanded substantially across GPT-5, GPT-4, gpt-3.5, and o-series model families using creator pricing data.
- Offerings now carry explicit `pricing_status` metadata so known, free, and unknown pricing states are distinguishable during catalog generation and validation.
- Build tooling now warns on unknown pricing and supports strict failure or filtered output via `--fail-on-unknown-pricing` and `--exclude-unknown-pricing`.
- Offering pricing can now be hydrated automatically from creator reference pricing for aligned services such as Codex.
- OpenRouter pricing classification now distinguishes explicit zero-cost offerings from offerings with missing pricing data.
- Local runtime offerings now mark pricing explicitly as free.

### Model Changes

- `catalog.json` was regenerated after OpenAI pricing enrichment and pricing-status propagation.
- OpenAI base offerings now expose explicit token pricing more consistently, including cached-input pricing where documented and `cache_write=0` where no separate write price is published.
- Codex offerings now inherit OpenAI base pricing where appropriate.
- Free offerings can now be identified explicitly in the generated catalog, while offerings with unclear pricing are surfaced for follow-up or optional exclusion.

## v0.11.5 - 2026-04-19

### Changed

- Anthropic exposure metadata is now more internally consistent: tool use and
  temperature support are reflected in supported parameters and parameter
  mappings for the Anthropic Messages surface.
- Anthropic reasoning availability now reflects either `thinking` or `effort`
  support instead of only `thinking` support.
- Anthropic `caching` is no longer inferred from the broader
  `context_management` capability, avoiding an over-broad semantic mapping.

### Model Changes

- `catalog.json` was regenerated after Anthropic capability/exposure cleanup.
- Anthropic offerings now expose `tools`, `tool_choice`, and `temperature`
  consistently alongside existing reasoning and structured-output metadata.
- Anthropic offerings no longer overclaim prompt-caching support from generic
  context-management capability flags alone.

## v0.11.4 - 2026-04-19

### Changed

- OpenAI static enrichment is now organized around more explicit GPT-5 and
  o-series family boundaries instead of relying only on broad line-level
  defaults or reviewer-driven per-model patches.
- OpenAI GPT-5 text-model summary support and effort controls were refined using
  creator-native API reference evidence, with tighter family splits for
  pre-5.1, 5.1, 5.2+, and Codex-backed descendants where corroboration exists.
- OpenAI o-series reasoning metadata was split into conservative `o1` and
  richer `o3+` family groups so summary support is only inherited where current
  source evidence supports it.
- The old per-model OpenAI docs-capture remnants were removed; the curated
  static manifest is now the only maintained OpenAI enrichment input.
- Added maintainer documentation explaining the evidence and reasoning behind
  current OpenAI family groupings and known remaining gaps.
- OpenRouter `openai-responses` exposures now advertise richer reasoning
  controls via normalized `reasoning_effort` and `reasoning_summary` support,
  using OpenRouter-documented broker value sets.
- Added `minimal` as a valid normalized reasoning effort level to represent the
  broker-surface OpenRouter responses API reasoning control vocabulary.

### Model Changes

- `catalog.json` was regenerated after OpenAI family/category normalization.
- OpenAI GPT-5.1 / GPT-5.2 / GPT-5.4 family members now expose more consistent
  `reasoning_summary` metadata where supported by creator-native evidence.
- OpenAI o3 / o4 family members now expose richer summary metadata than the
  more conservative `o1` family.
- OpenRouter `openai-responses` offerings now expose explicit reasoning effort
  and summary controls with broker-documented parameter values.

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
