#!/usr/bin/env bash

# Copyright 2026 The Kubermatic Kubernetes Platform contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

### Validates the configs rendered by the MLA Helm charts (the golden
### test/*.yaml.out files) with the validator of the same version that
### the chart deploys. The version comes from the image tag in the
### rendered output, so a chart bump also checks the new config schema.
###
### The fixture lists can be overridden for testing, for example to point
### at a modified copy of a .out file. Each variable holds one or more
### space separated paths or globs:
###   MLA_VERIFY_PROMETHEUS_FILES, MLA_VERIFY_ALERTMANAGER_FILES,
###   MLA_VERIFY_LOKI_FILES, MLA_VERIFY_LOKI_DISTRIBUTED_FILES

set -euo pipefail

cd "$(dirname "$0")/../.."
source hack/lib.sh

# Known failures that are accepted for now, as "component|source file".
# Keep this empty unless a validator rejects a config that is known to work
# and the fix is tracked elsewhere. Add a comment with the reason per entry.
ALLOWED_FAILURES=()

PROMETHEUS_FILES="${MLA_VERIFY_PROMETHEUS_FILES-charts/monitoring/prometheus/test/*.yaml.out}"
ALERTMANAGER_FILES="${MLA_VERIFY_ALERTMANAGER_FILES-charts/monitoring/alertmanager/test/*.yaml.out}"
LOKI_FILES="${MLA_VERIFY_LOKI_FILES-charts/logging/loki/test/*.yaml.out}"
LOKI_DISTRIBUTED_FILES="${MLA_VERIFY_LOKI_DISTRIBUTED_FILES-charts/mla/loki-distributed/test/*.yaml.out}"

TOOLS_DIR="$(pwd)/_build/tools"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
x86_64) ARCH=amd64 ;;
aarch64) ARCH=arm64 ;;
esac

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

RESULTS=()
FAILED=0

# image_version prints the tag (without a leading "v") of the container
# image whose repository ends with $2 (for example "/prometheus/prometheus")
# in the rendered manifests $1. It fails if there is none or more than one.
image_version() {
  local file="$1"
  local repo="$2"
  local versions

  versions="$(yq -N 'select(.spec.template.spec != null) | .spec.template.spec | ((.containers // []) + (.initContainers // [])) | .[].image' "$file" |
    grep -E "${repo}:" | sed -E 's/^.*:v?//' | sort -u)"

  if [ -z "$versions" ]; then
    echodate "No ${repo} image found in $file." >&2
    return 1
  fi
  if [ "$(echo "$versions" | wc -l | tr -d ' ')" != "1" ]; then
    echodate "More than one ${repo} version found in $file: $(echo "$versions" | tr '\n' ' ')" >&2
    return 1
  fi
  echo "$versions"
}

# ensure_tool downloads the release binary $1 (promtool, amtool or loki) of
# version $2 into _build/tools/<name>-<version>/ unless it is already there,
# and prints the path to the binary.
ensure_tool() {
  local name="$1"
  local version="$2"
  local dir="$TOOLS_DIR/$name-$version"
  local bin="$dir/$name"
  local url member tmp

  if [ -x "$bin" ]; then
    echo "$bin"
    return 0
  fi

  case "$name" in
  promtool)
    url="https://github.com/prometheus/prometheus/releases/download/v$version/prometheus-$version.$OS-$ARCH.tar.gz"
    member="prometheus-$version.$OS-$ARCH/promtool"
    ;;
  amtool)
    url="https://github.com/prometheus/alertmanager/releases/download/v$version/alertmanager-$version.$OS-$ARCH.tar.gz"
    member="alertmanager-$version.$OS-$ARCH/amtool"
    ;;
  loki)
    url="https://github.com/grafana/loki/releases/download/v$version/loki-$OS-$ARCH.zip"
    member="loki-$OS-$ARCH"
    ;;
  *)
    echodate "Unknown tool $name." >&2
    return 1
    ;;
  esac

  echodate "Downloading $name $version from $url ..." >&2
  tmp="$(mktemp -d)"
  if ! download_archive "$url" -fsSL -o "$tmp/archive"; then
    rm -rf "$tmp"
    echodate "Failed to download $url." >&2
    return 1
  fi

  case "$url" in
  *.zip) unzip -q "$tmp/archive" "$member" -d "$tmp" ;;
  *) tar -xzf "$tmp/archive" -C "$tmp" "$member" ;;
  esac

  mkdir -p "$dir"
  mv "$tmp/$member" "$bin"
  chmod +x "$bin"
  rm -rf "$tmp"
  echo "$bin"
}

# config_docs prints the document indexes of all ConfigMaps and Secrets in
# $1 that carry the key $2.
config_docs() {
  KEY="$2" yq -N 'select((.kind == "ConfigMap" and .data[strenv(KEY)] != null) or (.kind == "Secret" and (.data[strenv(KEY)] != null or .stringData[strenv(KEY)] != null))) | document_index' "$1"
}

# extract_config writes key $3 of document $2 in $1 to stdout. Secret data
# is base64 decoded.
extract_config() {
  KEY="$3" yq -N "select(document_index == $2) | (select(.kind == \"ConfigMap\") | .data[strenv(KEY)]), (select(.kind == \"Secret\" and .stringData[strenv(KEY)] != null) | .stringData[strenv(KEY)]), (select(.kind == \"Secret\" and .stringData[strenv(KEY)] == null) | .data[strenv(KEY)] | @base64d)" "$1"
}

# materialize_mounts recreates, under the directory $3, the files that the
# ConfigMap volumes of the container running image $2 would see in the pod.
# Only ConfigMaps rendered in the same file $1 are known. subPath mounts are
# skipped, none of the charts uses them for config or rule files.
materialize_mounts() {
  local file="$1"
  local repo="$2"
  local root="$3"
  local mountPath cmName key

  REPO="$repo:" yq -N 'select(.spec.template.spec != null) | .spec.template.spec as $s | $s.containers[] | select(.image | contains(strenv(REPO))) | .volumeMounts[] as $m | $s.volumes[] | select(.name == $m.name and .configMap != null and $m.subPath == null) | $m.mountPath + " " + .configMap.name' "$file" |
    while read -r mountPath cmName; do
      mkdir -p "$root$mountPath"
      for key in $(NAME="$cmName" yq -N 'select(.kind == "ConfigMap" and .metadata.name == strenv(NAME)) | .data | keys | .[]' "$file"); do
        NAME="$cmName" KEY="$key" yq -N 'select(.kind == "ConfigMap" and .metadata.name == strenv(NAME)) | .data[strenv(KEY)]' "$file" > "$root$mountPath/$key"
      done
    done
}

# rewrite_paths prefixes all absolute file references in the Prometheus or
# Alertmanager config $1 with $2 and creates empty files for references that
# do not exist yet (service account token, CA and client certificates).
# promtool and amtool require referenced files to exist, and writing them
# below a temp root avoids the need for root access to /etc and /var/run,
# which the older verify-user-cluster-prometheus-configs.sh needs.
rewrite_paths() {
  local config="$1"
  local root="$2"
  local ref

  ROOT="$root" yq -i '
    with(select(.rule_files != null); .rule_files[] |= strenv(ROOT) + .) |
    with(select(.scrape_config_files != null); .scrape_config_files[] |= strenv(ROOT) + .) |
    with(select(.templates != null); .templates[] |= strenv(ROOT) + .) |
    ((.. | select(key == "*_file" and tag == "!!str" and test("^/"))) |= strenv(ROOT) + .)
  ' "$config"

  for ref in $(yq -N '[.. | select(key == "*_file" and tag == "!!str")] | unique | .[]' "$config"); do
    if [ ! -e "$ref" ]; then
      mkdir -p "$(dirname "$ref")"
      touch "$ref"
    fi
  done
}

is_allowed() {
  local entry
  for entry in ${ALLOWED_FAILURES[@]+"${ALLOWED_FAILURES[@]}"}; do
    if [ "$entry" = "$1|$2" ]; then
      return 0
    fi
  done
  return 1
}

# record runs the remaining arguments as a check and stores the outcome for
# the summary table. Output is only printed when the check fails.
record() {
  local component="$1"
  local version="$2"
  local source="$3"
  shift 3
  local log="$WORKDIR/check.log"
  local result

  if "$@" > "$log" 2>&1; then
    result=PASS
  elif is_allowed "$component" "$source"; then
    result="FAIL (allowed)"
    cat "$log"
  else
    result=FAIL
    FAILED=1
    echodate "$component check failed for $source:"
    cat "$log"
  fi
  RESULTS+=("$component|$version|$source|$result")
}

check_prometheus() {
  local file="$1"
  local version promtool root idx config

  version="$(image_version "$file" /prometheus/prometheus)"
  promtool="$(ensure_tool promtool "$version")"

  for idx in $(config_docs "$file" prometheus.yaml); do
    root="$WORKDIR/prometheus-$idx"
    rm -rf "$root"
    mkdir -p "$root"
    materialize_mounts "$file" /prometheus/prometheus "$root"
    config="$root/prometheus.yaml"
    extract_config "$file" "$idx" prometheus.yaml > "$config"
    rewrite_paths "$config" "$root"

    record prometheus-config "$version" "$file" "$promtool" check config "$config"

    # check every rule file shipped by the chart, including the ones not
    # listed in rule_files (the usercluster-mla-* rules are loaded by the
    # Cortex ruler, which uses the same rule format).
    set -- "$root"/etc/prometheus/rules/*.yaml
    if [ -e "$1" ]; then
      record prometheus-rules "$version" "$file" "$promtool" check rules "$@"
    fi
  done
}

check_alertmanager() {
  local file="$1"
  local version amtool root idx config

  version="$(image_version "$file" /prometheus/alertmanager)"
  amtool="$(ensure_tool amtool "$version")"

  for idx in $(config_docs "$file" alertmanager.yml); do
    root="$WORKDIR/alertmanager-$idx"
    rm -rf "$root"
    mkdir -p "$root"
    materialize_mounts "$file" /prometheus/alertmanager "$root"
    config="$root/alertmanager.yml"
    extract_config "$file" "$idx" alertmanager.yml > "$config"
    rewrite_paths "$config" "$root"

    record alertmanager "$version" "$file" "$amtool" check-config "$config"
  done
}

# loki_verify runs "loki -verify-config". If the config uses ${VAR}
# references, they are expanded with dummy values, because the real values
# only exist in the pod.
loki_verify() {
  local loki="$1"
  local config="$2"
  local var

  if grep -q '\${' "$config"; then
    for var in $(grep -oE '\$\{[A-Za-z_][A-Za-z0-9_]*' "$config" | sed 's/^\${//' | sort -u); do
      export "$var=dummy"
    done
    "$loki" -config.file="$config" -config.expand-env=true -verify-config
  else
    "$loki" -config.file="$config" -verify-config
  fi
}

check_loki() {
  local component="$1"
  local file="$2"
  local version loki idx config

  version="$(image_version "$file" /grafana/loki)"
  loki="$(ensure_tool loki "$version")"

  # loki-distributed renders one config.yaml that all components share
  # (they only differ by -target), so each distinct document is checked once.
  for idx in $(config_docs "$file" config.yaml); do
    config="$WORKDIR/$component-$idx.yaml"
    extract_config "$file" "$idx" config.yaml > "$config"
    record "$component" "$version" "$file" loki_verify "$loki" "$config"
  done
}

for file in $PROMETHEUS_FILES; do
  echodate "Checking Prometheus config in $file ..."
  check_prometheus "$file"
done

for file in $ALERTMANAGER_FILES; do
  echodate "Checking Alertmanager config in $file ..."
  check_alertmanager "$file"
done

for file in $LOKI_FILES; do
  echodate "Checking seed Loki config in $file ..."
  check_loki loki "$file"
done

for file in $LOKI_DISTRIBUTED_FILES; do
  echodate "Checking user cluster MLA Loki config in $file ..."
  check_loki loki-distributed "$file"
done

echodate "Cortex has no offline config validator, its config is covered by the MLA e2e jobs."

echo
printf '%-18s %-9s %-16s %s\n' COMPONENT VERSION RESULT SOURCE
for row in ${RESULTS[@]+"${RESULTS[@]}"}; do
  IFS='|' read -r component version source result <<< "$row"
  printf '%-18s %-9s %-16s %s\n' "$component" "$version" "$result" "$source"
done
echo

if [ "$FAILED" -ne 0 ]; then
  echodate "Some MLA chart configs are invalid."
  exit 1
fi

echodate "All MLA chart configs are valid."
