#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${repo_root}"

bash tools/compliance/proto.sh
bash tools/compliance/openapi.sh
go test ./tools/compliance/docsserver -count=1 -v
bash tools/compliance/contract-test.sh
bash tools/compliance/auth-test.sh
bash tools/compliance/quota-test.sh
bash tools/compliance/negative-test.sh
