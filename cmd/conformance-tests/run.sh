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

set -xeuo pipefail

cd $(dirname $0)

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic
docker build -t conformance-tester-image containers/conformance-tests
cd -

go build github.com/kubermatic/kubermatic/api/cmd/conformance-tests

docker run --rm -it \
  -v $PWD:/bin/mounted \
  -v $(go env GOPATH)/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/kubeconfig:/kubeconfig \
  -v $(go env GOPATH)/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/datacenters.yaml:/datacenters.yaml \
  -v $(go env GOPATH)/src/github.com/kubermatic/secrets:/go/src/github.com/kubermatic/secrets \
  -v $(go env GOPATH)/src/github.com/kubermatic/kubermatic/reports:/reports \
  -v $(go env GOPATH)/src/github.com/kubermatic/kubermatic:/go/src/github.com/kubermatic/kubermatic \
  conformance-tester-image \
    /bin/mounted/conformance-tests \
        -debug \
        -worker-name=$USER \
        -kubeconfig=/kubeconfig \
        -datacenters=/datacenters.yaml \
        -kubermatic-nodes=3 \
        -kubermatic-parallel-clusters=11 \
        -kubermatic-delete-cluster=true \
        -name-prefix=$USER-e2e \
        -providers=aws \
        -reports-root=/reports \
        -cleanup-on-start=false \
        -aws-access-key-id="$(vault kv get -field=accessKeyID dev/e2e-aws)" \
        -aws-secret-access-key="$(vault kv get -field=secretAccessKey dev/e2e-aws)" \
        -digitalocean-token="$(vault kv get -field=token dev/e2e-digitalocean)" \
        -hetzner-token="$(vault kv get -field=token dev/e2e-hetzner)" \
        -openstack-domain="$(vault kv get -field=OS_USER_DOMAIN_NAME dev/syseleven-openstack)" \
        -openstack-tenant="$(vault kv get -field=OS_TENANT_NAME dev/syseleven-openstack)" \
        -openstack-username="$(vault kv get -field=username dev/syseleven-openstack)" \
        -openstack-password="$(vault kv get -field=password dev/syseleven-openstack)" \
        -vsphere-username="$(vault kv get -field=username dev/vsphere)" \
        -vsphere-password="$(vault kv get -field=password dev/vsphere)" \
        -azure-client-id="$(vault kv get -field=clientID dev/e2e-azure)" \
        -azure-client-secret="$(vault kv get -field=clientSecret dev/e2e-azure)" \
        -azure-tenant-id="$(vault kv get -field=tenantID dev/e2e-azure)" \
        -azure-subscription-id="$(vault kv get -field=subscriptionID dev/e2e-azure)" \
        -exclude-kubernetes-versions="9,10,11" \
        -exclude-distributions="ubuntu,centos,sles,rhel"

rm ./conformance-tests
