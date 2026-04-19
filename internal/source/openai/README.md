# OpenAI static enrichment

`source_openai_api.go` provides live OpenAI inventory from `/v1/models`.

`source_openai_static.go` provides curated, checked-in enrichment layered on top
of that inventory.

## Scope

The static manifest currently focuses on OpenAI text models only.

That means models may still appear in the catalog from live inventory even when
we do not statically enrich them. This is expected for categories we are not
currently maintaining in detail, including:

- audio
- realtime
- image generation / image-only
- transcription / TTS
- moderation
- other non-text-specialized surfaces

## Source of truth policy

The checked-in manifest is the source of truth for OpenAI enrichment.

Use:

- official OpenAI docs / API reference / pricing pages
- direct OpenAI API behavior when clearly documented
- corroboration from broker views like OpenRouter as a cross-check
- corroboration from checked-in creator-adjacent static sources such as the
  Codex fixture when they describe the same OpenAI model line or close sibling

Do not:

- scrape HTML into the repo as a primary maintenance workflow
- guess unsupported capabilities
- copy broker-specific quirks as native OpenAI truth

## Family reasoning used in the current manifest

The current manifest uses explicit family/category groupings to reduce
boilerplate while staying source-backed.

### GPT-5 families

We currently distinguish:

- `gpt-5-pre-5.1-reasoning`
- `gpt-5-5.1-reasoning`
- `gpt-5-5.2plus-reasoning`
- `gpt-5-5.2plus-codex-backed`

Reasoning behind this split:

- OpenAI shared API reference indicates models before `gpt-5.1` default to
  `medium` reasoning effort and do not support `none`.
- OpenAI shared API reference indicates `gpt-5.1` supports reasoning effort
  values `none|low|medium|high`.
- OpenAI shared API reference indicates `xhigh` is supported for models after
  `gpt-5.1-codex-max`.
- OpenAI shared API reference indicates `concise` summaries are supported for
  all reasoning models after `gpt-5`.
- The checked-in Codex fixture corroborates summary support and `xhigh` for
  specific GPT-5 descendants such as `gpt-5.2`, `gpt-5.3-codex`, `gpt-5.4`, and
  `gpt-5.4-mini`.

Because of that:

- `gpt-5` / `gpt-5-pro` remain conservative and do not inherit post-5.1 summary
  support by default.
- `gpt-5.1*` inherits summary support, but keeps model-specific effort values.
- `gpt-5.2+` inherits summary support and `xhigh` only where the source-backed
  family boundary supports it.
- The `gpt-5-5.2plus-codex-backed` family is used only where Codex fixture data
  corroborates richer support directly.

### O-series families

We currently distinguish:

- `o1-reasoning`
- `o3plus-reasoning`

Reasoning behind this split:

- OpenAI shared API reference indicates reasoning controls apply to GPT-5 and
  o-series models.
- The same source indicates `concise` summaries are supported for all reasoning
  models after `gpt-5`.

Because of that:

- `o1` / `o1-pro` remain conservative and do not inherit summary support.
- `o3` / `o4`-era models inherit summary support and visible summary metadata.

## Potential gaps / open questions

The manifest is still intentionally conservative. Known areas that may need more
work when stronger evidence is available:

- exact effort tiers for all `gpt-5.4*` variants, especially `gpt-5.4-pro`
- exact effort tiers for all `gpt-5.1-codex*` variants
- whether some generic `gpt-5.2+` non-Codex variants should move into more
  specific families or keep model-specific overrides
- whether `o-series` effort tiers differ materially by subfamily beyond the
  current conservative defaults
- whether OpenRouter-backed parameter exposure can be enriched further when the
  broker publishes richer `supported_parameters`

## Maintenance rule

When uncertain, omit the field.

Prefer conservative, high-confidence metadata over broad but brittle coverage.

## Cleanup note

The old per-model docs-derived JSON/HTML artifacts are no longer used as the
source of truth for OpenAI enrichment. The curated static manifest under
`internal/source/openai/testdata/static.json` is the maintained input.
