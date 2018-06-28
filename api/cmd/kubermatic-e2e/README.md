# Automated conformance testing of kubermatic managed clusters

## What is this
It's a tool to automate creating of custome resources for kubermatic managed
clusters and machines and running conformance tests on top of them.

## Usage
Run docker container

    $ docker run --rm -it \
        -v /path/to/kubeconfig:/config/kubeconfig \
        -v /path/to/cluster_template.yaml:/manifests/cluster.yaml \
        -v /path/to/cluster_template.yaml:/manifests/machine.yaml \
        kubermatic-e2e

Image ships with dependant binaries like `ginkgo` and `e2e.test` from kubernetes
test suit.

## Flags
All flags have reasonable defaults

        --addons stringArray               comma separated list of addons (default [canal,dns,kube-proxy,openvpn,rbac])
        --alsologtostderr                  log to standard error as well as files
        --cluster string                   path to Cluster yaml (default "/manifests/cluster.yaml")
        --cluster-timeout duration         cluster creation timeout (default 3m0s)
        --e2e.test-bin string              path to e2e.test binary (default "/usr/local/bin/e2e.test")
        --focus string                     tests focus (default "[Conformance]")
        --ginkgo-bin string                path to ginkgo binary (default "/usr/local/bin/ginkgo")
        --kubeconfig string                path to kubeconfig file (default "/config/kubeconfig")
        --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
        --log_dir string                   If non-empty, write log files in this directory
        --logtostderr                      log to standard error instead of files
        --machine string                   path to Machine yaml (default "/manifests/machine.yaml")
        --nodes int                        number of worker nodes (default 1)
        --nodes-timeout duration           nodes creation timeout (default 10m0s)
        --parallel string                  parallelism of tests (default "1")
        --provider string                  cloud provider to use in tests (default "local")
        --reoprts-dir string               directory to save test results (default "/tmp/results")
        --skip string                      skip those groups of tests (default "Alpha|Kubectl|[(Disruptive|Feature:[^]]+|Flaky)]")
        --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
    -v, --v Level                          log level for V logs
        --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
