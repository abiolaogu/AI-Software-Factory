# ADR-014: Build State Model — Dual-Track (Stage + Status)

**Status:** Accepted
**Date:** 2026-03-21
**Deciders:** Chief Software Factory Architect

## Context

The Build aggregate has a 16-stage pipeline and an 11-value BuildStatus enum. The architecture review identified ambiguity: does the Build transition through numbered stages or named statuses? Use-cases describe stage-by-stage progression; the state machine describes status transitions. These are two different models that can diverge.

## Decision

Both are first-class. The Build uses a **dual-track model**:

1. `current_stage` (StageNumber 1-16): The **source of truth** for pipeline position. Incremented by `AdvanceStage()`.
2. `status` (BuildStatus enum): The **derived, coarse-grained state** visible on the API and dashboard. Computed via a deterministic `stage_to_status()` mapping function.

**Invariant:** `build.status == stage_to_status(build.current_stage)` at all times (except FAILED and RECOVERING, which override the derived status).

## Mapping

```
Stage 1  (Ingestion)          → INITIATED
Stage 2  (DomainModelling)    → DOMAIN_MODELLING
Stage 3  (ArchitectureDesign) → ARCHITECTING
Stage 4  (ContractGeneration) → CONTRACTING
Stage 5  (TestSpecification)  → TESTING_RED
Stage 6  (CodeGeneration)     → CODING_GREEN
Stages 7-9  (Analysis/Test/Security) → VERIFYING
Stages 10-13 (Compliance/Perf/Docs/IaC) → CERTIFYING
Stage 14 (ArtifactSigning)    → SIGNING
Stages 15-16 (Deploy/Learn)   → DEPLOYING
```

## Consequences

- **No ambiguity for Codex.** Status is always a function of stage.
- **Remediation loops** revert `current_stage`; status follows automatically.
- **Recovery from checkpoint** restores `current_stage`; status follows.
- **API consumers** see status (simple). **Internal systems** use current_stage (precise).
