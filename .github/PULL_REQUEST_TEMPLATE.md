## Summary

- What changed:
- Why it changed:
- How it was verified:

## Factory Checks

- [ ] `make proto`
- [ ] `make test`
- [ ] `make test-integration`
- [ ] `make test-contract`
- [ ] `bash scripts/compliance/openapi-drift.sh`
- [ ] `make swagger`
- [ ] `make redoc`
- [ ] `make coverage`
- [ ] `make lint`
- [ ] `make complexity`
- [ ] `bash scripts/compliance/migration-safety.sh`
- [ ] `make helm-validate`
- [ ] `make compliance-test`
- [ ] Trivy, Semgrep, and secret scans passed in GitHub Actions

## Risk Review

- [ ] Migrations are safe or explicitly justified.
- [ ] Helm chart changes were rendered and reviewed.
- [ ] OpenAPI and proto artifacts match the implementation.
- [ ] New endpoints, fields, and examples are documented.
- [ ] Auth, tenant isolation, rate-limit headers, and idempotency were considered.
- [ ] Swagger and Redoc routes were verified.

## Release Notes

- Customer-visible change:
- Ops or rollout notes:
