# Title of the Poposal: **user-cluster-controller**

**Author**: Tobias Hintze (thz)

**Status**: Drafting

## Status Quo

This is about resources in the user-clusters which are not added by the user. Currently these resources are created in different places:

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

The scope of this proposal is on user-cluster resources and only. Most user-cluster resources are currently created as addons by the AddonManager some few are created directly from the cluster-controller (seed).
The list of addons which are installed by default can be changed on kubermatic level (values.yaml). Also addons can be changed or more addons can be made available by specifying a different addon image on kubermatic level.
Some of those resources are strictly required (as opposed to "addons") and should have a decoupled lifecycle. So it is proposed to have a dedicated controller for the user-cluster..

## Implementation

The envisioned improvement is started by establishing a user-cluster-controller-manager created by the (seed's) cluster-controller. The manager will create a user-cluster-controller inside the cluster-namespace.
Resources of the user-cluster which are not considered "addons" or should not be controlled by the (seed's) cluster-controller will be moved into the control of the user-cluster-controller.

## Task & effort:

* create user-cluster-controller-manager
	* deploy it to user-cluster by cluster-controller
* implement user-cluster-controller skeleton in `api/pkg/controller/usercluster`
	* check/employ kubebuilder
	* add to user-cluster-controller-manager
* move user-cluster code in cluster-controller to controller/userresources
	* userClusterEnsure...
	* some launchingCreate...
* move addons which should be no addons into controlled resources:
	* kube-proxy
	* kubelet-configmap
	* openvpn
	* rbac
