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
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/version"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateClusterTemplate(template *kubermaticv1.ClusterTemplate, dc *kubermaticv1.Datacenter, enabledFeatures features.FeatureGate, versions []*version.Version, parentFieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// validate cluster spec passed in ClusterTemplate
	if errs := ValidateClusterSpec(&template.Spec, dc, enabledFeatures, versions, parentFieldPath.Child("spec")); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	return allErrs
}
