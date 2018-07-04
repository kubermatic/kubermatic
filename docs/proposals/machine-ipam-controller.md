**Machine IPAM-controller**

**Author**: Henrik Schmidt

**Status**: Draft

*short description of the topic e.g.*
The machine IPAM controller will take care of allocating IP's from a given CIDR and assigning them to Machine objects.

## Motivation and Background

Currently the machine-controller required DHCP to exist in every network it deploys instances in.
As some environments do not have DHCP available or it is even completely forbidden, we need something which handles the IP address allocation.
A requirement will be that a CIDR will be provided together with settings like Gateway, DNS-server.

## Implementation

During creation of a cluster the user will provide settings like:
- CIDR's (`192.168.1.128/25`)
- Gateway
- DNS servers

During runtime the user will be able to add additional CIDR's

### machine-controller

The Machine object will be extended to include a dedicated `Network` object inside the `ProviderConfig`:
```yaml
apiVersion: "machine.k8s.io/v1alpha1"
kind: Machine
metadata:
  name: machine1
spec:
  metadata:
    name: node1
  providerConfig:
    network:
      ip: "192.168.1.129"
      netmask: "255.255.255.128"
      gateway: "192.168.1.1"
      # Explicitly making it a dedicated object to enable extending the dns settings
      dns:
        servers:
        - "192.168.1.1"
``` 

The machine-controller will use this information when generating the cloud-init/ignition data to configure the network of the instance.
`machine.Spec.ProviderConfig.Network` will be a pointer to express the network configuration is optional. Following the kubernetes documentation on `Optional vs. Required` https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#optional-vs-required
 
### cluster-controller

The cluster object will be extended to include a property to set the required CIDR for the machines.

```yaml
apiVersion: kubermatic.k8s.io/v1
kind: Cluster
metadata:
  name: 8scc6wc6wb
spec:
  # explicitly not part of clusterNetwork as that is a upstream type we should not modify. 
  # Otherwise migration to the cluster-api will get really tricky
  machineNetwork:
    cidrBlocks:
    - 192.168.1.128/25
```

The user will be able to specify the CIDR's during cluster creation.
It should be possible to add additional CIDR's later. Removing or modifying must be forbidden.

`cluster.Spec.machineNetwork` will be a pointer to express the network configuration is optional. Following the kubernetes documentation on `Optional vs. Required` https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#optional-vs-required

### machine-ipam-controller

A new controller will be deployed inside the cluster-namespace for each cluster when `cluster.spec.`.
It is responsible to allocate IP addresses from the cidr's and configure the `Network` settings in the Machine objects.
As the network settings must be set before the machine-controller provisions the instance, the `machine-ipam-controller` will use a Initializer on the machine objects.
After the network settings have been configured on the machine object, the `machine-ipam-controller` will remove the Initializer.

To prevent concurrency issues, the `machine-ipam-controller` will use leader-election & have exactly one worker routine.

The different assigned CIDR's & the network configuration will be passed to the `machine-ipam-controller` via flags.
`machine-ipam-controller` will exclusively talk to the user-cluster.

## Workflow

1. machine gets created
1. kubernetes adds Initializer (prevents processing by machine-controller)
1. `machine-ipam-controller` allocates an IP & configures the network settings on the machine
1. `machine-ipam-controller` removes the Initializer
1. `machine-controller` creates & provisions the machine


## Tasks:
*   Implement `machine-ipam-controller`
  * Tests:
    * allocates IP from CIDR's
    * allocates IP from second CIDR's when first is full
    * fails when no free ip is left
*   Add support for network configuration in `machine-controller`
  * Configuration implemented for every OS
    * CoreOS
    * CentOS
    * Ubuntu
* Add handling in `cluster-controller`
  * Deployment of `machine-ipam-controller`
  * Tests for manifest generation (Different & Multiple CIDR's)
* Implement UI handling for optional network configuration (vSphere only for now)
