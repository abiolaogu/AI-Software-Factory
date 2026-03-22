#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${repo_root}"

bash scripts/swagger/sync.sh
bash scripts/swagger/validate.sh
go test ./test/contract -run TestOpenAPISpecIsValid -count=1 -v
