# E2E conformance tester

Runs static test scenarios against a kubermatic cluster.

### Running

#### Docker

TODO: Define

#### Locally

Requires the `containers/conformance-tests/install.sh` to be run before.
```bash
go build github.com/kubermatic/kubermatic/api/cmd/conformance-tests
./conformance-tests \
-kubeconfig=$(go env GOPATH)/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
-datacenters=$(go env GOPATH)/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/datacenters.yaml \
-kubermatic-nodes=5 \
-kubermatic-parallel-clusters=10 \
-kubermatic-delete-cluster=true \
-name-prefix=e2e \
-reports-root=$(go env GOPATH)/src/github.com/kubermatic/kubermatic/reports \
-v=4 \
-cleanup-on-start=true \
-aws-access-key-id=<<AWS_ACCESS_KEY_ID>> \
-aws-secret-access-key=<<AWS_SECRET_ACCESS_KEY_ID>> \
-digitalocean-token=<<DIGITALOCEAN_TOKEN>> \
-hetzner-token=<<HETZNER_TOKEN>> \
-openstack-domain=<<OPENSTACK_DOMAIN>> \
-openstack-tenant=<<OPENSTACK_TENANT>> \
-openstack-username=<<OPENSTACK_USERNAME>> \
-openstack-password=<<OPENSTACK_PASSWORD>>
```
