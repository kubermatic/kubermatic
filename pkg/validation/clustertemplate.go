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
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/version"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateClusterTemplate validates a kubermaticv1.ClusterTemplate resource.
func ValidateClusterTemplate(ctx context.Context, template *kubermaticv1.ClusterTemplate, dc *kubermaticv1.Datacenter, cloudProvider provider.CloudProvider, enabledFeatures features.FeatureGate, versionManager *version.Manager, parentFieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	scope := template.Labels[kubermaticv1.ClusterTemplateScopeLabelKey]

	if scope == "" {
		allErrs = append(allErrs, field.Required(
			parentFieldPath.Child("metadata", "labels", "scope"),
			fmt.Sprintf("label '%s' is required", kubermaticv1.ClusterTemplateScopeLabelKey),
		))
		return allErrs
	}

	// ensure that a ClusterTemplate has a project reference
	if scope == kubermaticv1.ProjectClusterTemplateScope {
		projectID, ok := template.Labels[kubermaticv1.ClusterTemplateProjectLabelKey]
		if !ok || projectID == "" {
			allErrs = append(allErrs, field.Required(
				parentFieldPath.Child("metadata", "labels", kubermaticv1.ClusterTemplateProjectLabelKey),
				fmt.Sprintf("label '%s' is required", kubermaticv1.ClusterTemplateProjectLabelKey),
			))
			return allErrs
		}
	}

	// validate SSH keys having an ID that is not empty
	if template.UserSSHKeys != nil {
		for i, key := range template.UserSSHKeys {
			path := parentFieldPath.Child("userSSHKeys").Index(i)
			if key.ID == "" {
				allErrs = append(allErrs, field.Invalid(path.Child("id"), key.ID, "SSH key ID needs to be set"))
			}
		}
	}

	// For seed scope ClusterTemplate, cloud provider specification configurations are not allowed.
	if scope == kubermaticv1.SeedTemplateScope {
		cloudSpecPath := field.NewPath("spec", "cloud")
		if template.Spec.Cloud.ProviderName != "" {
			allErrs = append(allErrs, field.Required(
				cloudSpecPath.Child("providerName"),
				"should be empty for default(seed scoped) Cluster Templates",
			))
		}

		providerName, err := kubermaticv1helper.ClusterCloudProviderName(template.Spec.Cloud)
		if err != nil {
			allErrs = append(allErrs, field.InternalError(cloudSpecPath, err))
		}
		if providerName != "" {
			allErrs = append(allErrs, field.Required(
				cloudSpecPath,
				"cloud provider configuration is not allowed for default(seed scoped) Cluster Templates",
			))
		}
		// Seed ClusterTemplates are special cases which are omitted from validations for Cluster Spec apart from the Cloud Spec, for now.
		return allErrs
	}

	// validate cluster spec passed in ClusterTemplate
	if errs := ValidateNewClusterSpec(ctx, &template.Spec, dc, cloudProvider, versionManager, enabledFeatures, parentFieldPath.Child("spec")); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	return allErrs
}
