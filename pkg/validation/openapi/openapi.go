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

package openapi

import (
	"fmt"

	"k8c.io/kubermatic/v3/pkg/crd"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kube-openapi/pkg/validation/validate"
)

// NewValidatorForCRD creates a new validator based on the supplied CRD.
func NewValidatorForCRD(crd *apiextensionsv1.CustomResourceDefinition, version string) (*validate.SchemaValidator, error) {
	crdr := &apiextensions.CustomResourceDefinition{}
	if err := apiextensionsv1.Convert_v1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(crd, crdr, nil); err != nil {
		return nil, err
	}

	var err error
	var sv *validate.SchemaValidator
	for _, ver := range crdr.Spec.Versions {
		if ver.Name == version {
			// If there is only one version in the CRD, the crdv1 to crd converter will move the validation
			// into the global .Spec.Validation. Therefore, we need to manually check if per-version or
			// global validation is enabled
			if ver.Schema != nil {
				sv, _, err = validation.NewSchemaValidator(ver.Schema)
			} else {
				sv, _, err = validation.NewSchemaValidator(crdr.Spec.Validation)
			}
			if err != nil {
				return nil, err
			}
			break
		}
	}
	if sv == nil {
		return nil, fmt.Errorf("could not find SchemaValidator for desired version %q", version)
	}

	return sv, nil
}

func NewValidatorForObject(obj runtime.Object) (*validate.SchemaValidator, error) {
	c, err := crd.CRDForObject(obj)
	if err != nil {
		return nil, err
	}

	version := obj.GetObjectKind().GroupVersionKind().Version

	return NewValidatorForCRD(c, version)
}
