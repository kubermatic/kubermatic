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

package validation

import (
	"fmt"

	semverlib "github.com/Masterminds/semver/v3"

	appkubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/validation/openapi"

	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateApplicationDefinition(ad appkubermaticv1.ApplicationDefinition) field.ErrorList {
	var parentFieldPath *field.Path = nil
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, ValidateApplicationDefinitionWithOpenAPI(ad, parentFieldPath)...)

	allErrs = append(allErrs, ValidateApplicationVersions(ad.Spec.Versions, parentFieldPath.Child("spec"))...)

	return allErrs
}

func ValidateApplicationVersions(vs []appkubermaticv1.ApplicationVersion, parentFieldPath *field.Path) []*field.Error {
	allErrs := field.ErrorList{}

	lookup := make(map[string]struct{}, len(vs))
	for i, v := range vs {
		curVField := fmt.Sprintf("versions[%d]", i)
		if e := validateSemverRange(string(v.Constraints.K8sVersion), parentFieldPath.Child(curVField+".constraints.k8sVersion")); e != nil {
			allErrs = append(allErrs, e)
		}
		if e := validateSemverRange(string(v.Constraints.KKPVersion), parentFieldPath.Child(curVField+".constraints.kkpVersion")); e != nil {
			allErrs = append(allErrs, e)
		}
		if _, ok := lookup[v.Version]; ok {
			allErrs = append(allErrs, field.Duplicate(parentFieldPath.Child(curVField+".Version"), v.Version))
		} else {
			lookup[v.Version] = struct{}{}
		}
	}

	return allErrs
}

func validateSemverRange(v string, f *field.Path) *field.Error {
	_, err := semverlib.NewConstraint(v)
	if err != nil {
		return field.Invalid(f, v, fmt.Sprintf("is not a valid semVer range: %q", err))
	}
	return nil
}

func ValidateApplicationDefinitionWithOpenAPI(ad appkubermaticv1.ApplicationDefinition, parentFieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	v, err := openapi.NewValidatorForType(&ad.TypeMeta)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(nil, fmt.Errorf("could not create OpenAPI Validator: %w", err)))
		return allErrs
	}
	allErrs = append(allErrs, validation.ValidateCustomResource(parentFieldPath, ad, v)...)

	return allErrs
}
