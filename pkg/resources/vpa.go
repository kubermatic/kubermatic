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

package resources

import (
	"context"
	"fmt"

	"k8c.io/kubermatic/v3/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func getVPAReconcilerForPodTemplate(name string, pod corev1.PodSpec, controllerRef metav1.OwnerReference, enabled bool) reconciling.NamedVerticalPodAutoscalerReconcilerFactory {
	var containerPolicies []vpav1.ContainerResourcePolicy
	for _, container := range pod.Containers {
		containerPolicies = append(containerPolicies, vpav1.ContainerResourcePolicy{
			ContainerName: container.Name,
			MaxAllowed:    container.Resources.Limits,
			MinAllowed:    container.Resources.Requests,
		})
	}

	updateMode := vpav1.UpdateModeAuto
	if !enabled {
		updateMode = vpav1.UpdateModeOff
	}

	return func() (string, reconciling.VerticalPodAutoscalerReconciler) {
		return name, func(vpa *vpav1.VerticalPodAutoscaler) (*vpav1.VerticalPodAutoscaler, error) {
			// We're doing this as we don't want to use the Cluster object as owner.
			// Instead we're using the actual target as owner - this way the VPA gets deleted when the Deployment/StatefulSet gets deleted as well
			vpa.OwnerReferences = []metav1.OwnerReference{controllerRef}
			vpa.Spec = vpav1.VerticalPodAutoscalerSpec{
				TargetRef: &autoscalingv1.CrossVersionObjectReference{
					Name:       controllerRef.Name,
					Kind:       controllerRef.Kind,
					APIVersion: controllerRef.APIVersion,
				},
				UpdatePolicy: &vpav1.PodUpdatePolicy{
					UpdateMode: &updateMode,
				},
				ResourcePolicy: &vpav1.PodResourcePolicy{
					ContainerPolicies: containerPolicies,
				},
			}
			return vpa, nil
		}
	}
}

// GetVerticalPodAutoscalersForAll will return functions to create VPA resource for all supplied Deployments or StatefulSets.
// If creator functions for VPA's for Deployments should be returned, a deployment store must be passed in. Otherwise a StatefulSet store.
// All resources must exist in the specified namespace.
// The VPA resource will have the same selector as the Deployment/StatefulSet. The pod container limits will be set as VPA limits.
func getVerticalPodAutoscalersForResource(ctx context.Context, client ctrlruntimeclient.Client, names []string, namespace string, obj ctrlruntimeclient.Object, enabled bool) ([]reconciling.NamedVerticalPodAutoscalerReconcilerFactory, error) {
	var creators []reconciling.NamedVerticalPodAutoscalerReconcilerFactory
	for _, name := range names {
		name := name
		key := types.NamespacedName{Namespace: namespace, Name: name}

		err := client.Get(ctx, key, obj)
		if err != nil {
			return nil, fmt.Errorf("failed to get object '%s' from store: %w", key, err)
		}

		gv := appsv1.SchemeGroupVersion
		switch obj := obj.(type) {
		case *appsv1.Deployment:
			creators = append(creators, getVPAReconcilerForPodTemplate(
				obj.Name,
				obj.Spec.Template.Spec,
				*metav1.NewControllerRef(obj, gv.WithKind("Deployment")),
				enabled),
			)
		case *appsv1.StatefulSet:
			creators = append(creators, getVPAReconcilerForPodTemplate(
				obj.Name,
				obj.Spec.Template.Spec,
				*metav1.NewControllerRef(obj, gv.WithKind("StatefulSet")),
				enabled),
			)
		default:
			return nil, fmt.Errorf("object '%s' from store is %T instead of a expected *appsv1.Deployment or *appsv1.StatefulSet", key, obj)
		}
	}

	return creators, nil
}

// GetVerticalPodAutoscalersForAll will return functions to create VPA resource for all supplied Deployments and StatefulSets.
// All resources must exist in the specified namespace.
// The VPA resource will have the same selector as the Deployment/StatefulSet. The pod container limits will be set as VPA limits.
func GetVerticalPodAutoscalersForAll(ctx context.Context, client ctrlruntimeclient.Client, deploymentNames, statefulSetNames []string, namespace string, enabled bool) ([]reconciling.NamedVerticalPodAutoscalerReconcilerFactory, error) {
	deploymentVPAReconcilers, err := getVerticalPodAutoscalersForResource(ctx, client, deploymentNames, namespace, &appsv1.Deployment{}, enabled)
	if err != nil {
		return nil, fmt.Errorf("failed to create VPA creator functions for Deployments: %w", err)
	}

	statefulSetVPAReconcilers, err := getVerticalPodAutoscalersForResource(ctx, client, statefulSetNames, namespace, &appsv1.StatefulSet{}, enabled)
	if err != nil {
		return nil, fmt.Errorf("failed to create VPA creator functions for StatefulSets: %w", err)
	}

	return append(deploymentVPAReconcilers, statefulSetVPAReconcilers...), nil
}
