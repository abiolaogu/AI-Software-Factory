#!/usr/bin/env bash
set -euo pipefail

required_files=(
  "factory.yaml"
  "api/openapi/openapi.yaml"
  "api/openapi/openapi.json"
  "api/proto/neuroforge/sdx/v1/sdx.proto"
  "docs/api/COMPLIANCE_TERMINAL.md"
  "docs/api/API_TESTING_PLAYBOOK.md"
  "docs/adr/0001-go-http-surface.md"
  "docs/domain/bounded-contexts.md"
  "deploy/helm/neuroforge-sdx/Chart.yaml"
  "deploy/kubevela/application.yaml"
  "deploy/fleet/fleet.yaml"
  "Dockerfile.api"
  "Dockerfile.worker"
  "tools/compliance/proto.sh"
  "tools/compliance/openapi.sh"
  "tools/compliance/mock-api.sh"
  "tools/compliance/compliance-test.sh"
  "scripts/compliance/openapi-drift.sh"
)

for path in "${required_files[@]}"; do
  if [[ ! -f "${path}" ]]; then
    echo "missing required compliance artifact: ${path}" >&2
    exit 1
  fi
done

bash tools/compliance/proto.sh
bash scripts/compliance/openapi-drift.sh
bash scripts/swagger/validate.sh
bash scripts/compliance/migration-safety.sh
go test ./test/contract -count=1 -v
bash tools/compliance/compliance-test.sh
