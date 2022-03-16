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
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/validation/openapi"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"
)

// ApplicationInstallationCRDCreator returns the ApplicationInstallation CRD definition.
func ApplicationInstallationCRDCreator() reconciling.NamedCustomResourceDefinitionCreatorGetter {
	return func() (string, reconciling.CustomResourceDefinitionCreator) {
		return resources.ApplicationInstallationCRDName, func(crd *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			f, err := openapi.Efs.ReadFile(resources.ApplicationInstallationCRDFilePath)
			if err != nil {
				return nil, fmt.Errorf("failed to get crd file: %w", err)
			}

			var fileCRD *apiextensionsv1.CustomResourceDefinition
			if err := yaml.UnmarshalStrict(f, &fileCRD); err != nil {
				return nil, err
			}

			crd.Labels = fileCRD.Labels
			crd.Annotations = fileCRD.Annotations
			crd.Spec = fileCRD.Spec

			if fileCRD.Spec.Conversion == nil {
				// reconcile fails if conversion is not set as it's set by default to None
				crd.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{Strategy: apiextensionsv1.NoneConverter}
			}

			return crd, nil
		}
	}
}
