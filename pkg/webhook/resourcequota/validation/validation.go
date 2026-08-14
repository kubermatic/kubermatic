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
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/resources"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// validator for validating Resource Quota CRD.
type validator struct {
	client   ctrlruntimeclient.Client
	features features.FeatureGate
}

// NewValidator returns a new Resource Quota validator.
func NewValidator(client ctrlruntimeclient.Client, enabledFeatures features.FeatureGate) *validator {
	return &validator{
		client:   client,
		features: enabledFeatures,
	}
}

type acceleratorAccountingValidator struct {
	*validator
}

// NewAcceleratorAccountingValidator returns a validator for the dedicated,
// fail-closed accelerator-accounting admission path.
func NewAcceleratorAccountingValidator(client ctrlruntimeclient.Client, enabledFeatures features.FeatureGate) *acceleratorAccountingValidator {
	return &acceleratorAccountingValidator{validator: NewValidator(client, enabledFeatures)}
}

var _ admission.Validator[*kubermaticv1.ResourceQuota] = &validator{}
var _ admission.Validator[*kubermaticv1.ResourceQuota] = &acceleratorAccountingValidator{}

func (v *validator) ValidateCreate(ctx context.Context, obj *kubermaticv1.ResourceQuota) (admission.Warnings, error) {
	if err := validateAccountingCreate(obj); err != nil {
		return nil, err
	}

	return nil, validateCreate(ctx, obj, v.client)
}

func (v *acceleratorAccountingValidator) ValidateCreate(_ context.Context, obj *kubermaticv1.ResourceQuota) (admission.Warnings, error) {
	return nil, validateAccountingCreate(obj)
}

func validateAccountingCreate(obj *kubermaticv1.ResourceQuota) error {
	if obj != nil {
		if _, exists := obj.Annotations[resources.AcceleratorAccountingEnabledAnnotation]; exists {
			return fmt.Errorf("ResourceQuota annotation %q cannot be set during creation; wait for the ResourceQuota controllers to prepare the object before enabling accelerator accounting", resources.AcceleratorAccountingEnabledAnnotation)
		}
	}
	return nil
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj *kubermaticv1.ResourceQuota) (admission.Warnings, error) {
	if oldObj != nil && newObj != nil {
		if err := v.validateAccountingUpdate(ctx, oldObj, newObj); err != nil {
			return nil, err
		}
	}

	return nil, validateUpdate(ctx, oldObj, newObj)
}

func (v *acceleratorAccountingValidator) ValidateUpdate(ctx context.Context, oldObj, newObj *kubermaticv1.ResourceQuota) (admission.Warnings, error) {
	if oldObj != nil && newObj != nil {
		return nil, v.validateAccountingUpdate(ctx, oldObj, newObj)
	}
	return nil, nil
}

func (v *validator) ValidateDelete(ctx context.Context, obj *kubermaticv1.ResourceQuota) (admission.Warnings, error) {
	if err := v.validateAccountingDelete(ctx, obj); err != nil {
		return nil, err
	}

	return nil, validateDelete(ctx, obj, v.client)
}

func (v *acceleratorAccountingValidator) ValidateDelete(ctx context.Context, obj *kubermaticv1.ResourceQuota) (admission.Warnings, error) {
	return nil, v.validateAccountingDelete(ctx, obj)
}

func (v *validator) validateAccountingDelete(ctx context.Context, obj *kubermaticv1.ResourceQuota) error {
	if obj != nil && obj.Annotations[resources.AcceleratorAccountingEnabledAnnotation] == resources.AcceleratorAccountingEnabledAnnotationValue {
		project := &kubermaticv1.Project{}
		if err := v.client.Get(ctx, types.NamespacedName{Name: obj.Spec.Subject.Name}, project); err != nil {
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to get ResourceQuota project %q: %w", obj.Spec.Subject.Name, err)
			}
		} else if project.DeletionTimestamp.IsZero() && project.Status.Phase != kubermaticv1.ProjectTerminating {
			return fmt.Errorf("activated ResourceQuota %q cannot be deleted while project %q is active", obj.Name, project.Name)
		}
	}
	return nil
}

func (v *validator) validateAccountingUpdate(ctx context.Context, oldQuota, newQuota *kubermaticv1.ResourceQuota) error {
	oldValue, oldHasAnnotation := oldQuota.Annotations[resources.AcceleratorAccountingEnabledAnnotation]
	newValue, newHasAnnotation := newQuota.Annotations[resources.AcceleratorAccountingEnabledAnnotation]
	// Once deletion has started, leave the existing lifecycle controllers free
	// to finish their normal cleanup updates.
	if !newQuota.DeletionTimestamp.IsZero() {
		return nil
	}

	if oldHasAnnotation {
		if oldValue == resources.AcceleratorAccountingEnabledAnnotationValue {
			return validateEnabledAcceleratorAccountingUpdate(oldQuota, newQuota, newValue, newHasAnnotation)
		}

		// The webhook never permits creating a non-canonical value. Still allow such a
		// value, if it was written while the webhook was unavailable, to be removed.
		if newHasAnnotation {
			return fmt.Errorf("ResourceQuota annotation %q has invalid value %q; remove it before enabling accelerator accounting", resources.AcceleratorAccountingEnabledAnnotation, oldValue)
		}
		return nil
	}

	if !newHasAnnotation {
		return nil
	}
	return v.validateAcceleratorAccountingActivation(ctx, oldQuota, newQuota, newValue)
}

func validateEnabledAcceleratorAccountingUpdate(oldQuota, newQuota *kubermaticv1.ResourceQuota, newValue string, newHasAnnotation bool) error {
	if !newHasAnnotation || newValue != resources.AcceleratorAccountingEnabledAnnotationValue {
		return fmt.Errorf("ResourceQuota annotation %q is immutable once accelerator accounting is enabled", resources.AcceleratorAccountingEnabledAnnotation)
	}
	if err := validateAcceleratorAccountingResourceQuota(newQuota); err != nil {
		return err
	}
	oldOwner := metav1.GetControllerOf(oldQuota)
	newOwner := metav1.GetControllerOf(newQuota)
	if !sameOwnerReference(oldOwner, newOwner) {
		return fmt.Errorf("ResourceQuota must retain its project controller owner reference while accelerator accounting is enabled")
	}
	if oldQuota.Spec.Subject != newQuota.Spec.Subject {
		return fmt.Errorf("ResourceQuota project subject is immutable while accelerator accounting is enabled")
	}
	if newOwner == nil || newOwner.Name != newQuota.Spec.Subject.Name || newOwner.Kind != kubermaticv1.ProjectKindName {
		return fmt.Errorf("ResourceQuota must retain a controller owner reference matching its project subject while accelerator accounting is enabled")
	}
	if !hasMatchingSubjectLabels(newQuota) {
		return fmt.Errorf("ResourceQuota must retain subject labels matching its project while accelerator accounting is enabled")
	}
	if len(newQuota.Spec.Quota.Accelerators) != 0 {
		return fmt.Errorf("ResourceQuota accelerator limits cannot be configured until accelerator accounting readiness and enforcement are implemented")
	}
	return nil
}

func (v *validator) validateAcceleratorAccountingActivation(ctx context.Context, oldQuota, newQuota *kubermaticv1.ResourceQuota, newValue string) error {
	if newValue != resources.AcceleratorAccountingEnabledAnnotationValue {
		return fmt.Errorf("ResourceQuota annotation %q must have the exact value %q", resources.AcceleratorAccountingEnabledAnnotation, resources.AcceleratorAccountingEnabledAnnotationValue)
	}
	if !acceleratorAccountingSupported() {
		return fmt.Errorf("ResourceQuota accelerator accounting activation is only supported in Kubermatic Enterprise Edition")
	}
	if !v.features.Enabled(features.KubeVirtAcceleratorQuota) {
		return fmt.Errorf("ResourceQuota annotation %q requires the %q feature gate", resources.AcceleratorAccountingEnabledAnnotation, features.KubeVirtAcceleratorQuota)
	}
	if newQuota.Spec.Subject.Kind != kubermaticv1.ProjectSubjectKind {
		return fmt.Errorf("ResourceQuota annotation %q is only supported for project quotas", resources.AcceleratorAccountingEnabledAnnotation)
	}
	if err := validateAcceleratorAccountingResourceQuota(oldQuota); err != nil {
		return err
	}
	if err := validateAcceleratorAccountingResourceQuota(newQuota); err != nil {
		return err
	}
	if len(newQuota.Spec.Quota.Accelerators) != 0 {
		return fmt.Errorf("ResourceQuota accelerator limits must be empty when enabling accelerator accounting; enable accounting first, then configure accelerator limits")
	}
	if !hasMatchingSubjectLabels(oldQuota) {
		return fmt.Errorf("ResourceQuota must already have subject labels matching its project before accelerator accounting can be enabled")
	}
	if !hasMatchingSubjectLabels(newQuota) {
		return fmt.Errorf("ResourceQuota must retain subject labels matching its project when accelerator accounting is enabled")
	}

	project := &kubermaticv1.Project{}
	if err := v.client.Get(ctx, types.NamespacedName{Name: newQuota.Spec.Subject.Name}, project); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("cannot enable accelerator accounting: project %q does not exist", newQuota.Spec.Subject.Name)
		}
		return fmt.Errorf("failed to get ResourceQuota project %q: %w", newQuota.Spec.Subject.Name, err)
	}
	if !project.DeletionTimestamp.IsZero() || project.Status.Phase == kubermaticv1.ProjectTerminating {
		return fmt.Errorf("cannot enable accelerator accounting while project %q is terminating", project.Name)
	}
	if !hasMatchingProjectControllerReference(oldQuota, project) {
		return fmt.Errorf("ResourceQuota must already have a controller owner reference matching project %q before accelerator accounting can be enabled", project.Name)
	}
	if !hasMatchingProjectControllerReference(newQuota, project) {
		return fmt.Errorf("ResourceQuota must retain its controller owner reference to project %q when accelerator accounting is enabled", project.Name)
	}

	return nil
}

func hasMatchingProjectControllerReference(quota *kubermaticv1.ResourceQuota, project *kubermaticv1.Project) bool {
	owner := metav1.GetControllerOf(quota)
	return owner != nil &&
		owner.APIVersion == kubermaticv1.SchemeGroupVersion.String() &&
		owner.Kind == kubermaticv1.ProjectKindName &&
		owner.Name == project.Name &&
		owner.UID == project.UID
}

func sameOwnerReference(oldOwner, newOwner *metav1.OwnerReference) bool {
	return oldOwner != nil && newOwner != nil &&
		oldOwner.APIVersion == newOwner.APIVersion &&
		oldOwner.Kind == newOwner.Kind &&
		oldOwner.Name == newOwner.Name &&
		oldOwner.UID == newOwner.UID
}

func hasMatchingSubjectLabels(quota *kubermaticv1.ResourceQuota) bool {
	return quota.Labels[kubermaticv1.ResourceQuotaSubjectNameLabelKey] == quota.Spec.Subject.Name &&
		quota.Labels[kubermaticv1.ResourceQuotaSubjectKindLabelKey] == quota.Spec.Subject.Kind
}
