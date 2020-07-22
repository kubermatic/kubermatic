# Adding an external cluster to Kubermatic

**Author**: Lukasz Zajaczkowski (@zreigz)

**Status**: Draft proposal; prototype in progress.

## Goals
The end user would like to add existing Kubernetes cluster to Kubermatic platform. Therefore, Kubermatic does not provision
Kubernetes, but only sets up the connection to communicate with the cluster and install controllers to enable some Kubermatic features.
Kubermatic features, including displaying cluster details, metrics and role-based access control, will be available for external clusters.
The cluster details will be displayed in the Dashboard in Kubermatic way. The Kubermatic doesn't manipulate or change control plane.
The configuration of an external cluster still has to be edited outside of Kubermatic.

## Prerequisites

To add cluster user has to deliver an admin kubeconfig. The API server must be accessible for the Kubermatic.

## Implementation

The external cluster is represented by `ExternalCluster` CRD with the reference to the `Secret` with kubeconfig.

```
// ExternalCluster is the object representing a external kubernetes cluster.
type ExternalCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ExternalClusterSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExternalClusterList specifies a list of external kubernetes clusters
type ExternalClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Cluster `json:"items"`
}

// ExternalClusterSpec specifies the data for a new external kubernetes cluster.
type ExternalClusterSpec struct {

	// HumanReadableName is the cluster name provided by the user
	HumanReadableName string `json:"humanReadableName"`

	KubeconfigReference *providerconfig.GlobalSecretKeySelector `json:"kubeconfigReference,omitempty"`
}

```

The Kubermatic implements a provider with all necessary methods to access and control the cluster. It uses delivered kubeconfig
for this purpose.
The Kubermatic exposes endpoints to get, list, and delete the cluster from the Kubermatic platform.

The Machine Deployment view is replaced by cluster Nodes view.