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
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func ValidateUser(u *kubermaticv1.User) field.ErrorList {
	allErrs := field.ErrorList{}
	specPath := field.NewPath("spec")

	if u.Spec.Email == "" {
		allErrs = append(allErrs, field.Required(specPath.Child("email"), "this field is required"))
	}

	if u.Spec.Name == "" {
		allErrs = append(allErrs, field.Required(specPath.Child("name"), "this field is required"))
	}

	saEmail := kubermaticv1helper.IsProjectServiceAccount(u.Spec.Email)

	if kubermaticv1helper.IsProjectServiceAccount(u.Name) {
		if u.Spec.Project == "" {
			allErrs = append(allErrs, field.Required(specPath.Child("project"), "service accounts must reference a project"))
		}
		if !saEmail {
			allErrs = append(allErrs, field.Forbidden(specPath.Child("email"), fmt.Sprintf("service account users must have an email address starting with %q", kubermaticv1helper.UserServiceAccountPrefix)))
		}
	} else {
		if u.Spec.Project != "" {
			allErrs = append(allErrs, field.Forbidden(specPath.Child("project"), "regular users must not set a project reference"))
		}
		if saEmail {
			allErrs = append(allErrs, field.Forbidden(specPath.Child("email"), fmt.Sprintf("regular users must not have an email address starting with %q", kubermaticv1helper.UserServiceAccountPrefix)))
		}
	}

	// Ensure that `isAdmin` and `globalViewer` are not both true at the same time.
	// These roles are mutually exclusive — a user can't be both an admin and a global viewer.
	if isAdminAndGlobalViewer(u.Spec) {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec"),
			u.Spec,
			"`isAdmin` and `globalViewer` cannot both be true",
		))
	}

	return allErrs
}

func ValidateUserCreate(user *kubermaticv1.User) field.ErrorList {
	allErrs := ValidateUser(user)

	if kubermaticv1helper.IsProjectServiceAccount(user.Name) {
		if _, exists := user.Labels[kubermaticv1.ServiceAccountInitialGroupLabel]; !exists {
			allErrs = append(allErrs, field.Required(field.NewPath("metadata", "labels"), fmt.Sprintf("service accounts must define their group using a %q label", kubermaticv1.ServiceAccountInitialGroupLabel)))
		}
	}

	return allErrs
}

func ValidateUserUpdate(oldUser, newUser *kubermaticv1.User) field.ErrorList {
	allErrs := ValidateUser(newUser)
	specPath := field.NewPath("spec")

	if oldUser.Spec.Email != newUser.Spec.Email {
		allErrs = append(allErrs, field.Invalid(specPath.Child("email"), newUser.Spec.Email, "this field is immutable"))
	}

	if oldUser.Spec.Project != newUser.Spec.Project {
		allErrs = append(allErrs, field.Invalid(specPath.Child("project"), newUser.Spec.Project, "this field is immutable"))
	}

	return allErrs
}

func ValidateUserDelete(ctx context.Context,
	user *kubermaticv1.User,
	client ctrlruntimeclient.Client,
	seedsGetter provider.SeedsGetter,
	seedClientGetter provider.SeedClientGetter) error {
	projects, err := util.GetUserOwnedProjects(ctx, client, user.Name)
	if err != nil {
		return err
	}

	for _, project := range projects {
		// if project has multiple owner users
		if len(project.OwnerReferences) > 1 {
			continue
		}

		// project has single owner user then check if project has resources

		// if project has externalclusters
		hasExtClusters, err := util.HasExternalClusters(ctx, client, project.Name)
		if err != nil {
			return err
		}
		if hasExtClusters {
			return fmt.Errorf("operation not permitted!: user project %s has resources i.e., externalclusters", project.Name)
		}

		// if project has clusters on any seed
		seeds, err := seedsGetter()
		if err != nil {
			return fmt.Errorf("failed to list the seeds: %w", err)
		}
		for _, seed := range seeds {
			seedClient, err := seedClientGetter(seed)
			if err != nil {
				return fmt.Errorf("failed to get Seed client: %w", err)
			}
			hasClusters, err := util.HasClusters(ctx, seedClient, project.Name)
			if err != nil {
				return err
			}
			if hasClusters {
				return fmt.Errorf("operation not permitted!: user project %s has resources i.e., clusters", project.Name)
			}
		}
	}

	return nil
}

func isAdminAndGlobalViewer(userSpec kubermaticv1.UserSpec) bool {
	if userSpec.IsAdmin && userSpec.IsGlobalViewer {
		return true
	}
	return false
}
