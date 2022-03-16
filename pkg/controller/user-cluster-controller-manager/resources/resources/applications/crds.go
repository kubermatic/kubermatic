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

package applications

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApplicationInstallationCRDCreator returns the gatekeeper ApplicationInstallation CRD definition.
func ApplicationInstallationCRDCreator() reconciling.NamedCustomResourceDefinitionCreatorGetter {
	return func() (string, reconciling.CustomResourceDefinitionCreator) {
		return resources.ApplicationInstallationCRDName, func(crd *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			metav1.SetMetaDataAnnotation(&crd.ObjectMeta, "api-approved.kubernetes.io", "unapproved, legacy API")

			crd.Spec.Group = "apps.kubermatic.k8c.io"
			crd.Spec.Scope = apiextensionsv1.ClusterScoped
			crd.Spec.Names.Kind = "ApplicationInstallation"
			crd.Spec.Names.ListKind = "ApplicationInstallationList"
			crd.Spec.Names.Plural = "applicationinstallations"
			crd.Spec.Names.Singular = "applicationinstallation"
			crd.Spec.Names.ShortNames = []string{}
			crd.Spec.Versions = []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1",
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							XPreserveUnknownFields: resources.Bool(true),
							Type:                   "object",
						},
					},
					Subresources: &apiextensionsv1.CustomResourceSubresources{
						Status: &apiextensionsv1.CustomResourceSubresourceStatus{},
					},
				},
			}

			return crd, nil
		}
	}
}
