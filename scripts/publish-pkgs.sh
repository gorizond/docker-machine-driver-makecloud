#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

UPSTREAM_PUBLISH="./node_modules/@rancher/shell/scripts/extension/publish"

bash "${UPSTREAM_PUBLISH}" "$@"

PKG="makecloud-node-driver"
TEMPLATE_SRC="pkg/${PKG}/chart-templates/nodedriver.yaml"

if [[ ! -f "${TEMPLATE_SRC}" ]]; then
  echo "WARN: ${TEMPLATE_SRC} not found; skipping NodeDriver chart injection" >&2
  exit 0
fi

if [[ ! -d "charts/${PKG}" ]]; then
  # Nothing to patch (e.g. publish did not build anything)
  exit 0
fi

found_chart="false"
for chart_dir in "charts/${PKG}/"*; do
  [[ -d "${chart_dir}" ]] || continue
  found_chart="true"

  version="$(basename "${chart_dir}")"
  values_file="${chart_dir}/values.yaml"

  echo "Injecting NodeDriver into Helm chart: ${chart_dir}" >&2

  mkdir -p "${chart_dir}/templates"
  cp -f "${TEMPLATE_SRC}" "${chart_dir}/templates/nodedriver.yaml"

  if [[ -f "${values_file}" ]]; then
    yq -i '
      .nodeDriver.enabled = true |
      .nodeDriver.name = "makecloud" |
      .nodeDriver.publicCredentialFields = "baseUrl,vdcId" |
      .nodeDriver.privateCredentialFields = "token" |
      .nodeDriver.active = true |
      .nodeDriver.addCloudCredential = true |
      .nodeDriver.builtin = false |
      .nodeDriver.checksum = "" |
      .nodeDriver.description = "" |
      .nodeDriver.displayName = "makecloud" |
      .nodeDriver.externalId = "" |
      .nodeDriver.uiUrl = "" |
      .nodeDriver.url = "" |
      .nodeDriver.versionOverride = "" |
      .nodeDriver.whitelistDomains = []
    ' "${values_file}"
  fi

  if [[ -d "assets" ]]; then
    mkdir -p "assets/${PKG}"
    rm -f "assets/${PKG}/${PKG}-${version}.tgz"
    helm package "${chart_dir}" -d "assets/${PKG}" > /dev/null
  fi
done

if [[ "${found_chart}" != "true" ]]; then
  exit 0
fi

if [[ -d "assets" ]]; then
  merge_args=()
  if [[ -f "assets/index.yaml" ]]; then
    merge_args+=(--merge "assets/index.yaml")
  elif [[ -f "index.yaml" ]]; then
    merge_args+=(--merge "index.yaml")
  fi

  helm repo index assets --url assets/ "${merge_args[@]}" > /dev/null
  cp -f assets/index.yaml index.yaml
fi

# The GH reusable workflow uploads ./tmp; rebuild it so it includes our patched chart + re-packed assets.
if [[ -d "tmp" ]]; then
  rm -rf tmp
  mkdir -p tmp/{assets,charts,extensions}

  if [[ -d "assets" ]]; then
    cp -R assets/* tmp/assets/
  fi

  if [[ -d "charts" ]]; then
    cp -R charts/* tmp/charts/
  fi

  if [[ -d "extensions" ]]; then
    cp -R extensions/* tmp/extensions/
  fi

  if [[ -f "index.yaml" ]]; then
    cp -f index.yaml tmp/index.yaml
  fi
fi
