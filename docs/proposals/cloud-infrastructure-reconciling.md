# Machine-controller integration

**Author**: Henrik Schmidt (@mrincompetent)

**Status**: Proposal

A kubernetes controller regularly checks the desired state & actual state. While comparing it tries during each iteration to bring the actual state closer to the desired state.
While this concept was already implemented for the kubermatic cluster controller in terms of Manifests and cluster properties, it is still lacking support for the cloud infrastructure.
During each cluster creation the controller might create resources at a cloud provider - despite the machines/instances.
Those resources are currently only being created during the initial cluster bootstrap. After this they stay as they are.
As it might be necessary (just reached that point) to change resources at the cloud provider we need to implement this actual/desired state behaviour for the cloud provider resources as well.

## Motivation and Background

The current way we manage cloud provider resource (despite from machines/instances):

*   Gets created during initial cluster bootstrap
*   Won't get updated

With the new approach the controller will automatically detect when it need to update.

## Implementation

Kubermatic-cluster-controller:
*   During each cluster synchronisation we check the cloud provider resources
*   Update/Recreate when necessary

## Task & effort:
*   Openstack - 0.5d
*   AWS 1d
