package resources

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/util/informer"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"

	ctrlruntimecache "sigs.k8s.io/controller-runtime/pkg/cache"
)

func getVPACreatorForPodTemplate(name, namespace string, pod corev1.PodSpec, controllerRef metav1.OwnerReference) VerticalPodAutoscalerCreator {
	var containerPolicies []autoscalingv1beta2.ContainerResourcePolicy
	for _, container := range pod.Containers {
		containerPolicies = append(containerPolicies, autoscalingv1beta2.ContainerResourcePolicy{
			ContainerName: container.Name,
			MaxAllowed:    container.Resources.Limits,
			MinAllowed:    container.Resources.Requests,
		})
	}

	return func(existing *autoscalingv1beta2.VerticalPodAutoscaler) (*autoscalingv1beta2.VerticalPodAutoscaler, error) {
		var pdb *autoscalingv1beta2.VerticalPodAutoscaler
		if existing != nil {
			pdb = existing
		} else {
			pdb = &autoscalingv1beta2.VerticalPodAutoscaler{}
		}

		pdb.Name = name
		pdb.Namespace = namespace
		pdb.OwnerReferences = []metav1.OwnerReference{controllerRef}

		updateMode := autoscalingv1beta2.UpdateModeAuto
		pdb.Spec = autoscalingv1beta2.VerticalPodAutoscalerSpec{
			TargetRef: &autoscalingv1.CrossVersionObjectReference{
				Name:       controllerRef.Name,
				Kind:       controllerRef.Kind,
				APIVersion: controllerRef.APIVersion,
			},
			UpdatePolicy: &autoscalingv1beta2.PodUpdatePolicy{
				UpdateMode: &updateMode,
			},
			ResourcePolicy: &autoscalingv1beta2.PodResourcePolicy{
				ContainerPolicies: containerPolicies,
			},
		}

		return pdb, nil
	}
}

// GetVerticalPodAutoscalersForAll will return functions to create VPA resource for all supplied Deployments or StatefulSets.
// If creator functions for VPA's for Deployments should be returned, a deployment store must be passed in. Otherwise a StatefulSet store.
// All resources must exist in the specified namespace.
// The VPA resource will have the same selector as the Deployment/StatefulSet. The pod container limits will be set as VPA limits.
func getVerticalPodAutoscalersForResource(names []string, namespace string, store cache.Store) ([]VerticalPodAutoscalerCreator, error) {
	var creators []VerticalPodAutoscalerCreator
	for _, name := range names {
		name := name
		key := fmt.Sprintf("%s/%s", namespace, name)
		obj, exists, err := store.GetByKey(key)
		if err != nil {
			return nil, fmt.Errorf("failed to get object '%s' from store: %v", key, err)
		}
		if !exists {
			return nil, fmt.Errorf("object '%s' does not exist in the store", key)
		}

		gv := appsv1.SchemeGroupVersion
		switch obj := obj.(type) {
		case *appsv1.Deployment:
			creators = append(creators, getVPACreatorForPodTemplate(
				obj.Name,
				obj.Namespace,
				obj.Spec.Template.Spec,
				*metav1.NewControllerRef(obj, gv.WithKind("Deployment"))),
			)
		case *appsv1.StatefulSet:
			creators = append(creators, getVPACreatorForPodTemplate(
				obj.Name,
				obj.Namespace,
				obj.Spec.Template.Spec,
				*metav1.NewControllerRef(obj, gv.WithKind("StatefulSet"))),
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
func GetVerticalPodAutoscalersForAll(deploymentNames, statefulSetNames []string, namespace string, dynamicCache ctrlruntimecache.Cache) ([]VerticalPodAutoscalerCreator, error) {
	deploymentStore, err := informer.GetSyncedStoreFromDynamicFactory(dynamicCache, &appsv1.Deployment{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Deployment store: %v", err)
	}

	deploymentVPACreators, err := getVerticalPodAutoscalersForResource(deploymentNames, namespace, deploymentStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create VPA creator functions for Deployments: %v", err)
	}

	statefulSetStore, err := informer.GetSyncedStoreFromDynamicFactory(dynamicCache, &appsv1.StatefulSet{})
	if err != nil {
		return nil, fmt.Errorf("failed to get StatefulSet store: %v", err)
	}

	statefulSetVPACreators, err := getVerticalPodAutoscalersForResource(statefulSetNames, namespace, statefulSetStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create VPA creator functions for StatefulSets: %v", err)
	}

	return append(deploymentVPACreators, statefulSetVPACreators...), nil
}
