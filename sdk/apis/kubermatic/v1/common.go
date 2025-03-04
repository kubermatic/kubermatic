/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

// +kubebuilder:validation:Enum=NodePort;LoadBalancer;Tunneling

// ExposeStrategy is the strategy used to expose a cluster control plane.
// Possible values are `NodePort`, `LoadBalancer` or `Tunneling` (requires a feature gate).
type ExposeStrategy string

const (
	// ExposeStrategyNodePort creates a NodePort with a "nodeport-proxy.k8s.io/expose": "true" annotation to expose
	// all clusters on one central Service of type LoadBalancer via the NodePort proxy.
	ExposeStrategyNodePort ExposeStrategy = "NodePort"
	// ExposeStrategyLoadBalancer creates a LoadBalancer service per cluster.
	ExposeStrategyLoadBalancer ExposeStrategy = "LoadBalancer"
	// ExposeStrategyTunneling allows to reach the control plane components by
	// tunneling L4 traffic (TCP only is supported at the moment).
	// The traffic is encapsulated by mean of an agent deployed on the worker
	// nodes.
	// The traffic is decapsulated and forwarded to the recipients by
	// mean of a proxy deployed on the Seed Cluster.
	// The same proxy is also capable of routing TLS traffic without
	// termination, this is to allow the Kubelet to reach the control plane
	// before the agents are running.
	//
	// This strategy has the inconvenience of requiring an agent on worker
	// nodes, but has the notable advantage of requiring one single entry point
	// (e.g. Service of type LoadBalancer) without consuming one or more ports
	// for each user cluster.
	ExposeStrategyTunneling ExposeStrategy = "Tunneling"
)

// Finalizers should be kept to their controllers. Only if a finalizer is
// used by multiple controllers should it be placed here.

const (
	// NodeDeletionFinalizer indicates that the nodes still need cleanup.
	NodeDeletionFinalizer = "kubermatic.k8c.io/delete-nodes"
	// NamespaceCleanupFinalizer indicates that the cluster namespace still exists and the owning Cluster object
	// must not yet be deleted.
	NamespaceCleanupFinalizer = "kubermatic.k8c.io/cleanup-namespace"
	// InClusterPVCleanupFinalizer indicates that the PVs still need cleanup.
	InClusterPVCleanupFinalizer = "kubermatic.k8c.io/cleanup-in-cluster-pv"
	// InClusterLBCleanupFinalizer indicates that the LBs still need cleanup.
	InClusterLBCleanupFinalizer = "kubermatic.k8c.io/cleanup-in-cluster-lb"
	// CredentialsSecretsCleanupFinalizer indicates that secrets for credentials still need cleanup.
	CredentialsSecretsCleanupFinalizer = "kubermatic.k8c.io/cleanup-credentials-secrets"
	// ExternalClusterKubeOneNamespaceCleanupFinalizer indicates that kubeone cluster namespace still need cleanup.
	ExternalClusterKubeOneNamespaceCleanupFinalizer = "kubermatic.k8c.io/cleanup-kubeone-namespace"
	// ExternalClusterKubeconfigCleanupFinalizer indicates that secrets for kubeconfig still need cleanup.
	ExternalClusterKubeconfigCleanupFinalizer = "kubermatic.k8c.io/cleanup-kubeconfig-secret"
	// ExternalClusterKubeOneCleanupFinalizer indicates that secrets for kubeone cluster still need cleanup.
	ExternalClusterKubeOneSecretsCleanupFinalizer = "kubermatic.k8c.io/cleanup-kubeone-secret"
	// EtcdBackConfigCleanupFinalizer indicates that EtcdBackupConfigs for the cluster still need cleanup.
	EtcdBackupConfigCleanupFinalizer = "kubermatic.k8c.io/cleanup-etcdbackupconfigs"
	// GatekeeperConstraintCleanupFinalizer indicates that gatkeeper constraints on the user cluster need cleanup.
	GatekeeperConstraintCleanupFinalizer = "kubermatic.k8c.io/cleanup-gatekeeper-constraints"
	// KubermaticConstraintCleanupFinalizer indicates that Kubermatic constraints for the cluster need cleanup.
	KubermaticConstraintCleanupFinalizer = "kubermatic.k8c.io/cleanup-kubermatic-constraints"
)

const (
	InitialMachineDeploymentRequestAnnotation        = "kubermatic.io/initial-machinedeployment-request"
	InitialApplicationInstallationsRequestAnnotation = "kubermatic.io/initial-application-installations-request"
	InitialCNIValuesRequestAnnotation                = "kubermatic.io/initial-cni-values-request"
)

type MachineFlavorFilter struct {
	// +kubebuilder:default=0
	// +kubebuilder:validation:Minimum:=0

	// Minimum number of vCPU
	MinCPU int `json:"minCPU"`

	// +kubebuilder:default=0
	// +kubebuilder:validation:Minimum:=0

	// Maximum number of vCPU
	MaxCPU int `json:"maxCPU"`

	// +kubebuilder:default=0
	// +kubebuilder:validation:Minimum:=0

	// Minimum RAM size in GB
	MinRAM int `json:"minRAM"`

	// +kubebuilder:default=0
	// +kubebuilder:validation:Minimum:=0

	// Maximum RAM size in GB
	MaxRAM int `json:"maxRAM"`

	// Include VMs with GPU
	EnableGPU bool `json:"enableGPU"` //nolint:tagliatelle
}
