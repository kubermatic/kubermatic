# Multi-Cluster IP Address Management (IPAM) for User Cluster Applications / Addons

**Authors**: Rastislav Szabo (@rastislavs)

**Status**: Draft proposal.

## Goals
Provide an automated per-user-cluster IP address range allocation from a predefined larger CIDR range for applications
running in user clusters. Expose the allocated range in the user cluster (e.g. in a ConfigMap), and in KKP Addon
`TemplateData`, so that various applications / Addons running in the user cluster can consume the allocated ranges easily.

Do not tie the implementation to a specific application / Addon or 3rd party IP address management tools,
to provide vendor neutrality.

## Motivation and Background
**User Story**: https://github.com/kubermatic/kubermatic/issues/6421

Networking applications deployed in KKP user clusters need automated IP Address Management (IPAM) for IP ranges that
they use, in a way that prevents address overlaps between multiple user clusters. An example for such an application
is MetalLB load-balancer, for which a unique IP range from a larger CIDR range needs to be configured in each user
cluster in the same datacenter. This is currently mostly done manually, by keeping evidence of allocated subnets in
an external document and copy-pasting from it when deploying the MetalLB addon.

The goal is to provide a simple solution that is automated and less prone to human errors.

## Implementation
KKP will expose a global-scoped Custom Resource Definition `IPAMPool` in the seed cluster. The administrators will be
able to define the `IPAMPool` CR with a specific name with multiple pool CIDRs with predefined allocation ranges tied
to specific data centers, e.g.:

```yaml
apiVersion: kubermatic.k8s.io/v1
kind: IPAMPool
metadata:
  name: metallb
spec:
  - datacenter: vsphere-hamburg
    poolCIDR: 192.168.1.0/16
    allocationRange: 10
  - datacenter: vsphere-berlin
    poolCIDR: 192.168.1.0/16
    allocationRange: 10
```

If there are multiple pools available for a datacenter, an allocation can be made from any of them.
Note that `poolCIDR` can be the same in different datacenters.

The IPAM controller in the seed-controller-manager will be in charge of allocation of IP ranges from the defined pools
for user clusters. It should run with 1 worker to make sure that there are never two entities trying to make an allocation
concurrently. See [Possible Further Enhancements](#possible-further-enhancements) for more options of providing better guaranties.
For each user cluster which runs in a datacenter for which an `IPAMPool` is defined, it will
automatically allocate a free IP range from the available `poolCIDR` long enough to accommodate the `allocationRage`.
To persist the allocation, it will create a an `IPAMAllocation` CR in the user cluster’s namespace,
e.g. for the `metallb` `IPAMPool`:

```yaml
apiVersion: kubermatic.k8s.io/v1
kind: IPAMAllocation
metadata:
  name: metallb
  namespace: cluster-xyz
spec:
  addresses:
  - 192.168.1.240-192.168.1.250
```

Note that the ranges of addresses may be disjoint, e.g.:
```yaml
spec:
  addresses:
  - 192.168.1.200-192.168.1.210
  - 192.168.1.240-192.168.1.250
```

See [Possible Further Enhancements](#possible-further-enhancements) for reasons why disjoint ranges may be generated.

Whenever a cluster is deleted, the allocation will be implicitly released by deleting the `IPAMAllocation` CR.
Note that `poolCIDR` and `allocationRange` updates may not be supported in the first version (and potentially denied
by a validation webhook). [Possible Further Enhancements](#possible-further-enhancements) describe handling oh these updates.

## KKP Addon Infra Integration
To expose the allocated subnet for addon templating in the KKP’s Addon `TemplateData`, the addon-controller needs to
know about a dependency on a particular `IPAMAllocation`(s), so that an addon cannot be installed before the allocation
is made for the cluster. For this, the `AddonSpec` struct of the `Addon` CRD will be extended with `RequiredIPAMAllocations` slice:

```go
type AddonSpec struct {
    ...
    RequiredIPAMAllocations []string `json:"requiredPAMAllocations,omitempty"`
    ...
}
```

If there will be any `RequiredIPAMAllocations` set for an Addon, the addon-controller will make sure they are available
before proceeding with the addon installation.

The allocated subnets will be made available in Addon templates in `.ClusterData.IPAMAllocations` map indexed by `IPAMPool` name.

## AddonConfig / Addon UI Integration
In order to have the `RequiredIPAMAllocations` in the `AddonSpec` automatically set for accessible Addons installed
from the KKP UI, we will allow to define them in the `AddonConfig` CR for accessible addons, e.g.:

```yaml
apiVersion: kubermatic.k8s.io/v1
kind: AddonConfig
metadata:
  name: metallb
spec:
  requiredIPAMAllocations:
  - metallb
...
```

The KKP UI / api-server will automatically propagate the `requiredIPAMAllocations` from an `AddonConfig` CR to the
respective `Addon` CR. Optionally, the KKP UI may display the list of `requiredIPAMAllocations` in the Addon UI, or
even allow the users to specify/modify the list from the UI.


## Possible Further Enhancements
This section covers some future enhancements that do not necessarily have to be implemented in the first working version.

### ConfigMap in the User Clusters
The IPAM controller could also create/update a ConfigMap in the user cluster (e.g. `kubermatic-ipam-allocation`),
that could be used to easily consume the allocated pool in any application running in the user cluster, e.g.:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kubermatic-ipam-allocation
  namespace: kube-system
data:
  metallb: 192.168.1.200-192.168.1.210
```

In case of disjoint address ranges, these will be separated by commas (`,`), e.g.:
```yaml
data:
  metallb: 192.168.1.200-192.168.1.210,192.168.1.240-192.168.1.250
```

Whenever an allocation is modified the ConfigMap will be updated as well.

### Handling Modifications to IPAMPool
If the infrastructure administrator decides to modify the `IPAMPool` from which some allocations were already made,
it needs to be handled with special care - already allocated addresses should be kept whenever possible. If the
`allocationRange` becomes higher, we should just allocate a new block for extra addresses that we need and create
a disjoint range rather than trying to allocate a new continuous range.

Also, we should probably deny removal of a `poolCIDR` using a validation webhook, if it is still used by some clusters.
To allow migrations to a new `poolCIDR`, we may introduce a `deprecated` flag, which will cause that a `poolCIDR`
won’t be used for any new clusters.

### Consistency Checking
Having an IPAM controller with 1 worker in conjunction with leader election should make sure that there are no parallel
allocations, but leader election [does not seem to provide strong guarantees](https://github.com/kubernetes/client-go/blob/93ce9718ffcde5a4465be99b1e0238a3a042ad22/tools/leaderelection/leaderelection.go#L17).
To avoid any potential race conditions, we could introduce a validation webhook for `IPAMAllocation`
resources, that would perform a consistency check.


## Alternatives Considered
Another considered approach was to do IP address allocation for each LoadBalancer service ourselves in a controller
that would watch all LoadBalancer Services, and set the allocated IP to `svc.spec.loadBalancerIP` of the service.
That should prevent MetalLB from doing their allocation, the MetalLB speaker should just pick up the specified address
(which still has to be from a predefined range). This approach would be a bit more complex, as it would probably require
a controller per each user cluster, and some centralized IPAM entity that would act as a single source of truth from
which the controllers would request IPs. Also, this approach could result in some timing issues (unnecessary changes
of LoadBalancer IP).

We also considered leveraging DHCP for allocating IP addresses / IP ranges for KKP user clusters. Although it would help
with allocation, releasing and tracking of allocated IPs, it would bring several disadvantages. If it was our controller
who requests addresses from DHCP:

 - we would probably need to use DHCP relay, as DHCP operates on L2 network layer,
 - we would need to issue multiple DHCP requests to obtain a larger range of IP addresses,
 - we would need to deal with renewing DHCP leases periodically,
 - there might be outages in case that DHCP server is not available,
 - it wouldn’t be working out of box in any setup.

Given the amount of problems, having IPAM fully under our control seems like a better solution.

We also considered integration with some external IPAM management tools commonly used in complex networking deployments.
But since there is no standard IPAM protocol supported by more of them, that would mean a dependency on a specific
protocol of a given software/vendor.

There is a possibility that MetalLB will provide integration with some external IPAM systems, or even with DHCP in future
(see the related issue). In that case, even DHCP would make sense, since the requests would be made per individual
LoadBalancer service, not for the larger blocks of addresses. That is however not yet reality, and it would solve the
problem for MetalLB only.


## Tasks & Efforts
For the simple version (NOT including the [Possible Further Enhancements](#possible-further-enhancements)):
 - Implement the IPAM Controller which consumes `IPAMPool` CRs and creates `IPAMAllocation` CRs: _5 days_
 - Implement changes in the Addon controller - add support for `RequiredIPAMAllocations` in `AddonSpec`,
   expose tha allocation data in `ClusterData` for templating: _5 days_
 - Add `requiredIPAMAllocations` to `AddonConfig` and project it to `AddonSpec`: _2 days_
