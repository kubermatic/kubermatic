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
cp -r dashboards $tmpdir

echodate "Verifying dashboards..."
for dashboard in $tmpdir/*/*.json; do
	format_dashboard "$dashboard"
done
diff -rdu dashboards $tmpdir
echodate "Dashboards are properly formatted."
