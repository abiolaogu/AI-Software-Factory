#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${repo_root}"

files=(
  "api/openapi/openapi.yaml"
  "api/openapi/openapi.json"
)

temp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "${temp_dir}"
}
trap cleanup EXIT

for path in "${files[@]}"; do
  if [[ ! -f "${path}" ]]; then
    echo "missing OpenAPI artifact before drift check: ${path}" >&2
    exit 1
  fi
  cp "${path}" "${temp_dir}/$(basename "${path}")"
done

bash tools/compliance/openapi.sh

drift_detected=0
for path in "${files[@]}"; do
  snapshot="${temp_dir}/$(basename "${path}")"
  if ! cmp -s "${snapshot}" "${path}"; then
    echo "OpenAPI drift detected in ${path}" >&2
    drift_detected=1
  fi
done

if [[ "${drift_detected}" -ne 0 ]]; then
  echo "Run 'make openapi' and commit the generated artifacts." >&2
  exit 1
fi

echo "OpenAPI artifacts are in sync"
