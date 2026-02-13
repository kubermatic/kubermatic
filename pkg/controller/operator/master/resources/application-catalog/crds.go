/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package applicationcatalogmanager

import (
	_ "embed"

	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	// ApplicationCatalogCRDName is the name of the ApplicationCatalog CRD.
	ApplicationCatalogCRDName = "applicationcatalogs.applicationcatalog.k8c.io"
)

var (
	//go:embed static/crd-applicationcatalog.yaml
	applicationCatalogCRDYAML string
)

// ApplicationCatalogCRDReconciler returns the ApplicationCatalog CRD definition.
func ApplicationCatalogCRDReconciler() reconciling.NamedCustomResourceDefinitionReconcilerFactory {
	return func() (string, reconciling.CustomResourceDefinitionReconciler) {
		return ApplicationCatalogCRDName, func(crd *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			var fileCRD *apiextensionsv1.CustomResourceDefinition
			err := yaml.UnmarshalStrict([]byte(applicationCatalogCRDYAML), &fileCRD)
			if err != nil {
				return nil, err
			}

			crd.Labels = fileCRD.Labels
			crd.Annotations = fileCRD.Annotations
			crd.Spec = fileCRD.Spec
			crd.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{Strategy: apiextensionsv1.NoneConverter}

			return crd, nil
		}
	}
}
