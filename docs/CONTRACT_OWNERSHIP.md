# Contract Ownership

`AI-Software-Factory` owns the Software Factory backend contract surface.

## Owned

- `software-factory-openapi`
- `software-factory-proto`
- `software-factory-connect-typescript`
- `software-factory-schemas`
- `tenant-context`
- `autonomous-mission-request`
- `design-build-handoff`
- `build-artifact-handoff`
- `qualification-evidence-envelope`

## Downstream consumers

- `AI-Frontend-Experience` consumes the OpenAPI bundle, proto bundle, and generated Connect-ES types.
- `AI-Companion-Mobile` consumes the OpenAPI bundle.
- `Nvuto-AICO`, `Designer2`, `Droplet`, `AppForge`, `SkyForge`, and `Nvuto-Forge` consume the versioned boundary schemas listed in `compatibility/cross-repo-contracts.json`. Domain producers remain responsible for creating valid payloads; this repository owns schema publication and compatibility policy.

## Change policy

- Breaking contract changes require a semver major increment.
- Additive non-breaking changes require a semver minor increment.
- Example or documentation fixes require a semver patch increment when published as contract artifacts.
