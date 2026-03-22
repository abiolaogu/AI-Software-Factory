#!/usr/bin/env bash
set -euo pipefail

docker_available() {
  command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1
}

TOOLS_BIN_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/neuroforge-sdx-tools/bin"
TRIVY_VERSION="0.69.3"
TRIVY_DOCKER_IMAGE="aquasec/trivy:${TRIVY_VERSION}"
GITLEAKS_VERSION="8.24.2"
GITLEAKS_DOCKER_IMAGE="ghcr.io/gitleaks/gitleaks:v${GITLEAKS_VERSION}"

trivy_archive_suffix() {
  case "$(uname -s)-$(uname -m)" in
    Darwin-arm64)
      printf 'macOS-ARM64'
      ;;
    Darwin-x86_64)
      printf 'macOS-64bit'
      ;;
    Linux-aarch64 | Linux-arm64)
      printf 'Linux-ARM64'
      ;;
    Linux-x86_64)
      printf 'Linux-64bit'
      ;;
    *)
      echo "unsupported platform for Trivy bootstrap: $(uname -s)-$(uname -m)" >&2
      exit 1
      ;;
  esac
}

gitleaks_archive_suffix() {
  case "$(uname -s)-$(uname -m)" in
    Darwin-arm64)
      printf 'darwin_arm64'
      ;;
    Darwin-x86_64)
      printf 'darwin_x64'
      ;;
    Linux-aarch64 | Linux-arm64)
      printf 'linux_arm64'
      ;;
    Linux-x86_64)
      printf 'linux_x64'
      ;;
    *)
      echo "unsupported platform for Gitleaks bootstrap: $(uname -s)-$(uname -m)" >&2
      exit 1
      ;;
  esac
}

download_asset() {
  local url="$1"
  local destination="$2"

  mkdir -p "$(dirname "$destination")"
  curl -fL --retry 3 --retry-delay 2 --continue-at - -o "$destination" "$url"
}

bootstrap_trivy() {
  local archive_suffix
  archive_suffix="$(trivy_archive_suffix)"
  local archive_url="https://github.com/aquasecurity/trivy/releases/download/v${TRIVY_VERSION}/trivy_${TRIVY_VERSION}_${archive_suffix}.tar.gz"
  local archive_path="$TOOLS_BIN_DIR/trivy_${TRIVY_VERSION}_${archive_suffix}.tar.gz"

  mkdir -p "$TOOLS_BIN_DIR"
  if [ ! -x "$TOOLS_BIN_DIR/trivy" ]; then
    download_asset "$archive_url" "$archive_path"
    tar -xzf "$archive_path" -C "$TOOLS_BIN_DIR" trivy
  fi
  "$TOOLS_BIN_DIR/trivy" fs --severity HIGH,CRITICAL --exit-code 1 .
}

run_trivy() {
  if command -v trivy >/dev/null 2>&1; then
    trivy fs --severity HIGH,CRITICAL --exit-code 1 .
    return
  fi

  if docker_available; then
    docker run --rm -v "$PWD:/workspace" -w /workspace "$TRIVY_DOCKER_IMAGE" fs --severity HIGH,CRITICAL --exit-code 1 .
    return
  fi

  bootstrap_trivy
}

run_semgrep() {
  if command -v semgrep >/dev/null 2>&1; then
    semgrep --config p/ci --error .
    return
  fi

  if docker_available; then
    docker run --rm -v "$PWD:/workspace" -w /workspace semgrep/semgrep:1.123.0 semgrep --config p/ci --error .
    return
  fi

  if command -v python3 >/dev/null 2>&1; then
    python3 -m pip install --user --disable-pip-version-check --quiet semgrep
    python3 -m semgrep --config p/ci --error .
    return
  fi

  echo "semgrep is not installed and no supported fallback is available" >&2
  exit 1
}

bootstrap_gitleaks() {
  local archive_suffix
  archive_suffix="$(gitleaks_archive_suffix)"
  local archive_url="https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_${archive_suffix}.tar.gz"
  local archive_path="$TOOLS_BIN_DIR/gitleaks_${GITLEAKS_VERSION}_${archive_suffix}.tar.gz"

  mkdir -p "$TOOLS_BIN_DIR"
  if [ ! -x "$TOOLS_BIN_DIR/gitleaks" ]; then
    download_asset "$archive_url" "$archive_path"
    tar -xzf "$archive_path" -C "$TOOLS_BIN_DIR" gitleaks
  fi
  "$TOOLS_BIN_DIR/gitleaks" dir . --no-git --redact
}

run_gitleaks() {
  if command -v gitleaks >/dev/null 2>&1; then
    gitleaks dir . --no-git --redact
    return
  fi

  if docker_available; then
    docker run --rm -v "$PWD:/workspace" -w /workspace "$GITLEAKS_DOCKER_IMAGE" dir . --no-git --redact
    return
  fi

  bootstrap_gitleaks
}

if command -v gosec >/dev/null 2>&1; then
  gosec ./...
else
  go run github.com/securego/gosec/v2/cmd/gosec@v2.22.3 ./...
fi

if command -v govulncheck >/dev/null 2>&1; then
  govulncheck ./...
else
  go run golang.org/x/vuln/cmd/govulncheck@latest ./...
fi

run_trivy
run_semgrep
run_gitleaks
