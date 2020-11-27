# Cluster default component settings webhook

**Author**: Furkhat @furkhat

**Status**: Draft proposal; prototype in progress.

## Goals

*short description of the topic e.g.*
Add mutating admission webhook that reads configured configmap with default components settings and sets `.spec.componentsOverride` for new clusters.

## Motivation and Background

*What is the background and why do we want to deplyo it e.g.*

Clusters have `.spec.componentsSettings` field which describes configuration of cluster components (like apiserver replicas, etcd disk size etc).
The only way to do it right now is to deploy new binary with adjusted constants.
We need a flexible way te set components setting for new clusters that do not require redeployment.

## Implementation

I propose to
- Configure default components setting in ConfigMap
- Add mutating admission webhook that will intercept creation of clusters
- Use seed controller manager server to host the endpoint
- Read the ConfigMap and set new clusters `.spec.componentsOverride`
- Delete existing clustercomponentdefaulter controller.

## Alternatives considered

- Extend clustercomponentdefaulter controller to accept configuration (through ConfigMap or cli flags)
