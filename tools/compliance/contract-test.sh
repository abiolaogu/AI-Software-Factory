#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${repo_root}"

go test ./test/contract -run 'TestContractRoutesMatchSpec|TestDocumentationEndpoints' -count=1 -v
go test ./tools/compliance/suite -run 'TestComplianceContractSmokeForDocumentedEndpoints|TestComplianceVersionedRoutesAreDocumented|TestComplianceDocsRoutesServe' -count=1 -v
