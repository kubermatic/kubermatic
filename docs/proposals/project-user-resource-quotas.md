# Project resource quotas

**Authors**: Lovro Sviben (@lsviben)

**Status**: Draft proposal.

**Issue**: https://github.com/kubermatic/kubermatic/issues/8042

## Goals
The idea is to allow KKP-Admins to control how many resources are projects using by introducing resource quotas.

Resources that will be under quota:
- vCPU
- RAM
- storage (node disk size) - TBD, we can remove this as it's not a strict requirement

## Non-Goals
- The quotas won't work for imported external clusters.
- The quotas won't restrict the usage of resources for the control plane on the seed cluster.
- Nodes which join the cluster using other means than through KKP are not supported in the quotas.

## Motivation and Background

KKP-Admins would like to have options to fine-grain restrict resource usage per project.
The reasons vary from being able to stop the depletion of resources in a shared datacenter, to being able to sell packages (projects) with a set 
number of resources available. 

This will be an EE feature.

## How will the resource quotas work

Project quota - amount of resources (vCPU, RAM, storage) that can allocated for nodes in a project.

Rules:

- The quotas are for node resources, which KKP controls through the Machine CRD
- The quotas are enforced(we should block creation of new Machines that would exceed the quota)
- The resource quotas management will be admin-only.
- By default, projects will not have a quota, meaning they can use as many resources they want
- There will be a way to set a default resource quota for all projects
- When a machine is deleted, the quota usage needs to decrease for the project
- If the quota is exceeded(by lowering the quota under current capacity for example), it is not the responsibility of KKP to remove machines. Admins should fix that in communication with the project/user.
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

The safest way would be to check the Node capacity on user clusters, but as the Nodes get created
based on the MachineDeployments, which in turn create Machines that correspond to/create Nodes. We can calculate
the usage based on the Machines, as the Nodes take some time to get created. This way we could avoid races.

So we need to focus on the Machines and control the quota through them. The nodes get created anyway based 
on the Machine spec, and we can get the desired node size based its spec, although for each cloud provider it's a bit different.

An idea would be to extend the Cluster status with ResourceConsumption which would be calculated based on watching its
Machines.(an addition to `user-cluster-controller-manager`). Then a seed-wide controller can calculate the usage of all clusters
in a project and update the related ResourceQuota.

### How to get node size

The Machines for different providers only have node flavours in their spec. So the question is how to get the real node size.

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


### Enforcing Quotas - Machine Webhook

To enforce the resource quotas, we can use a validating webhook for Machines on the user cluster. The admission server
itself would run on the control plane so that cluster users won't have access to it. 

The webhook would be:
- calculate requested Machine resource usage (needs to use user-cluster credentials to talk with providers)
- get the latest project ResourceQuota
- if it fits, allow
- if it doesn't fit:
  - don't allow, with message that it's too big
  - create an event that could be visible for users 
  
The problem could be that the MachineSet controller will try to create the Machine again and again, which won't work unless either
the ResourceQuota or requested Machine changes. Maybe we should have a way to mark/flag this so that we control it. - TBD

### Default Project Quotas

The idea here is to allow admins to set a default project resource quota, that will be applied to all projects which don't have a resource quota set.

The default project resource quota can be set in the Global Settings, where we already have similar settings like `maxProjects`, and min/max node size.
The trick now is how to apply it. We could either create ResourceQuotas for every project that doesn't have it, or
not, but then we would have to calculate every time in the quota webhook how many resources a project is currently consuming.

That's why creating ResourceQuotas for all projects that don't have them set seems like a simpler idea. To support editing/removing
the default resource quota, we could label all default resource quotas with `default=true`, so we know to distinguish them from
custom resource quotas.

### Possible Issues

1. How to deal with autoscalers?
2. Possible race when updating quotas

#### Race when updating quotas

There is still a chance of a race, in which users could create multiple MachineDeployments for a project in the same time,
and the Machines get created(could be on different Seeds) before the controller updates the resource usage, thus bypassing the limit. 
For now no solution to this, but the KKP admins will be able to see the Project resource usage(in the UI for example), get informed about it,
and react according to their company policy.

### Possible Enhancements in the future

1. Possibility to set `maxClusters` for a project. 

## Tasks and effort

1. Investigate project and user quotas
2. Implement ResourceQuotas CRD
4. Implement MRQC
5. Implement controllers for calculating cluster quota usage
6. Validating webhook for Machines based on resource quotas
7. ResourceQuota API endpoints
8. (Optional) Implement default project quotas
