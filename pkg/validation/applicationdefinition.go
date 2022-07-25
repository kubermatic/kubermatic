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

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/validation/openapi"

	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateApplicationDefinitionSpec(ad appskubermaticv1.ApplicationDefinition) field.ErrorList {
	var parentFieldPath *field.Path = nil
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, ValidateApplicationDefinitionWithOpenAPI(ad, parentFieldPath)...)

	allErrs = append(allErrs, ValidateApplicationVersions(ad.Spec.Versions, parentFieldPath.Child("spec"))...)

	return allErrs
}

func ValidateApplicationDefinitionUpdate(newAd appskubermaticv1.ApplicationDefinition, oldAd appskubermaticv1.ApplicationDefinition) field.ErrorList {
	var parentFieldPath *field.Path = nil

	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateApplicationDefinitionSpec(newAd)...)

	// Validate .Spec.Method for immutability
	allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
		newAd.Spec.Method,
		oldAd.Spec.Method,
		parentFieldPath.Child("spec.method"),
	)...)

	return allErrs
}

func ValidateApplicationVersions(vs []appskubermaticv1.ApplicationVersion, parentFieldPath *field.Path) []*field.Error {
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

		allErrs = append(allErrs, validateSource(v.Template.Source, parentFieldPath.Child(curVField+".template.source"))...)

		if _, ok := lookup[v.Version]; ok {
			allErrs = append(allErrs, field.Duplicate(parentFieldPath.Child(curVField+".Version"), v.Version))
		} else {
			lookup[v.Version] = struct{}{}
		}
	}

	return allErrs
}

func validateSource(source appskubermaticv1.ApplicationSource, f *field.Path) []*field.Error {
	allErrs := field.ErrorList{}

	switch {
	case source.Helm != nil && source.Git != nil:
		allErrs = append(allErrs, field.Forbidden(f, "only source type can be provided"))
	case source.Git != nil:
		allErrs = append(allErrs, validateGitSource(source.Git, f.Child("git"))...)
	case source.Helm != nil:
		break // all validations are in cr definition.
	default:
		allErrs = append(allErrs, field.Required(f, "no source provided"))
	}

	return allErrs
}

func validateGitSource(gitSource *appskubermaticv1.GitSource, f *field.Path) []*field.Error {
	allErrs := field.ErrorList{}

	if e := validateGitRef(gitSource.Ref, f.Child("ref")); e != nil {
		allErrs = append(allErrs, e)
	}

	allErrs = append(allErrs, validateGitCredentials(gitSource.Credentials, f.Child("credentials"))...)

	return allErrs
}

func validateGitRef(ref appskubermaticv1.GitReference, f *field.Path) *field.Error {
	if len(ref.Tag) == 0 && len(ref.Branch) == 0 && len(ref.Commit) == 0 {
		return field.Required(f, "at least a branch, a commit or tag must be defined")
	}

	if len(ref.Tag) > 0 && (len(ref.Branch) > 0 || len(ref.Commit) > 0) {
		return field.Forbidden(f.Child("tag"), "tag can not be used in conjunction with branch or commit")
	}
	return nil
}

func validateGitCredentials(credentials *appskubermaticv1.GitCredentials, f *field.Path) []*field.Error {
	allErrs := field.ErrorList{}
	if credentials != nil {
		switch credentials.Method {
		case appskubermaticv1.GitAuthMethodPassword:
			if credentials.Username == nil {
				allErrs = append(allErrs, field.Required(f.Child("username"), "username is required when method is "+string(credentials.Method)))
			}
			if credentials.Password == nil {
				allErrs = append(allErrs, field.Required(f.Child("password"), "password is required when method is "+string(credentials.Method)))
			}

		case appskubermaticv1.GitAuthMethodToken:
			if credentials.Token == nil {
				allErrs = append(allErrs, field.Required(f.Child("token"), "token is reuqied when method is "+string(credentials.Method)))
			}

		case appskubermaticv1.GitAuthMethodSSHKey:
			if credentials.SSHKey == nil {
				allErrs = append(allErrs, field.Required(f.Child("sshKey"), "sshKey is reuqied when method is "+string(credentials.Method)))
			}

		default: // This should never happen.
			allErrs = append(allErrs, field.Invalid(f.Child("method"), credentials.Method, "unknown method"))
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

func ValidateApplicationDefinitionWithOpenAPI(ad appskubermaticv1.ApplicationDefinition, parentFieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	v, err := openapi.NewValidatorForObject(&ad)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(nil, fmt.Errorf("could not create OpenAPI Validator: %w", err)))
		return allErrs
	}
	allErrs = append(allErrs, validation.ValidateCustomResource(parentFieldPath, ad, v)...)

	return allErrs
}
