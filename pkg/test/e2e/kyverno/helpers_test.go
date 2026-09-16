//go:build e2e && ee

/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package kyverno

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kyvernocontroller "k8c.io/kubermatic/v2/pkg/ee/kyverno"
	commonseedresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/common"
	userclusterresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/user-cluster"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func newPolicyBinding(bindingNamespace, name, policyNamespace string, namespaced bool) *kubermaticv1.PolicyBinding {
	binding := &kubermaticv1.PolicyBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: bindingNamespace,
		},
		Spec: kubermaticv1.PolicyBindingSpec{
			PolicyTemplateRef: corev1.ObjectReference{Name: name},
		},
	}
	if namespaced {
		binding.Spec.KyvernoPolicyNamespace = &kubermaticv1.KyvernoPolicyNamespace{Name: policyNamespace}
	}
	return binding
}

func newPolicyTemplate(name, policyNamespace string, namespaced, enforced bool) (*kubermaticv1.PolicyTemplate, error) {
	pattern, err := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"labels": map[string]string{requiredLabelKey: requiredLabelValue},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal validation pattern: %w", err)
	}

	resourceDescription := kyvernov1.ResourceDescription{
		Kinds: []string{"ConfigMap"},
		Names: []string{"kyverno-e2e-*"},
	}
	if !namespaced {
		resourceDescription.Namespaces = []string{policyNamespace}
	}

	failureAction := kyvernov1.Enforce
	background := false
	policySpec, err := json.Marshal(kyvernov1.Spec{
		Background: &background,
		Rules: []kyvernov1.Rule{{
			Name: "require-e2e-label",
			MatchResources: kyvernov1.MatchResources{
				Any: kyvernov1.ResourceFilters{{ResourceDescription: resourceDescription}},
			},
			Validation: &kyvernov1.Validation{
				FailureAction: &failureAction,
				Message:       policyDenyMessage,
				RawPattern:    &apiextensionsv1.JSON{Raw: pattern},
			},
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Kyverno policy spec: %w", err)
	}

	return &kubermaticv1.PolicyTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: kubermaticv1.PolicyTemplateSpec{
			Title:            "Kyverno e2e required label",
			Description:      "Requires a label on ConfigMaps created by the Kyverno integration e2e test.",
			Visibility:       kubermaticv1.PolicyTemplateVisibilityGlobal,
			Enforced:         enforced,
			NamespacedPolicy: namespaced,
			PolicySpec:       runtime.RawExtension{Raw: policySpec},
		},
	}, nil
}

func setKyvernoEnabled(ctx context.Context, client ctrlruntimeclient.Client, clusterName string, enabled bool) error {
	cluster := &kubermaticv1.Cluster{}
	if err := client.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	oldCluster := cluster.DeepCopy()
	cluster.Spec.Kyverno = &kubermaticv1.KyvernoSettings{Enabled: enabled}
	if err := client.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
		return fmt.Errorf("failed to patch Kyverno enabled=%t: %w", enabled, err)
	}

	return nil
}

func waitForSeedKyvernoControllersReady(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, namespace string) error {
	deployments := map[string]int32{
		commonseedresources.KyvernoAdmissionControllerDeploymentName:  commonseedresources.KyvernoAdmissionControllerReplicas,
		commonseedresources.KyvernoBackgroundControllerDeploymentName: commonseedresources.KyvernoBackgroundControllerReplicas,
		commonseedresources.KyvernoCleanupControllerDeploymentName:    commonseedresources.KyvernoCleanupControllerReplicas,
		commonseedresources.KyvernoReportsControllerDeploymentName:    commonseedresources.KyvernoReportsControllerReplicas,
	}

	return wait.PollLog(ctx, logger, waitInterval, 10*time.Minute, func(ctx context.Context) (error, error) {
		for name, expectedReady := range deployments {
			deployment := &appsv1.Deployment{}
			if err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, deployment); err != nil {
				return fmt.Errorf("failed to get Deployment %s/%s: %w", namespace, name, err), nil
			}
			if deployment.Status.ObservedGeneration != deployment.Generation || deployment.Status.ReadyReplicas < expectedReady || deployment.Status.UpdatedReplicas < expectedReady {
				return fmt.Errorf("Deployment %s/%s is not fully rolled out: generation=%d observedGeneration=%d ready=%d updated=%d expected=%d", namespace, name, deployment.Generation, deployment.Status.ObservedGeneration, deployment.Status.ReadyReplicas, deployment.Status.UpdatedReplicas, expectedReady), nil
			}
		}
		return nil, nil
	})
}

func waitForSeedKyvernoControllersRemoved(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, namespace string) error {
	names := []string{
		commonseedresources.KyvernoAdmissionControllerDeploymentName,
		commonseedresources.KyvernoBackgroundControllerDeploymentName,
		commonseedresources.KyvernoCleanupControllerDeploymentName,
		commonseedresources.KyvernoReportsControllerDeploymentName,
	}

	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		for _, name := range names {
			err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &appsv1.Deployment{})
			if err == nil {
				return fmt.Errorf("Deployment %s/%s still exists", namespace, name), nil
			}
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to get Deployment %s/%s: %w", namespace, name, err), nil
			}
		}
		return nil, nil
	})
}

func waitForKyvernoUserClusterResources(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, namespace string) error {
	if err := waitForKyvernoCRDsEstablished(ctx, client, logger); err != nil {
		return err
	}

	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		if err := client.Get(ctx, types.NamespacedName{Name: namespace}, &corev1.Namespace{}); err != nil {
			return fmt.Errorf("Kyverno namespace is not ready: %w", err), nil
		}
		for _, name := range []string{commonseedresources.KyvernoConfigMapName, commonseedresources.KyvernoMetricsConfigMapName} {
			if err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &corev1.ConfigMap{}); err != nil {
				return fmt.Errorf("Kyverno ConfigMap %s/%s is not ready: %w", namespace, name, err), nil
			}
		}
		return nil, nil
	})
}

func waitForKyvernoCRDsEstablished(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger) error {
	crds, err := userclusterresources.KyvernoCRDs()
	if err != nil {
		return fmt.Errorf("failed to load expected Kyverno CRDs: %w", err)
	}

	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		for _, expected := range crds {
			crd := &apiextensionsv1.CustomResourceDefinition{}
			if err := client.Get(ctx, types.NamespacedName{Name: expected.Name}, crd); err != nil {
				return fmt.Errorf("Kyverno CRD %s is not available: %w", expected.Name, err), nil
			}
			established := false
			for _, condition := range crd.Status.Conditions {
				if condition.Type == apiextensionsv1.Established && condition.Status == apiextensionsv1.ConditionTrue {
					established = true
					break
				}
			}
			if !established {
				return fmt.Errorf("Kyverno CRD %s is not established", expected.Name), nil
			}
		}
		return nil, nil
	})
}

func waitForKyvernoCRDsRemoved(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger) error {
	crds, err := userclusterresources.KyvernoCRDs()
	if err != nil {
		return fmt.Errorf("failed to load expected Kyverno CRDs: %w", err)
	}

	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		for _, expected := range crds {
			err := client.Get(ctx, types.NamespacedName{Name: expected.Name}, &apiextensionsv1.CustomResourceDefinition{})
			if err == nil {
				return fmt.Errorf("Kyverno CRD %s still exists", expected.Name), nil
			}
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to get Kyverno CRD %s: %w", expected.Name, err), nil
			}
		}
		return nil, nil
	})
}

func waitForActivePolicyBinding(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, key types.NamespacedName, templateEnforced bool) error {
	return waitForPolicyBindingState(ctx, client, logger, key, true, metav1.ConditionTrue, kubermaticv1.PolicyBindingReasonReady, true, templateEnforced)
}

func waitForInactivePolicyBinding(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, key types.NamespacedName) error {
	return waitForPolicyBindingState(ctx, client, logger, key, false, metav1.ConditionFalse, kubermaticv1.PolicyBindingReasonKyvernoDisabled, false, false)
}

func waitForPolicyBindingState(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, key types.NamespacedName, active bool, conditionStatus metav1.ConditionStatus, reason string, expectFinalizer, templateEnforced bool) error {
	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		binding := &kubermaticv1.PolicyBinding{}
		if err := client.Get(ctx, key, binding); err != nil {
			return fmt.Errorf("failed to get PolicyBinding %s: %w", key, err), nil
		}

		stateError := func(format string, args ...any) error {
			conditions := make([]string, 0, len(binding.Status.Conditions))
			for _, condition := range binding.Status.Conditions {
				conditions = append(conditions, fmt.Sprintf("%s=%s reason=%s message=%q observedGeneration=%d", condition.Type, condition.Status, condition.Reason, condition.Message, condition.ObservedGeneration))
			}
			return fmt.Errorf("PolicyBinding %s %s; active=%s templateEnforced=%s observedGeneration=%d generation=%d conditions=[%s]", key, fmt.Sprintf(format, args...), formatOptionalBool(binding.Status.Active), formatOptionalBool(binding.Status.TemplateEnforced), binding.Status.ObservedGeneration, binding.Generation, strings.Join(conditions, "; "))
		}

		if binding.Status.Active == nil || *binding.Status.Active != active {
			return stateError("expected active=%t", active), nil
		}
		if binding.Status.ObservedGeneration != binding.Generation {
			return stateError("has not observed the current generation"), nil
		}

		for _, conditionType := range []kubermaticv1.PolicyBindingConditionType{
			kubermaticv1.PolicyBindingConditionKyvernoPolicyApplied,
			kubermaticv1.PolicyBindingConditionReady,
		} {
			expectedReason := reason
			if active && conditionType == kubermaticv1.PolicyBindingConditionKyvernoPolicyApplied {
				expectedReason = kubermaticv1.PolicyBindingReasonPolicyApplied
			}
			condition := meta.FindStatusCondition(binding.Status.Conditions, string(conditionType))
			if condition == nil || condition.Status != conditionStatus || condition.Reason != expectedReason || condition.ObservedGeneration != binding.Generation {
				return stateError("expected condition %s status=%s reason=%s generation=%d", conditionType, conditionStatus, expectedReason, binding.Generation), nil
			}
		}
		if active || reason == kubermaticv1.PolicyBindingReasonPolicyNamespaceMissing {
			templateCondition := meta.FindStatusCondition(binding.Status.Conditions, string(kubermaticv1.PolicyBindingConditionTemplateValid))
			if templateCondition == nil || templateCondition.Status != metav1.ConditionTrue || templateCondition.Reason != kubermaticv1.PolicyBindingReasonPolicyApplied || templateCondition.ObservedGeneration != binding.Generation {
				return stateError("expected condition %s status=True reason=%s generation=%d", kubermaticv1.PolicyBindingConditionTemplateValid, kubermaticv1.PolicyBindingReasonPolicyApplied, binding.Generation), nil
			}
			if binding.Status.TemplateEnforced == nil || *binding.Status.TemplateEnforced != templateEnforced {
				return stateError("expected templateEnforced=%t", templateEnforced), nil
			}
			if templateEnforced && binding.Annotations[kubermaticv1.AnnotationPolicyEnforced] != "true" {
				return stateError("was not marked as generated from an enforced template"), nil
			}
		}

		hasFinalizer := slices.Contains(binding.Finalizers, kubermaticv1.PolicyBindingCleanupFinalizer)
		if hasFinalizer != expectFinalizer {
			return stateError("cleanup finalizer present=%t, expected %t", hasFinalizer, expectFinalizer), nil
		}

		return nil, nil
	})
}

func formatOptionalBool(value *bool) string {
	if value == nil {
		return "unset"
	}
	return fmt.Sprintf("%t", *value)
}

func waitForClusterPolicyReady(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, name string) error {
	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		policy := &kyvernov1.ClusterPolicy{}
		if err := client.Get(ctx, types.NamespacedName{Name: name}, policy); err != nil {
			return fmt.Errorf("failed to get ClusterPolicy %s: %w", name, err), nil
		}
		if !policy.Status.IsReady() {
			return fmt.Errorf("ClusterPolicy %s is not ready: %#v", name, policy.Status.Conditions), nil
		}
		return nil, nil
	})
}

func waitForPolicyReady(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, namespace, name string) error {
	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		policy := &kyvernov1.Policy{}
		if err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, policy); err != nil {
			return fmt.Errorf("failed to get Policy %s/%s: %w", namespace, name, err), nil
		}
		if !policy.Status.IsReady() {
			return fmt.Errorf("Policy %s/%s is not ready: %#v", namespace, name, policy.Status.Conditions), nil
		}
		return nil, nil
	})
}

func verifyPolicyAdmission(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, namespace, policyName, labelValue string) error {
	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		allowed := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kyverno-e2e-allowed-" + rand.String(8),
				Namespace: namespace,
				Labels:    map[string]string{requiredLabelKey: labelValue},
			},
		}
		if err := client.Create(ctx, allowed, ctrlruntimeclient.DryRunAll); err != nil {
			return fmt.Errorf("policy %s rejected a ConfigMap with %s=%s: %w", policyName, requiredLabelKey, labelValue, err), nil
		}

		denied := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "kyverno-e2e-denied-" + rand.String(8), Namespace: namespace}}
		err := client.Create(ctx, denied, ctrlruntimeclient.DryRunAll)
		if err == nil {
			return fmt.Errorf("policy %s allowed a ConfigMap without the required label", policyName), nil
		}
		// Kyverno formats denials as YAML and may wrap the validation message.
		denial := strings.Join(strings.Fields(err.Error()), " ")
		if !strings.Contains(err.Error(), policyName) || !strings.Contains(denial, policyDenyMessage) {
			return fmt.Errorf("expected Kyverno denial from policy %q containing %q, got: %w", policyName, policyDenyMessage, err), nil
		}
		return nil, nil
	})
}

func waitForAdmissionAllowed(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, namespace string) error {
	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "kyverno-e2e-unrestricted-" + rand.String(8), Namespace: namespace}}
		if err := client.Create(ctx, configMap, ctrlruntimeclient.DryRunAll); err != nil {
			return fmt.Errorf("ConfigMap without the required label is still rejected in namespace %s: %w", namespace, err), nil
		}
		return nil, nil
	})
}

func waitForPolicyTemplateFinalizers(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, name string) error {
	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		template := &kubermaticv1.PolicyTemplate{}
		if err := client.Get(ctx, types.NamespacedName{Name: name}, template); err != nil {
			return fmt.Errorf("failed to get PolicyTemplate %s: %w", name, err), nil
		}
		for _, finalizer := range []string{
			kubermaticv1.PolicyTemplatePolicyBindingCleanupFinalizer,
			kubermaticv1.PolicyTemplateSeedCleanupFinalizer,
		} {
			if !slices.Contains(template.Finalizers, finalizer) {
				return fmt.Errorf("PolicyTemplate %s does not have finalizer %s", name, finalizer), nil
			}
		}
		return nil, nil
	})
}

func waitForClusterKyvernoFinalizer(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, clusterName string, present bool) error {
	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		cluster := &kubermaticv1.Cluster{}
		if err := client.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
			return fmt.Errorf("failed to get Cluster %s: %w", clusterName, err), nil
		}
		hasFinalizer := slices.Contains(cluster.Finalizers, kyvernocontroller.CleanupFinalizer)
		if hasFinalizer != present {
			return fmt.Errorf("Cluster %s Kyverno cleanup finalizer present=%t, expected %t", clusterName, hasFinalizer, present), nil
		}
		return nil, nil
	})
}

func waitForObjectDeleted(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, object ctrlruntimeclient.Object) error {
	key := ctrlruntimeclient.ObjectKeyFromObject(object)
	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		current, ok := object.DeepCopyObject().(ctrlruntimeclient.Object)
		if !ok {
			return nil, fmt.Errorf("object %T does not implement controller-runtime client.Object", object)
		}
		err := client.Get(ctx, key, current)
		if apierrors.IsNotFound(err) || meta.IsNoMatchError(err) {
			return nil, nil
		}
		if err != nil {
			return fmt.Errorf("failed to check whether %T %s was deleted: %w", object, key, err), nil
		}
		return fmt.Errorf("%T %s still exists", object, key), nil
	})
}
