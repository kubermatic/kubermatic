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
	"strings"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/features"

	"github.com/kcp-dev/logicalcluster/v2"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateKubermaticConfigurationSpec(spec *kubermaticv1.KubermaticConfigurationSpec) field.ErrorList {
	allErrs := field.ErrorList{}

	if errs := ValidateKubermaticVersioningConfiguration(spec.Versions, field.NewPath("spec", "versions")); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	if spec.FeatureGates[features.KCPUserManagement] {
		if errs := ValidateKubermaticConfigurationKCPConfiguration(spec.KCP, field.NewPath("spec", "kcp")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	return allErrs
}

func ValidateKubermaticConfigurationKCPConfiguration(config kubermaticv1.KubermaticKCPConfiguration, parentFieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	name := logicalcluster.New(config.HomeRootPrefix)
	if !name.IsValid() {
		allErrs = append(allErrs, field.Invalid(parentFieldPath.Child("homeRootPrefix"), config.HomeRootPrefix, "not a valid logical cluster name"))
	} else if name == logicalcluster.Wildcard {
		allErrs = append(allErrs, field.Invalid(parentFieldPath.Child("homeRootPrefix"), config.HomeRootPrefix, "cannot use wildcard"))
	}

	return allErrs
}

func ValidateKubermaticVersioningConfiguration(config kubermaticv1.KubermaticVersioningConfiguration, parentFieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if config.Default == nil {
		allErrs = append(allErrs, field.Required(parentFieldPath.Child("default"), "no default version configured"))
	} else {
		validDefault := false

		for _, v := range config.Versions {
			if v.Equal(config.Default) {
				validDefault = true
				break
			}
		}

		if !validDefault {
			allErrs = append(allErrs, field.Invalid(parentFieldPath.Child("default"), config.Default, "default version is not listed in the list of supported versions"))
		}
	}

	// collect a sorted list of minor versions
	minorSet := sets.NewInt()
	for _, version := range config.Versions {
		minorSet.Insert(int(version.Semver().Minor()))
	}

	minors := minorSet.List()

	// if there are less than 2 versions, there is no point in checking for gaps
	if len(minors) < 2 {
		return allErrs
	}

	start := minors[0]
	end := minors[len(minors)-1]
	missing := []string{}

	for minor := start; minor < end; minor++ {
		if !minorSet.Has(minor) {
			missing = append(missing, fmt.Sprintf("v1.%d", minor))
		}
	}

	if len(missing) > 0 {
		msg := fmt.Sprintf("no versions for the minor releases %s configured, cannot have gaps", strings.Join(missing, ", "))
		allErrs = append(allErrs, field.Invalid(parentFieldPath.Child("versions"), minors, msg))
	}

	return allErrs
}
