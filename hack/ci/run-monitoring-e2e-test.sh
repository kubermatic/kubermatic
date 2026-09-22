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

### This script deploys the seed-MLA monitoring stack via the
### kubermatic-installer into a kind cluster and verifies the rule
### selector contract from issue #16535: rules must select series
### via app_kubernetes_io_name, never the dead app label.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

echodate "Building kubermatic-installer..."
go build -o _build/kubermatic-installer ./cmd/kubermatic-installer

export KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-monitoring-e2e}"
source hack/ci/setup-kind-cluster.sh

VALUES_FILE="$(mktemp)"
cat << EOF > "$VALUES_FILE"
prometheus:
  storageSize: 1Gi
  storageClass: ""
EOF

CONFIG_FILE="$(mktemp)"
cat << EOF > "$CONFIG_FILE"
apiVersion: kubermatic.k8c.io/v1
kind: KubermaticConfiguration
metadata:
  name: kubermatic
  namespace: kubermatic
spec: {}
EOF

echodate "Deploying monitoring stack via kubermatic-installer..."
KUBECONFIG="$KUBECONFIG" _build/kubermatic-installer deploy seed-mla --config "$CONFIG_FILE" --charts-directory charts --helm-values "$VALUES_FILE" --mla-skip-logging --skip-charts grafana,blackbox-exporter,alertmanager,karma,helm-exporter

NS=monitoring

echodate "Waiting for workloads to roll out..."
kubectl rollout status daemonset/node-exporter --namespace "$NS" --timeout=5m
kubectl rollout status deployment/kube-state-metrics --namespace "$NS" --timeout=5m
kubectl rollout status statefulset/prometheus --namespace "$NS" --timeout=5m

echodate "Polling Prometheus label contract..."
PROM_POD="$(kubectl get pods --namespace "$NS" -l app.kubernetes.io/name=prometheus -o name | head -n 1)"

attempt=0
ok=0
while [ "$attempt" -lt 20 ]; do
  attempt=$((attempt + 1))

  UP="$(kubectl exec --namespace "$NS" "$PROM_POD" -c prometheus -- promtool query instant http://localhost:9090 'up{job="node-exporter"}' 2>/dev/null || true)"
  DEAD="$(kubectl exec --namespace "$NS" "$PROM_POD" -c prometheus -- promtool query instant http://localhost:9090 'count({app="node-exporter"})' 2>/dev/null || true)"
  LIVE="$(kubectl exec --namespace "$NS" "$PROM_POD" -c prometheus -- promtool query instant http://localhost:9090 'count({app_kubernetes_io_name="node-exporter"})' 2>/dev/null || true)"
  CPUS="$(kubectl exec --namespace "$NS" "$PROM_POD" -c prometheus -- promtool query instant http://localhost:9090 'node:node_num_cpu:sum' 2>/dev/null || true)"
  UTIL="$(kubectl exec --namespace "$NS" "$PROM_POD" -c prometheus -- promtool query instant http://localhost:9090 'node:node_cpu_utilisation:avg1m' 2>/dev/null || true)"

  ok=1

  # every node-exporter target must be up
  if [ -z "$UP" ] || echo "$UP" | grep -qv '=> 1 @'; then
    ok=0
  fi

  # count() prints no output when the selector matches nothing,
  # so the dead label must yield empty output
  if [ -n "$DEAD" ]; then
    ok=0
  fi

  if [ -z "$LIVE" ]; then
    ok=0
  fi

  if [ -z "$CPUS" ] || [ -z "$UTIL" ]; then
    ok=0
  fi

  if [ "$ok" -eq 1 ]; then
    break
  fi

  sleep 15
done

if [ "$ok" -ne 1 ]; then
  echodate "Monitoring e2e FAILED, last query outputs:"
  echo "up{job=\"node-exporter\"}:"
  echo "$UP"
  echo "count({app=\"node-exporter\"}):"
  echo "$DEAD"
  echo "count({app_kubernetes_io_name=\"node-exporter\"}):"
  echo "$LIVE"
  echo "node:node_num_cpu:sum:"
  echo "$CPUS"
  echo "node:node_cpu_utilisation:avg1m:"
  echo "$UTIL"
  exit 1
fi

echodate "Monitoring e2e passed."
