#!/usr/bin/env bash
set -euo pipefail

factory_threshold="$(awk '
  /quality_gates:/ {in_quality_gates=1; next}
  in_quality_gates && /coverage:/ {in_coverage=1; next}
  in_quality_gates && in_coverage && /^[[:space:]]+line:/ {print $2; exit}
' factory.yaml)"
threshold="${COVERAGE_THRESHOLD:-${factory_threshold:-80}}"
packages=(
  ./api/openapi
  ./internal/...
)

go test "${packages[@]}" -count=1 -coverprofile=coverage.out
summary="$(go tool cover -func=coverage.out)"
printf '%s\n' "$summary"
go tool cover -html=coverage.out -o coverage.html

coverage="$(printf '%s\n' "$summary" | awk '/^total:/ {gsub(/%/, "", $3); print $3}')"
awk -v coverage="$coverage" -v threshold="$threshold" 'BEGIN {
  if ((coverage + 0) < (threshold + 0)) {
    printf("coverage %.1f%% is below required threshold %.1f%%\n", coverage + 0, threshold + 0)
    exit 1
  }
}'
