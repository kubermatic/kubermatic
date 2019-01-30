package machinecontroller

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/semver"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

const (
	clusterAPIGroup   = "cluster.k8s.io"
	clusterAPIVersion = "v1alpha1"
)

// MachineCRD returns the machine CRD definition
func MachineCRD(_ semver.Semver, existing *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	if existing == nil {
		existing = &apiextensionsv1beta1.CustomResourceDefinition{}
	}

	existing.Name = resources.MachineCRDName
	existing.Spec.Group = clusterAPIGroup
	existing.Spec.Version = clusterAPIVersion
	existing.Spec.Scope = apiextensionsv1beta1.NamespaceScoped
	existing.Spec.Names.Kind = "Machine"
	existing.Spec.Names.ListKind = "MachineList"
	existing.Spec.Names.Plural = "machines"
	existing.Spec.Names.Singular = "machine"
	existing.Spec.Names.ShortNames = []string{"ma"}

	return existing, nil
}

// MachineSetCRD returns the machineset CRD definition
func MachineSetCRD(clusterVersion semver.Semver, existing *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	if existing == nil {
		existing = &apiextensionsv1beta1.CustomResourceDefinition{}
	}

	existing.Name = resources.MachineSetCRDName
	existing.Spec.Group = clusterAPIGroup
	existing.Spec.Version = clusterAPIVersion
	existing.Spec.Scope = apiextensionsv1beta1.NamespaceScoped
	existing.Spec.Names.Kind = "MachineSet"
	existing.Spec.Names.ListKind = "MachineSetList"
	existing.Spec.Names.Plural = "machinesets"
	existing.Spec.Names.Singular = "machineset"
	existing.Spec.Names.ShortNames = []string{"ms"}

	if clusterVersion.Semver().Minor() > 9 {
		existing.Spec.Subresources = &apiextensionsv1beta1.CustomResourceSubresources{Status: &apiextensionsv1beta1.CustomResourceSubresourceStatus{}}
	}

	return existing, nil
}

// MachineDeploymentCRD returns the machinedeployments CRD definition
func MachineDeploymentCRD(clusterVersion semver.Semver, existing *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	if existing == nil {
		existing = &apiextensionsv1beta1.CustomResourceDefinition{}
	}

	existing.Name = resources.MachineDeploymentCRDName
	existing.Spec.Group = clusterAPIGroup
	existing.Spec.Version = clusterAPIVersion
	existing.Spec.Scope = apiextensionsv1beta1.NamespaceScoped
	existing.Spec.Names.Kind = "MachineDeployment"
	existing.Spec.Names.ListKind = "MachineDeploymentList"
	existing.Spec.Names.Plural = "machinedeployments"
	existing.Spec.Names.Singular = "machinedeployment"
	existing.Spec.Names.ShortNames = []string{"md"}

	if clusterVersion.Semver().Minor() > 9 {
		existing.Spec.Subresources = &apiextensionsv1beta1.CustomResourceSubresources{Status: &apiextensionsv1beta1.CustomResourceSubresourceStatus{}}
	}

	return existing, nil
}

// ClusterCRD returns the cluster crd definition
func ClusterCRD(clusterVersion semver.Semver, existing *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	if existing == nil {
		existing = &apiextensionsv1beta1.CustomResourceDefinition{}
	}

	existing.Name = resources.ClusterCRDName
	existing.Spec.Group = clusterAPIGroup
	existing.Spec.Version = clusterAPIVersion
	existing.Spec.Scope = apiextensionsv1beta1.NamespaceScoped
	existing.Spec.Names.Kind = "Cluster"
	existing.Spec.Names.ListKind = "ClusterList"
	existing.Spec.Names.Plural = "clusters"
	existing.Spec.Names.Singular = "cluster"
	existing.Spec.Names.ShortNames = []string{"cl"}

	if clusterVersion.Semver().Minor() > 9 {
		existing.Spec.Subresources = &apiextensionsv1beta1.CustomResourceSubresources{Status: &apiextensionsv1beta1.CustomResourceSubresourceStatus{}}
	}

	return existing, nil
}
