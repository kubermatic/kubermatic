# Cluster Dehydration

**Author**: Marvin Beckers (@embik)

**Status**: Draft proposal.

## Goals

Dehydration is a functionality for KKP user clusters that will allow to temporarily "suspend" a cluster to free up cloud resources used by the cluster. The feature will allow to optimize resource consumption and subsequent spending on cloud resources. State (e.g. storage) should not be affected by this functionality.

## Non-Goals

Migrating dehydrated user clusters between seeds is out of scope for this proposal. User clusters can only be re-started on the same seed.

## Motivation and Background

Many workloads running on Kubernetes clusters do not need to be online all day, especially if they are deployed for testing or staging purposes. While implementing manual scaling schedules within the user cluster can help with that, clusters will still occupy resources. If a complete cluster is not needed during off-hours (e.g. because it is a testing environment used by employees to verify a software stack before releasing it), shutting down that cluster for 14-16h a day might yield significant savings in both energy and costs.

Of course clusters could be deleted in the evening, but this comes with several downsides:

* Any state/data is lost if the cluster and its storage is fully deleted.
* This is self-written automation that needs to be developed and maintained.
* Re-provisioning a new cluster with the same application stack might take a considerable amount of time at the beginning of each shift.

Having a "one click" solution to dehydrate your user cluster will provide the following upsides:

* Quickly free up resources in a resource-constrained environment.
* Restart work right where it was left off maybe days or even weeks ago while staying cost-efficient.
* No maintenance for custom scripts or other automation that does similar things.

To make sure this feature is useful for end users, we have made the following key decisions:

* The feature will focus on VMs as the primary driver for cost and energy optimization.
* Storage and Load Balancers remain untouched, so that application state and public endpoints remain stable over a de-/hydration cycle.
* Other cloud provider resources (e.g. security groups, route tables, etc) will not be deleted either, as they usually incur little cost.
* The control plane running in the seed will be suspended as well, since a reduction in resource consumption on the seed is also useful and the cluster will not be usable anyway.

## Implementation

Implementation will rely on a new field added to our CRDs for `Cluster`. This would look like the example below (the field name can be subject to discussion):

```yaml
apiVersion: kubermatic.k8c.io/v1
kind: Cluster
metadata:
  name: xyz1234
spec:
  humanReadableName: dehydration-cluster
  dehydrated: true
  [...]
```

### UI

The user experience for this feature should be pretty straightforward: User clusters have a button / UI interface to deyhdrate a cluster. It gives proper warning what the feature does, worded something like this:

> Dehydrating a user cluster will evict all workloads, terminate all active nodes and make the Kubernetes API unavailable. Persistent storage, load balancers and similar cloud resources will not be affected and will continue to exist. Make sure that this cluster is not actively used before dehydrating it. Dehydrated user clusters can be restored at any given time, assuming resource quotas allow the re-creation of nodes. Are you sure you want to dehydrate your cluster?

The key takeaway here is that because of its destructive nature, this should require confirmation when done from the UI.

When a cluster is dehydrated, the UI should show this, perhaps even with a special icon in the cluster overview.

### Controller

If `spec.dehydrated=true` is set, a `(de-)hydration-controller` needs to ensure the following things:

1. All nodes are drained completely, evicting all running pods. In particular, this needs to deal with `PodDisruptionBudgets` properly (by not honoring them). Because of this limitation, the controller might opt for cordoning off all nodes and directly deleting pods (in contrast to using the eviction API to gracefully evict pods), which is a highly disruptive process.
1. All MachineDeployments are scaled down to zero to remove the nodes and associated VMs on the cloud provider side. The controller needs to store the pre-dehydration replica count in an annotation, so the state before dehydrating the cluster is remembered and can be restored later.
1. The control plane components in the seed are scaled to zero, either by the `dehydration-controller` or by the controllers usually managing them.
1. Controllers need to be aware of a cluster being dehydrated and should skip reconciling them.

It is likely that for the first two steps, a dehydration phase indicator needs to be added to the `Cluster` status, since the third step will not allow to interact with objects on the user cluster anymore and the previous steps need to be skipped at that point. It could look something like:

```yaml
status:
  dehydration:
    phase: MachinesScaledDown
---
status:
  dehydration:
    phase: Active
```

The condition that other controllers would react to could look like this:

```yaml
status:
  conditions:
    ClusterDehydrated:
      kubermaticVersion: v2.22.0-23-gbc5f9c5f6
      lastHeartbeatTime: "2023-03-08T09:28:12Z"
      status: "True"
    [...]
```

To reverse the dehydration (to "hydrate" a cluster again), the changes to control plane components need to be undone. This can be achieved by removing the condition that prevents controllers from reconciling the dehydrated cluster, which should cause the control plane components to be scaled up again. Once the cluster is up again, the `(de-)hydration-controller` needs to restore previous state of `MachineDeployments` by reading the annotation and scaling them as appropriate.

#### Cluster Status before Dehydration

Of particular concern is how to keep users and admins informed of what a user cluster was like before it got dehydrated. This includes key information like how many `PersistentVolumes`, `LoadBalancers` and `Machines` existed in the cluster, since shutting down the control plane means there is no way to enumerate them before hydrating again.

Because of that, adding a status field to `Clusters` might be beneficial to include some basic information. This will not give the full details, but rather an "executive summary" of what the cluster looks like, what is still incurring costs and what it means to hydrate the cluster again. The status field could look something like this:

```yaml
apiVersion: kubermatic.k8c.io/v1
kind: Cluster
metadata:
  name: xyz1234
spec:
  humanReadableName: dehydration-cluster
  dehydrated: true
  [...]
status:
  dehydration:
    phase: Active
    remaining:
      volumes: 3
      loadBalancers: 2
    scaledDownMachines: 5
```

This information should be shown in the UI in some way, to highlight that some resources associated with this cluster likely still exist (which is what the feature is supposed to ensure). `status.dehydration.scaledDownMachines` can be used in the dialog to re-hydrate a cluster again, letting the user know that x machines will be created when hydrating the user cluster.

### Limitations

* "Bring Your Own" clusters cannot be dehydrated because KKP does not control the nodes and cannot scale them down. Dehydrating such a cluster is therefore useless.
* Dehydrated user clusters will not count towards resource quotas, since the nodes will be removed.

## Alternatives Considered

The alternatives below have been discussed and considered and at the current state of the proposal are still deemed valid alternatives. Please leave comments if you believe them to be better than the implementation proposal above.

### Pausing VMs

Instead of scaling down `MachineDeployments`, it was proposed to pause/suspend VMs via cloud provider functionality. This would allow to bring back the original nodes in a re-hydrated user cluster. The following challenges stem from that idea:

* Support to properly suspend VMs via the cloud provider's individual functionality needs to be implemented in machine-controller. Not all providers might support such functionality and we would need an alternative implementation (most likely the implementation proposal) to fall back to anyway.
* Pausing VMs might be less cost-efficient given that the VM itself will not be billed for most providers, but there is more ambiguity around paying for allocated IPs or OS disks.
* It encourages relying on specific nodes in cloud environments. "Cattle, not pets" is a common principle in the Kubernetes community and would be violated by such implementation. If you rely on specific nodes, dehydrating a cluster is probably not a good idea.

### More Details for Scaled Down Machines

As discussed in [Cluster Status before Dehydration](#cluster-status-before-dehydration), knowing what the cluster looked like before dehydrating it can be quite useful. Instead of giving an executive summary, it might make sense to store more detailed information about the `MachineDeployments` that have been scaled down - This could lead to better integration with the [resource quota (EE) feature](https://docs.kubermatic.com/kubermatic/v2.22/architecture/concept/kkp-concepts/resource-quotas/), giving users a detailed overview of what impact a re-hydrated cluster will have on the project's quotas.

## Tasks & Effort

TBD
