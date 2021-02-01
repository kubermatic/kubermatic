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

package multus

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	configAPIGroup   = "k8s.cni.cncf.io"
	configAPIVersion = "v1"
)

// ConfigCRDCreator returns the multus config CRD definition
func ConfigCRDCreator() reconciling.NamedCustomResourceDefinitionCreatorGetterv1 {
	return func() (string, reconciling.CustomResourceDefinitionCreatorv1) {
		return resources.MultusConfigCRDName, func(crd *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			crd.Spec.Group = configAPIGroup
			crd.Spec.Scope = apiextensionsv1.NamespaceScoped
			crd.Spec.Names.Kind = "NetworkAttachmentDefinition"
			crd.Spec.Names.Plural = "network-attachment-definitions"
			crd.Spec.Names.Singular = "network-attachment-definition"
			crd.Spec.Names.ShortNames = []string{"net-attach-def"}
			crd.Spec.Versions = []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    configAPIVersion,
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Description: "NetworkAttachmentDefinition is a CRD schema specified by the Network Plumbing Working Group to express the intent for attaching pods to one or more logical or physical networks. More information available at: https://github.com/k8snetworkplumbingwg/multi-net-spec",
							Type:        "object",
							Properties: map[string]apiextensionsv1.JSONSchemaProps{
								"apiVersion": {
									Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources",
									Type:        "string",
								},
								"kind": {
									Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds",
									Type:        "string",
								},
								"metadata": {
									Type: "object",
								},
								"spec": {
									Description: "NetworkAttachmentDefinition spec defines the desired state of a network attachment",
									Type:        "object",
									Properties: map[string]apiextensionsv1.JSONSchemaProps{
										"config": {
											Description: "NetworkAttachmentDefinition config is a JSON-formatted CNI configuration",
											Type:        "string",
										},
									},
								},
							},
						},
					},
				},
			}

			return crd, nil
		}
	}
}
