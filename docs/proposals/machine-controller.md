# Machine-controller integration

**Author**: Henrik Schmidt (@mrincompetent)

**Status**: Done

The machine-controller uses the machine-api discussed in https://github.com/kubernetes/kube-deploy.
It offers a simple and stable solution to machine-management.
The approach the machine-controller takes, differs from the node-controller.

## Motivation and Background

The current way we manage nodes has some downsides:

*   There's a new "official" api definition for machine-management
*   We extended the core node object via annotations
*   Node.spec.providerID is read-only. Currently we do a buffered deletion when a node comes up
*   We used docker-machine which is poorly written and was not flexible enough
*   Uses ssh to provision nodes

With the machine-controller, we have a more cloud-native way and more flexibility:

*   New extensible machine object
*   Machine provisioning via cloud-init
*   Support for specific container runtime (docker & cri-o atm)

## Implementation

Kubermatic-api:
*   We create new v2 endpoints (backwards compatibility)
*   Only one node per call (clear restful api)
*   New API type for the node (abstracted version of the machine+node)
*   The api creates machine resources in the customer cluster

kubermatic-controller:
*   The machine-controller gets deployed next to the existing node-controller (backwards compatibility)
*   The node-controller will stay for the next 6 month to offer a migration way

## Task & effort:
*   Define new api types - 0.5d
*   Add initial swagger documentation - 0.5d
*   Handle wrapped machine&node type via the api - 1d
*   Add machine-controller to cluster-deployment - 0.5d
*   Add new version & update for rollout - 0.5d
