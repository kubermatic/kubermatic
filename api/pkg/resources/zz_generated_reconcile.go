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
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	autoscalingv1beta1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta1"
)

// ServiceCreator defines an interface to create/update Services
type ServiceCreator = func(existing *corev1.Service) (*corev1.Service, error)

// NamedServiceCreatorGetter returns the name of the resource and the corresponding creator function
type NamedServiceCreatorGetter = func() (name string, create ServiceCreator)

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
func ReconcileServices(namedGetters []NamedServiceCreatorGetter, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, objectModifiers ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &corev1.Service{})
	if err != nil {
		return fmt.Errorf("failed to get Service informer: %v", err)
	}

	for _, get := range namedGetters {
		name, create := get()
		createObject := ServiceObjectWrapper(create)
		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(name, namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure Service: %v", err)
		}
	}

	return nil
}

// SecretCreator defines an interface to create/update Secrets
type SecretCreator = func(existing *corev1.Secret) (*corev1.Secret, error)

// NamedSecretCreatorGetter returns the name of the resource and the corresponding creator function
type NamedSecretCreatorGetter = func() (name string, create SecretCreator)

// SecretObjectWrapper adds a wrapper so the SecretCreator matches ObjectCreator
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

// ConfigMapObjectWrapper adds a wrapper so the ConfigMapCreator matches ObjectCreator
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

// ServiceAccountCreator defines an interface to create/update ServiceAccounts
type ServiceAccountCreator = func(existing *corev1.ServiceAccount) (*corev1.ServiceAccount, error)

// ServiceAccountObjectWrapper adds a wrapper so the ServiceAccountCreator matches ObjectCreator
// This is needed as golang does not support function interface matching
func ServiceAccountObjectWrapper(create ServiceAccountCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*corev1.ServiceAccount))
		}
		return create(&corev1.ServiceAccount{})
	}
}

// ReconcileServiceAccounts will create and update the ServiceAccounts coming from the passed ServiceAccountCreator slice
func ReconcileServiceAccounts(creators []ServiceAccountCreator, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, objectModifiers ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &corev1.ServiceAccount{})
	if err != nil {
		return fmt.Errorf("failed to get ServiceAccount informer: %v", err)
	}

	for _, create := range creators {
		createObject := ServiceAccountObjectWrapper(create)
		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureObject(namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure ServiceAccount: %v", err)
		}
	}

	return nil
}

// StatefulSetCreator defines an interface to create/update StatefulSets
type StatefulSetCreator = func(existing *appsv1.StatefulSet) (*appsv1.StatefulSet, error)

// NamedStatefulSetCreatorGetter returns the name of the resource and the corresponding creator function
type NamedStatefulSetCreatorGetter = func() (name string, create StatefulSetCreator)

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
func ReconcileStatefulSets(namedGetters []NamedStatefulSetCreatorGetter, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, objectModifiers ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &appsv1.StatefulSet{})
	if err != nil {
		return fmt.Errorf("failed to get StatefulSet informer: %v", err)
	}

	for _, get := range namedGetters {
		name, create := get()
		createObject := StatefulSetObjectWrapper(create)
		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(name, namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure StatefulSet: %v", err)
		}
	}

	return nil
}

// DeploymentCreator defines an interface to create/update Deployments
type DeploymentCreator = func(existing *appsv1.Deployment) (*appsv1.Deployment, error)

// NamedDeploymentCreatorGetter returns the name of the resource and the corresponding creator function
type NamedDeploymentCreatorGetter = func() (name string, create DeploymentCreator)

// DeploymentObjectWrapper adds a wrapper so the DeploymentCreator matches ObjectCreator
// This is needed as golang does not support function interface matching
func DeploymentObjectWrapper(create DeploymentCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*appsv1.Deployment))
		}
		return create(&appsv1.Deployment{})
	}
}

// ReconcileDeployments will create and update the Deployments coming from the passed DeploymentCreator slice
func ReconcileDeployments(namedGetters []NamedDeploymentCreatorGetter, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, objectModifiers ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &appsv1.Deployment{})
	if err != nil {
		return fmt.Errorf("failed to get Deployment informer: %v", err)
	}

	for _, get := range namedGetters {
		name, create := get()
		createObject := DeploymentObjectWrapper(create)
		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(name, namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure Deployment: %v", err)
		}
	}

	return nil
}

// PodDisruptionBudgetCreator defines an interface to create/update PodDisruptionBudgets
type PodDisruptionBudgetCreator = func(existing *policyv1beta1.PodDisruptionBudget) (*policyv1beta1.PodDisruptionBudget, error)

// PodDisruptionBudgetObjectWrapper adds a wrapper so the PodDisruptionBudgetCreator matches ObjectCreator
// This is needed as golang does not support function interface matching
func PodDisruptionBudgetObjectWrapper(create PodDisruptionBudgetCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*policyv1beta1.PodDisruptionBudget))
		}
		return create(&policyv1beta1.PodDisruptionBudget{})
	}
}

// ReconcilePodDisruptionBudgets will create and update the PodDisruptionBudgets coming from the passed PodDisruptionBudgetCreator slice
func ReconcilePodDisruptionBudgets(creators []PodDisruptionBudgetCreator, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, objectModifiers ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &policyv1beta1.PodDisruptionBudget{})
	if err != nil {
		return fmt.Errorf("failed to get PodDisruptionBudget informer: %v", err)
	}

	for _, create := range creators {
		createObject := PodDisruptionBudgetObjectWrapper(create)
		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureObject(namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure PodDisruptionBudget: %v", err)
		}
	}

	return nil
}

// VerticalPodAutoscalerCreator defines an interface to create/update VerticalPodAutoscalers
type VerticalPodAutoscalerCreator = func(existing *autoscalingv1beta1.VerticalPodAutoscaler) (*autoscalingv1beta1.VerticalPodAutoscaler, error)

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
func ReconcileVerticalPodAutoscalers(creators []VerticalPodAutoscalerCreator, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, objectModifiers ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &autoscalingv1beta1.VerticalPodAutoscaler{})
	if err != nil {
		return fmt.Errorf("failed to get VerticalPodAutoscaler informer: %v", err)
	}

	for _, create := range creators {
		createObject := VerticalPodAutoscalerObjectWrapper(create)
		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureObject(namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure VerticalPodAutoscaler: %v", err)
		}
	}

	return nil
}

// ClusterRoleBindingCreator defines an interface to create/update ClusterRoleBindings
type ClusterRoleBindingCreator = func(existing *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error)

// ClusterRoleBindingObjectWrapper adds a wrapper so the ClusterRoleBindingCreator matches ObjectCreator
// This is needed as golang does not support function interface matching
func ClusterRoleBindingObjectWrapper(create ClusterRoleBindingCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*rbacv1.ClusterRoleBinding))
		}
		return create(&rbacv1.ClusterRoleBinding{})
	}
}

// ReconcileClusterRoleBindings will create and update the ClusterRoleBindings coming from the passed ClusterRoleBindingCreator slice
func ReconcileClusterRoleBindings(creators []ClusterRoleBindingCreator, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, objectModifiers ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &rbacv1.ClusterRoleBinding{})
	if err != nil {
		return fmt.Errorf("failed to get ClusterRoleBinding informer: %v", err)
	}

	for _, create := range creators {
		createObject := ClusterRoleBindingObjectWrapper(create)
		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureObject(namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure ClusterRoleBinding: %v", err)
		}
	}

	return nil
}

// RoleCreator defines an interface to create/update Roles
type RoleCreator = func(existing *rbacv1.Role) (*rbacv1.Role, error)

// NamedRoleCreatorGetter returns the name of the resource and the corresponding creator function
type NamedRoleCreatorGetter = func() (name string, create RoleCreator)

// RoleObjectWrapper adds a wrapper so the RoleCreator matches ObjectCreator
// This is needed as golang does not support function interface matching
func RoleObjectWrapper(create RoleCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*rbacv1.Role))
		}
		return create(&rbacv1.Role{})
	}
}

// ReconcileRoles will create and update the Roles coming from the passed RoleCreator slice
func ReconcileRoles(namedGetters []NamedRoleCreatorGetter, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, objectModifiers ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &rbacv1.Role{})
	if err != nil {
		return fmt.Errorf("failed to get Role informer: %v", err)
	}

	for _, get := range namedGetters {
		name, create := get()
		createObject := RoleObjectWrapper(create)
		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(name, namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure Role: %v", err)
		}
	}

	return nil
}

// RoleBindingCreator defines an interface to create/update RoleBindings
type RoleBindingCreator = func(existing *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error)

// RoleBindingObjectWrapper adds a wrapper so the RoleBindingCreator matches ObjectCreator
// This is needed as golang does not support function interface matching
func RoleBindingObjectWrapper(create RoleBindingCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*rbacv1.RoleBinding))
		}
		return create(&rbacv1.RoleBinding{})
	}
}

// ReconcileRoleBindings will create and update the RoleBindings coming from the passed RoleBindingCreator slice
func ReconcileRoleBindings(creators []RoleBindingCreator, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, objectModifiers ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &rbacv1.RoleBinding{})
	if err != nil {
		return fmt.Errorf("failed to get RoleBinding informer: %v", err)
	}

	for _, create := range creators {
		createObject := RoleBindingObjectWrapper(create)
		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureObject(namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure RoleBinding: %v", err)
		}
	}

	return nil
}

// CustomResourceDefinitionCreator defines an interface to create/update CustomResourceDefinitions
type CustomResourceDefinitionCreator = func(existing *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error)

// NamedCustomResourceDefinitionCreatorGetter returns the name of the resource and the corresponding creator function
type NamedCustomResourceDefinitionCreatorGetter = func() (name string, create CustomResourceDefinitionCreator)

// CustomResourceDefinitionObjectWrapper adds a wrapper so the CustomResourceDefinitionCreator matches ObjectCreator
// This is needed as golang does not support function interface matching
func CustomResourceDefinitionObjectWrapper(create CustomResourceDefinitionCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*apiextensionsv1beta1.CustomResourceDefinition))
		}
		return create(&apiextensionsv1beta1.CustomResourceDefinition{})
	}
}

// ReconcileCustomResourceDefinitions will create and update the CustomResourceDefinitions coming from the passed CustomResourceDefinitionCreator slice
func ReconcileCustomResourceDefinitions(namedGetters []NamedCustomResourceDefinitionCreatorGetter, namespace string, client ctrlruntimeclient.Client, informerFactory ctrlruntimecache.Cache, objectModifiers ...ObjectModifier) error {
	store, err := informerutil.GetSyncedStoreFromDynamicFactory(informerFactory, &apiextensionsv1beta1.CustomResourceDefinition{})
	if err != nil {
		return fmt.Errorf("failed to get CustomResourceDefinition informer: %v", err)
	}

	for _, get := range namedGetters {
		name, create := get()
		createObject := CustomResourceDefinitionObjectWrapper(create)
		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(name, namespace, createObject, store, client); err != nil {
			return fmt.Errorf("failed to ensure CustomResourceDefinition: %v", err)
		}
	}

	return nil
}
