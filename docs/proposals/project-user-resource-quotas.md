# Project/User/Datacenter resource quotas

**Authors**: Lovro Sviben (@lsviben)

**Status**: Draft proposal.

**Issue**: https://github.com/kubermatic/kubermatic/issues/8042

## Goals
The idea is to allow admins to control how many resources are users using by introducing resource quotas.

The quotas would restrict:
- resource usage per project
- resource usage of a datacenter per project
- user resource usage

Resources that will be under quota:
- vCPU
- RAM
- storage

## Non-Goals


## Motivation and Background

Clients would like to have options to fine-grain restrict resource usage per project, of a datacenter per project and individual resource usage.
The reasons vary from being able to stop the depletion of resources in a shared datacenter, to being able to sell packages (projects) with a set 
number of resources available. 

This will be an EE feature.

## How will the resource quotas work

Three types of resource quotas:
- Project quota - amount of resources (vCPU, RAM, storage) that can allocated for nodes in a project
- Project datacenter quota - amount of resources (vCPU, RAM, storage) that can allocated for nodes in a datacenter for a project. 
- User quota - amount of resources (vCPU, RAM, storage) that can allocated for nodes when a user is creating them


Rules:

- The quotas are for node resources
- The resource quotas management will be admin-only.
- By default, projects, project datacenters and users will not have a quota, meaning they can make as many resources they want
- There will be a way to set a default resource quota for every project (should we do the same for users?)
- When a user A creates a cluster in a project B, in datacenter C, the quotas for all of those are checked, and get filled up
- When a node is deleted, the quota usage needs to decrease for:
  - project
  - user who created the node (not the one who deleted it)
  - project datacenter

Questions?
- how should this work for external clusters?
  - For example importing a cluster to a project will change its resource quotas or not?
  - should we block imports that go over the quota
  - what about node creation for external clusters

## Implementation

### How to ensure that we get proper resource usage and that it stays in sync?

The obvious way seems to get the amount of resources requested when a KKP NodeDeployment is being created, and then check against
the user, datacenter and project quota and fill them up accordingly. But this would only work through the API, and if there are some issues
we could have resource quotas which are out of sync.

The safest way would be to check the Node capacity on user clusters, but there are some issues with that as well, as the Nodes get created
based on the NodeDeployments. So there could be races in between when the Node is created(which can take some time) and when we check if user can create a NodeDeployment.

So the best would be to create a controller which watches NodeDeployments, and fills out the project, datacenter and user quotas accordingly.
And the check for resource quotas could then be in the webhook, so to cover both kubectl and API creation.

There is still a chance of a race, in which multiple users could create NodeDeployments for a project in the same time, before the
controller updates the resource usage, thus bypassing the limit. In this case we have 2 options:
- admins could notice/get informed about it and react according to their company policy
- add a "pending"/"reserved" resource mechanism to the API. So as soon as a NodeDeployment request is received in the API, it would reserve some of the quota. Later the NodeDeployment controller could move this from reserved to real usage.


### CRD Changes

Add a new flexible ResourceQuota CRD which will hold the desired quota and current consumption of the quota. 

The ResourceQuota will need to have an OwnerReference to a Project or User to which the quota applies to. If a specific datacenter is targeted for a project,
the Datacenter field needs to be set.

```go
type ResourceQuota struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   ResourceQuotaSpec   `json:"spec,omitempty"`
    Status ResourceQuotaStatus `json:"status,omitempty"`
}

type ResourceQuotaSpec struct {
	// Datacenter is the name of the datacenter for which a resource quota should be set.
    Datacenter     string          `json:"datacenter,omitempty"`
	// ResourceQuotas is a map of maximum resource quotas per resource
    ResourceQuotas corev1.ResourceList `json:"resourceQuotas"`
}

type ResourceQuotaStatus struct {
	// ResourceConsumption is map which holds the current usage of resources per resource 
    ResourceConsumption corev1.ResourceList `json:"resourceConsumption"`
}
```

### Possible Enhancements in the future

1. Try to make the resourcequtas bulletproof by adding a mechanism of "pending"/"reserved" quota which is filled before the NodeDeployment creation
2. Add a possibility to set user groups and set quotas per group

## Tasks and effort

1. Investigate project and user quotas
2. Implement resource quotas CRD
3. Implement default project quotas 
4. ResourceQuota API endpoints
5. Implement controller for NodeDeployments which fills the resource quotas
6. Webhook validation for resource quotas
7. (Optional) Implement a "pending/reserved" mechanism to decrease chances of races

