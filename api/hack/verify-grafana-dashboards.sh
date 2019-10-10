#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../../config/monitoring/grafana

tmpdir=tmp-dashboards

function cleanup() {
    rm -rf $tmpdir
}
trap cleanup EXIT SIGINT SIGTERM

cleanup
mkdir -p $tmpdir

echo "Extracting dashboards archive..."
tar vxjf dashboards.tar.bz2 -C $tmpdir

echo "Comparing archive contents against repository..."
diff -rdu dashboards tmp-dashboards
