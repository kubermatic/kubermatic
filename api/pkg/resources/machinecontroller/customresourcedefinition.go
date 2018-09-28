package machinecontroller

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

const (
	clusterAPIGroup   = "cluster.k8s.io"
	clusterAPIVersion = "v1alpha1"
)

// MachineCRD returns the machine CRD definition
func MachineCRD(existing *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	if existing == nil {
		existing = &apiextensionsv1beta1.CustomResourceDefinition{}
	}

	existing.Name = resources.MachineCRDName
	existing.Spec.Group = clusterAPIGroup
	existing.Spec.Version = clusterAPIVersion
	existing.Spec.Scope = apiextensionsv1beta1.NamespaceScoped
	existing.Spec.Names.Kind = "Machine"
	existing.Spec.Names.Plural = "machines"

	return existing, nil
}

// MachineSetCRD returns the machineset CRD definition
func MachineSetCRD(existing *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	if existing == nil {
		existing = &apiextensionsv1beta1.CustomResourceDefinition{}
	}

	existing.Name = resources.MachineSetCRDName
	existing.Spec = apiextensionsv1beta1.CustomResourceDefinitionSpec{}
	existing.Spec.Group = clusterAPIGroup
	existing.Spec.Version = clusterAPIVersion
	existing.Spec.Scope = apiextensionsv1beta1.NamespaceScoped
	existing.Spec.Names.Kind = "MachineSet"
	existing.Spec.Names.Plural = "machinesets"
	existing.Spec.Subresources = &apiextensionsv1beta1.CustomResourceSubresources{Status: &apiextensionsv1beta1.CustomResourceSubresourceStatus{}}

	return existing, nil
}

// MachineDeploymentCRD returns the machinedeployments CRD definition
func MachineDeploymentCRD(existing *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	if existing == nil {
		existing = &apiextensionsv1beta1.CustomResourceDefinition{}
	}

	existing.Name = resources.MachineDeploymentCRDName
	existing.Spec = apiextensionsv1beta1.CustomResourceDefinitionSpec{}
	existing.Spec.Group = clusterAPIGroup
	existing.Spec.Version = clusterAPIVersion
	existing.Spec.Scope = apiextensionsv1beta1.NamespaceScoped
	existing.Spec.Names.Kind = "MachineDeployment"
	existing.Spec.Names.Plural = "machinedeployments"
	existing.Spec.Subresources = &apiextensionsv1beta1.CustomResourceSubresources{Status: &apiextensionsv1beta1.CustomResourceSubresourceStatus{}}

	return existing, nil
}

// ClusterCRD returns the cluster crd definition
func ClusterCRD(existing *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	if existing == nil {
		existing = &apiextensionsv1beta1.CustomResourceDefinition{}
	}

	existing.Name = resources.ClusterCRDName
	existing.Spec = apiextensionsv1beta1.CustomResourceDefinitionSpec{}
	existing.Spec.Group = clusterAPIGroup
	existing.Spec.Version = clusterAPIVersion
	existing.Spec.Scope = apiextensionsv1beta1.NamespaceScoped
	existing.Spec.Names.Kind = "Cluster"
	existing.Spec.Names.Plural = "clusters"
	existing.Spec.Subresources = &apiextensionsv1beta1.CustomResourceSubresources{Status: &apiextensionsv1beta1.CustomResourceSubresourceStatus{}}

	return existing, nil
}
