# Governance Mode

Default declared mode for `AI-Software-Factory` is **HOTL**.

- Shared baseline: `../../AI Platform Automation/docs/ecosystem/GOVERNANCE_MODE.md`
- Canonical package reference: `/Users/AbiolaOgunsakin1/Workflows/Claude-Master-Architecture-Package/docs/GOVERNANCE_MODE.md`
- Repo-local switch: `governance/manifest.json`

## What changes by mode

| Mode | Repo behavior |
|------|---------------|
| HOTL | PRD-to-software generation, contract publication, and release automation proceed inside approved guardrails with human supervision by exception. |
| HITL | Human review is required at more explicit checkpoints before publishing contracts, promoting releases, or advancing sensitive generation flows. |
| FULL_AUTONOMY | Automation may advance further inside approved boundaries, but Swagger/OpenAPI compliance, contract validation, security checks, and release docs still gate execution. |

## Actions affected by governance mode

- PRD-to-service generation flows
- contract and schema publication
- release promotion and rollback orchestration
- high-impact agent execution paths

## Unchanged in every mode

- `AI-Software-Factory` remains the owner of software-factory backend contracts
- final design handoff remains outside this repo
- compatibility validation remains mandatory
- release documentation remains mandatory

## Cannot be bypassed even in FULL_AUTONOMY

- policy and security validation
- compatibility validation
- Swagger/OpenAPI compliance
- contract export validation
- release documentation updates

## Safe switching

1. Update `governance/manifest.json`
2. Update this file in the same PR
3. Run `node scripts/integration/validate-governance-mode.mjs`
4. Keep HOTL as the default even if `activeMode` changes temporarily
