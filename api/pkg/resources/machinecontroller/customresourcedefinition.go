package machinecontroller

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

// MachineCRD returns the machine CRD definition
func MachineCRD(existing *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	if existing == nil {
		existing = &apiextensionsv1beta1.CustomResourceDefinition{}
	}

	existing.Name = resources.MachineCRDName
	existing.Spec = apiextensionsv1beta1.CustomResourceDefinitionSpec{}

	return existing, nil
}

// MachineSetCRD returns the machineset CRD definition
func MachineSetCRD(existing *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	if existing == nil {
		existing = &apiextensionsv1beta1.CustomResourceDefinition{}
	}

	existing.Name = resources.MachineSetCRDName
	existing.Spec = apiextensionsv1beta1.CustomResourceDefinitionSpec{}
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
	existing.Spec.Subresources = &apiextensionsv1beta1.CustomResourceSubresources{Status: &apiextensionsv1beta1.CustomResourceSubresourceStatus{}}

	return existing, nil
}

// ClusterCRD returns the cluster crd definition
func ClusterCRD(existing *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	if existing == nil {
		existing = &apiextensionsv1beta1.CustomResourceDefinition{}
	}

	existing.Name = resources.MachineSetCRDName
	existing.Spec = apiextensionsv1beta1.CustomResourceDefinitionSpec{}
	existing.Spec.Subresources = &apiextensionsv1beta1.CustomResourceSubresources{Status: &apiextensionsv1beta1.CustomResourceSubresourceStatus{}}

	return existing, nil
}
