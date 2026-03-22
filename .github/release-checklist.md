# Release Checklist

- [ ] `make ci`
- [ ] `make test-integration`
- [ ] `make compliance-test`
- [ ] `make helm-validate`
- [ ] `bash scripts/compliance/openapi-drift.sh`
- [ ] Release notes are written and reviewed.
- [ ] OpenAPI, proto, and docs artifacts are current.
- [ ] Swagger and Redoc routes are reachable in the published docs build.
- [ ] Container images build successfully.
- [ ] Container scan workflow is green for the release candidate.
- [ ] SBOM artifacts were generated and attached.
- [ ] Security findings for this release are reviewed or accepted.
- [ ] Rollback steps are documented.
