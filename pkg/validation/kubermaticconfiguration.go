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
	"errors"
	"fmt"
	"strconv"
	"strings"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/distribution/reference"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/version"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateKubermaticConfigurationSpec(spec *kubermaticv1.KubermaticConfigurationSpec) field.ErrorList {
	allErrs := field.ErrorList{}

	// Validate the MirrorImages field
	if err := ValidateMirrorImages(spec.MirrorImages); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "mirrorImages"), spec.MirrorImages, err.Error()))
	}

	// Validate ApplicationDefinitions configuration
	if errs := ValidateApplicationDefinitionsConfiguration(spec.Applications, field.NewPath("spec", "applications")); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	// general cloud spec logic
	if errs := ValidateKubermaticVersioningConfiguration(spec.Versions, field.NewPath("spec", "versions")); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
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

	// ensure that the update rules make sense
	allErrs = append(allErrs, validateAutomaticUpdateRulesOnlyPointToValidVersions(config, parentFieldPath)...)

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

// "valid versions" is here defined as the output of versionManager.GetVersions(), which
// will only return those versions that would _not_ cause an automated update.
// Having an update rule point to a version which then would also immediately be upgraded
// by another rule is invalid, as the cluster webhook will reject such "temporary" versions
// and require users to choose the final version already.
func validateAutomaticUpdateRulesOnlyPointToValidVersions(config kubermaticv1.KubermaticVersioningConfiguration, parentFieldPath *field.Path) field.ErrorList {
	manager := version.NewFromConfiguration(&kubermaticv1.KubermaticConfiguration{
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Versions: config,
		},
	})

	allErrs := field.ErrorList{}

	validVersions, err := manager.GetVersions()
	if err != nil {
		allErrs = append(allErrs, field.Invalid(parentFieldPath.Child("versions"), nil, err.Error()))
		return allErrs
	}

	for i, update := range config.Updates {
		is := strconv.Itoa(i)

		// only test automatic rules
		if (update.Automatic == nil || !*update.Automatic) && (update.AutomaticNodeUpdate == nil || !*update.AutomaticNodeUpdate) {
			continue
		}

		toVersion, err := semverlib.NewVersion(update.To)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(parentFieldPath.Child("updates", is), update, err.Error()))
			continue
		}

		found := false
		for _, v := range validVersions {
			if v.Version.Equal(toVersion) {
				found = true
			}
		}

		if !found {
			err := errors.New("this update rule points to a version which is not configured as allowed or for which another update rule also exists")
			allErrs = append(allErrs, field.Invalid(parentFieldPath.Child("updates", is), update, err.Error()))
		}
	}

	return allErrs
}

func ValidateMirrorImages(images []string) error {
	for _, img := range images {
		// Parse the image reference using distribution/reference
		named, err := reference.Parse(img)
		if err != nil {
			return fmt.Errorf("invalid image reference %q: %w", img, err)
		}

		// Ensure the image is tagged
		_, ok := named.(reference.NamedTagged)
		if !ok {
			return fmt.Errorf("image reference %q must include a tag", img)
		}
	}
	return nil
}

func ValidateApplicationDefinitionsConfiguration(config kubermaticv1.ApplicationDefinitionsConfiguration, parentFieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Validate DefaultApplicationCatalog settings
	if errs := ValidateDefaultApplicationCatalogSettings(config.DefaultApplicationCatalog, parentFieldPath.Child("defaultApplicationCatalog")); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	return allErrs
}

func ValidateDefaultApplicationCatalogSettings(config kubermaticv1.DefaultApplicationCatalogSettings, parentFieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Validate that HelmCredentials and HelmRegistryConfigFile are not both set
	if config.HelmCredentials != nil && config.HelmRegistryConfigFile != nil {
		allErrs = append(allErrs, field.Invalid(parentFieldPath.Child("helmCredentials"), config.HelmCredentials, "helmCredentials and helmRegistryConfigFile cannot be set simultaneously"))
	}

	// Validate HelmCredentials if provided
	if config.HelmCredentials != nil {
		if errs := ValidateHelmCredentials(config.HelmCredentials, parentFieldPath.Child("helmCredentials")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	return allErrs
}

func ValidateHelmCredentials(credentials *appskubermaticv1.HelmCredentials, parentFieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Validate that either username/password or registryConfigFile is set, but not both
	hasUsernamePassword := credentials.Username != nil || credentials.Password != nil
	hasRegistryConfig := credentials.RegistryConfigFile != nil

	if hasUsernamePassword && hasRegistryConfig {
		allErrs = append(allErrs, field.Invalid(parentFieldPath, credentials, "either username/password or registryConfigFile can be defined, but not both"))
	}

	if !hasUsernamePassword && !hasRegistryConfig {
		allErrs = append(allErrs, field.Invalid(parentFieldPath, credentials, "either username/password or registryConfigFile must be defined"))
	}

	// If username is set, password must also be set
	if credentials.Username != nil && credentials.Password == nil {
		allErrs = append(allErrs, field.Required(parentFieldPath.Child("password"), "password is required when username is provided"))
	}

	// If password is set, username must also be set
	if credentials.Password != nil && credentials.Username == nil {
		allErrs = append(allErrs, field.Required(parentFieldPath.Child("username"), "username is required when password is provided"))
	}

	return allErrs
}
