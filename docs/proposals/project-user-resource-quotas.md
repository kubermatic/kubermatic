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
- storage (node disk size)

## Non-Goals
- The quotas won't work for imported external clusters.
- The quotas won't restrict the usage of resources for the control plane on the seed cluster.
- Nodes which join the cluster using other means then through KKP are not supported in the quotas.

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
- For simplicity, the user which is the owner of a cluster, is used when calculating quotas for the cluster nodes. Even though another user can create nodes in that cluster.
- When a node is deleted, the quota usage needs to decrease for:
  - project
  - the cluster owner
  - project datacenter
- If the quota is exceeded(by lowering the quota under current capacity for example), it is not the responsibility of KKP to remove nodes. Admins should fix that in communication with the project/user.
- The storage quota targets just the disk size of the node. It will only affect node-local PV storage.

## Implementation

### CRD Changes

Add a new flexible ResourceQuota CRD which will hold the desired quota and current consumption of the quota.

```go
type ResourceQuota struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   ResourceQuotaSpec   `json:"spec,omitempty"`
    Status ResourceQuotaStatus `json:"status,omitempty"`
}

type ResourceQuotaSpec struct {
	// QuotaSubject describes the object (user or project) for which the quota is applied to.
	QuotaSubject *QuotaSubject `json:"quotaSubject"`
	// Datacenter is the name of the datacenter for which a resource quota should be set.
    Datacenter     string          `json:"datacenter,omitempty"`
	// ResourceQuotas is a map of maximum resource quotas per resource
    ResourceQuotas corev1.ResourceList `json:"resourceQuotas"`
}

type QuotaSubject struct {
    // Type of the subject, can be `user` or `project`
    Type string `json:"type"`
    // Name of the quota subject
    Name string `json:"name"`
}

type ResourceQuotaStatus struct {
	// ResourceConsumption is map which holds the current usage of resources for all seeds
    ResourceConsumption corev1.ResourceList `json:"resourceConsumption"`
	// LocalConsumption is a map of the current usage of resources for the local seed
    LocalConsumption corev1.ResourceList `json:"localConsumption"`

}
```

The ResourceQuota is a master-cluster resource. It will need to be synced to seed clusters to allow webhooks access.
As we have a distributed platform over multiple seeds, each seed will have to report its usage in the status. Then the master quota controller
can recalculate the total resource consumption and re-sync the quotas on the seeds.

### How to ensure that we get proper resource usage and that it stays in sync?

The obvious way seems to get the amount of resources requested when a KKP MachineDeployment is being created, and then check against
the user, datacenter and project quota and fill them up accordingly. But this would only work through the API, and if there are some issues
we could have resource quotas which are out of sync.

The safest way would be to check the Node capacity on user clusters, but there are some issues with that as well, as the Nodes get created
based on the MachineDeployments. So there could be races in between when the Node is created(which can take some time) and when we check if user can create a MachineDeployment.

So the best would be to create a controller which watches MachineDeployments, and fills out the project, datacenter and user quotas accordingly.
And the check for resource quotas could then be in the webhook, so to cover both kubectl and API creation. There is an issue here
that MachineDeployments are user-cluster resources, so the webhook on the seed-cluster control plane won't be able to work. And we need
the webhook to be in the seed-cluster, as it needs access to quotas, projects, users and provider credentials.

TBD how to solve this. Ideally, MachineDeployments would be namespaced-scoped resources in the control-plane. Then we could easily control access to them,
and control the quotas through the webhook.

#### How to sync quota info from seed clusters and master cluster and back

We would need a controller for ResourceQuotas in the master cluster: `master-resource-quota-controller` - MRQC
And a controller for MachineDeployments in the seed cluster: `seed-md-controller` - SMDC

The MRQC would watch master cluster ResourceQuotas. These are the ones that are created through the API, and maybe one day through `kubectl`.
The role of the MRQC would be to reconcile the master ResourceQuotas to the seed clusters.
It also needs to watch seed cluster ResourceQuotas for changes in LocalConsumption, and update the master ResourceQuota ResourceConsumption based on it.

The SMDC would watch the seed cluster MachineDeployments(assuming we go with the plan to move MDs to the seed cluster, otherwise we 
need to have it watch MDs on user clusters, and do it a bit differently), calculate the resource usage, and update quotas for that clusters
project, owner and project datacenter (if such a quota exists). Also it will watch for new ResourceQuotas, so that for new ones it 
calculates the usage.

### Moving MachineDeployments to the seed-clusters user-cluster namespaces 

The issue here is that MachineDeployments are user-cluster resources, so cluster users can easily edit them and increase replica size to circumvent the quota.

One option is to move the MachineDeployments to the control plane, but as both KubeOne and KKP use machine-controller/MachineDeployments,
it is not certain if that is possible without breaking KubeOne. We could eventually dynamically set the kubeconfig of the cluster
where the machine-controller looks for MachineDeployments, and set it to the seed-cluster in case of KKP. But then, as
MachineDeployments are cluster-scoped we will need additional cluster info on them.

Ideally, MachineDeployments would be namespaced-scoped resources in the control-plane. Then we could easily control access to them,
and control the quotas through webhooks.

How to go about making the move is TBD

### How to get node size

The MachineDeployments for different providers only have node flavours in their spec. So the question is how to get the real node size.

Fortunately, we already have a feature which uses this, the ability for admins to limit the minimum and maximum node CPU and RAM. 
It uses provider clients to get node sizes and the filters if they fit. We can use the same provider clients, although not all providers
have this option.

Below is a table of providers and how to get the node size.

| Provider     | Provider-client node size | Alternative                                            | Comment                          |
|--------------|---------------------------|--------------------------------------------------------|----------------------------------|
| Alibaba      | Y                         |                                                        |                                  |
| AWS          | Y                         |                                                        | Not dynamic, loaded from library |
| Azure        | Y                         |                                                        |                                  |
| DigitalOcean | Y                         |                                                        |                                  |
| GCP          | Y                         |                                                        |                                  |
| Hetzner      | Y                         |                                                        |                                  |
| Openstack    | Y                         |                                                        |                                  |
| KubeVirt     | N                         | We set the requested size directly into MachineDeployment |                                  |
| Nutanix      | TBD                       | TBD                                                    |                                  |
| Equinox      | Y                         |                                                        |                                  |
| vSphere      | N                         | We set the requested size directly into MachineDeployment |                                  |

### Default Project Quotas

The idea here is to allow admins to set a default project resource quota, that will be applied to all projects which don't have a resource quota set.

The default project resource quota can be set in the Global Settings, where we already have similar settings like `maxProjects`, and min/max node size.
The trick now is how to apply it. We could either create ResourceQuotas for every project that doesn't have it, or
not, but then we would have to calculate everytime in the quota webhook how many resources a project is currently consuming.

That's why creating ResourceQuotas for all projects that don't have them set seems like a simpler idea. To support editing/removing
the default resource quota, we could label all default resource quotas with `default=true`, so we know to distinguish them from
custom resource quotas.

### Possible Issues

#### Can't move MachineDeployments to seed cluster

Currently, the whole plan kinda hangs on the fact that we would like to move MachineDeployments to the seed cluster so that we can:
- validate them through the webhook based on quotas
- hide them from users on user cluster
- be able to easily watch them from a controller in the seed cluster

#### Race when updating quotas

There is still a chance of a race, in which users could create multiple MachineDeployments for a project in the same time, before the
controller updates the resource usage, thus bypassing the limit. In this case we have 2 options:
- admins could notice/get informed about it and react according to their company policy
- add a "pending"/"reserved" resource mechanism to the API. So as soon as a MachineDeployment request is received in the API, it would reserve some of the quota. Later the MachineDeployment controller could move this from reserved to real usage.

### Possible Enhancements in the future

1. Try to make the resourcequotas bulletproof by adding a mechanism of "pending"/"reserved" quota which is filled before the MachineDeployment creation
2. Add a possibility to set user groups and set quotas per group
3. Possibility to set `maxClusters` for a project/user

## Tasks and effort

1. Investigate project and user quotas
2. Implement resource quotas CRD
3. Move MachineDeployments to the seed cluster
4. Implement MRQC
5. Implement SMDC
6. Validating webhook for MDs based on resource quotas
7. ResourceQuota API endpoints
8. (Optional) Implement default project quotas
9. (Optional) Implement a "pending/reserved" mechanism to decrease chances of races

