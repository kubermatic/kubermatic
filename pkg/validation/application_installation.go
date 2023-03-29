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
	"encoding/json"
	"fmt"

	appskubermaticv1 "k8c.io/api/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/cni/cilium"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ValidateApplicationInstallationSpec validates the ApplicationInstallation Spec.
func ValidateApplicationInstallationSpec(ctx context.Context, client ctrlruntimeclient.Client, ai appskubermaticv1.ApplicationInstallation) field.ErrorList {
	specPath := field.NewPath("spec")
	allErrs := field.ErrorList{}
	spec := ai.Spec

	if spec.ReconciliationInterval.Duration < 0 {
		allErrs = append(allErrs, field.Invalid(specPath.Child("reconciliationInterval"), spec.ReconciliationInterval.Duration.String(), "should be a positive value, or zero to disable"))
	}

	// Ensure that the referenced ApplicationDefinition exists only if applicationInstallation is not deleting (removing finalizer raise an UPDATE event)
	if ai.DeletionTimestamp.IsZero() {
		ad := &appskubermaticv1.ApplicationDefinition{}
		err := client.Get(ctx, types.NamespacedName{Name: spec.ApplicationRef.Name}, ad)
		if err != nil {
			if apierrors.IsNotFound(err) {
				allErrs = append(allErrs, field.NotFound(specPath.Child("applicationRef", "name"), spec.ApplicationRef.Name))
			} else {
				allErrs = append(allErrs, field.InternalError(specPath.Child("applicationRef", "name"), err))
			}
			return allErrs
		}

		// Ensure that there is matching version defined in ApplicationDefinition
		exists := false
		desiredVersion := spec.ApplicationRef.Version
		for _, version := range ad.Spec.Versions {
			if version.Version == desiredVersion {
				exists = true
			}
		}

		if !exists {
			allErrs = append(allErrs, field.NotFound(specPath.Child("applicationRef", "version"), spec.ApplicationRef.Version))
		}
	}
	allErrs = append(allErrs, ValidateDeployOpts(spec.DeployOptions, specPath.Child("deployOptions"))...)
	return allErrs
}

func ValidateDeployOpts(deployOpts *appskubermaticv1.DeployOptions, f *field.Path) []*field.Error {
	allErrs := field.ErrorList{}
	if deployOpts != nil && deployOpts.Helm != nil {
		if deployOpts.Helm.Atomic && !deployOpts.Helm.Wait {
			allErrs = append(allErrs, field.Forbidden(f.Child("helm"), "if atomic=true then wait must also be true"))
		}
		// note: deployOpts.Helm.Timeout is time metav1.Duration which guarantee no negative value
		if deployOpts.Helm.Wait && deployOpts.Helm.Timeout.Duration == 0 {
			allErrs = append(allErrs, field.Forbidden(f.Child("helm"), "if wait = true then timeout must be greater than 0"))
		}
		if !deployOpts.Helm.Wait && deployOpts.Helm.Timeout.Duration > 0 {
			allErrs = append(allErrs, field.Forbidden(f.Child("helm"), "if timeout is defined then wait must be true"))
		}
	}
	return allErrs
}

// ValidateApplicationInstallationUpdate validates the update event on an ApplicationInstallation.
func ValidateApplicationInstallationUpdate(ctx context.Context, client ctrlruntimeclient.Client, newAI, oldAI appskubermaticv1.ApplicationInstallation) field.ErrorList {
	specPath := field.NewPath("spec")
	allErrs := field.ErrorList{}

	// Validation for new ApplicationInstallation Spec
	if errs := ValidateApplicationInstallationSpec(ctx, client, newAI); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	// Validate .Spec.Namespace.Create for immutability
	allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
		newAI.Spec.Namespace.Create,
		oldAI.Spec.Namespace.Create,
		specPath.Child("namespace", "create"),
	)...)

	// Validate .Spec.Namespace.Name for immutability
	allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
		newAI.Spec.Namespace.Name,
		oldAI.Spec.Namespace.Name,
		specPath.Child("namespace", "name"),
	)...)

	// Validate .Spec.ApplicationRef.Name for immutability
	allErrs = append(allErrs, apimachineryvalidation.ValidateImmutableField(
		newAI.Spec.ApplicationRef.Name,
		oldAI.Spec.ApplicationRef.Name,
		specPath.Child("applicationRef", "name"),
	)...)

	// Validate managed-by label immutability
	allErrs = append(allErrs, validateImmutableLabel(
		newAI.Labels,
		oldAI.Labels,
		appskubermaticv1.ApplicationManagedByLabel,
	)...)

	// Validate update on ApplicationInstallation managed by KKP
	if oldAI.Labels[appskubermaticv1.ApplicationManagedByLabel] == appskubermaticv1.ApplicationManagedByKKPValue {
		allErrs = append(allErrs, ValidateKKPManagedApplicationInstallationUpdate(newAI, oldAI)...)
	}

	return allErrs
}

// ValidateKKPManagedApplicationInstallationUpdate validates the update event on a KKP-managed ApplicationInstallation.
func ValidateKKPManagedApplicationInstallationUpdate(newAI, oldAI appskubermaticv1.ApplicationInstallation) field.ErrorList {
	allErrs := field.ErrorList{}
	valuesPath := field.NewPath("spec").Child("values")

	// Validate type label immutability
	allErrs = append(allErrs, validateImmutableLabel(
		newAI.Labels,
		oldAI.Labels,
		appskubermaticv1.ApplicationTypeLabel,
	)...)

	// Validate CNI values
	if newAI.Labels[appskubermaticv1.ApplicationTypeLabel] == appskubermaticv1.ApplicationTypeCNIValue {
		newValues := make(map[string]any)
		oldValues := make(map[string]any)
		if len(newAI.Spec.Values.Raw) > 0 {
			if err := json.Unmarshal(newAI.Spec.Values.Raw, &newValues); err != nil {
				allErrs = append(allErrs, field.Invalid(valuesPath, string(newAI.Spec.Values.Raw), fmt.Sprintf("unable to unmarshal values: %s", err)))
			}
		}
		if len(oldAI.Spec.Values.Raw) > 0 {
			if err := json.Unmarshal(oldAI.Spec.Values.Raw, &oldValues); err != nil {
				allErrs = append(allErrs, field.Invalid(valuesPath, string(oldAI.Spec.Values.Raw), fmt.Sprintf("unable to unmarshal values: %s", err)))
			}
		}
		if newAI.Name == kubermaticv1.CNIPluginTypeCilium.String() {
			// Validate Cilium values update
			allErrs = append(allErrs, cilium.ValidateValuesUpdate(newValues, oldValues, valuesPath)...)
		}
	}

	return allErrs
}

func validateImmutableLabel(newLabels, oldLabels map[string]string, labelName string) field.ErrorList {
	allErrs := field.ErrorList{}
	if newLabels[labelName] != oldLabels[labelName] {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("labels"), newLabels, fmt.Sprintf("label \"%s\" is immutable", labelName)))
	}
	return allErrs
}
