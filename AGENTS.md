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

## Development

Use the provided `Taskfile.yaml` to automate common development tasks:

- `task install` - Install the modeldb binary with version information
- `task build` - Build the modeldb binary locally to `./bin/modeldb`
- `task test` - Run all tests
- `task clean` - Clean build artifacts
- `task dev` - Full development workflow (clean → install)

The install and build tasks automatically embed version information from git tags or use "dev" as fallback.
Access version info with `modeldb version`.

## Release Process

To cut a release:

1. **Check diff against last release**: Review changes since the last tagged release with:
   ```bash
   git diff <last-tag>..HEAD --stat
   git log <last-tag>..HEAD --oneline
   ```

2. **Update CHANGELOG.md**: Add a new version section under `## Unreleased` with:
   - **Added** section for new features (e.g., new CLI commands, new fields)
   - **Changed** section for behavior changes or refinements
   - **Removed** section for deletions
   - **Fixed** section for bug fixes
   - **Model Changes** section if `catalog.json` contents changed

3. **Commit changes**: Stage all files and commit with a `release: vX.Y.Z` message:
   ```bash
   git add .
   git commit -m "release: vX.Y.Z"
   ```

4. **Tag the commit**: Create an annotated git tag:
   ```bash
   git tag -a vX.Y.Z -m "Release vX.Y.Z"
   ```

5. **Create GitHub release**: Push the tag and create a release on GitHub:
   ```bash
   git push origin main --tags
   ```
   Then use the GitHub web interface to create a release from the tag, using the CHANGELOG entry as release notes.

### Version Strategy

- Use semantic versioning: `vMAJOR.MINOR.PATCH`
- MAJOR: Breaking changes to the public API or snapshot format
- MINOR: New user-facing features or CLI commands
- PATCH: Bug fixes and refinements

## Docs

- Keep `README.md` end-user centric.
- Update `CHANGELOG.md` for user-visible snapshot/build behavior changes.
- If snapshot contents materially change, document them under a `Model Changes` section.
