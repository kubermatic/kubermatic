#!/usr/bin/env bash
set -xeuo pipefail

go build github.com/kubermatic/kubermatic/api/cmd/conformance-tests

./conformance-tests \
    -debug \
    -kubeconfig=$(go env GOPATH)/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
    -datacenters=$(go env GOPATH)/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/datacenters.yaml \
    -kubermatic-nodes=3 \
    -kubermatic-parallel-clusters=11 \
    -kubermatic-delete-cluster=true \
    -name-prefix=henrik-e2e \
    -reports-root=$(go env GOPATH)/src/github.com/kubermatic/kubermatic/reports \
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
    -azure-subscription-id="$(vault kv get -field=subscriptionID dev/e2e-azure)"

rm ./conformance-tests
