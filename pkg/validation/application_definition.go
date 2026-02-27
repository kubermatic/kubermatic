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
	"net/url"

	"github.com/containerd/containerd/v2/core/remotes/docker"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	applicationcatalogmanager "k8c.io/kubermatic/v2/pkg/controller/operator/master/resources/application-catalog"
	"k8c.io/kubermatic/v2/pkg/validation/openapi"

	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"
)

func ValidateApplicationDefinitionSpec(ad appskubermaticv1.ApplicationDefinition) field.ErrorList {
	var parentFieldPath *field.Path = nil
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, ValidateApplicationDefinitionWithOpenAPI(ad, parentFieldPath)...)
	allErrs = append(allErrs, ValidateApplicationVersions(ad.Spec.Versions, parentFieldPath.Child("spec"))...)
	allErrs = append(allErrs, ValidateDeployOpts(ad.Spec.DefaultDeployOptions, parentFieldPath.Child("spec.defaultDeployOptions"))...)
	allErrs = append(allErrs, ValidateApplicationValues(ad.Spec, parentFieldPath.Child("spec"))...)
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

func deleteSystemAppErrorMsg() string {
	return fmt.Sprintf("ApplicationDefinition for system applications are managed by KKP and should not be deleted. "+
		"Deletion can impact user clusters where these Applications are installed. "+
		"If you would still like to remove the ApplicationDefinition, remove the %q label and then delete it.",
		appskubermaticv1.ApplicationManagedByLabel,
	)
}

func deleteAppCatalogManagedAppErrorMsg() string {
	return fmt.Sprintf("ApplicationDefinition for applications managed by ApplicationCatalog cannot be deleted until the " +
		"ApplicationDefinition declaration is removed from the ApplicationCatalog. " +
		"If you would still like to remove the ApplicationDefinition, remove its entry from the ApplicationCatalog first.",
	)
}

func ValidateApplicationDefinitionDelete(ad appskubermaticv1.ApplicationDefinition) field.ErrorList {
	labels := ad.GetLabels()
	if labels != nil {
		if val := labels[appskubermaticv1.ApplicationManagedByLabel]; val == appskubermaticv1.ApplicationManagedByKKPValue {
			return field.ErrorList{
				field.Invalid(
					field.NewPath("metadata").Child("labels"),
					labels,
					deleteSystemAppErrorMsg(),
				),
			}
		}

		// if the application is managed by ApplicationCatalog, the deletion is blocked until the label
		// is removed or the ApplicationDefinition declaration is removed from the ApplicationCatalog.
		if _, exists := labels[applicationcatalogmanager.LabelManagedByApplicationCatalog]; exists {
			return field.ErrorList{
				field.Invalid(
					field.NewPath("metadata").Child("labels"),
					labels,
					deleteAppCatalogManagedAppErrorMsg(),
				),
			}
		}
	}

	var parentFieldPath *field.Path = nil
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, ValidateApplicationDefinitionWithOpenAPI(ad, parentFieldPath)...)
	return allErrs
}

func ValidateApplicationVersions(vs []appskubermaticv1.ApplicationVersion, parentFieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	lookup := make(map[string]struct{}, len(vs))
	for i, v := range vs {
		curVField := fmt.Sprintf("versions[%d]", i)

		allErrs = append(allErrs, validateSource(v.Template.Source, parentFieldPath.Child(curVField+".template.source"))...)

		if _, ok := lookup[v.Version]; ok {
			allErrs = append(allErrs, field.Duplicate(parentFieldPath.Child(curVField+".Version"), v.Version))
		} else {
			lookup[v.Version] = struct{}{}
		}
	}

	return allErrs
}

func validateSource(source appskubermaticv1.ApplicationSource, f *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	switch {
	case source.Helm != nil && source.Git != nil:
		allErrs = append(allErrs, field.Forbidden(f, "only one source type can be provided"))
	case source.Git != nil:
		allErrs = append(allErrs, validateGitSource(source.Git, f.Child("git"))...)
	case source.Helm != nil:
		if errs := validateHelmSource(source.Helm, f.Child("helm")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}

	default:
		allErrs = append(allErrs, field.Required(f, "no source provided"))
	}

	return allErrs
}

func validateHelmSource(helmSource *appskubermaticv1.HelmSource, f *field.Path) field.ErrorList {
	allErrs := validateHelmSourceURL(helmSource, f)

	if e := validateHelmCredentials(helmSource.Credentials, f.Child("credentials")); e != nil {
		allErrs = append(allErrs, e)
	}

	return allErrs
}

func validateHelmSourceURL(helmSource *appskubermaticv1.HelmSource, f *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// url.Parse is _extremely_ forgiving and happily accepts nonsense like "[" or "123" or even "'"
	// as valid URLs.
	parsed, err := url.Parse(helmSource.URL)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(f.Child("url"), helmSource.URL, err.Error()))
		return allErrs
	}

	if parsed.Host == "" || parsed.Scheme == "" {
		allErrs = append(allErrs, field.Invalid(f.Child("url"), helmSource.URL, "value must be a valid URL"))
		return allErrs
	}

	// containerd, if not explicitly configured with a set of host rules, will always use HTTP to
	// communicate with an oci://localhost registry, regardless of any setting in the Helm client.
	// Since installing applications from localhost (i.e. the usercluster-ctrl-mgr Pod) is nonsense,
	// KKP simply forbids HTTPS on localhost; it's easier and less maintenance burden than
	// configuring a custom Helm resolver that has a custom Helm fetcher that configures containerd.
	isLocalhost, err := docker.MatchLocalhost(parsed.Host)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(f.Child("url"), helmSource.URL, fmt.Sprintf("value uses an invalid host section: %v", err)))
		return allErrs
	}

	switch parsed.Scheme {
	case "http":
		if helmSource.Insecure != nil {
			allErrs = append(allErrs, field.Forbidden(f.Child("insecure"), "insecure flag can not be used with http URLs"))
		}

		if u := helmSource.PlainHTTP; u != nil && !*u {
			allErrs = append(allErrs, field.Forbidden(f.Child("plainHTTP"), "plainHTTP flag can not be disabled with http URLs"))
		}

	case "https":
		if u := helmSource.PlainHTTP; u != nil && *u {
			allErrs = append(allErrs, field.Forbidden(f.Child("plainHTTP"), "plainHTTP flag can not be enabled with http URLs"))
		}

		if isLocalhost {
			allErrs = append(allErrs, field.Invalid(f, helmSource.URL, "localhost/loopback URLs cannot use HTTPS"))
		}

	case "oci":
		if plainHTTP := helmSource.PlainHTTP; plainHTTP != nil {
			if *plainHTTP {
				if helmSource.Insecure != nil {
					allErrs = append(allErrs, field.Forbidden(f.Child("insecure"), "insecure flag can not be used with OCI URLs using HTTP"))
				}
			} else if isLocalhost {
				allErrs = append(allErrs, field.Invalid(f, helmSource.URL, "localhost/loopback URLs always use plain HTTP"))
			}
		}
	}

	return allErrs
}

func validateHelmCredentials(credential *appskubermaticv1.HelmCredentials, f *field.Path) *field.Error {
	if credential != nil {
		if credential.RegistryConfigFile != nil && (credential.Username != nil || credential.Password != nil) {
			return field.Forbidden(f.Child("registryConfigFile"), "registryConfigFile can not be used in conjunction with username / password")
		}

		if credential.Username != nil && credential.Password == nil {
			return field.Forbidden(f.Child("password"), "password must be specified with username")
		}
		if credential.Password != nil && credential.Username == nil {
			return field.Forbidden(f.Child("username"), "username must be specified  with password")
		}
	}
	return nil
}

func validateGitSource(gitSource *appskubermaticv1.GitSource, f *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if e := validateGitRef(gitSource.Ref, f.Child("ref")); e != nil {
		allErrs = append(allErrs, e)
	}

	allErrs = append(allErrs, validateGitCredentials(gitSource.Credentials, f.Child("credentials"))...)

	return allErrs
}

func validateGitRef(ref appskubermaticv1.GitReference, f *field.Path) *field.Error {
	if len(ref.Tag) == 0 && len(ref.Branch) == 0 && len(ref.Commit) == 0 {
		return field.Required(f, "at least a branch, a tag  or a commint and branch must be defined")
	}

	if len(ref.Tag) > 0 && (len(ref.Branch) > 0 || len(ref.Commit) > 0) {
		return field.Forbidden(f.Child("tag"), "tag can not be used in conjunction with branch or commit")
	}

	if len(ref.Commit) > 0 && len(ref.Branch) == 0 {
		return field.Forbidden(f.Child("commit"), "commit must be used in conjunction with branch")
	}

	return nil
}

func validateGitCredentials(credentials *appskubermaticv1.GitCredentials, f *field.Path) field.ErrorList {
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

func ValidateApplicationValues(spec appskubermaticv1.ApplicationDefinitionSpec, parentFieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if spec.DefaultValues != nil && len(spec.DefaultValues.Raw) > 0 && spec.DefaultValuesBlock != "" {
		allErrs = append(allErrs, field.Forbidden(parentFieldPath.Child("defaultValues"), "Only defaultValues or defaultValuesBlock can be set, but not both simultaneously"))
		allErrs = append(allErrs, field.Forbidden(parentFieldPath.Child("defaultValuesBlock"), "Only defaultValues or defaultValuesBlock can be set, but not both simultaneously"))
	}

	// we need to verify that the defaultValuesBlock is valid yaml as it is a free-text string
	if spec.DefaultValuesBlock != "" {
		if err := yaml.Unmarshal([]byte(spec.DefaultValuesBlock), struct{}{}); err != nil {
			allErrs = append(allErrs, field.TypeInvalid(parentFieldPath.Child("defaultValues"), nil, fmt.Sprintf("invalid yaml %v", err)))
		}
	}

	return allErrs
}
