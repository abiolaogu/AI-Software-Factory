# Local Integration Guide

Use [`.env.integration.example`](/Users/AbiolaOgunsakin1/Workflows/AI-Software-Factory/neuroforge-sdx/.env.integration.example) as the local baseline when validating this repo with frontend and mobile consumers.

## Local integration flow

1. Start the API locally with integration-safe settings.
2. Ensure OpenAPI and proto manifests are current.
3. Point `AI-Frontend-Experience` and `AI-Companion-Mobile` at the local API URLs.
4. Run the consumer sync and drift checks after any contract change.

## Required outputs

- `api/openapi/openapi.json`
- `api/proto`
- `gen/typescript/sdx`
- `compatibility/manifest.json`
