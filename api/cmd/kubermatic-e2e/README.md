# Automate creation of kubermatic managed clusters

## What is this
It's a tool to automate creating of custom resources for kubermatic managed
clusters and machine objects

## Usage
Run docker container

    $ docker run --rm -it \
        -w /app \
        -e KUBECONFIG=/config/kubeconfig \
        -v /host_path/to/kubeconfig:/config/kubeconfig \
        -v /host_path/to/cluster_template.yaml:/app/cluster.yaml \
        -v /host_path/to/node_template.yaml:/app/node.yaml \
        kubermatic/kubermatic user-cluster

## Flags

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
          path to kubeconfig file (default /Users/kron/DEV/loodse/secrets/seed-clusters/dev.kubermatic.io/kubeconfig)
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

## Examples

### Example AWS Cluster spec

```yaml
apiVersion: kubermatic.k8s.io/v1
kind: Cluster
metadata:
  name: aws-cicd-tmp-cluster
  labels:
    user: << Optional UserID to see cluster in UI>>
    worker-name: ""
spec:
cloud:
  dc: "aws-eu-central-1a" # Datacenter key from datacenters.yaml
  aws:
    accessKeyId: << your AWS accessKeyId >>
    secretAccessKey: << your AWS secretAccessKey >>
humanReadableName: aws-cicd-tmp-test-runner
pause: false
version: 1.10.5
```

### Example AWS Node spec

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
    kubelet: 1.10.5
    containerRuntime:
      name: docker
      version: 17.03.2
```
