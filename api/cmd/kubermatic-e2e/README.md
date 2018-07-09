# Automated conformance testing of kubermatic managed clusters

## What is this
It's a tool to automate creating of custome resources for kubermatic managed
clusters and machines and running conformance tests on top of them.

## Usage
Run docker container

    $ docker run --rm -it \
        -v /path/to/kubeconfig:/config/kubeconfig \
        -v /path/to/cluster_template.yaml:/manifests/cluster.yaml \
        -v /path/to/node_template.yaml:/manifests/node.yaml \
        kubermatic-e2e

Image ships with dependant binaries like `ginkgo` and `e2e.test` from kubernetes
test suite.

## Flags
All flags have reasonable defaults

    -alsologtostderr
        log to standard error as well as files
    -e2e-provider string
        cloud provider to use in tests (default "local")
    -e2e-results-dir string
        directory to save test results (default "/tmp/results")
    -e2e-test-bin string
        path to e2e.test binary (default "/usr/local/bin/e2e.test")
    -ginkgo-bin string
        path to ginkgo binary (default "/usr/local/bin/ginkgo")
    -ginkgo-focus string
        tests focus (default "\\[Conformance\\]")
    -ginkgo-nocolor
        don't show colors
    -ginkgo-parallel int
        parallelism of tests (default 25)
    -ginkgo-skip string
        skip those groups of tests (default "Alpha|\\[(Disruptive|Feature:[^\\]]+|Flaky)\\]")
    -ginkgo-timeout duration
        ginkgo execution timeout (default 1h30m0s)
    -kubeconfig string
        path to kubeconfig file (default "/config/kubeconfig")
    -kubermatic-addons value
        comma separated list of addons (default canal,dns,kube-proxy,openvpn,rbac,kubelet-configmap)
    -kubermatic-cluster string
        path to Cluster yaml (default "/manifests/cluster.yaml")
    -kubermatic-cluster-timeout duration
        cluster creation timeout (default 3m0s)
    -kubermatic-delete-cluster
        delete test cluster at the exit (default true)
    -kubermatic-namespace string
        namespace where kubermatic and it's configs deployed (default "kubermatic")
    -kubermatic-node string
        path to Node yaml (default "/manifests/node.yaml")
    -kubermatic-nodes int
        number of worker nodes (default 3)
    -kubermatic-nodes-timeout duration
        nodes creation timeout (default 10m0s)
    -log_backtrace_at value
        when logging hits line file:N, emit a stack trace
    -log_dir string
        If non-empty, write log files in this directory
    -logtostderr
        log to standard error instead of files
    -stderrthreshold value
        logs at or above this threshold go to stderr
    -v value
        log level for V logs
    -vmodule value
        comma-separated list of pattern=N settings for file-filtered logging

## Examples

### Example AWS Cluster spec

    apiVersion: kubermatic.k8s.io/v1
    kind: Cluster
    metadata:
      name: aws-e2e-cluster
      labels:
        user: << Optional UserID to see cluster in UI>>
        worker-name: ""
    spec:
    cloud:
      dc: "aws-eu-central-1a" # Datacenter key from datacenters.yaml
      aws:
        accessKeyId: << your AWS accessKeyId >>
        secretAccessKey: << your AWS secretAccessKey >>
    humanReadableName: aws-e2e-test-runner
    pause: false
    version: 1.10.5

### Example AWS Node spec

    spec:
      cloud:
        aws:
          instanceType: t2.medium
          diskSize: 25
          volumeType: gp2
      operatingSystem:
        ubuntu:
          distUpgradeOnBoot: false
      versions:
        kubelet: 1.10.5
        containerRuntime:
          name: docker
          version: 17.03.2
