//go:build ee

package validation

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func validateCreate(ctx context.Context, policyInstance *kubermaticv1.PolicyInstance, client ctrlruntimeclient.Client) error {
	var allErrs field.ErrorList

	if err := validatePolicyTemplateRef(ctx, policyInstance, client); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := validateEnabledState(ctx, policyInstance, client); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := validateScope(policyInstance); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := validateTarget(ctx, policyInstance); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		return allErrs.ToAggregate()
	}
	return nil
}

func validatePolicyTemplateRef(ctx context.Context, instance *kubermaticv1.PolicyInstance, client ctrlruntimeclient.Client) *field.Error {
	if instance.Spec.PolicyTemplateRef.Name == "" {
		return field.Required(field.NewPath("spec", "policyTemplateRef", "name"), "policy template reference name is required")
	}

	template := &kubermaticv1.PolicyTemplate{}
	if err := client.Get(ctx, types.NamespacedName{Name: instance.Spec.PolicyTemplateRef.Name}, template); err != nil {
		if errors.IsNotFound(err) {
			return field.NotFound(field.NewPath("spec", "policyTemplateRef", "name"),
				fmt.Sprintf("policy template %q not found", instance.Spec.PolicyTemplateRef.Name))
		}
		return field.InternalError(field.NewPath("spec", "policyTemplateRef", "name"), err)
	}

	return nil
}

func validateEnabledState(ctx context.Context, instance *kubermaticv1.PolicyInstance, client ctrlruntimeclient.Client) *field.Error {
	template := &kubermaticv1.PolicyTemplate{}
	if err := client.Get(ctx, types.NamespacedName{Name: instance.Spec.PolicyTemplateRef.Name}, template); err != nil {
		return field.InternalError(field.NewPath("spec", "enabled"), err)
	}

	if template.Spec.Enforced && !instance.Spec.Enabled {
		return field.Forbidden(field.NewPath("spec", "enabled"), "cannot disable policy instance when template is enforced")
	}

	return nil
}

func validateScope(instance *kubermaticv1.PolicyInstance) *field.Error {
	switch instance.Spec.Scope {
	case "cluster", "namespaced":
		return nil
	default:
		return field.NotSupported(field.NewPath("spec", "scope"),
			instance.Spec.Scope, []string{"cluster", "namespaced"})
	}
}

func validateTarget(ctx context.Context, instance *kubermaticv1.PolicyInstance) *field.Error {
	if instance.Spec.Target.AllProjects && instance.Spec.Target.ProjectSelector != nil {
		return field.Invalid(
			field.NewPath("spec", "target"),
			instance.Spec.Target,
			"allProjects and projectSelector cannot be used together",
		)
	}

	if instance.Spec.Target.AllClusters && instance.Spec.Target.ClusterSelector != nil {
		return field.Invalid(
			field.NewPath("spec", "target"),
			instance.Spec.Target,
			"allClusters and clusterSelector cannot be used together",
		)
	}

	return nil
}

func validateUpdate(ctx context.Context, oldObj runtime.Object, newObj runtime.Object, client ctrlruntimeclient.Client) error {
	return nil
}

func validateDelete(ctx context.Context, obj runtime.Object, client ctrlruntimeclient.Client) error {
	return nil
}
