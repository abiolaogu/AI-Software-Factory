# Integration Architecture

Canonical source of truth: the workspace-level `Claude-Master-Architecture-Package`, anchored by `docs/MASTER_ECOSYSTEM_ARCHITECTURE.md` and its machine-readable manifests.

Authoritative baseline references:

- `ECOSYSTEM_REPO_CATALOG.md`
- `ECOSYSTEM_LAYER_MODEL.md`
- `ECOSYSTEM_FOLDER_TREE.md`
- `CONTRACT_OWNERSHIP.md`
- `COMPATIBILITY_MATRIX.md`
- `CROSS_REPO_RELEASE_FLOW.md`
- `LOCAL_INTEGRATION_GUIDE.md`
- `GOVERNANCE_MODE.md`

`AI-Software-Factory` is a producer repository in the StratoGroup AI ecosystem. Its integration responsibility is to publish stable, versioned Software Factory API contracts that can be consumed by:

- `AI-Frontend-Experience`
- `AI-Companion-Mobile`

## Owned exports

- OpenAPI contract bundle in [`api/openapi`](/Users/AbiolaOgunsakin1/Workflows/AI-Software-Factory/neuroforge-sdx/api/openapi)
- Proto contract bundle in [`api/proto`](/Users/AbiolaOgunsakin1/Workflows/AI-Software-Factory/neuroforge-sdx/api/proto)
- Schema bundle in [`api/schemas`](/Users/AbiolaOgunsakin1/Workflows/AI-Software-Factory/neuroforge-sdx/api/schemas)
- Generated Connect-ES TypeScript artifacts in [`gen/typescript/sdx`](/Users/AbiolaOgunsakin1/Workflows/AI-Software-Factory/neuroforge-sdx/gen/typescript/sdx)
- Compatibility metadata in [`compatibility/manifest.json`](/Users/AbiolaOgunsakin1/Workflows/AI-Software-Factory/neuroforge-sdx/compatibility/manifest.json)

## Integration rules

- OpenAPI and proto artifacts are versioned independently from service runtime releases.
- Contract manifests are the machine-readable source of truth for downstream sync and validation.
- Generated TypeScript artifacts are published as part of the contract surface, not as an implementation detail.
- CI validates that manifests, source artifacts, and release metadata remain coherent.
