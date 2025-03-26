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
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateUserSSHKey(key *kubermaticv1.UserSSHKey) field.ErrorList {
	allErrs := field.ErrorList{}

	if key.Spec.PublicKey == "" {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "publicKey"), "no public key specified"))
	}

	if key.Spec.Fingerprint == "" {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "fingerprint"), "no fingerprint specified"))
	}

	if key.Spec.Project == "" {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "project"), "no project specified"))
	}

	return allErrs
}

func ValidateUserSSHKeyCreate(key *kubermaticv1.UserSSHKey) field.ErrorList {
	return ValidateUserSSHKey(key)
}

func ValidateUserSSHKeyUpdate(oldKey, newKey *kubermaticv1.UserSSHKey) field.ErrorList {
	allErrs := ValidateUserSSHKey(newKey)

	if oldKey.Spec.Project != newKey.Spec.Project {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "project"), newKey.Spec.Project, "this field is immutable"))
	}

	return allErrs
}
