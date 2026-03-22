#!/usr/bin/env bash
set -euo pipefail

chart_dir="deploy/helm/neuroforge-sdx"
release_name="${HELM_RELEASE_NAME:-neuroforge-sdx}"
namespace="${HELM_NAMESPACE:-default}"

if ! command -v helm >/dev/null 2>&1; then
  echo "helm is required to validate charts" >&2
  exit 1
fi

helm lint "${chart_dir}"
helm template "${release_name}" "${chart_dir}" --namespace "${namespace}" >/tmp/neuroforge-sdx-rendered.yaml

if [[ ! -s /tmp/neuroforge-sdx-rendered.yaml ]]; then
  echo "helm template did not render any manifests" >&2
  exit 1
fi

echo "helm chart validation passed"
