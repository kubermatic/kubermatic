#!/usr/bin/env bash

# Copyright 2020 The Kubermatic Kubernetes Platform contributors.
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

### This script sets up a local KKP installation in kind, deploys a
### couple of test Presets and Users and then runs the e2e tests for the
### API.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

TEST_NAME="Pre-warm Go build cache"
echodate "Attempting to pre-warm Go build cache"

beforeGocache=$(nowms)
make download-gocache
pushElapsed gocache_download_duration_milliseconds $beforeGocache

export KIND_CLUSTER_NAME="${SEED_NAME:-kubermatic}"

source hack/ci/setup-kind-cluster.sh

# gather the logs of all things in the Kubermatic namespace
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/kubermatic" --namespace kubermatic > /dev/null 2>&1 &

# As dex doesn't support groups in static users configuration we configure minimal LDAP server.
# More details: https://github.com/dexidp/dex/issues/1080
echodate "Setting up openldap server..."

LDAP_NAMESPACE="ldap"
export KUBERMATIC_LDAP_LOGIN="janedoe@example.com"
export KUBERMATIC_LDAP_PASSWORD="foo"

# Append Dex configuration with ldap connector
cat << EOF >> hack/ci/testdata/oauth_values.yaml
  connectors:
  - type: ldap
    name: OpenLDAP
    id: ldap
    config:
      host: openldap.${LDAP_NAMESPACE}.svc.cluster.local:389
      insecureNoSSL: true
      bindDN: cn=admin,dc=example,dc=org
      bindPW: admin
      usernamePrompt: Email Address
      userSearch:
        baseDN: ou=People,dc=example,dc=org
        filter: "(objectClass=person)"
        username: mail
        idAttr: DN
        emailAttr: mail
        nameAttr: cn
      groupSearch:
        baseDN: ou=Groups,dc=example,dc=org
        filter: "(objectClass=groupOfNames)"
        userMatchers:
          - userAttr: DN
            groupAttr: member
        nameAttr: cn
EOF

retry 2 kubectl create ns ${LDAP_NAMESPACE}
retry 2 kubectl apply -f hack/ci/testdata/openldap.yaml

source hack/ci/setup-kubermatic-in-kind.sh

echo "Seed is OK, tests would now continue to run."
