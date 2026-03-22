#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${repo_root}"

engine="${COMPLIANCE_MOCK_ENGINE:-internal}"
port="${COMPLIANCE_MOCK_PORT:-8090}"

case "${engine}" in
  prism)
    if command -v prism >/dev/null 2>&1; then
      exec prism mock api/openapi/openapi.yaml --host 0.0.0.0 --port "${port}"
    fi

    if command -v npx >/dev/null 2>&1; then
      exec npx -y @stoplight/prism-cli mock api/openapi/openapi.yaml -h 0.0.0.0 -p "${port}"
    fi

    if command -v docker >/dev/null 2>&1; then
      exec docker run --rm -p "${port}:${port}" -v "${repo_root}:/workspace" stoplight/prism:5 mock -h 0.0.0.0 -p "${port}" /workspace/api/openapi/openapi.yaml
    fi

    echo "Prism requested but neither prism, npx, nor docker is available." >&2
    exit 1
    ;;
  internal|equivalent)
    if [[ -z "${NF_MOCK_HTTP_ADDR:-}" ]]; then
      export NF_MOCK_HTTP_ADDR=":${port}"
    fi
    exec go run ./cmd/mock-api
    ;;
  *)
    echo "Unknown COMPLIANCE_MOCK_ENGINE=${engine}. Use internal or prism." >&2
    exit 1
    ;;
esac
