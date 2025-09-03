# Ginkgo Conformance Tests

This document describes how to run, configure, and debug the Ginkgo-based conformance tests for Kubermatic.

## Installation

Before you can run the tests using the `ginkgo` CLI, you need to install it:

```bash
go install github.com/onsi/ginkgo/v2/ginkgo@latest
```

## Running Tests

There are two ways to run the conformance tests: using `go test` or the `ginkgo` CLI. Before running the tests, make sure to set the `CONFORMANCE_TESTER_CONFIG_FILE` environment variable to point to your configuration file.

```bash
export CONFORMANCE_TESTER_CONFIG_FILE=./config.yaml
```

### Using `go test`

To run the conformance tests, you can use the standard `go test` command.

```bash
go test -v ./pkg/ginkgo/...
```

### Using the `ginkgo` CLI

Alternatively, you can use the `ginkgo` CLI to run the tests. This provides more options for controlling the test execution and reporting.

```bash
ginkgo -v ./pkg/ginkgo/...
```

You can also run tests in parallel:

```bash
ginkgo -p -v ./pkg/ginkgo/...
```

## Configuration

The conformance tests are configured using a YAML file. The path to this file must be specified via the `CONFORMANCE_TESTER_CONFIG_FILE` environment variable.

### Example Configuration

```yaml
# A prefix for all created resources.
namePrefix: "ginkgo-test"

# A list of providers to test.
providers:
- KubeVirt

# A list of Kubernetes releases to test.
releases:
- "1.23"

# A list of enabled operating system distributions.
enableDistributions:
- ubuntu

# A list of enabled tests. If empty, all tests are run.
enableTests: []

# A list of excluded tests.
excludeTests: []

# The file to write the test results to.
resultsFile: "results.json"

# If set, only scenarios that failed in a previous run will be executed.
retryFailedScenarios: false

# If set, the tests will be run in a dual-stack environment.
dualStackEnabled: false

# If set, the tests will be run with Konnectivity enabled.
konnectivityEnabled: true

# If set, the tests will include a cluster update.
testClusterUpdate: false

# Timeouts
controlPlaneReadyWaitTimeout: 10m
nodeReadyTimeout: 20m
customTestTimeout: 10m

# Cluster settings
deleteClusterAfterTests: true
nodeCount: 1

# Paths
reportsRoot: "_reports"
logDirectory: "_logs"

# Kubermatic settings
kubermaticNamespace: "kubermatic"
kubermaticSeedName: "kubermatic"
kubermaticProject: "" # will be created if empty

# Secrets for the providers
secrets:
  kubevirt:
    kubeconfig: "/path/to/your/kubevirt-kubeconfig"
    kkpDatacenter: "kubevirt-dc"
```

## Secrets Management

Provider secrets can be provided directly in the configuration file or loaded from external files. To load a secret from a file, append `File` to the secret key and provide a path to the file.

**Example:**

```yaml
secrets:
  kubevirt:
    # Provide the Kubeconfig directly
    kubeconfig: "apiVersion: v1..."
  hetzner:
    # Load the token from a file
    tokenFile: "/path/to/hetzner-token"
```

## Test Reporting

The tests generate reports in JUnit XML format. By default, these reports are saved in the `_reports` directory. You can change this location using the `reportsRoot` key in your configuration file. The name of the file will be `junit_ginkgo.xml`.

When using the `ginkgo` CLI, you can also generate other types of reports:

```bash
# Generate a JSON report
ginkgo --json-report=report.json ./pkg/ginkgo/...

# Generate a TeamCity report
ginkgo --teamcity-report=report.teamcity ./pkg/ginkgo/...
```

## Debugging

To debug the tests, you can use the Delve debugger.

1.  Set the `CONFORMANCE_TESTER_CONFIG_FILE` environment variable.

    ```bash
    export CONFORMANCE_TESTER_CONFIG_FILE=./config.yaml
    ```

2.  Build the test binary:

    ```bash
    go test -c ./pkg/ginkgo/... -o ginkgo.test
    ```

3.  Run the test binary with Delve:

    ```bash
    dlv exec ./ginkgo.test -- -test.v -ginkgo.v
    ```

You can then set breakpoints and inspect the state of the application as you would with any other Go program.
