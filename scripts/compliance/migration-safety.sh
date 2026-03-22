#!/usr/bin/env bash
set -euo pipefail

status=0

while IFS= read -r -d '' file; do
  if grep -Eiq '\bDROP\s+TABLE\b|\bDROP\s+COLUMN\b|\bTRUNCATE\b' "${file}"; then
    if ! grep -Eq 'MIGRATION_SAFE_OVERRIDE' "${file}"; then
      echo "unsafe migration detected: ${file}" >&2
      status=1
    fi
  fi
done < <(find internal/adapters/storage/migrations -type f -name '*.sql' -print0 | sort -z)

exit "${status}"
