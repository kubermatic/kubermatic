# Title of the Proposal: e.g. **kube-node**

**Author**: Sebastian Scheele (@scheeles)

**Status**: Draft proposal; prototype in progress.

*short description of the topic e.g.*
kube-node is a minimalistic API which enables Kubernetes to manage its nodes by itself. The goal is to have a higher abstraction layer for managed k8s nodes and to be able to integrate Kubernetes with different providers in a generic way. It is intended to live outside of core Kubernetes and add optional node management features to Kubernetes clusters.


## Motivation and Background

*What is the background and why do we want to deplyo it e.g.*


The current approaches to deploy and manage k8s nodes come with several downsides:

*   Require Ops to scale the cluster
*   No generic approach for different providers
*   Each implementation requires specific knowledge
*   Scaling of new nodes requires external dependencies
*   No generic node auto-scaling across different providers available
*   Huge effort to deploy k8s on a different platform

The idea is to make the deployment of kubernetes much simpler and create a native integration for kubernetes node deployment.

With kube-node, to create a native integration for kubernetes nodes and achieve that:

*   Dev can scale the cluster
*   Nodes become cattles, not pets
*   Implementation of generic auto scaling possible
*   Very similar setup for different providers

## Implementation

*How to implment the idea e.g.*

This proposal enhances the node API (available as Node kind in core/v1 in Kubernetes) and introduces two new API types: NodeClass and NodeSet. See the full definition in[ types.go](https://github.com/kube-node/nodeset/blob/master/pkg/nodeset/v1alpha1/types.go).

Similar to the PersistentVolumes API, the goal is:

 *   A higher-level node abstraction which is isolated from/independent of   the  cloud environment
*   Empower admins to create the configuration of the nodes and users to provision them
*   Empower Kubernetes to control the lifecycle of nodes
*   Dynamically "scheduled" and managed nodes

A **Node** is a worker machine in Kubernetes. A node may be a VM or physical machine, depending on the cluster. Each node has the services necessary to run pods. The services on a node include container runtime, kubelet and kube-proxy.

In today's Kubernetes the approach is to create the node objects after a machine joins a cluster. This is done by the kubelet. With kube-node, the node objects are created before a machine is added.

**NodeController** watches for node objects. When a new node object is created, the NodeController provisions the machine via cloud APIs. After the machine joined the cluster, the kubelet updates the node object. When a node object is deleted, the NodeController deletes the machine via cloud APIs.The intention is that similar to storage provisioner, different implementations of a NodeController can exist. A first reference implementation is[ kube-machine](https://github.com/kube-node/kube-machine) which reuses parts of the library of docker-machine and[ archon-nodeset](https://github.com/kube-node/archon-nodeset). There could be also implementation for terraform or python or even cloud provider-specific implementation.

A **NodeClass** provides a way for administrators to describe the "classes" of nodes they offer. Different classes might map to types of machines, quality-of-service levels or to arbitrary policies determined by the cluster administrators. Kubernetes itself is unopinionated about what classes represent. This concept is sometimes called "Instance templates" by different cloud provider.

A NodeClass contains cloud provider & OS specific details:

*   Cloud provider credentials
*   Machine type (e.g. t2.medium)
*   Provisioning
    *   files e.g. systemd unit, ssh keys
    *   kubelet version
    *   container runtime/version
    *   ssh command

A **NodeSet** ensures that a specified number of nodes is running at any one time. In other words, a NodeSet makes sure that a node or a homogeneous set of nodes is always up and available. Each NodeSet refers  to a NodeClass, were the details are described in nodes templates. The NodeSet should be integrated with the Kubernetes autoscaler. With this integration, Kubernetes gets the possibility to generically auto-scale its nodes on different cloud provider. Based on the specified replicas in the NodeSet, a NodeSetController will create the specified node objects. The NodeSetController will ensure that the number of nodes specified in the replicas will meet the existing node objects. If the specified number does not match, the NodeSetController creates or destroys node objects.

Additionally, when the reference NodeClass gets changed, e.g. the kubelet version is updated, the NodeSetController executes a rolling update of the nodes. The update takes place with zero downtime by incrementally updating nodes with new ones.


## Task & effort:
*Specify the taks and the effort in days (samples unit 0.5days) e.g.*
* Implement NodeClass - 1d
* Enhance kubermatic controller - 2d
