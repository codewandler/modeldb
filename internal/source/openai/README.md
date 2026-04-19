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

Do not:

- scrape HTML into the repo as a primary maintenance workflow
- guess unsupported capabilities
- copy broker-specific quirks as native OpenAI truth

## Maintenance rule

When uncertain, omit the field.

Prefer conservative, high-confidence metadata over broad but brittle coverage.
