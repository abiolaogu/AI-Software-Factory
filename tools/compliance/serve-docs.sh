#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${repo_root}"

if [[ -z "${COMPLIANCE_DOCS_ADDR:-}" && -n "${COMPLIANCE_DOCS_PORT:-}" ]]; then
  export COMPLIANCE_DOCS_ADDR=":${COMPLIANCE_DOCS_PORT}"
fi

exec go run ./tools/compliance/cmd/docs-server
