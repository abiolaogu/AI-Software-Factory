# ADR-015: Spectral Phased Enforcement

**Status:** Accepted
**Date:** 2026-03-21
**Deciders:** Chief Software Factory Architect

## Context

grpc-gateway generates OpenAPI specs with patterns that conflict with StratoGroup's custom Spectral rules (e.g., `$ref` siblings, missing `info.contact`, OAS2-specific rules). Running the full custom ruleset against grpc-gateway output produces false positives.

## Decision

**M1: Minimal OAS baseline.** `.spectral.yaml` extends `spectral:oas` with OAS2-specific rules and cosmetic rules disabled. This eliminates false positives from grpc-gateway output.

**M2: Re-add custom rules.** After the OpenAPI spec stabilises, implement StratoGroup-specific checks as custom Spectral JavaScript functions that understand grpc-gateway output format:
- `operation-summary` (every endpoint must have a summary)
- `operation-security` (every endpoint must define a security scheme)
- `operation-error-responses` (every endpoint must define 400/401/500)
- Rate limit header enforcement
- Tenant scoping annotation validation

## Consequences

- **M1 has weaker API lint coverage.** Compensated by `buf lint` on proto definitions.
- **M2 achieves full spec compliance.** Custom functions avoid false positives.
- **CI gate:** `make api-compliance-full` uses whatever rules are active. M1 gate is softer; M2 gate is strict.
