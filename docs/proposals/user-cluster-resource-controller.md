# Title of the Poposal: **user-cluster-resource-controller**

**Author**: Tobias Hintze (thz)

**Status**: Drafting

## Status Quo

This is about resources inside the user clusters. Currently resources are created the following places:

### Seed Components

The following master components are running in a cluster-xyz namespace in the seed cluster:

* apiserver
* controller-manager
* etcd
* machine-controller
* openvpn-server
* scheduler
* prometheus

### User Components (within the user-cluster)

The following user components are running in the user cluster without the user actually installing them. Some of them are strictly required, some are not.

* as addons by AddonManager:
	* canal
	* openvpn-client
	* heapster/metrics-server
	* rbac
	* default-storage-class
	* dashboard
* by cluster.Controller:
	* launchingCreateClusterInfoConfigMap et al: cluster-info ConfigMap, some rbac, OpenVPN certs
	* userClusterEnsureXyz: roles and bindings, configmap for vpn
* apiserver
	* machine crd

## Motivation and Background

Most user resources are currently created as addons by the AddonManager. The list of addons which are installed by default can be changed on kubermatic level (values.yaml). Also addons can be changed or more addons can be made available by specifying a different addon image on kubermatic level. For some resources this makes no sense because they are no addons. These resources are no addons (perhaps because they are strictly required) and in the lack of a better option the AddonManager is only abused to install them.
Then there are some cases of user-resource installation in the cluster-controller, which is just another abuse.

## Implementation

The envisioned improvement is to create a UserClusterResourceController similar to the ClusterController in the seed cluster. The UserClusterResourceController will run in cluster-namespace too.

The UserClusterResourceController will do reconciliation like the ClusterController does. This allows for more flexible resource management.

The UserClusterResourceController will be able to control resources in the user-cluster (example: metrics-server) by talking to the apiserver and it will be controlling resources in the cluster-namespace as well (example: machine CRDs).

## Task & effort:

* implement controller skeleton in `api/pkg/controller/userresources`
	* add to kubermatic-controller-manager
* move user-cluster code in cluster-controller to controller/userresources
	* userClusterEnsure...
	* some launchingCreate...
* move addons which should be no addons into controlled resources:
	* kube-proxy
	* kubelet-configmap
	* openvpn
	* rbac
