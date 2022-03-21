# Project resource quotas

**Authors**: Lovro Sviben (@lsviben)

**Status**: Draft proposal.

**Issue**: https://github.com/kubermatic/kubermatic/issues/8042

## Goals
The idea is to allow admins to control how many resources are projects using by introducing resource quotas.

Resources that will be under quota:
- vCPU
- RAM
- storage (node disk size) - TBD, we can remove this as it's not a strict requirement

## Non-Goals
- The quotas won't work for imported external clusters.
- The quotas won't restrict the usage of resources for the control plane on the seed cluster.
- Nodes which join the cluster using other means then through KKP are not supported in the quotas.

## Motivation and Background

Clients would like to have options to fine-grain restrict resource usage per project.
The reasons vary from being able to stop the depletion of resources in a shared datacenter, to being able to sell packages (projects) with a set 
number of resources available. 

This will be an EE feature.

## How will the resource quotas work

Project quota - amount of resources (vCPU, RAM, storage) that can allocated for nodes in a project.

Rules:

- The quotas are for node resources
- The quotas are enforced(we should block creation of new nodes that would exceed the quota)
- The resource quotas management will be admin-only.
- By default, projects will not have a quota, meaning they can use as many resources they want
- There will be a way to set a default resource quota for every project 
- When a node is deleted, the quota usage needs to decrease for the project
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
    // QuotaSubject describes the object for which the quota is applied to.
    QuotaSubject *QuotaSubject `json:"quotaSubject"`
    // ResourceQuotas is a map of maximum resource quotas per resource
    ResourceQuotas corev1.ResourceList `json:"resourceQuotas"`
}

type QuotaSubject struct {
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

### How to sync quota info from seed clusters and master cluster and back

We would need a controller for ResourceQuotas in the master cluster: `master-resource-quota-controller` - MRQC

The MRQC would watch master cluster ResourceQuotas. These are the ones that are created through the API, and maybe one day through `kubectl`.
The role of the MRQC would be to reconcile the master ResourceQuotas to the seed clusters.
It also needs to watch seed cluster ResourceQuotas for changes in LocalConsumption, and update the master ResourceQuota ResourceConsumption based on it.

### How to ensure that we get proper resource usage and that it stays in sync?

The obvious way seems to get the amount of resources requested when a KKP MachineDeployment is being created, and then check against
the project quota and fill it up accordingly. But this would only work through the API, and if there are some issues
we could have resource quotas which are out of sync.

The safest way would be to check the Node capacity on user clusters, but there are some issues with that as well, as the Nodes get created
based on the MachineDeployments. So there could be races in between when the Node is created(which can take some time) and 
when we check if user can create a MachineDeployment.

So we need to focus on the MachineDeployments and control the quota through them. The nodes get created anyway based 
on the MachineDeployment spec, and we can get the desired node size based on the MD, although for each cloud provider its a bit different.

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


### Enforcing Quotas - Implementation variants

I see 2 possible ways to implement the enforcing of project quotas through MachineDeployments. 

One is through a MachineDeployment webhook, although it requires moving the MD's to the seed cluster, because the webhook 
needs to be out of reach for the user-cluster users. The pros of that is that users can get directly notified if they 
can't create a MachineDeployment, but the con is that we would need to do a migration and change the shared code with KubeOne.

In the other approach, the machine-controller would be responsible for checking the quota before creating nodes for a MachineDeployment.
The pro is that it wouldn't be a big change, the con is that it won't be so visible to users, we could get questions "why are my nodes not being created".
We can show the info that there is not enough resource quota left to create the nodes, but users would only see that in MachineDeployment
Events. So it would be something like Pods not getting scheduled because there is no space.


#### 1. Webhook for MachineDeployments

This variant is dependent on migrating the MachineDeployments from being user-cluster resources to being seed-cluster resources.

If the MD's are on the seed cluster, we can control their creation/modification through a validation webhook which would check with the 
project's resource quota if the MD fits. If not, we would just deny the request.

The problem here is if its possible and at what cost to migrate the MD's, while thinking about KubeOne as well.
We could eventually dynamically set the kubeconfig of the cluster where the machine-controller watches for MachineDeployments, 
and set it to the seed-cluster in case of KKP. But then, as MachineDeployments are cluster-scoped we will need additional 
cluster info label or something on them.

Another issue could be that cluster users won't have direct kubectl access to MD's anymore, at least until we start 
supporting it somewhere in the future. But they can use the KKP API. 

#### 2. Machine-controller checks quota

In this variant, the machine-controller is checking the quota before it creates nodes for a MachineDeployment. 
If there is no quota left, it should not schedule the node, and emit some event and set status that there is not enough resources
left in the quota.

### How to ensure that we get proper resource usage and that it stays in sync?

The obvious way seems to get the amount of resources requested when a KKP MachineDeployment is being created, and then check against
the project quota and fill it up accordingly. But this would only work through the API, and if there are some issues
we could have resource quotas which are out of sync.

The safest way would be to check the Node capacity on user clusters, but there are some issues with that as well, as the Nodes get created
based on the MachineDeployments. So there could be races in between when the Node is created(which can take some time) and when we check if user can create a MachineDeployment.

So the best would be to create a controller which watches MachineDeployments, and fills out the project quotas accordingly.

The seed-machine-deployment-controller(SMDC) would watch the seed cluster MachineDeployments(assuming we go with the plan to move MDs to the seed cluster, otherwise we
need to have it watch MDs on user clusters, and do it a bit differently), calculate the resource usage, and update quotas for that clusters
project (if such a quota exists). Also it needs to watch for new ResourceQuotas, so that for new ones it calculates the usage.

### Default Project Quotas

The idea here is to allow admins to set a default project resource quota, that will be applied to all projects which don't have a resource quota set.

The default project resource quota can be set in the Global Settings, where we already have similar settings like `maxProjects`, and min/max node size.
The trick now is how to apply it. We could either create ResourceQuotas for every project that doesn't have it, or
not, but then we would have to calculate everytime in the quota webhook how many resources a project is currently consuming.

That's why creating ResourceQuotas for all projects that don't have them set seems like a simpler idea. To support editing/removing
the default resource quota, we could label all default resource quotas with `default=true`, so we know to distinguish them from
custom resource quotas.

### Possible Issues

1. How to deal with autoscalers?
2. Possible race when updating quotas

#### Race when updating quotas

There is still a chance of a race, in which users could create multiple MachineDeployments for a project in the same time, before the
controller updates the resource usage, thus bypassing the limit. In this case we have 2 options:
- admins could notice/get informed about it and react according to their company policy
- add a "pending"/"reserved" resource mechanism to the API. So as soon as a MachineDeployment request is received in the API, it would reserve some of the quota. Later the MachineDeployment controller could move this from reserved to real usage.

### Possible Enhancements in the future

1. Try to make the resourcequotas bulletproof by adding a mechanism of "pending"/"reserved" quota which is filled before the MachineDeployment creation
2. Possibility to set `maxClusters` for a project. 

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

