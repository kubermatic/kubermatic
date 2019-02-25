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

// ServiceCreator defines an interface to create/update Services
type ServiceCreator = func(existing *corev1.Service) (*corev1.Service, error)

// ServiceObjectWrapper adds a wrapper so the ServiceCreator matches ObjectCreator
// This is needed as golang does not support function interface matching
func serviceObjectWrapper(create ServiceCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*corev1.Service))
		}
		return create(&corev1.Service{})
	}
}

// ReconcileServices will create and update the Services coming from the passed ServiceCreator slice
func ReconcileServices(creators []ServiceCreator, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, objectModifiers ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &corev1.Service{})
	if err != nil {
		return fmt.Errorf("failed to get Service informer: %v", err)
	}

	for _, create := range creators {
		createObject := serviceObjectWrapper(create)
		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureObject(namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure Service: %v", err)
		}
	}

	return nil
}

// SecretCreator defines an interface to create/update Secrets
type SecretCreator = func(existing *corev1.Secret) (*corev1.Secret, error)

// NamedSecretCreatorGetter returns the name of the resource and the corresponding creator function
type NamedSecretCreatorGetter = func() (name string, create SecretCreator)

// secretObjectWrapper adds a wrapper so the SecretCreator matches ObjectCreator
// This is needed as golang does not support function interface matching
func SecretObjectWrapper(create SecretCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*corev1.Secret))
		}
		return create(&corev1.Secret{})
	}
}

// ReconcileSecrets will create and update the Secrets coming from the passed SecretCreator slice
func ReconcileSecrets(namedGetters []NamedSecretCreatorGetter, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, objectModifiers ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &corev1.Secret{})
	if err != nil {
		return fmt.Errorf("failed to get Secret informer: %v", err)
	}

	for _, get := range namedGetters {
		name, create := get()
		createObject := SecretObjectWrapper(create)
		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(name, namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure Secret: %v", err)
		}
	}

	return nil
}

// ConfigMapCreator defines an interface to create/update ConfigMaps
type ConfigMapCreator = func(existing *corev1.ConfigMap) (*corev1.ConfigMap, error)

// NamedConfigMapCreatorGetter returns the name of the resource and the corresponding creator function
type NamedConfigMapCreatorGetter = func() (name string, create ConfigMapCreator)

// configMapObjectWrapper adds a wrapper so the ConfigMapCreator matches ObjectCreator
// This is needed as golang does not support function interface matching
func ConfigMapObjectWrapper(create ConfigMapCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*corev1.ConfigMap))
		}
		return create(&corev1.ConfigMap{})
	}
}

// ReconcileConfigMaps will create and update the ConfigMaps coming from the passed ConfigMapCreator slice
func ReconcileConfigMaps(namedGetters []NamedConfigMapCreatorGetter, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, objectModifiers ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &corev1.ConfigMap{})
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap informer: %v", err)
	}

	for _, get := range namedGetters {
		name, create := get()
		createObject := ConfigMapObjectWrapper(create)
		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(name, namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure ConfigMap: %v", err)
		}
	}

	return nil
}

// StatefulSetCreator defines an interface to create/update StatefulSets
type StatefulSetCreator = func(existing *appsv1.StatefulSet) (*appsv1.StatefulSet, error)

// StatefulSetObjectWrapper adds a wrapper so the StatefulSetCreator matches ObjectCreator
// This is needed as golang does not support function interface matching
func statefulSetObjectWrapper(create StatefulSetCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*appsv1.StatefulSet))
		}
		return create(&appsv1.StatefulSet{})
	}
}

// ReconcileStatefulSets will create and update the StatefulSets coming from the passed StatefulSetCreator slice
func ReconcileStatefulSets(creators []StatefulSetCreator, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, objectModifiers ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &appsv1.StatefulSet{})
	if err != nil {
		return fmt.Errorf("failed to get StatefulSet informer: %v", err)
	}

	for _, create := range creators {
		createObject := statefulSetObjectWrapper(create)
		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureObject(namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure StatefulSet: %v", err)
		}
	}

	return nil
}

// DeploymentCreator defines an interface to create/update Deployments
type DeploymentCreator = func(existing *appsv1.Deployment) (*appsv1.Deployment, error)

// DeploymentObjectWrapper adds a wrapper so the DeploymentCreator matches ObjectCreator
// This is needed as golang does not support function interface matching
func deploymentObjectWrapper(create DeploymentCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*appsv1.Deployment))
		}
		return create(&appsv1.Deployment{})
	}
}

// ReconcileDeployments will create and update the Deployments coming from the passed DeploymentCreator slice
func ReconcileDeployments(creators []DeploymentCreator, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, objectModifiers ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &appsv1.Deployment{})
	if err != nil {
		return fmt.Errorf("failed to get Deployment informer: %v", err)
	}

	for _, create := range creators {
		createObject := deploymentObjectWrapper(create)
		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureObject(namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure Deployment: %v", err)
		}
	}

	return nil
}

// VerticalPodAutoscalerCreator defines an interface to create/update VerticalPodAutoscalers
type VerticalPodAutoscalerCreator = func(existing *autoscalingv1beta1.VerticalPodAutoscaler) (*autoscalingv1beta1.VerticalPodAutoscaler, error)

// VerticalPodAutoscalerObjectWrapper adds a wrapper so the VerticalPodAutoscalerCreator matches ObjectCreator
// This is needed as golang does not support function interface matching
func verticalPodAutoscalerObjectWrapper(create VerticalPodAutoscalerCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*autoscalingv1beta1.VerticalPodAutoscaler))
		}
		return create(&autoscalingv1beta1.VerticalPodAutoscaler{})
	}
}

// ReconcileVerticalPodAutoscalers will create and update the VerticalPodAutoscalers coming from the passed VerticalPodAutoscalerCreator slice
func ReconcileVerticalPodAutoscalers(creators []VerticalPodAutoscalerCreator, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, objectModifiers ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &autoscalingv1beta1.VerticalPodAutoscaler{})
	if err != nil {
		return fmt.Errorf("failed to get VerticalPodAutoscaler informer: %v", err)
	}

	for _, create := range creators {
		createObject := verticalPodAutoscalerObjectWrapper(create)
		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureObject(namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure VerticalPodAutoscaler: %v", err)
		}
	}

	return nil
}
