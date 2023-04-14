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

package reconciling

import (
	"context"
	"errors"
	"fmt"

	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// helmReleaseAnnotation is the indicator for the ownership modifier to
	// not touch the object.
	helmReleaseAnnotation = "meta.helm.sh/release-name"
)

// NewOwnershipModifier is generating a new ObjectModifier that wraps an ObjectReconciler
// and takes care of applying the ownership and other labels for all managed objects.
func NewOwnershipModifier(owner metav1.Object, scheme *runtime.Scheme) reconciling.ObjectModifier {
	return func(create reconciling.ObjectReconciler) reconciling.ObjectReconciler {
		return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
			obj, err := create(existing)
			if err != nil {
				return obj, err
			}

			o, ok := obj.(metav1.Object)
			if !ok {
				return obj, nil
			}

			// Sometimes, the KKP operator needs to deal with objects that are owned by Helm
			// and then re-appropriated by KKP. This will however interfere with Helm's own
			// ownership concept. Also, reconciling resources owned by Helm will just lead to
			// increased resourceVersions, which might then trigger Deployments to be reconciled
			// due to the VolumeVersion annotations.
			// To prevent this, if an object is already owned by Helm, we never touch it.
			if _, exists := o.GetAnnotations()[helmReleaseAnnotation]; exists {
				return obj, nil
			}

			// try to set an owner reference; on shared resources this would fail to set
			// the second owner ref, we ignore this error and rely on the existing
			// KubermaticConfiguration ownership
			err = controllerutil.SetControllerReference(owner, o, scheme)
			if err != nil {
				var cerr *controllerutil.AlreadyOwnedError // do not use errors.Is() on this error type
				if !errors.As(err, &cerr) {
					return obj, fmt.Errorf("failed to set owner reference: %w", err)
				}
			}

			return obj, nil
		}
	}
}

// NewVolumeRevisionLabelsModifier scans volume mounts for pod templates for ConfigMaps
// and Secrets and will then put new labels for these mounts onto the pod template, causing
// restarts when the volumes changed.
func NewVolumeRevisionLabelsModifier(ctx context.Context, client ctrlruntimeclient.Client) reconciling.ObjectModifier {
	return func(create reconciling.ObjectReconciler) reconciling.ObjectReconciler {
		return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
			obj, err := create(existing)
			if err != nil {
				return obj, err
			}

			deployment, ok := obj.(*appsv1.Deployment)
			if !ok {
				return obj, nil
			}

			volumeLabels, err := volumeRevisionLabels(ctx, client, deployment.Namespace, deployment.Spec.Template.Spec.Volumes)
			if err != nil {
				return obj, fmt.Errorf("failed to determine revision labels for volumes: %w", err)
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

// secretRevision returns the resource version of the Secret specified by name.
func secretRevision(ctx context.Context, key types.NamespacedName, client ctrlruntimeclient.Client) (string, error) {
	secret := &corev1.Secret{}
	if err := client.Get(ctx, key, secret); err != nil {
		return "", fmt.Errorf("could not get Secret %s: %w", key, err)
	}
	return secret.ResourceVersion, nil
}

// configMapRevision returns the resource version of the ConfigMap specified by name.
func configMapRevision(ctx context.Context, key types.NamespacedName, client ctrlruntimeclient.Client) (string, error) {
	cm := &corev1.ConfigMap{}
	if err := client.Get(ctx, key, cm); err != nil {
		return "", fmt.Errorf("could not get ConfigMap %s: %w", key, err)
	}
	return cm.ResourceVersion, nil
}

// volumeRevisionLabels returns a set of labels for the given volumes, with one label per
// ConfigMap or Secret, containing the objects' revisions.
// When used for pod template labels, this will force pods being restarted as soon as one
// of the secrets/configmaps get updated.
func volumeRevisionLabels(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	namespace string,
	volumes []corev1.Volume,
) (map[string]string, error) {
	labels := make(map[string]string)

	for _, v := range volumes {
		if v.VolumeSource.Secret != nil {
			key := types.NamespacedName{Namespace: namespace, Name: v.VolumeSource.Secret.SecretName}
			revision, err := secretRevision(ctx, key, client)
			if err != nil {
				return nil, err
			}
			labels[fmt.Sprintf("%s-secret-revision", v.VolumeSource.Secret.SecretName)] = revision
		}
		if v.VolumeSource.ConfigMap != nil {
			key := types.NamespacedName{Namespace: namespace, Name: v.VolumeSource.ConfigMap.Name}
			revision, err := configMapRevision(ctx, key, client)
			if err != nil {
				return nil, err
			}
			labels[fmt.Sprintf("%s-configmap-revision", v.VolumeSource.ConfigMap.Name)] = revision
		}
	}

	return labels, nil
}
