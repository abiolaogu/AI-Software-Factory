#!/usr/bin/env bash
set -euo pipefail

find . -type f -name '*.go' -not -path './vendor/*' -print0 | xargs -0 gofmt -w

if command -v shfmt >/dev/null 2>&1; then
  find scripts -type f -name '*.sh' -print0 | xargs -0 shfmt -w
fi
