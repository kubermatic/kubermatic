# Integrate upstream `machineSet` and `machineDeployment` controller into our `machine-controller`

**Author**: Alvaro Aleman (@alvaroaleman)
**Status**: Draft proposal

## Motivation and Background

In order to simplify management of nodes, we want to offer to our customers the possibility
of grouping identical machines into `machineSets` and manage configuration changes of `machineSets`
via `machineDeployments`. This in turn means, that we need controllers for both `machineSets` and
`machineDeployments`.

These controllers already exist within the [sig cluster-api repo](https://github.com/kubernetes-sigs/cluster-api/tree/c8f5046fb0b9a3a16b7f8b92f6dda7b0f65b4f55):

* [machineSet controlller]: https://github.com/kubernetes-sigs/cluster-api/tree/c8f5046fb0b9a3a16b7f8b92f6dda7b0f65b4f55/pkg/controller/machineset
* [machineDeployment controller]: https://github.com/kubernetes-sigs/cluster-api/tree/c8f5046fb0b9a3a16b7f8b92f6dda7b0f65b4f55/pkg/controller/machinedeployment

To reduce development efforts we would like to use the upstream controllers. However simply importing them is
currently not possible, because the upstream types have diverged from what we imported.

The focus of this proposal is to outline the required steps to be able to use the `machineSet` and
`machineDeployment` controller from the upstream repository.

## Non-Goals

### Moving CRDs into the seed cluster

There was a discussion about moving the machine-related CRDs into the seed cluster. The background is that
`machines` may eventually reference attributes of the upstream `cluster` CR, e.G. Pod or service cidr. When our
`cluster-controller` at some point moves to using the `sig-cluster-api` `cluster` CRD, the `cluster` CR must be
inside the seed-cluster, because otherwise we have a chicken-egg problem.

However at this point in time, we are far away from using upstream types within the `cluster` controller. Moving
types into the seed cluster would complicate things a lot for the `machine-controller` because:

* We can not use `ownerReferences` on the nodes anymore which means we would have to introduce a finalizer to clean them up
* We would have to talk to two distinct clusters

Since there are currently no plans to move our cluster `CRD` to upstream types, we should refrain from doing any changes
related to that within the `machine-controller`.

## Implementation

### Change apigroup of `machines.k8s.io` to `cluster.k8s.io` and adapt the upstream type

We have to adapt our types to use the same apigroup, be namespaced and while being at it, it makes
sense to move the type itself to what is currently in upstream.

This is defined as being done when:

* There is a conversion that converts our current `machine` type to the upstream `machine` type
* There is a migration that executes the conversion and saves the machines in as a namespaced CR in the `cluster.k8s.io` apigroup
* The migration will only be executed by an elected leader
* The migration is written in a way that it can be interrupted at any point and will successfully continue on the
  next start
* After successfully finishing, the migration deletes the `machine.machines.k8s.io` CRD from the cluster
* All examples contain a label `upstream-type-version`: `commit-sha1-from-which-we-imported-the-types` to be able to
  write another migration if upstream changes the types in a backwards-incompatible way without incrementing the
  version
* The machine-controller does not process machines that do not have the correct `upstream-type-version`: `commit-sha1-from-which-we-imported-the-types`
  label
* All currently existing functionality is preserved, we have the same set of e2e tests, they just create different
  types now

### Import upstreams `machineSet` controller and plumb it in

This is defined as being done when:

* All currently existing e2e tests create a `machineSet` instead of a `machine` and pass

### Import upstreams `machineDeployment` controller and plumb it in

This is defined as being done when:

* All currently existing e2e tests create a `machineDeployment` instead of a `machine` and pass
* There is an e2e test which checks the recration of a machine after its `machineDeployment` was altered
