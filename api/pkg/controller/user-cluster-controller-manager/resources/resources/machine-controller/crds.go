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

package machinecontroller

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

const (
	clusterAPIGroup   = "cluster.k8s.io"
	clusterAPIVersion = "v1alpha1"
)

// MachineCRD returns the machine CRD definition
func MachineCRDCreator() reconciling.NamedCustomResourceDefinitionCreatorGetter {
	return func() (string, reconciling.CustomResourceDefinitionCreator) {
		return resources.MachineCRDName, func(crd *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
			crd.Spec.Group = clusterAPIGroup
			crd.Spec.Version = clusterAPIVersion
			crd.Spec.Scope = apiextensionsv1beta1.NamespaceScoped
			crd.Spec.Names.Kind = "Machine"
			crd.Spec.Names.ListKind = "MachineList"
			crd.Spec.Names.Plural = "machines"
			crd.Spec.Names.Singular = "machine"
			crd.Spec.Names.ShortNames = []string{"ma"}
			crd.Spec.AdditionalPrinterColumns = []apiextensionsv1beta1.CustomResourceColumnDefinition{
				{
					Name:     "Age",
					Type:     "date",
					JSONPath: ".metadata.creationTimestamp",
				},
				{
					Name:     "Deleted",
					Type:     "date",
					JSONPath: ".metadata.deletionTimestamp",
				},
				{
					Name:     "MachineSet",
					Type:     "string",
					JSONPath: ".metadata.ownerReferences[0].name",
				},
				{
					Name:     "Address",
					Type:     "string",
					JSONPath: ".status.addresses[0].address",
				},
				{
					Name:     "Node",
					Type:     "string",
					JSONPath: ".status.nodeRef.name",
				},
				{
					Name:     "Provider",
					Type:     "string",
					JSONPath: ".spec.providerSpec.value.cloudProvider",
				},
				{
					Name:     "OS",
					Type:     "string",
					JSONPath: ".spec.providerSpec.value.operatingSystem",
				},
				{
					Name:     "Version",
					Type:     "string",
					JSONPath: ".spec.versions.kubelet",
				},
			}

			return crd, nil
		}
	}

}

// MachineSetCRD returns the machineset CRD definition
func MachineSetCRDCreator() reconciling.NamedCustomResourceDefinitionCreatorGetter {
	return func() (string, reconciling.CustomResourceDefinitionCreator) {
		return resources.MachineSetCRDName, func(crd *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
			crd.Spec.Group = clusterAPIGroup
			crd.Spec.Version = clusterAPIVersion
			crd.Spec.Scope = apiextensionsv1beta1.NamespaceScoped
			crd.Spec.Names.Kind = "MachineSet"
			crd.Spec.Names.ListKind = "MachineSetList"
			crd.Spec.Names.Plural = "machinesets"
			crd.Spec.Names.Singular = "machineset"
			crd.Spec.Names.ShortNames = []string{"ms"}
			crd.Spec.Subresources = &apiextensionsv1beta1.CustomResourceSubresources{
				Status: &apiextensionsv1beta1.CustomResourceSubresourceStatus{},
				Scale: &apiextensionsv1beta1.CustomResourceSubresourceScale{
					SpecReplicasPath:   ".spec.replicas",
					StatusReplicasPath: ".status.replicas",
				}}
			crd.Spec.AdditionalPrinterColumns = []apiextensionsv1beta1.CustomResourceColumnDefinition{
				{
					Name:     "Age",
					Type:     "date",
					JSONPath: ".metadata.creationTimestamp",
				},
				{
					Name:     "Deleted",
					Type:     "date",
					JSONPath: ".metadata.deletionTimestamp",
				},
				{
					Name:     "Replicas",
					Type:     "string",
					JSONPath: ".spec.replicas",
				},
				{
					Name:     "AvailableReplicas",
					Type:     "string",
					JSONPath: ".status.availableReplicas",
				},
				{
					Name:     "MachineDeployment",
					Type:     "string",
					JSONPath: ".metadata.ownerReferences[0].name",
				},
				{
					Name:     "Provider",
					Type:     "string",
					JSONPath: ".spec.template.spec.providerSpec.value.cloudProvider",
				},
				{
					Name:     "OS",
					Type:     "string",
					JSONPath: ".spec.template.spec.providerSpec.value.operatingSystem",
				},
				{
					Name:     "Version",
					Type:     "string",
					JSONPath: ".spec.template.spec.versions.kubelet",
				},
			}

			return crd, nil
		}
	}
}

// MachineDeploymentCRD returns the machinedeployments CRD definition
func MachineDeploymentCRDCreator() reconciling.NamedCustomResourceDefinitionCreatorGetter {
	return func() (string, reconciling.CustomResourceDefinitionCreator) {
		return resources.MachineDeploymentCRDName, func(crd *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
			crd.Spec.Group = clusterAPIGroup
			crd.Spec.Version = clusterAPIVersion
			crd.Spec.Scope = apiextensionsv1beta1.NamespaceScoped
			crd.Spec.Names.Kind = "MachineDeployment"
			crd.Spec.Names.ListKind = "MachineDeploymentList"
			crd.Spec.Names.Plural = "machinedeployments"
			crd.Spec.Names.Singular = "machinedeployment"
			crd.Spec.Names.ShortNames = []string{"md"}
			crd.Spec.Subresources = &apiextensionsv1beta1.CustomResourceSubresources{
				Status: &apiextensionsv1beta1.CustomResourceSubresourceStatus{},
				Scale: &apiextensionsv1beta1.CustomResourceSubresourceScale{
					SpecReplicasPath:   ".spec.replicas",
					StatusReplicasPath: ".status.replicas",
				}}
			crd.Spec.AdditionalPrinterColumns = []apiextensionsv1beta1.CustomResourceColumnDefinition{
				{
					Name:     "Age",
					Type:     "date",
					JSONPath: ".metadata.creationTimestamp",
				},
				{
					Name:     "Deleted",
					Type:     "date",
					JSONPath: ".metadata.deletionTimestamp",
				},
				{
					Name:     "Replicas",
					Type:     "string",
					JSONPath: ".spec.replicas",
				},
				{
					Name:     "AvailableReplicas",
					Type:     "string",
					JSONPath: ".status.availableReplicas",
				},
				{
					Name:     "Provider",
					Type:     "string",
					JSONPath: ".spec.template.spec.providerSpec.value.cloudProvider",
				},
				{
					Name:     "OS",
					Type:     "string",
					JSONPath: ".spec.template.spec.providerSpec.value.operatingSystem",
				},
				{
					Name:     "Version",
					Type:     "string",
					JSONPath: ".spec.template.spec.versions.kubelet",
				},
			}

			return crd, nil
		}
	}
}

// ClusterCRD returns the cluster crd definition
func ClusterCRDCreator() reconciling.NamedCustomResourceDefinitionCreatorGetter {
	return func() (string, reconciling.CustomResourceDefinitionCreator) {
		return resources.ClusterCRDName, func(crd *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
			crd.Spec.Group = clusterAPIGroup
			crd.Spec.Version = clusterAPIVersion
			crd.Spec.Scope = apiextensionsv1beta1.NamespaceScoped
			crd.Spec.Names.Kind = "Cluster"
			crd.Spec.Names.ListKind = "ClusterList"
			crd.Spec.Names.Plural = "clusters"
			crd.Spec.Names.Singular = "cluster"
			crd.Spec.Names.ShortNames = []string{"cl"}
			crd.Spec.Subresources = &apiextensionsv1beta1.CustomResourceSubresources{Status: &apiextensionsv1beta1.CustomResourceSubresourceStatus{}}

			return crd, nil
		}
	}
}
