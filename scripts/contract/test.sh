#!/usr/bin/env bash
set -euo pipefail

go test ./test/contract/... -count=1 -v
