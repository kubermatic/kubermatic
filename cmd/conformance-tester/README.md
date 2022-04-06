# E2E conformance tester

Runs static test scenarios against a kubermatic cluster.

The conformance tester will, by default, test all supported versions on providers using all supported operating systems.

## Tests

The command which will execute the following tests:
- Simple PVC (Only: AWS, Azure, OpenStack, vSphere)
  A StatefulSet with a PVC template will be created. The Pod, which mounts the PV has a ReadinessProbe in place
  which will only report ready when the pod was able to write something to the mounted PV.
  The PVC has no StorageClass set, so the default StorageClass, which gets deployed via the default kubermatic Addon `default-storage-class`, will be used.
- Simple LB (Only: AWS & Azure)
  The [Hello Kubernetes](https://kubernetes.io/docs/tasks/access-application-cluster/service-access-application-cluster/#creating-a-service-for-an-application-running-in-two-pods) Pod will be deployed with a Service of type LoadBalancer.
  The test will wait until the LoadBalancer is available and only report a success when the "Hello Kubernetes" Pod could be reached via the LoadBalancer IP(Or DNS).
- [Kubernetes Conformance tests](https://github.com/kubernetes/community/blob/master/contributors/devel/conformance-tests.md#running-conformance-tests) (First all parallel, afterwards all serial tests)

## Caveats

All providers have custom quotas. Hitting the quota is fairly easy when testing too many clusters at once.

## Running

The tester needs either a predefined login token (e.g. from copying the token cookie from your local KKP setup, after
logging in) or username/password to perform the login itself.

The tester by default uses dev.kubermatic.io as the seed, but this can be changed via `KUBERMATIC_API_ENDPOINT`.

Depending on the cloud provider, additional credentials must be provided. See the `hack/run-conformance-tests.sh`
script for more information.

You can specify a fixed project (`-kubermatic-project-id`) or let the tester create a temporary project on-the-fly.
Note that you need to cleanup projects after failed tests, like removing any added SSH keys to prevent conflicts
on the next run.

Run `./hack/run-conformance-tests.sh -help` for more information.

### With Token

```bash
./hack/run-conformance-tests.sh \
  -kubermatic-project-id=YOUR_PROJECT_ID_HERE \
  -kubermatic-oidc-token=OIDC_TOKEN_HERE \
  -versions=v1.19.0
```

### With Login

For this to work, the tester needs to create a client to communicate with Dex. The configuration for this
client happens by reading a Helm values.yaml file, usually the file used to install the `oauth` chart. In
addition, you need to specify the username (`KUBERMATIC_OIDC_LOGIN`) and password (`KUBERMATIC_OIDC_PASSWORD`)
as environment variables.

As the tests themselves run inside a Docker container, pay attention to the path you configure for
`-dex-helm-values-file`. It's recommended to copy the file into the KKP repository root, so it becomes
available automatically.

```bash
export KUBERMATIC_OIDC_LOGIN=testuser@example.com
export KUBERMATIC_OIDC_PASSWORD=testpassword

cp /your/oauth-helm-values.yaml oauth.yaml

./hack/run-conformance-tests.sh \
  -kubermatic-project-id=YOUR_PROJECT_ID_HERE \
  -kubermatic-oidc-token=OIDC_TOKEN_HERE \
  -dex-helm-values-file=oauth.yaml \
  -versions=v1.19.0
```

### Common customizations

**Debug logs**

Setting `-log-debug=true` will enable the debug logs.

**Only test a specific provider**

The provider which should be covered can be set via the `-provider` flag.

For example, setting `-provider=aws` will only test AWS clusters. This is also the default.

**Parallelism**

To configure the number of clusters which should be tested in parallel, the `-kubermatic-parallel-clusters=4`
flag can be use. Pay attention to not overload the seed cluster.

**Node count**

More nodes tend to improve test performance by some degree, though a higher node count might
lead to reaching the quota. The number of nodes, each cluster will have can be set via
`-kubermatic-nodes=3`.

**Keep clusters after test**

To be able to debug clusters, they must remain after a test has been run.
For this `-kubermatic-delete-cluster=false` can be specified, which will simply not delete the
cluster after testing.

**Delete existing clusters from a previous run**

In case a previous run left some clusters behind - maybe due to the use of `-kubermatic-delete-cluster=false` -
they can be deleting during the next execution by setting `-cleanup-on-start=true`.
