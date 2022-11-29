/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// CRDCreator returns a creator that will reconcile a CustomResourceDefinition.
func CRDCreator(crd *apiextensionsv1.CustomResourceDefinition) reconciling.NamedCustomResourceDefinitionReconcilerFactory {
	return func() (string, reconciling.CustomResourceDefinitionCreator) {
		return crd.Name, func(obj *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			obj.Labels = crd.Labels
			obj.Annotations = crd.Annotations
			obj.Spec = crd.Spec

			if crd.Spec.Conversion == nil {
				// reconcile fails if conversion is not set as it's set by default to None
				obj.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{Strategy: apiextensionsv1.NoneConverter}
			}

			return obj, nil
		}
	}
}
