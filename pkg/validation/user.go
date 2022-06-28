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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	"k8s.io/apimachinery/pkg/util/validation/field"
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

	return allErrs
}

func ValidateUserCreate(user *kubermaticv1.User) field.ErrorList {
	allErrs := ValidateUser(user)

	if kubermaticv1helper.IsProjectServiceAccount(user.Name) {
		if _, exists := user.Labels[kubernetes.ServiceAccountLabelGroup]; !exists {
			allErrs = append(allErrs, field.Required(field.NewPath("metadata", "labels"), fmt.Sprintf("service accounts must define their group using a %q label", kubernetes.ServiceAccountLabelGroup)))
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
