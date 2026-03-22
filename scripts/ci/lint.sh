#!/usr/bin/env bash
set -euo pipefail

go vet ./...

if command -v golangci-lint >/dev/null 2>&1; then
  golangci-lint run
else
  go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8 run
fi
