#!/usr/bin/env bash

# Copyright 2023 The Kubermatic Kubernetes Platform contributors.
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

set -euo pipefail

export CABUNDLE_CONFIGMAP="${CABUNDLE_CONFIGMAP:-ca-bundle}"
export CABUNDLE_CONFIGMAP_KEY="${CABUNDLE_CONFIGMAP_KEY:-ca-bundle.pem}"
export CABUNDLE_MOUNTPATH="${CABUNDLE_MOUNTPATH:-/etc/kubermatic/certs}"

for kind in Deployment StatefulSet DaemonSet; do
   export K8S_KIND="$kind"

   # inject volume
   yq -i '(
      . |
      select(.kind==env(K8S_KIND)) |
      .spec.template.spec.volumes // (.spec.template.spec.volumes = [])
   ) += {
      "name": "ca-bundle",
      "configMap": {
         "name": env(CABUNDLE_CONFIGMAP)
      }
   }' "$1"

   # mount the volume into each container
   yq -i '(
      . |
      select(.kind==env(K8S_KIND)) |
      .spec.template.spec.containers.[] | .volumeMounts
   ) += {
      "name": "ca-bundle",
      "mountPath": env(CABUNDLE_MOUNTPATH),
      "readOnly": true
   }' "$1"

   # override the ssl ca env
   yq -i '(
      . |
      select(.kind==env(K8S_KIND)) |
      .spec.template.spec.containers.[] | .env
   ) += {
      "name": "SSL_CERT_FILE",
      "value": env(CABUNDLE_MOUNTPATH) + "/" + env(CABUNDLE_CONFIGMAP_KEY)
   }' "$1"
done
