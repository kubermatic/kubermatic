/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package common

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling/modifier"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// OperatorName is used as the value for ManagedBy labels to establish
	// a weak ownership to reconciled resources.
	OperatorName = "kubermatic-operator"
)

var (
	// ManagedByOperatorPredicate is a predicate that matches all resources created by
	// the Kubermatic Operator, based on the ManagedBy label.
	ManagedByOperatorPredicate = predicate.Factory(func(o ctrlruntimeclient.Object) bool {
		for _, ref := range o.GetOwnerReferences() {
			if isKubermaticConfiguration(ref) || isSeed(ref) {
				return true
			}
		}

		return false
	})

	// ManagedByOperatorSelector is a label selector that matches all resources created by
	// the Kubermatic Operator.
	ManagedByOperatorSelector, _ = labels.NewRequirement(modifier.ManagedByLabel, selection.Equals, []string{OperatorName})
)

func isKubermaticConfiguration(ref metav1.OwnerReference) bool {
	return ref.APIVersion == kubermaticv1.SchemeGroupVersion.String() && ref.Kind == "KubermaticConfiguration"
}

func isSeed(ref metav1.OwnerReference) bool {
	return ref.APIVersion == kubermaticv1.SchemeGroupVersion.String() && ref.Kind == "Seed"
}

// StringifyFeatureGates takes a set of enabled features and returns a comma-separated
// key=value list like "featureA=true,featureB=true,...". The list of feature gates is
// sorted, so the output of this function is stable.
func StringifyFeatureGates(cfg *kubermaticv1.KubermaticConfiguration) string {
	// use a set to ensure that the result is sorted, otherwise reconciling code that
	// uses this will end up in endless loops
	features := sets.New[string]()
	for feature, enabled := range cfg.Spec.FeatureGates {
		features.Insert(fmt.Sprintf("%s=%v", feature, enabled))
	}

	return strings.Join(sets.List(features), ",")
}

func createSecretData(s *corev1.Secret, data map[string]string) *corev1.Secret {
	if s.Data == nil {
		s.Data = make(map[string][]byte)
	}

	for k, v := range data {
		s.Data[k] = []byte(v)
	}

	return s
}

// CleanupClusterResource attempts to find a cluster-wide resource and
// deletes it if it was found. If no resource with the given name exists,
// nil is returned.
func CleanupClusterResource(ctx context.Context, client ctrlruntimeclient.Client, obj ctrlruntimeclient.Object, name string) error {
	key := types.NamespacedName{Name: name}

	if err := client.Get(ctx, key, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to probe for %s: %w", key, err)
		}

		return nil
	}

	if err := client.Delete(ctx, obj); err != nil {
		return fmt.Errorf("failed to delete %s: %w", key, err)
	}

	return nil
}

// KubermaticProxyEnvironmentVars returns ProxySettings from Kubermatic configuration as env vars.
func KubermaticProxyEnvironmentVars(p *kubermaticv1.KubermaticProxyConfiguration) (result []corev1.EnvVar) {
	if p == nil || (p.HTTP == "" && p.HTTPS == "") {
		return
	}

	if p.HTTP != "" {
		result = append(result, corev1.EnvVar{
			Name:  "HTTP_PROXY",
			Value: p.HTTP,
		})
	}

	if p.HTTPS != "" {
		result = append(result, corev1.EnvVar{
			Name:  "HTTPS_PROXY",
			Value: p.HTTPS,
		})
	}

	noProxy := []string{
		defaulting.DefaultNoProxy,
	}

	if p.NoProxy != "" {
		noProxy = append(noProxy, p.NoProxy)
	}

	result = append(result, corev1.EnvVar{
		Name:  "NO_PROXY",
		Value: strings.Join(noProxy, ","),
	})

	return
}

// SeedProxyEnvironmentVars returns ProxySettings from Seed as env vars.
func SeedProxyEnvironmentVars(p *kubermaticv1.ProxySettings) (result []corev1.EnvVar) {
	if p == nil || p.Empty() {
		return
	}

	result = append(result, corev1.EnvVar{
		Name:  "HTTP_PROXY",
		Value: p.HTTPProxy.String(),
	})

	result = append(result, corev1.EnvVar{
		Name:  "HTTPS_PROXY",
		Value: p.HTTPProxy.String(),
	})

	noProxy := []string{
		defaulting.DefaultNoProxy,
	}

	if p.NoProxy.String() != "" {
		noProxy = append(noProxy, p.NoProxy.String())
	}

	result = append(result, corev1.EnvVar{
		Name:  "NO_PROXY",
		Value: strings.Join(noProxy, ","),
	})

	return
}

func DeleteObject(ctx context.Context, client ctrlruntimeclient.Client, name, namespace string, obj ctrlruntimeclient.Object) error {
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	if err := client.Get(ctx, key, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("failed to get object %v: %w", key, err)
	}

	return client.Delete(ctx, obj)
}

func DeleteService(ctx context.Context, client ctrlruntimeclient.Client, name, namespace string) error {
	return DeleteObject(ctx, client, name, namespace, &corev1.Service{})
}

// CleanupWebhookServices removes the unused webhook services. It's here because
// we need to exact same logic on master and seed clusters.
func CleanupWebhookServices(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, namespace string) {
	for _, name := range []string{SeedWebhookServiceName, ClusterWebhookServiceName} {
		if err := DeleteService(ctx, client, name, namespace); err != nil {
			logger.Warnw("Failed to delete unused Service", zap.Error(err))
		}
	}
}
