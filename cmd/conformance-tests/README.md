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

All providers have custom quota's.
Hitting the quota is fairly easy when testing too many clusters at once.

## Running

### Locally

**Requires**
- vault cli to be installed locally & configured for https://vault.loodse.com/

#### Testing all providers
```bash
./run.sh
```

#### Common customizations

**Debug logs**

Setting `-debug` will enable the debug logs.

**Only test a specific provider**

The providers which should be covered can be set via the `-providers` flag.

For example, setting `-providers=aws` will only test AWS clusters.

**Parallelism**

To configure the number of clusters which should be tested in parallel, the `-kubermatic-parallel-clusters=4` can be use.

**Node count**

More nodes tend to improve test performance by some degree, though a higher node count might lead to reaching the quota.

The number of nodes, each cluster will have can be set via `-kubermatic-nodes=3`

**Keep clusters after test**

To be able to debug clusters, they must remain after a test has been run.
For this `-kubermatic-delete-cluster=false` can be specified, which will simply not delete the cluster after testing.

**Delete existing clusters from a previous run**

In case a previous run left some clusters behind - maybe due to the use of `-kubermatic-delete-cluster=false` -
they can be deleting during the next execution by setting `-cleanup-on-start=true`.

### Docker

TODO: Define
