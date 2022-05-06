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
	"fmt"

	appkubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/crd"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// ApplicationInstallationCRDCreator returns the ApplicationInstallation CRD definition.
func ApplicationInstallationCRDCreator() reconciling.NamedCustomResourceDefinitionCreatorGetter {
	return func() (string, reconciling.CustomResourceDefinitionCreator) {
		return resources.ApplicationInstallationCRDName, func(crdObj *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			c, err := crd.CRDForObject(&appkubermaticv1.ApplicationDefinition{})
			if err != nil {
				return nil, fmt.Errorf("failed to get CRD: %w", err)
			}

			crdObj.Labels = c.Labels
			crdObj.Annotations = c.Annotations
			crdObj.Spec = c.Spec

			if c.Spec.Conversion == nil {
				// reconcile fails if conversion is not set as it's set by default to None
				crdObj.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{Strategy: apiextensionsv1.NoneConverter}
			}

			return crdObj, nil
		}
	}
}
