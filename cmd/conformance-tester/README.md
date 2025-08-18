# E2E conformance tester

Runs static test scenarios against a set of KKP userclusters.

The conformance tester will, by default, test all supported versions on providers using all supported operating systems,
by creting one cluster for each combination, waiting for it to become healthy, adding nodes and then executing the
selected tests. Optionally the tests can be executed in a pre-existing cluster.

The conformance tester is used both in the KKP e2e tests as well as manually to perform pre-release tests.

## Tests

The command which will execute the following tests:

- Simple PVC (Only: AWS, Azure, OpenStack, vSphere):
  A StatefulSet with a PVC template will be created. The Pod, which mounts the PV has a ReadinessProbe in place
  which will only report ready when the pod was able to write something to the mounted PV.
  The PVC has no StorageClass set, so the default StorageClass, which gets deployed via the default kubermatic Addon `default-storage-class`, will be used.
- Simple LB (Only: AWS & Azure):
  The [Hello Kubernetes](https://kubernetes.io/docs/tasks/access-application-cluster/service-access-application-cluster/#creating-a-service-for-an-application-running-in-two-pods) Pod will be deployed with a Service of type LoadBalancer.
  The test will wait until the LoadBalancer is available and only report a success when the "Hello Kubernetes" Pod could be reached via the LoadBalancer IP(Or DNS).
- [Kubernetes Conformance tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md): First all parallel, afterwards all serial tests
- Telemetry: Verify that telemetry data is sent by the usercluster.
- Metrics: Verify that all components expose their expected metrics.

Check `pkg/tests/` for all the individual testcases.

## Caveats

All providers have custom quotas. Hitting the quota is fairly easy when testing too many clusters at once.

## Running

The tester needs Kube API access to a specific KKP _seed_ cluster (can be a shared master/seed). All clusters will
be scheduled onto this given seed.

Depending on the cloud provider, additional credentials must be provided. See the `hack/run-conformance-tests.sh`
script for more information.

You can specify a fixed project (`-kubermatic-project`) or let the tester create a temporary project on-the-fly.
Note that you need to cleanup projects after failed tests, like removing any added SSH keys to prevent conflicts
on the next run.

Run `./hack/run-conformance-tests.sh -help` for more information.

### Running a full provider tests

The following is an example script to help in running a variety of test scenarios on AWS. The flags are
optimized to run a larger number of test scenarios, compared to the `hack/run-conformance-tests.sh`, which
is geared more towards a single scenario.

Note how `-exclude-tests` is used to skip the expensive and lengthy Kubernetes conformance tests.

```bash
#!/usr/bin/env bash

make clean conformance-tester

# In a QA scenario, there is usually a Preset available with the credentials
# for that QA environment.
accessKey="$(kubectl get presets qa -o json | jq '.spec.aws.accessKeyID' -r)"
secretAccessKey="$(kubectl get presets qa -o json | jq '.spec.aws.secretAccessKey' -r)"

_build/conformance-tester \
  -aws-access-key-id "$accessKey" \
  -aws-secret-access-key "$secretAccessKey" \
  -aws-kkp-datacenter "aws-eu-west-1a" \
  -providers "aws" \
  -distributions "${DISTRIBUTIONS:-}" \
  -releases "${RELEASES:-}" \
  -container-runtimes "${RUNTIMES:-}" \
  -client "kube" \
  -log-format "Console" \
  -name-prefix "qa" \
  -exclude-tests "conformance,telemetry" \
  -wait-for-cluster-deletion=false \
  -kubermatic-seed-cluster "kkp-qa-env" \
  -reports-root "$(realpath reports)" \
  -kubermatic-parallel-clusters ${PARALLEL:-3}
```

This script can be used like so:

```bash
DISTRIBUTIONS=ubuntu RELEASES=1.27 runtests.sh
```

### Common customizations

**Debug logs**

Setting `-log-debug=true` will enable the debug logs.

**Only test a specific provider / OS**

The providers which should be covered can be set via the `-providers` flag, which is a comma-separated list of
provider names. For example, setting `-providers=aws` will only test AWS clusters.

The same goes for `-distributions`, which can be used like `ubuntu, flatcar`.

`-releases` is likewise a comma-separated list of Kubernetes releases to test (usually just major.minor).
The tester will automatically choose the most recent version supported for each given release. You can also
explicitly give a full version like `1.26.1`.

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

### Scenario-based testing

Instead of providing all test matrix options via the command line, it's possible to define test scenarios in a YAML file. This is useful for running a fixed set of tests or for complex test configurations.

Use the `-scenarios-file` flag to specify the path to the scenario file.

Execution model with scenarios-file:
- One cluster is created per scenario (provider + operatingSystem + version).
- All configured flavors for that scenario are created as separate MachineDeployments in the same cluster.
- If a provider does not support flavors, its scenario will produce the default MachineDeployment(s).
- Versions can be specified as minors (e.g. "1.31"); at runtime they are resolved to the latest supported patch version.

Example `scenarios.yaml`:

```yaml
versions:
  - "1.31"
  - "1.32"
scenarios:
  - provider: kubevirt
    operatingSystem: ubuntu
    flavors:
      - name: small
        value:
          virtualMachine:
            template:
              cpus: "2"
              memory: "2Gi"
              primaryDisk:
                size: "20Gi"
                storageClassName: "local"
                osImage: "http://image-repo.kube-system.svc/images/ubuntu-22.04.qcow2"
      - name: medium
        value:
          virtualMachine:
            template:
              cpus: "4"
              memory: "4Gi"
              primaryDisk:
                size: "40Gi"
                storageClassName: "local"
                osImage: "http://image-repo.kube-system.svc/images/ubuntu-22.04.qcow2"
  - provider: hetzner
    operatingSystem: ubuntu
    # no flavors: uses provider defaults
```

### Generating Scenarios

The conformance tester can generate a scenario file from a template. This is useful for creating a comprehensive test plan that covers all possible combinations of providers, operating systems, versions, and provider-specific flavors.

The `generate` subcommand is used for this purpose. It takes a template file and generates a `scenarios.yaml` file.

Usage:

```bash
_build/conformance-tester generate --from template.yaml --to scenarios.yaml
```

Template example (excerpt):

```yaml
providers:
  - kubevirt
  - hetzner

distributions:  # operating systems
  - ubuntu
  - rockylinux

versions:       # minors; resolved to latest patches at runtime
  - "1.31"
  - "1.32"

kubevirt:
  virtualMachine:
    template:
      cpus: ["2", "4"]
      memory: ["2Gi", "4Gi"]
      primaryDisk:
        size: ["20Gi", "40Gi"]
        storageClassName: ["local"]
        osImage: [
          "http://image-repo.kube-system.svc/images/ubuntu-22.04.qcow2"
        ]
```

Behavior:
- The generator computes the Cartesian product of providers × distributions and expands any lists under provider-specific sections into flavor combinations.
- For KubeVirt flavors, the image should be provided at `virtualMachine.template.primaryDisk.osImage`.
- The generated `scenarios.yaml` contains a top-level `versions:` list and per-scenario `flavors:`.
- YAML keys are sanitized to strings to avoid invalid keys.
