#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${repo_root}"

bash tools/compliance/openapi.sh
go test ./tools/compliance/docsserver -run TestDocsServerSwaggerRoute -count=1 -v
