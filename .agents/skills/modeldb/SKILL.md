---
name: modeldb
description: Use when working with the modeldb CLI or catalog snapshot, including querying canonical model identities, service offerings, API surfaces, pricing, normalized parameters, and building or validating catalog.json.
metadata:
  short-description: Query and build the modeldb catalog
---

# modeldb CLI Skill

## What modeldb is

`modeldb` is a CLI for querying and building a typed model catalog across AI
providers, brokers, and local runtimes. It gives you a consistent, cross-service
graph for questions like:

- Which services expose Claude Sonnet 4.5?
- What is the canonical identity of `openrouter/anthropic/claude-sonnet-4.5`?
- Which API types and normalized parameters does a given model support?
- What is the wire model ID I need to send to Bedrock for Sonnet?

The binary ships with a built-in `catalog.json` snapshot that is always
available without any network access.

---

## Commands

```
modeldb models     # Query logical models and their offerings
modeldb build      # Rebuild the catalog snapshot from upstream sources
modeldb validate   # Validate a catalog JSON file
modeldb completion # Shell completion (bash/zsh/fish/powershell)
```

---

## `modeldb models` — query the catalog

### Basic listing (all models with at least one offering)

```bash
modeldb models
```

### Free-text search across IDs, names, aliases, services, and wire IDs

```bash
modeldb models --query sonnet
modeldb models --query gpt-5.4
modeldb models -q claude
```

### Filter by structured fields

```bash
# By logical name or alias
modeldb models --name sonnet
modeldb models --name opus

# By name + version
modeldb models --name sonnet --version 4.5

# By exact canonical model ID
modeldb models --id anthropic/claude/sonnet/4.5@2025-09-29

# By creator
modeldb models --creator openai
modeldb models --creator anthropic

# By model family
modeldb models --family claude

# By model series
modeldb models --series sonnet

# By version only
modeldb models --version 4.6

# By release date
modeldb models --release-date 2025-09-29

# By service (implies --offerings)
modeldb models --service openrouter
modeldb models --service anthropic --name sonnet
```

### Expand offerings per model

```bash
# Show which wire model ID to use on each service
modeldb models --name sonnet --version 4.5 --offerings

# Filter offerings by API surface type
modeldb models --name sonnet --api-type anthropic-messages --offerings
modeldb models --creator openai --api-type openai-responses --offerings

# Filter offerings by a normalized parameter name
modeldb models --parameter reasoning_effort --offerings
modeldb models --parameter cache_control --offerings
modeldb models --creator openai --parameter reasoning_effort --offerings
```

### Detailed view

```bash
# Rich text output: capabilities, caching, pricing, offerings
modeldb models --name sonnet --version 4.5 --details
modeldb models --name sonnet --version 4.5 --details --offerings

# Require exactly one match (errors on 0 or 2+ results)
modeldb models --id anthropic/claude/sonnet/4.5@2025-09-29 --select --details
```

### Machine-readable JSON output

```bash
# Always emits full detail (model record + offerings + exposures)
modeldb models --name sonnet --version 4.5 --json
modeldb models --creator anthropic --json
modeldb models --service openrouter --json | jq '.[].offerings[].offering.wire_model_id'
```

---

## `modeldb validate` — check a catalog file

```bash
# Validate the default catalog.json in the current directory
modeldb validate

# Validate a specific file
modeldb validate --in /path/to/catalog.json
```

Returns exit code 0 on success, non-zero with an error message on failure.

---

## `modeldb build` — rebuild the catalog snapshot

Rebuilds `catalog.json` from upstream sources (live API calls by default).

```bash
# Rebuild using live upstream sources, write to catalog.json
modeldb build

# Write to a custom path
modeldb build --out /path/to/my-catalog.json

# Use local fixture files instead of live API calls (reproducible/offline)
modeldb build \
  --anthropic-file internal/source/anthropic/testdata/models.json \
  --modelsdev-file internal/source/modelsdev/testdata/api.json

# Use a single local fixture per source while leaving others live
modeldb build --codex-file /tmp/codex-models.json
modeldb build --openai-static-file /tmp/openai-static.json

# Use the bundled models.dev fixture
modeldb build --modelsdev-fixture

# Warn about offerings with unknown pricing (default: warn to stderr)
modeldb build --out catalog.json

# Fail the build if any text-model offering has unknown pricing
modeldb build --fail-on-unknown-pricing

# Strip offerings with unknown pricing from the output
modeldb build --exclude-unknown-pricing
```

---

## Key concepts

### Canonical model ID formats

```
anthropic/claude/sonnet/4.5           # line identity (no release date)
anthropic/claude/sonnet/4.5@2025-09-29  # release identity (with date)
```

Fields: `creator / family / series / version [@release-date]`

### Service IDs

Common values seen in the catalog:

| ID           | Description                  |
|--------------|------------------------------|
| `anthropic`  | Anthropic direct API         |
| `openai`     | OpenAI direct API            |
| `openrouter` | OpenRouter broker            |
| `bedrock`    | AWS Bedrock platform         |
| `codex`      | OpenAI Codex                 |
| `minimax`    | MiniMax                      |
| `ollama-local` | Local Ollama runtime       |

### API types (`--api-type`)

| Value                  | Surface                                      |
|------------------------|----------------------------------------------|
| `anthropic-messages`   | Anthropic `/v1/messages` API                 |
| `openai-messages`      | OpenAI-compatible chat messages format       |
| `openai-responses`     | OpenAI Responses API (`/v1/responses`)       |
| `openai-chat`          | Legacy OpenAI chat completions               |

### Normalized parameters (`--parameter`)

Common values for `--parameter` filtering:

| Parameter                | Meaning                                     |
|--------------------------|---------------------------------------------|
| `reasoning_effort`       | Reasoning effort control                    |
| `reasoning_summary`      | Reasoning summary output control            |
| `thinking`               | Anthropic extended thinking block           |
| `thinking.mode`          | Thinking mode (`enabled`, `adaptive`, `off`)|
| `temperature`            | Sampling temperature                        |
| `tools`                  | Function/tool definitions                   |
| `tool_choice`            | Tool selection control                      |
| `response_format`        | Structured output / JSON mode               |
| `parallel_tool_calls`    | Parallel tool execution                     |
| `seed`                   | Deterministic sampling seed                 |
| `logprobs`               | Log probability output                      |
| `web_search`             | Built-in web search                         |
| `cache_control`          | Prompt caching (generic)                    |
| `top_level_cache_control`| Request-level cache control                 |
| `block_cache_control`    | Per-message/block cache control             |
| `prompt_cache_retention` | Cache retention duration                    |
| `prompt_cache_key`       | Explicit cache key                          |

---

## Common workflows

### Find the wire model ID for a specific service

```bash
# What do I send to Bedrock for Claude Sonnet 4.5?
modeldb models --name sonnet --version 4.5 --service bedrock

# What wire IDs does OpenRouter use for all Sonnet models?
modeldb models --name sonnet --service openrouter
```

### Discover which models support a capability

```bash
# Which models support reasoning_effort on the OpenAI Responses API?
modeldb models --api-type openai-responses --parameter reasoning_effort --offerings

# Which Anthropic models support extended thinking?
modeldb models --creator anthropic --parameter thinking --offerings

# Which models support prompt caching?
modeldb models --parameter cache_control --offerings
```

### Look up full details on a known model

```bash
modeldb models --id anthropic/claude/sonnet/4.5@2025-09-29 --select --details --offerings
```

### Export all OpenAI offerings as JSON for scripting

```bash
modeldb models --service openai --json
modeldb models --creator openai --json | jq '.[].id'
```

### Validate a freshly built snapshot

```bash
modeldb build --out my-catalog.json && modeldb validate --in my-catalog.json
```

---

## Output formats

### Default (summary table)

```
MODEL                                    SERVICES
anthropic/claude/sonnet/4.5@2025-09-29  anthropic, bedrock, openrouter
```

### `--offerings` (flat table)

```
MODEL                                    SERVICE      API TYPES                          WIRE MODEL ID
anthropic/claude/sonnet/4.5@2025-09-29  anthropic    anthropic-messages                 claude-sonnet-4-5-20250929
anthropic/claude/sonnet/4.5@2025-09-29  bedrock      anthropic-messages                 anthropic.claude-sonnet-4-5-20250929-v1:0
anthropic/claude/sonnet/4.5@2025-09-29  openrouter   openai-messages,openai-responses   anthropic/claude-sonnet-4.5
```

### `--details` (rich text block)

```
anthropic/claude/sonnet/4.5@2025-09-29
  name: Claude Sonnet 4.5
  creator: anthropic
  family: claude
  series: sonnet
  version: 4.5
  release_date: 2025-09-29
  aliases: claude-sonnet-4-5, ...
  services: anthropic, bedrock, openrouter
  limits: context_window=200000 max_output=16000
  capabilities: reasoning, tool_use, vision, streaming, caching, ...
  caching: available=true mode=explicit configurable=true ...
  pricing: input=3e-06 output=1.5e-05 cached_input=3e-07 cache_write=3.75e-06
```

### `--json` (structured array)

Each element contains `id`, `model` (full `ModelRecord`), `services` (list of
service IDs), and `offerings` (list of `{service, offering}` pairs including
all `exposures`).

---

## Shell completions

Enable completions once for your shell:

```bash
# bash (add to ~/.bashrc)
source <(modeldb completion bash)

# zsh (add to ~/.zshrc)
source <(modeldb completion zsh)

# fish
modeldb completion fish | source

# PowerShell
modeldb completion powershell | Out-String | Invoke-Expression
```

After enabling, `--id`, `--service`, `--creator`, `--family`, `--series`,
`--version`, `--release-date`, `--api-type`, and `--parameter` all support
tab completion against the live catalog.

---

## Error handling

| Situation | Exit code | Message |
|-----------|-----------|---------|
| No models match selector | 1 | `model not found: ...` |
| `--select` matches multiple models | 1 | `ambiguous model selector: ...` |
| Catalog load failure | 1 | `load built-in catalog: ...` |
| Validation failure | 1 | `validate catalog: ...` |
| Build failure | 1 | `build catalog: ...` |
| `--fail-on-unknown-pricing` triggered | 1 | `build catalog: unknown pricing for N offerings` |

Unknown-pricing warnings (without `--fail-on-unknown-pricing`) go to stderr
and do not affect the exit code.
