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

	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// OperatorName is used as the value for ManagedBy labels to establish
	// a weak ownership to reconciled resources.
	OperatorName = "kubermatic-operator"

	// ManagedByLabel is the label used to identify the resources
	// created by this controller.
	ManagedByLabel = "app.kubernetes.io/managed-by"
)

var (
	// ManagedByOperatorPredicate is a predicate that matches all resources created by
	// the Kubermatic Operator, based on the ManagedBy label.
	ManagedByOperatorPredicate = predicate.Factory(func(o client.Object) bool {
		for _, ref := range o.GetOwnerReferences() {
			if isKubermaticConfiguration(ref) || isSeed(ref) {
				return true
			}
		}

		return false
	})

	// ManagedByOperatorSelector is a label selector that matches all resources created by
	// the Kubermatic Operator.
	ManagedByOperatorSelector, _ = labels.NewRequirement(ManagedByLabel, selection.Equals, []string{OperatorName})
)

func isKubermaticConfiguration(ref metav1.OwnerReference) bool {
	return ref.APIVersion == operatorv1alpha1.SchemeGroupVersion.String() && ref.Kind == "KubermaticConfiguration"
}

func isSeed(ref metav1.OwnerReference) bool {
	return ref.APIVersion == kubermaticv1.SchemeGroupVersion.String() && ref.Kind == "Seed"
}

// StringifyFeatureGates takes a set of enabled features and returns a comma-separated
// key=value list like "featureA=true,featureB=true,...".
func StringifyFeatureGates(cfg *operatorv1alpha1.KubermaticConfiguration) string {
	features := make([]string, 0)
	for _, feature := range cfg.Spec.FeatureGates.List() {
		features = append(features, fmt.Sprintf("%s=true", feature))
	}

	return strings.Join(features, ",")
}

// OwnershipModifierFactory is generating a new ObjectModifier that wraps an ObjectCreator
// and takes care of applying the ownership and other labels for all managed objects.
func OwnershipModifierFactory(owner metav1.Object, scheme *runtime.Scheme) reconciling.ObjectModifier {
	return func(create reconciling.ObjectCreator) reconciling.ObjectCreator {
		return func(existing runtime.Object) (runtime.Object, error) {
			obj, err := create(existing)
			if err != nil {
				return obj, err
			}

			o, ok := obj.(metav1.Object)
			if !ok {
				return obj, nil
			}

			// try to set an owner reference; on shared resources this would fail to set
			// the second owner ref, we ignore this error and rely on the existing
			// KubermaticConfiguration ownership
			err = controllerutil.SetControllerReference(owner, o, scheme)
			if err != nil {
				if _, ok := err.(*controllerutil.AlreadyOwnedError); !ok {
					return obj, fmt.Errorf("failed to set owner reference: %v", err)
				}
			}

			labels := o.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels[ManagedByLabel] = OperatorName
			o.SetLabels(labels)

			return obj, nil
		}
	}
}

// VolumeRevisionLabelsModifierFactory scans volume mounts for pod templates for ConfigMaps
// and Secrets and will then put new labels for these mounts onto the pod template, causing
// restarts when the volumes changed.
func VolumeRevisionLabelsModifierFactory(ctx context.Context, client ctrlruntimeclient.Client) reconciling.ObjectModifier {
	return func(create reconciling.ObjectCreator) reconciling.ObjectCreator {
		return func(existing runtime.Object) (runtime.Object, error) {
			obj, err := create(existing)
			if err != nil {
				return obj, err
			}

			deployment, ok := obj.(*appsv1.Deployment)
			if !ok {
				return obj, nil
			}

			volumeLabels, err := resources.VolumeRevisionLabels(ctx, client, deployment.Namespace, deployment.Spec.Template.Spec.Volumes)
			if err != nil {
				return obj, fmt.Errorf("failed to determine revision labels for volumes: %v", err)
			}

			// switch to a new map in case the deployment used the same map for selector.matchLabels and labels
			oldLabels := deployment.Spec.Template.Labels
			deployment.Spec.Template.Labels = volumeLabels

			for k, v := range oldLabels {
				deployment.Spec.Template.Labels[k] = v
			}

			return obj, nil
		}
	}
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
func CleanupClusterResource(client ctrlruntimeclient.Client, obj runtime.Object, name string) error {
	key := types.NamespacedName{Name: name}
	ctx := context.Background()

	if err := client.Get(ctx, key, obj); err != nil {
		if !kerrors.IsNotFound(err) {
			return fmt.Errorf("failed to probe for %s: %v", key, err)
		}

		return nil
	}

	if err := client.Delete(ctx, obj); err != nil {
		return fmt.Errorf("failed to delete %s: %v", key, err)
	}

	return nil
}

func ProxyEnvironmentVars(cfg *operatorv1alpha1.KubermaticConfiguration) []corev1.EnvVar {
	result := []corev1.EnvVar{}
	settings := cfg.Spec.Proxy

	if settings.HTTP == "" && settings.HTTPS == "" {
		return result
	}

	if settings.HTTP != "" {
		result = append(result, corev1.EnvVar{
			Name:  "HTTP_PROXY",
			Value: settings.HTTP,
		})
	}

	if settings.HTTPS != "" {
		result = append(result, corev1.EnvVar{
			Name:  "HTTPS_PROXY",
			Value: settings.HTTPS,
		})
	}

	noProxy := []string{
		DefaultNoProxy,
	}

	if settings.NoProxy != "" {
		noProxy = append(noProxy, settings.NoProxy)
	}

	result = append(result, corev1.EnvVar{
		Name:  "NO_PROXY",
		Value: strings.Join(noProxy, ","),
	})

	return result
}
