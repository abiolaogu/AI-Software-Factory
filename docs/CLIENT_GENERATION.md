# Client Generation

This repository publishes the backend contracts used for downstream client generation.

## Producer artifacts

- OpenAPI bundle: [`api/openapi/manifest.json`](/Users/AbiolaOgunsakin1/Workflows/AI-Software-Factory/neuroforge-sdx/api/openapi/manifest.json)
- Proto bundle: [`api/proto/manifest.json`](/Users/AbiolaOgunsakin1/Workflows/AI-Software-Factory/neuroforge-sdx/api/proto/manifest.json)
- Generated TypeScript: [`gen/typescript/sdx`](/Users/AbiolaOgunsakin1/Workflows/AI-Software-Factory/neuroforge-sdx/gen/typescript/sdx)

## Consumer flow

1. Frontend and mobile sync the manifest and artifacts into their local snapshot directories.
2. Consumer generation scripts derive endpoint registries and version metadata from those snapshots.
3. Consumer CI fails if generated bindings or compatibility manifests drift from the producer contracts.
