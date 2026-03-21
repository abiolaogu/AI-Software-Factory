# NeuroForge SDX — State Machines

> Every aggregate with a `status` field has a formally defined state machine.
> Codex MUST enforce these transitions in aggregate methods. Invalid transitions throw ErrInvalidStateTransition.

---

## 1. WorkItem State Machine

```
                    ┌──────────────────────────────────────────────┐
                    │                                              │
                    ▼                                              │
  ┌───────┐    ┌────────┐    ┌─────────────────┐    ┌──────────┐  │
  │ DRAFT │───►│ PARSED │───►│ MANIFEST_REVIEW │───►│ APPROVED │  │
  └───────┘    └────────┘    └─────────────────┘    └──────────┘  │
      │            │               │                     │         │
      │            │               │                     ▼         │
      │            │               │              ┌────────────┐   │
      │            │               │              │ MODELLING  │   │
      │            │               │              └────────────┘   │
      │            │               │                     │         │
      │            │               │                     ▼         │
      │            │               │              ┌───────────┐    │
      │            │               │              │ COMPLETED │    │
      │            │               │              └───────────┘    │
      │            │               │                               │
      ▼            ▼               ▼                               │
  ┌────────┐                                                       │
  │ FAILED │───────────────────────────────────────────────────────┘
  └────────┘  (restart: FAILED → DRAFT only)
```

**Transition Table:**

| From | To | Trigger | Guard Condition |
|------|----|---------|-----------------|
| DRAFT | PARSED | `MarkAsParsed()` | ≥ 1 Requirement extracted |
| PARSED | MANIFEST_REVIEW | `SubmitForReview()` | All ambiguities resolved |
| MANIFEST_REVIEW | APPROVED | `ApproveManifest()` | Manifest has ≥ 1 language, ≥ 1 compliance framework; languages within tenant plan |
| APPROVED | MODELLING | `StartModelling()` | Build initiated |
| MODELLING | COMPLETED | `MarkCompleted()` | Build completed successfully |
| ANY | FAILED | `Fail()` | Unrecoverable error |
| FAILED | DRAFT | `Restart()` | Manual restart; resets requirements and ambiguities |

---

## 2. Build State Machine

```
  ┌───────────┐
  │ INITIATED │
  └─────┬─────┘
        ▼
  ┌─────────────────┐
  │ DOMAIN_MODELLING │ (Stage 2)
  └───────┬─────────┘
          ▼
  ┌──────────────┐
  │ ARCHITECTING │ (Stage 3)
  └──────┬───────┘
         ▼
  ┌──────────────┐
  │ CONTRACTING  │ (Stage 4)
  └──────┬───────┘
         ▼
  ┌──────────────┐
  │ TESTING_RED  │ (Stage 5)
  └──────┬───────┘
         ▼
  ┌──────────────┐
  │ CODING_GREEN │ (Stage 6)
  └──────┬───────┘
         ▼
  ┌───────────┐
  │ VERIFYING │ (Stages 7-8-9-10-11)
  └─────┬─────┘
        ▼
  ┌────────────┐
  │ CERTIFYING │ (Stage 10 compliance)
  └─────┬──────┘
        ▼
  ┌──────────┐
  │ SIGNING  │ (Stage 14)
  └────┬─────┘
       ▼
  ┌───────────┐
  │ DEPLOYING │ (Stage 15)
  └─────┬─────┘
        ▼
  ┌───────────┐
  │ COMPLETED │ (terminal)
  └───────────┘

  Any active state ──► FAILED (terminal, except → RECOVERING)
  FAILED ──► RECOVERING ──► (resume at checkpoint stage)
  RECOVERING ──► any active state (at checkpoint)
  RECOVERING ──► FAILED (if max recovery exceeded)
```

**Transition Table:**

| From | To | Trigger | Guard |
|------|----|---------|-------|
| INITIATED | DOMAIN_MODELLING | `AdvanceStage(2)` | Stage 1 completed |
| DOMAIN_MODELLING | ARCHITECTING | `AdvanceStage(3)` | Stage 2 completed, domain model ready |
| ARCHITECTING | CONTRACTING | `AdvanceStage(4)` | Stage 3 completed, blueprint approved |
| CONTRACTING | TESTING_RED | `AdvanceStage(5)` | Stage 4 completed, contracts generated |
| TESTING_RED | CODING_GREEN | `AdvanceStage(6)` | Stage 5 completed, test stubs ready |
| CODING_GREEN | VERIFYING | `AdvanceStage(7)` | Stage 6 completed, code generated |
| VERIFYING | CERTIFYING | `AdvanceStage(10)` | Stages 7-9 completed, all gates passed |
| CERTIFYING | SIGNING | `AdvanceStage(14)` | Stages 10-13 completed, compliance certified |
| SIGNING | DEPLOYING | `AdvanceStage(15)` | Stage 14 completed, artifact signed + SBOM attached |
| DEPLOYING | COMPLETED | `Complete()` | Stage 15-16 completed, deployment successful |
| ANY active | FAILED | `Fail()` | Unrecoverable error or max escalation |
| FAILED | RECOVERING | `RecoverFromCheckpoint()` | Valid checkpoint exists, recovery_attempts < 5 |
| RECOVERING | (checkpoint state) | Internal | Checkpoint hash verified |
| RECOVERING | FAILED | `Fail()` | Checkpoint corrupted or recovery_attempts >= 5 |

---

## 3. Artifact State Machine

```
  ┌─────────┐    ┌────────┐    ┌───────────┐    ┌──────────┐
  │ CREATED │───►│ SIGNED │───►│ PUBLISHED │───►│ DEPLOYED │
  └─────────┘    └────────┘    └───────────┘    └──────────┘
                                     │                │
                                     ▼                ▼
                               ┌─────────┐     ┌─────────┐
                               │ REVOKED │     │ REVOKED │
                               └─────────┘     └─────────┘
```

| From | To | Trigger | Guard |
|------|----|---------|-------|
| CREATED | SIGNED | `Sign()` | Valid SignatureRef, ProvenanceRef, SBOMRef |
| SIGNED | PUBLISHED | `Publish()` | Uploaded to registry |
| PUBLISHED | DEPLOYED | `MarkDeployed()` | Deployment successful |
| PUBLISHED | REVOKED | `Revoke()` | Security finding or policy violation |
| DEPLOYED | REVOKED | `Revoke()` | Post-deployment security finding |
| REVOKED | (terminal) | — | No transitions out of REVOKED |

---

## 4. Tenant State Machine

```
  ┌───────────────┐    ┌────────┐    ┌───────────┐    ┌──────────────────┐
  │ PROVISIONING  │───►│ ACTIVE │───►│ SUSPENDED │───►│ DEPROVISIONED   │
  └───────────────┘    └────────┘    └───────────┘    └──────────────────┘
                           ▲              │
                           └──────────────┘
                           (reactivate)
```

| From | To | Trigger | Guard |
|------|----|---------|-------|
| PROVISIONING | ACTIVE | `Activate()` | K8s namespace ready, Vault mount created, DragonflyDB partition ready |
| ACTIVE | SUSPENDED | `Suspend()` | Payment failure, policy violation, or admin action |
| SUSPENDED | ACTIVE | `Reactivate()` | Payment resolved or admin action |
| SUSPENDED | DEPROVISIONED | `Deprovision()` | 30-day grace period expired or explicit admin action |
| DEPROVISIONED | (terminal) | — | Data retained 30 days, then purged |

---

## 5. Agent State Machine

```
  ┌────────────┐    ┌──────┐    ┌──────────┐    ┌────────────┐
  │ REGISTERED │───►│ IDLE │───►│ ASSIGNED │───►│ PROCESSING │
  └────────────┘    └──────┘    └──────────┘    └────────────┘
                       ▲              │                │
                       │              │                │
                       └──────────────┘                │
                       (release)                       │
                       ▲                               │
                       └───────────────────────────────┘
                       (complete task)

  PROCESSING ──► RECOVERING (agent error, auto-recovery)
  RECOVERING ──► IDLE (recovery successful)
  RECOVERING ──► UNHEALTHY (recovery failed)

  ANY ──► UNHEALTHY (3 missed heartbeats)
  UNHEALTHY ──► IDLE (health restored)
  UNHEALTHY ──► TERMINATED (SPIFFE SVID expired or admin kill)
  ANY ──► TERMINATED (SPIFFE SVID expired)
```

| From | To | Trigger | Guard |
|------|----|---------|-------|
| REGISTERED | IDLE | `Ready()` | Health check passes, SPIFFE SVID valid |
| IDLE | ASSIGNED | `AssignToBuild()` | Build exists, stage matches agent type |
| ASSIGNED | PROCESSING | `StartProcessing()` | Activity started |
| PROCESSING | IDLE | `ReleaseFromBuild()` | Task completed successfully |
| PROCESSING | RECOVERING | `RecoverFromError()` | Transient error detected |
| RECOVERING | IDLE | `RecoverySucceeded()` | Agent health restored |
| RECOVERING | UNHEALTHY | `RecoveryFailed()` | Recovery timed out |
| ANY active | UNHEALTHY | `MarkUnhealthy()` | 3 consecutive missed heartbeats (30s) |
| UNHEALTHY | IDLE | `RestoreHealth()` | Health check passes again |
| UNHEALTHY | TERMINATED | `Terminate()` | SVID expired or admin kill |
| ANY | TERMINATED | `Terminate()` | SVID expired |

---

## 6. Deployment State Machine

```
  ┌─────────┐    ┌──────────┐    ┌───────────────┐    ┌─────────────────┐    ┌──────────┐
  │ PENDING │───►│ APPLYING │───►│ SMOKE_TESTING │───►│ SLO_VALIDATING │───►│ DEPLOYED │
  └─────────┘    └──────────┘    └───────────────┘    └─────────────────┘    └──────────┘
                      │                │                      │
                      ▼                ▼                      ▼
                 ┌────────┐     ┌─────────────┐       ┌─────────────┐
                 │ FAILED │     │ ROLLED_BACK │       │ ROLLED_BACK │
                 └────────┘     └─────────────┘       └─────────────┘
```

| From | To | Trigger | Guard |
|------|----|---------|-------|
| PENDING | APPLYING | `Apply()` | Fleet GitOps bundle pushed |
| APPLYING | SMOKE_TESTING | `RunSmokeTests()` | Fleet reconciliation complete |
| APPLYING | FAILED | `Fail()` | Fleet apply error |
| SMOKE_TESTING | SLO_VALIDATING | `StartSLOValidation()` | All smoke tests passed |
| SMOKE_TESTING | ROLLED_BACK | `Rollback()` | Smoke test failure |
| SLO_VALIDATING | DEPLOYED | `Confirm()` | 15-min SLO validation passed |
| SLO_VALIDATING | ROLLED_BACK | `Rollback()` | SLO violation detected |
| DEPLOYED | ROLLED_BACK | `Rollback()` | Post-deployment issue (manual or auto) |

---

## 7. Circuit Breaker State Machine (LLM Gateway)

```
  ┌────────┐    ┌──────┐    ┌───────────┐
  │ CLOSED │───►│ OPEN │───►│ HALF_OPEN │
  └────────┘    └──────┘    └───────────┘
      ▲              ▲            │
      │              │            │
      │              └────────────┘ (probe fails)
      │                           │
      └───────────────────────────┘ (probe succeeds × 2)
```

| From | To | Trigger | Guard |
|------|----|---------|-------|
| CLOSED | OPEN | `RecordFailure()` | consecutive_failures >= 3 |
| OPEN | HALF_OPEN | Timer | 30s cooldown elapsed |
| HALF_OPEN | CLOSED | `RecordSuccess()` | 2 consecutive successes |
| HALF_OPEN | OPEN | `RecordFailure()` | Probe request failed |
