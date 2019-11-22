#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../..
source api/hack/lib.sh

for dashboard in config/monitoring/grafana/dashboards/*/*.json; do
	echodate "$dashboard"
	format_dashboard "$dashboard"
done
