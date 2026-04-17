# Changelog

## Unreleased

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
