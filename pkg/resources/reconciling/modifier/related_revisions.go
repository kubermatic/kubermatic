/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package modifier

import (
	"context"
	"fmt"

	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// RelatedRevisionsLabels scans volume mounts for pod templates for ConfigMaps
// and Secrets and will then put new labels for these mounts onto the pod template, causing
// restarts when the volumes changed.
func RelatedRevisionsLabels(ctx context.Context, client ctrlruntimeclient.Client) reconciling.ObjectModifier {
	return func(reconciler reconciling.ObjectReconciler) reconciling.ObjectReconciler {
		return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
			obj, err := reconciler(existing)
			if err != nil {
				return obj, err
			}

			return addRevisionLabelsToObject(ctx, client, obj)
		}
	}
}

func addRevisionLabelsToObject(ctx context.Context, client ctrlruntimeclient.Client, obj ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
	switch asserted := obj.(type) {
	case *appsv1.Deployment:
		if err := addRevisionLabelsToPodSpec(ctx, client, asserted.Namespace, &asserted.Spec.Template); err != nil {
			return nil, fmt.Errorf("failed to determine revision labels: %w", err)
		}

	case *appsv1.StatefulSet:
		if err := addRevisionLabelsToPodSpec(ctx, client, asserted.Namespace, &asserted.Spec.Template); err != nil {
			return nil, fmt.Errorf("failed to determine revision labels: %w", err)
		}

	case *appsv1.DaemonSet:
		if err := addRevisionLabelsToPodSpec(ctx, client, asserted.Namespace, &asserted.Spec.Template); err != nil {
			return nil, fmt.Errorf("failed to determine revision labels: %w", err)
		}

	default:
		panic(fmt.Sprintf("RelatedRevisionsLabels modifier used on incompatible type %T", obj))
	}

	return obj, nil
}

func addRevisionLabelsToPodSpec(ctx context.Context, client ctrlruntimeclient.Client, namespace string, ps *corev1.PodTemplateSpec) error {
	allContainers := []corev1.Container{}
	allContainers = append(allContainers, ps.Spec.InitContainers...)
	allContainers = append(allContainers, ps.Spec.Containers...)

	revisionLabels, err := getRelatedRevisionLabels(ctx, client, namespace, ps.Spec.Volumes, allContainers)
	if err != nil {
		return err
	}

	kubernetes.EnsureLabels(ps, revisionLabels)

	return nil
}

// getRelatedRevisionLabels returns a set of labels for the given volumes/containers,
// with one label per ConfigMap or Secret, containing the objects' revisions.
// When used for pod template labels, this will force pods being restarted as soon as one
// of the secrets/configmaps get updated.
func getRelatedRevisionLabels(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	namespace string,
	volumes []corev1.Volume,
	containers []corev1.Container,
) (map[string]string, error) {
	labels := make(map[string]string)

	loadSecret := func(name string, optional bool) error {
		labelName := fmt.Sprintf("%s-secret-revision", name)

		// skip checking the same secret again
		if _, ok := labels[labelName]; ok {
			return nil
		}

		key := types.NamespacedName{Namespace: namespace, Name: name}
		revision, err := secretRevision(ctx, client, key)
		if err != nil {
			if optional && apierrors.IsNotFound(err) {
				return nil
			}

			return err
		}

		labels[labelName] = revision

		return nil
	}

	loadConfigMap := func(name string, optional bool) error {
		labelName := fmt.Sprintf("%s-configmap-revision", name)

		// skip checking the same ConfigMap again
		if _, ok := labels[labelName]; ok {
			return nil
		}

		key := types.NamespacedName{Namespace: namespace, Name: name}
		revision, err := configMapRevision(ctx, client, key)
		if err != nil {
			if optional && apierrors.IsNotFound(err) {
				return nil
			}

			return err
		}

		labels[labelName] = revision

		return nil
	}

	for _, v := range volumes {
		if v.Secret != nil {
			if err := loadSecret(v.Secret.SecretName, false); err != nil {
				return nil, err
			}
		}

		if v.ConfigMap != nil {
			if err := loadConfigMap(v.ConfigMap.Name, false); err != nil {
				return nil, err
			}
		}
	}

	for _, c := range containers {
		for _, e := range c.Env {
			if source := e.ValueFrom; source != nil {
				if cm := source.ConfigMapKeyRef; cm != nil {
					if err := loadConfigMap(cm.Name, cm.Optional != nil && *cm.Optional); err != nil {
						return nil, err
					}
				}

				if sec := source.SecretKeyRef; sec != nil {
					if err := loadSecret(sec.Name, sec.Optional != nil && *sec.Optional); err != nil {
						return nil, err
					}
				}
			}
		}

		for _, e := range c.EnvFrom {
			if cm := e.ConfigMapRef; cm != nil {
				if err := loadConfigMap(cm.Name, cm.Optional != nil && *cm.Optional); err != nil {
					return nil, err
				}
			}

			if sec := e.SecretRef; sec != nil {
				if err := loadSecret(sec.Name, sec.Optional != nil && *sec.Optional); err != nil {
					return nil, err
				}
			}
		}
	}

	return labels, nil
}

// secretRevision returns the resource version of the Secret specified by name.
func secretRevision(ctx context.Context, client ctrlruntimeclient.Client, key types.NamespacedName) (string, error) {
	secret := &corev1.Secret{}
	err := client.Get(ctx, key, secret)

	return secret.ResourceVersion, err
}

// configMapRevision returns the resource version of the ConfigMap specified by name.
func configMapRevision(ctx context.Context, client ctrlruntimeclient.Client, key types.NamespacedName) (string, error) {
	cm := &corev1.ConfigMap{}
	err := client.Get(ctx, key, cm)

	return cm.ResourceVersion, err
}
