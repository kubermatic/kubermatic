#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../..
source ./api/hack/lib.sh

cd config/monitoring/grafana

tmpdir=tmp-dashboards

function cleanup() {
  rm -rf $tmpdir
}
trap cleanup EXIT SIGINT SIGTERM

cleanup
mkdir -p $tmpdir

echodate "Verifying dashboard file format..."
for dashboard in dashboards/*/*.json; do
  folder=$(basename $(dirname "$dashboard"))
  name=$(basename "$dashboard")

  mkdir -p "$tmpdir/$folder"
  cat "$dashboard" | jq --sort-keys '.' > "$tmpdir/$folder/$name"
done
diff -rdu dashboards $tmpdir
echodate "Dashboards are properly formatted."
