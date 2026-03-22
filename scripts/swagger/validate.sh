#!/usr/bin/env bash
set -euo pipefail

go test ./test/contract -run '^TestOpenAPISpecIsValid$' -count=1 -v
