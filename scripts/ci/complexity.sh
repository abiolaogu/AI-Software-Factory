#!/usr/bin/env bash
set -euo pipefail

threshold="${CYCLO_THRESHOLD:-10}"

if command -v gocyclo >/dev/null 2>&1; then
  output="$(gocyclo -over "${threshold}" . 2>&1 || true)"
else
  output="$(go run github.com/fzipp/gocyclo/cmd/gocyclo@v0.6.0 -over "${threshold}" . 2>&1 || true)"
fi

output="$(printf '%s\n' "${output}" | grep -v '_test\.go' || true)"
output="$(printf '%s\n' "${output}" | grep -vE '(^|[[:space:]]|/)gen/.*\.go:' || true)"
output="$(printf '%s\n' "${output}" | grep -v '^exit status 1$' || true)"

if [[ -n "${output}" ]]; then
  printf '%s\n' "${output}"
  echo "cyclomatic complexity threshold exceeded (>${threshold})" >&2
  exit 1
fi

echo "cyclomatic complexity is within threshold (${threshold})"
