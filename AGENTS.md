# AGENTS.md

## Repo purpose
- This repo is the AI-first software generation system for turning PRDs into production-ready services, APIs, tests, docs, and Swagger/OpenAPI compliance workflows.
- Preserve the 3-layer model from `../../AI Platform Automation/docs/ecosystem/ECOSYSTEM_LAYER_MODEL.md`. This repo is in the backend/control layer and must not absorb design-layer or experience-layer ownership.
- Treat `../../AI Platform Automation/docs/ecosystem/CONTRACT_OWNERSHIP.md`, `../../AI Platform Automation/docs/ecosystem/COMPATIBILITY_MATRIX.md`, `../../AI Platform Automation/docs/ecosystem/CROSS_REPO_RELEASE_FLOW.md`, `../../AI Platform Automation/docs/ecosystem/GOVERNANCE_MODE.md`, and `/Users/AbiolaOgunsakin1/Workflows/Claude-Master-Architecture-Package/docs/MASTER_ECOSYSTEM_ARCHITECTURE.md` as authoritative before changing cross-repo behavior.

## Commands to run
- `make proto`
- `make openapi`
- `make contract-test`
- `make compliance-test`
- `make test`
- `make build`
- `node scripts/integration/validate-exports.mjs`
- `node scripts/integration/validate-release-artifacts.mjs`

## Where contracts live
- OpenAPI: `api/openapi/`
- Proto: `api/proto/`
- Schemas: `api/schemas/`
- Compatibility: `compatibility/manifest.json`
- Release packaging rules: `scripts/integration/package-release-artifacts.mjs`
- Repo-local integration docs: `docs/INTEGRATION_ARCHITECTURE.md` and `docs/CONTRACT_OWNERSHIP.md`
- Governance mode: `governance/manifest.json` and `docs/GOVERNANCE_MODE.md`

## What not to change
- Do not redefine route maps, component specs, state maps, or design tokens here. Final design handoff belongs to `Design-Factory_Figma`.
- Do not move web or mobile implementation concerns into this repo. `AI-Frontend-Experience` and `AI-Companion-Mobile` consume these contracts.
- Do not weaken Swagger/OpenAPI compliance workflows; they are first-class for this repo.
- HOTL is the default governance mode. HITL and FULL_AUTONOMY are controlled modes and must not become the default path in code, prompts, or docs.

## What this repo owns vs consumes
- Owns: authoritative software-factory backend contracts, contract manifests, compatibility manifests, release artifacts, and compliance validation for software-factory APIs.
- Consumes: shared ecosystem standards, auth/session expectations, tenant/workspace identity rules, and root architecture baselines from the workspace docs.
