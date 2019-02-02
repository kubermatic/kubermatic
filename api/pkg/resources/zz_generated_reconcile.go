// This file is generated. DO NOT EDIT.
package resources

import (
	"fmt"

	informerutil "github.com/kubermatic/kubermatic/api/pkg/util/informer"

	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimecache "sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	autoscalingv1beta1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta1"
)

// ServiceObjectWrapper adds a wrapper so the ServiceCreator matches ObjectCreator
// This is needed as golang does not support function interface matching
func ServiceObjectWrapper(create ServiceCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*corev1.Service))
		}
		return create(&corev1.Service{})
	}
}

// ReconcileServices will create and update the Services coming from the passed ServiceCreator slice
func ReconcileServices(creators []ServiceCreator, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, wrapper ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &corev1.Service{})
	if err != nil {
		return fmt.Errorf("failed to get Service informer: %v", err)
	}

	for _, create := range creators {
		createObject := ServiceObjectWrapper(create)
		for _, wrap := range wrapper {
			createObject = wrap(createObject)
		}

		if err := EnsureObject(namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure Service: %v", err)
		}
	}

	return nil
}

// StatefulSetObjectWrapper adds a wrapper so the StatefulSetCreator matches ObjectCreator
// This is needed as golang does not support function interface matching
func StatefulSetObjectWrapper(create StatefulSetCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*appsv1.StatefulSet))
		}
		return create(&appsv1.StatefulSet{})
	}
}

// ReconcileStatefulSets will create and update the StatefulSets coming from the passed StatefulSetCreator slice
func ReconcileStatefulSets(creators []StatefulSetCreator, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, wrapper ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &appsv1.StatefulSet{})
	if err != nil {
		return fmt.Errorf("failed to get StatefulSet informer: %v", err)
	}

	for _, create := range creators {
		createObject := StatefulSetObjectWrapper(create)
		for _, wrap := range wrapper {
			createObject = wrap(createObject)
		}

		if err := EnsureObject(namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure StatefulSet: %v", err)
		}
	}

	return nil
}

// VerticalPodAutoscalerObjectWrapper adds a wrapper so the VerticalPodAutoscalerCreator matches ObjectCreator
// This is needed as golang does not support function interface matching
func VerticalPodAutoscalerObjectWrapper(create VerticalPodAutoscalerCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*autoscalingv1beta1.VerticalPodAutoscaler))
		}
		return create(&autoscalingv1beta1.VerticalPodAutoscaler{})
	}
}

// ReconcileVerticalPodAutoscalers will create and update the VerticalPodAutoscalers coming from the passed VerticalPodAutoscalerCreator slice
func ReconcileVerticalPodAutoscalers(creators []VerticalPodAutoscalerCreator, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, wrapper ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &autoscalingv1beta1.VerticalPodAutoscaler{})
	if err != nil {
		return fmt.Errorf("failed to get VerticalPodAutoscaler informer: %v", err)
	}

	for _, create := range creators {
		createObject := VerticalPodAutoscalerObjectWrapper(create)
		for _, wrap := range wrapper {
			createObject = wrap(createObject)
		}

		if err := EnsureObject(namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure VerticalPodAutoscaler: %v", err)
		}
	}

	return nil
}
