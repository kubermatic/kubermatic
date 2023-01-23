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

package operatingsystemmanager

import (
	_ "embed"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var (
	//go:embed static/crd-operatingsystemprofile.yaml
	operatingSystemProfileYAML string

	//go:embed static/crd-operatingsystemconfig.yaml
	operatingSystemConfigYAML string
)

// OperatingSystemProfileCRDReconciler returns the OperatingSystemManager operatingSystemProfile CRD definition.
func OperatingSystemProfileCRDReconciler() reconciling.NamedCustomResourceDefinitionReconcilerFactory {
	return func() (string, reconciling.CustomResourceDefinitionReconciler) {
		return resources.OperatingSystemManagerOperatingSystemProfileCRDName, func(crd *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			var fileCRD *apiextensionsv1.CustomResourceDefinition
			err := yaml.UnmarshalStrict([]byte(operatingSystemProfileYAML), &fileCRD)
			if err != nil {
				return nil, err
			}

			crd.Labels = fileCRD.Labels
			crd.Annotations = fileCRD.Annotations
			crd.Spec = fileCRD.Spec

			// reconcile fails if conversion is not set as it's set by default to None
			crd.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{Strategy: apiextensionsv1.NoneConverter}

			return crd, nil
		}
	}
}

// OperatingSystemConfigCRDReconciler returns the OperatingSystemManager operatingSystemConfig CRD definition.
func OperatingSystemConfigCRDReconciler() reconciling.NamedCustomResourceDefinitionReconcilerFactory {
	return func() (string, reconciling.CustomResourceDefinitionReconciler) {
		return resources.OperatingSystemManagerOperatingSystemConfigCRDName, func(crd *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			var fileCRD *apiextensionsv1.CustomResourceDefinition
			err := yaml.UnmarshalStrict([]byte(operatingSystemConfigYAML), &fileCRD)
			if err != nil {
				return nil, err
			}

			crd.Labels = fileCRD.Labels
			crd.Annotations = fileCRD.Annotations
			crd.Spec = fileCRD.Spec

			// reconcile fails if conversion is not set as it's set by default to None
			crd.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{Strategy: apiextensionsv1.NoneConverter}

			return crd, nil
		}
	}
}
