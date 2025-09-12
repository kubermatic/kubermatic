#!/usr/bin/env bash

# Copyright 2025 The Kubermatic Kubernetes Platform contributors.
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

# For usage guide see hack/README.md#run-kubermatic-webhook.sh

set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

KUBERMATIC_EDITION="${KUBERMATIC_EDITION:-ce}"
WEBHOOK_PORT=${WEBHOOK_PORT:-443}

# Generate self-signed certificates if they don't exist
CERT_ROOT_DIR="_certs"
CERT_DIR="$CERT_ROOT_DIR/webhook-serving-cert"
CA_BUNDLE_DIR="$CERT_ROOT_DIR/ca-bundle"

if [ ! -f "$CERT_DIR/serving.crt" ] || [ ! -f "$CERT_DIR/serving.key" ]; then
  echo "Generating self-signed certificates..."
  mkdir -p "$CERT_DIR" "$CA_BUNDLE_DIR"

  openssl req -new -newkey rsa:4096 -x509 -sha256 -days 365 -nodes \
    -out "$CERT_DIR/serving.crt" \
    -keyout "$CERT_DIR/serving.key" \
    -extensions san \
    -config <(
      echo "[req]"
      echo distinguished_name=req
      echo "[san]"
      echo subjectAltName=DNS:localhost,IP:127.0.0.1
    ) \
    -subj "/CN=*"

  # Copy certificate as CA bundle (for self-signed certs)
  cp "$CERT_DIR/serving.crt" "$CA_BUNDLE_DIR/ca-bundle.pem"

  echo "Certificates generated successfully!"
else
  echo "Certificates already exist, skipping generation."
fi

echodate "Compiling webhook..."
make kubermatic-webhook

echodate "Starting Kubermatic webhook..."
./_build/kubermatic-webhook \
  -namespace=kubermatic \
  -feature-gates=DevelopmentEnvironment=true,EtcdLauncher=true,KonnectivityService=true,OIDCKubeCfgEndpoint=true,OpenIDAuthPlugin=true,TunnelingExposeStrategy=true,UserClusterMLA=true,VerticalPodAutoscaler=true \
  -pprof-listen-address=:6600 \
  -v=2 \
  -seed-name=shared \
  -webhook-listen-port=$WEBHOOK_PORT \
  -webhook-cert-dir="$CERT_DIR/" \
  -webhook-cert-name=serving.crt \
  -webhook-key-name=serving.key \
  -ca-bundle="$CA_BUNDLE_DIR/ca-bundle.pem"
