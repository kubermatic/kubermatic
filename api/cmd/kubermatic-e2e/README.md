# Automate creation of kubermatic managed clusters

## What is this
It's a tool to automate creating of kubermatic managed clusters

## Usage

### Run

For this tool to work at least 3 files has to exists:
* kubeconfig (with kubermatic installed)
* cluster.yaml definition (conform HTTP API model `github.com/kubermatic/kubermatic/api/pkg/api/v1.Cluster`)
* node.yaml template definition (conform HTTP API model `github.com/kubermatic/kubermatic/api/pkg/api/v1.Node`)

See definition examples below

```shell
# this will create user-cluster with 3 nodes (according to definition in node.yaml)
# and will dump user-cluster kubeconfig into ./user-cluster-kubeconfig
kubermatic-e2e \
      -kubeconfig=./kubermaticseedkubeconfig \
      -output=./user-cluster-kubeconfig
      -cluster=./cluster.yaml \
      -node=./node.yaml \
      -nodes=3
# now run any workloads on fresh user-cluster you'd like
kubectl --kubeconfig=./user-cluster-kubeconfig apply -f ./some-manifests

# cleanup
kubectl \
      --kubeconfig=./kubermaticseedkubeconfig delete cluster \
      $(kubectl --kubeconfig=./user-cluster-kubeconfig config view -ojsonpath='{.clusters[0].name}')
```

### Flags

```
Usage of kubermatic-e2e:
  -addons value
        comma separated list of addons (default canal,dns,kube-proxy,openvpn,rbac,kubelet-configmap,default-storage-class,metrics-server)
  -alsologtostderr
        log to standard error as well as files
  -cluster string
        path to Cluster yaml (default "cluster.yaml")
  -cluster-timeout duration
        cluster creation timeout (default 5m0s)
  -delete-on-error
        try to delete cluster on error (default true)
  -kubeconfig value
        path to kubeconfig file (default $HOME/.kube/config)
  -log_backtrace_at value
        when logging hits line file:N, emit a stack trace
  -log_dir string
        If non-empty, write log files in this directory
  -logtostderr
        log to standard error instead of files
  -namespace string
        namespace where kubermatic and it's configs deployed (default "kubermatic")
  -node string
        path to Node yaml (default "node.yaml")
  -nodes int
        number of worker nodes (default 3)
  -nodes-timeout duration
        nodes creation timeout (default 10m0s)
  -output string
        path to generated usercluster kubeconfig (default "usercluster_kubeconfig")
  -stderrthreshold value
        logs at or above this threshold go to stderr
  -v value
        log level for V logs
  -vmodule value
        comma-separated list of pattern=N settings for file-filtered logging
```

## Examples

### Example AWS Cluster

cluster.yaml

```yaml
name: mycoolcluster
spec:
  cloud:
    dc: "aws-eu-central-1a" # Datacenter key from datacenters.yaml
    aws:
      accessKeyId: << your AWS accessKeyId >>
      secretAccessKey: << your AWS secretAccessKey >>
  version: 1.12.1
```

### Example AWS Node

node.yaml

```yaml
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
    kubelet: 1.12.1
```
