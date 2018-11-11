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
TBD

## Examples

### Example AWS Cluster spec

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
