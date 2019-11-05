// This file is generated. DO NOT EDIT.
package reconciling

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
)

// NamespaceCreator defines an interface to create/update Namespaces
type NamespaceCreator = func(existing *corev1.Namespace) (*corev1.Namespace, error)

// NamedNamespaceCreatorGetter returns the name of the resource and the corresponding creator function
type NamedNamespaceCreatorGetter = func() (name string, create NamespaceCreator)

// NamespaceObjectWrapper adds a wrapper so the NamespaceCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func NamespaceObjectWrapper(create NamespaceCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*corev1.Namespace))
		}
		return create(&corev1.Namespace{})
	}
}

// ReconcileNamespaces will create and update the Namespaces coming from the passed NamespaceCreator slice
func ReconcileNamespaces(ctx context.Context, namedGetters []NamedNamespaceCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := NamespaceObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &corev1.Namespace{}, false); err != nil {
			return fmt.Errorf("failed to ensure Namespace %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// ServiceCreator defines an interface to create/update Services
type ServiceCreator = func(existing *corev1.Service) (*corev1.Service, error)

// NamedServiceCreatorGetter returns the name of the resource and the corresponding creator function
type NamedServiceCreatorGetter = func() (name string, create ServiceCreator)

// ServiceObjectWrapper adds a wrapper so the ServiceCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func ServiceObjectWrapper(create ServiceCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*corev1.Service))
		}
		return create(&corev1.Service{})
	}
}

// ReconcileServices will create and update the Services coming from the passed ServiceCreator slice
func ReconcileServices(ctx context.Context, namedGetters []NamedServiceCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := ServiceObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &corev1.Service{}, false); err != nil {
			return fmt.Errorf("failed to ensure Service %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// SecretCreator defines an interface to create/update Secrets
type SecretCreator = func(existing *corev1.Secret) (*corev1.Secret, error)

// NamedSecretCreatorGetter returns the name of the resource and the corresponding creator function
type NamedSecretCreatorGetter = func() (name string, create SecretCreator)

// SecretObjectWrapper adds a wrapper so the SecretCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func SecretObjectWrapper(create SecretCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*corev1.Secret))
		}
		return create(&corev1.Secret{})
	}
}

// ReconcileSecrets will create and update the Secrets coming from the passed SecretCreator slice
func ReconcileSecrets(ctx context.Context, namedGetters []NamedSecretCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := SecretObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &corev1.Secret{}, false); err != nil {
			return fmt.Errorf("failed to ensure Secret %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// ConfigMapCreator defines an interface to create/update ConfigMaps
type ConfigMapCreator = func(existing *corev1.ConfigMap) (*corev1.ConfigMap, error)

// NamedConfigMapCreatorGetter returns the name of the resource and the corresponding creator function
type NamedConfigMapCreatorGetter = func() (name string, create ConfigMapCreator)

// ConfigMapObjectWrapper adds a wrapper so the ConfigMapCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func ConfigMapObjectWrapper(create ConfigMapCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*corev1.ConfigMap))
		}
		return create(&corev1.ConfigMap{})
	}
}

// ReconcileConfigMaps will create and update the ConfigMaps coming from the passed ConfigMapCreator slice
func ReconcileConfigMaps(ctx context.Context, namedGetters []NamedConfigMapCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := ConfigMapObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &corev1.ConfigMap{}, false); err != nil {
			return fmt.Errorf("failed to ensure ConfigMap %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// ServiceAccountCreator defines an interface to create/update ServiceAccounts
type ServiceAccountCreator = func(existing *corev1.ServiceAccount) (*corev1.ServiceAccount, error)

// NamedServiceAccountCreatorGetter returns the name of the resource and the corresponding creator function
type NamedServiceAccountCreatorGetter = func() (name string, create ServiceAccountCreator)

// ServiceAccountObjectWrapper adds a wrapper so the ServiceAccountCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func ServiceAccountObjectWrapper(create ServiceAccountCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*corev1.ServiceAccount))
		}
		return create(&corev1.ServiceAccount{})
	}
}

// ReconcileServiceAccounts will create and update the ServiceAccounts coming from the passed ServiceAccountCreator slice
func ReconcileServiceAccounts(ctx context.Context, namedGetters []NamedServiceAccountCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := ServiceAccountObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &corev1.ServiceAccount{}, false); err != nil {
			return fmt.Errorf("failed to ensure ServiceAccount %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// StatefulSetCreator defines an interface to create/update StatefulSets
type StatefulSetCreator = func(existing *appsv1.StatefulSet) (*appsv1.StatefulSet, error)

// NamedStatefulSetCreatorGetter returns the name of the resource and the corresponding creator function
type NamedStatefulSetCreatorGetter = func() (name string, create StatefulSetCreator)

// StatefulSetObjectWrapper adds a wrapper so the StatefulSetCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func StatefulSetObjectWrapper(create StatefulSetCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*appsv1.StatefulSet))
		}
		return create(&appsv1.StatefulSet{})
	}
}

// ReconcileStatefulSets will create and update the StatefulSets coming from the passed StatefulSetCreator slice
func ReconcileStatefulSets(ctx context.Context, namedGetters []NamedStatefulSetCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		create = DefaultStatefulSet(create)
		createObject := StatefulSetObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &appsv1.StatefulSet{}, false); err != nil {
			return fmt.Errorf("failed to ensure StatefulSet %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// DeploymentCreator defines an interface to create/update Deployments
type DeploymentCreator = func(existing *appsv1.Deployment) (*appsv1.Deployment, error)

// NamedDeploymentCreatorGetter returns the name of the resource and the corresponding creator function
type NamedDeploymentCreatorGetter = func() (name string, create DeploymentCreator)

// DeploymentObjectWrapper adds a wrapper so the DeploymentCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func DeploymentObjectWrapper(create DeploymentCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*appsv1.Deployment))
		}
		return create(&appsv1.Deployment{})
	}
}

// ReconcileDeployments will create and update the Deployments coming from the passed DeploymentCreator slice
func ReconcileDeployments(ctx context.Context, namedGetters []NamedDeploymentCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		create = DefaultDeployment(create)
		createObject := DeploymentObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &appsv1.Deployment{}, false); err != nil {
			return fmt.Errorf("failed to ensure Deployment %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// DaemonSetCreator defines an interface to create/update DaemonSets
type DaemonSetCreator = func(existing *appsv1.DaemonSet) (*appsv1.DaemonSet, error)

// NamedDaemonSetCreatorGetter returns the name of the resource and the corresponding creator function
type NamedDaemonSetCreatorGetter = func() (name string, create DaemonSetCreator)

// DaemonSetObjectWrapper adds a wrapper so the DaemonSetCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func DaemonSetObjectWrapper(create DaemonSetCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*appsv1.DaemonSet))
		}
		return create(&appsv1.DaemonSet{})
	}
}

// ReconcileDaemonSets will create and update the DaemonSets coming from the passed DaemonSetCreator slice
func ReconcileDaemonSets(ctx context.Context, namedGetters []NamedDaemonSetCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		create = DefaultDaemonSet(create)
		createObject := DaemonSetObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &appsv1.DaemonSet{}, false); err != nil {
			return fmt.Errorf("failed to ensure DaemonSet %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// PodDisruptionBudgetCreator defines an interface to create/update PodDisruptionBudgets
type PodDisruptionBudgetCreator = func(existing *policyv1beta1.PodDisruptionBudget) (*policyv1beta1.PodDisruptionBudget, error)

// NamedPodDisruptionBudgetCreatorGetter returns the name of the resource and the corresponding creator function
type NamedPodDisruptionBudgetCreatorGetter = func() (name string, create PodDisruptionBudgetCreator)

// PodDisruptionBudgetObjectWrapper adds a wrapper so the PodDisruptionBudgetCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func PodDisruptionBudgetObjectWrapper(create PodDisruptionBudgetCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*policyv1beta1.PodDisruptionBudget))
		}
		return create(&policyv1beta1.PodDisruptionBudget{})
	}
}

// ReconcilePodDisruptionBudgets will create and update the PodDisruptionBudgets coming from the passed PodDisruptionBudgetCreator slice
func ReconcilePodDisruptionBudgets(ctx context.Context, namedGetters []NamedPodDisruptionBudgetCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := PodDisruptionBudgetObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &policyv1beta1.PodDisruptionBudget{}, true); err != nil {
			return fmt.Errorf("failed to ensure PodDisruptionBudget %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// VerticalPodAutoscalerCreator defines an interface to create/update VerticalPodAutoscalers
type VerticalPodAutoscalerCreator = func(existing *autoscalingv1beta2.VerticalPodAutoscaler) (*autoscalingv1beta2.VerticalPodAutoscaler, error)

// NamedVerticalPodAutoscalerCreatorGetter returns the name of the resource and the corresponding creator function
type NamedVerticalPodAutoscalerCreatorGetter = func() (name string, create VerticalPodAutoscalerCreator)

// VerticalPodAutoscalerObjectWrapper adds a wrapper so the VerticalPodAutoscalerCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func VerticalPodAutoscalerObjectWrapper(create VerticalPodAutoscalerCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*autoscalingv1beta2.VerticalPodAutoscaler))
		}
		return create(&autoscalingv1beta2.VerticalPodAutoscaler{})
	}
}

// ReconcileVerticalPodAutoscalers will create and update the VerticalPodAutoscalers coming from the passed VerticalPodAutoscalerCreator slice
func ReconcileVerticalPodAutoscalers(ctx context.Context, namedGetters []NamedVerticalPodAutoscalerCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := VerticalPodAutoscalerObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &autoscalingv1beta2.VerticalPodAutoscaler{}, false); err != nil {
			return fmt.Errorf("failed to ensure VerticalPodAutoscaler %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// ClusterRoleBindingCreator defines an interface to create/update ClusterRoleBindings
type ClusterRoleBindingCreator = func(existing *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error)

// NamedClusterRoleBindingCreatorGetter returns the name of the resource and the corresponding creator function
type NamedClusterRoleBindingCreatorGetter = func() (name string, create ClusterRoleBindingCreator)

// ClusterRoleBindingObjectWrapper adds a wrapper so the ClusterRoleBindingCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func ClusterRoleBindingObjectWrapper(create ClusterRoleBindingCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*rbacv1.ClusterRoleBinding))
		}
		return create(&rbacv1.ClusterRoleBinding{})
	}
}

// ReconcileClusterRoleBindings will create and update the ClusterRoleBindings coming from the passed ClusterRoleBindingCreator slice
func ReconcileClusterRoleBindings(ctx context.Context, namedGetters []NamedClusterRoleBindingCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := ClusterRoleBindingObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &rbacv1.ClusterRoleBinding{}, false); err != nil {
			return fmt.Errorf("failed to ensure ClusterRoleBinding %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// ClusterRoleCreator defines an interface to create/update ClusterRoles
type ClusterRoleCreator = func(existing *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error)

// NamedClusterRoleCreatorGetter returns the name of the resource and the corresponding creator function
type NamedClusterRoleCreatorGetter = func() (name string, create ClusterRoleCreator)

// ClusterRoleObjectWrapper adds a wrapper so the ClusterRoleCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func ClusterRoleObjectWrapper(create ClusterRoleCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*rbacv1.ClusterRole))
		}
		return create(&rbacv1.ClusterRole{})
	}
}

// ReconcileClusterRoles will create and update the ClusterRoles coming from the passed ClusterRoleCreator slice
func ReconcileClusterRoles(ctx context.Context, namedGetters []NamedClusterRoleCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := ClusterRoleObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &rbacv1.ClusterRole{}, false); err != nil {
			return fmt.Errorf("failed to ensure ClusterRole %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// RoleCreator defines an interface to create/update Roles
type RoleCreator = func(existing *rbacv1.Role) (*rbacv1.Role, error)

// NamedRoleCreatorGetter returns the name of the resource and the corresponding creator function
type NamedRoleCreatorGetter = func() (name string, create RoleCreator)

// RoleObjectWrapper adds a wrapper so the RoleCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func RoleObjectWrapper(create RoleCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*rbacv1.Role))
		}
		return create(&rbacv1.Role{})
	}
}

// ReconcileRoles will create and update the Roles coming from the passed RoleCreator slice
func ReconcileRoles(ctx context.Context, namedGetters []NamedRoleCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := RoleObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &rbacv1.Role{}, false); err != nil {
			return fmt.Errorf("failed to ensure Role %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// RoleBindingCreator defines an interface to create/update RoleBindings
type RoleBindingCreator = func(existing *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error)

// NamedRoleBindingCreatorGetter returns the name of the resource and the corresponding creator function
type NamedRoleBindingCreatorGetter = func() (name string, create RoleBindingCreator)

// RoleBindingObjectWrapper adds a wrapper so the RoleBindingCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func RoleBindingObjectWrapper(create RoleBindingCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*rbacv1.RoleBinding))
		}
		return create(&rbacv1.RoleBinding{})
	}
}

// ReconcileRoleBindings will create and update the RoleBindings coming from the passed RoleBindingCreator slice
func ReconcileRoleBindings(ctx context.Context, namedGetters []NamedRoleBindingCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := RoleBindingObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &rbacv1.RoleBinding{}, false); err != nil {
			return fmt.Errorf("failed to ensure RoleBinding %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// CustomResourceDefinitionCreator defines an interface to create/update CustomResourceDefinitions
type CustomResourceDefinitionCreator = func(existing *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error)

// NamedCustomResourceDefinitionCreatorGetter returns the name of the resource and the corresponding creator function
type NamedCustomResourceDefinitionCreatorGetter = func() (name string, create CustomResourceDefinitionCreator)

// CustomResourceDefinitionObjectWrapper adds a wrapper so the CustomResourceDefinitionCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func CustomResourceDefinitionObjectWrapper(create CustomResourceDefinitionCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*apiextensionsv1beta1.CustomResourceDefinition))
		}
		return create(&apiextensionsv1beta1.CustomResourceDefinition{})
	}
}

// ReconcileCustomResourceDefinitions will create and update the CustomResourceDefinitions coming from the passed CustomResourceDefinitionCreator slice
func ReconcileCustomResourceDefinitions(ctx context.Context, namedGetters []NamedCustomResourceDefinitionCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := CustomResourceDefinitionObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &apiextensionsv1beta1.CustomResourceDefinition{}, false); err != nil {
			return fmt.Errorf("failed to ensure CustomResourceDefinition %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// CronJobCreator defines an interface to create/update CronJobs
type CronJobCreator = func(existing *batchv1beta1.CronJob) (*batchv1beta1.CronJob, error)

// NamedCronJobCreatorGetter returns the name of the resource and the corresponding creator function
type NamedCronJobCreatorGetter = func() (name string, create CronJobCreator)

// CronJobObjectWrapper adds a wrapper so the CronJobCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func CronJobObjectWrapper(create CronJobCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*batchv1beta1.CronJob))
		}
		return create(&batchv1beta1.CronJob{})
	}
}

// ReconcileCronJobs will create and update the CronJobs coming from the passed CronJobCreator slice
func ReconcileCronJobs(ctx context.Context, namedGetters []NamedCronJobCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		create = DefaultCronJob(create)
		createObject := CronJobObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &batchv1beta1.CronJob{}, false); err != nil {
			return fmt.Errorf("failed to ensure CronJob %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// MutatingWebhookConfigurationCreator defines an interface to create/update MutatingWebhookConfigurations
type MutatingWebhookConfigurationCreator = func(existing *admissionregistrationv1beta1.MutatingWebhookConfiguration) (*admissionregistrationv1beta1.MutatingWebhookConfiguration, error)

// NamedMutatingWebhookConfigurationCreatorGetter returns the name of the resource and the corresponding creator function
type NamedMutatingWebhookConfigurationCreatorGetter = func() (name string, create MutatingWebhookConfigurationCreator)

// MutatingWebhookConfigurationObjectWrapper adds a wrapper so the MutatingWebhookConfigurationCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func MutatingWebhookConfigurationObjectWrapper(create MutatingWebhookConfigurationCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*admissionregistrationv1beta1.MutatingWebhookConfiguration))
		}
		return create(&admissionregistrationv1beta1.MutatingWebhookConfiguration{})
	}
}

// ReconcileMutatingWebhookConfigurations will create and update the MutatingWebhookConfigurations coming from the passed MutatingWebhookConfigurationCreator slice
func ReconcileMutatingWebhookConfigurations(ctx context.Context, namedGetters []NamedMutatingWebhookConfigurationCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := MutatingWebhookConfigurationObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &admissionregistrationv1beta1.MutatingWebhookConfiguration{}, false); err != nil {
			return fmt.Errorf("failed to ensure MutatingWebhookConfiguration %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// ValidatingWebhookConfigurationCreator defines an interface to create/update ValidatingWebhookConfigurations
type ValidatingWebhookConfigurationCreator = func(existing *admissionregistrationv1beta1.ValidatingWebhookConfiguration) (*admissionregistrationv1beta1.ValidatingWebhookConfiguration, error)

// NamedValidatingWebhookConfigurationCreatorGetter returns the name of the resource and the corresponding creator function
type NamedValidatingWebhookConfigurationCreatorGetter = func() (name string, create ValidatingWebhookConfigurationCreator)

// ValidatingWebhookConfigurationObjectWrapper adds a wrapper so the ValidatingWebhookConfigurationCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func ValidatingWebhookConfigurationObjectWrapper(create ValidatingWebhookConfigurationCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*admissionregistrationv1beta1.ValidatingWebhookConfiguration))
		}
		return create(&admissionregistrationv1beta1.ValidatingWebhookConfiguration{})
	}
}

// ReconcileValidatingWebhookConfigurations will create and update the ValidatingWebhookConfigurations coming from the passed ValidatingWebhookConfigurationCreator slice
func ReconcileValidatingWebhookConfigurations(ctx context.Context, namedGetters []NamedValidatingWebhookConfigurationCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := ValidatingWebhookConfigurationObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &admissionregistrationv1beta1.ValidatingWebhookConfiguration{}, false); err != nil {
			return fmt.Errorf("failed to ensure ValidatingWebhookConfiguration %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// APIServiceCreator defines an interface to create/update APIServices
type APIServiceCreator = func(existing *apiregistrationv1beta1.APIService) (*apiregistrationv1beta1.APIService, error)

// NamedAPIServiceCreatorGetter returns the name of the resource and the corresponding creator function
type NamedAPIServiceCreatorGetter = func() (name string, create APIServiceCreator)

// APIServiceObjectWrapper adds a wrapper so the APIServiceCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func APIServiceObjectWrapper(create APIServiceCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*apiregistrationv1beta1.APIService))
		}
		return create(&apiregistrationv1beta1.APIService{})
	}
}

// ReconcileAPIServices will create and update the APIServices coming from the passed APIServiceCreator slice
func ReconcileAPIServices(ctx context.Context, namedGetters []NamedAPIServiceCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := APIServiceObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &apiregistrationv1beta1.APIService{}, false); err != nil {
			return fmt.Errorf("failed to ensure APIService %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// IngressCreator defines an interface to create/update Ingresss
type IngressCreator = func(existing *extensionsv1beta1.Ingress) (*extensionsv1beta1.Ingress, error)

// NamedIngressCreatorGetter returns the name of the resource and the corresponding creator function
type NamedIngressCreatorGetter = func() (name string, create IngressCreator)

// IngressObjectWrapper adds a wrapper so the IngressCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func IngressObjectWrapper(create IngressCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*extensionsv1beta1.Ingress))
		}
		return create(&extensionsv1beta1.Ingress{})
	}
}

// ReconcileIngresses will create and update the Ingresses coming from the passed IngressCreator slice
func ReconcileIngresses(ctx context.Context, namedGetters []NamedIngressCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := IngressObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &extensionsv1beta1.Ingress{}, false); err != nil {
			return fmt.Errorf("failed to ensure Ingress %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// SeedCreator defines an interface to create/update Seeds
type SeedCreator = func(existing *kubermaticv1.Seed) (*kubermaticv1.Seed, error)

// NamedSeedCreatorGetter returns the name of the resource and the corresponding creator function
type NamedSeedCreatorGetter = func() (name string, create SeedCreator)

// SeedObjectWrapper adds a wrapper so the SeedCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func SeedObjectWrapper(create SeedCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*kubermaticv1.Seed))
		}
		return create(&kubermaticv1.Seed{})
	}
}

// ReconcileSeeds will create and update the Seeds coming from the passed SeedCreator slice
func ReconcileSeeds(ctx context.Context, namedGetters []NamedSeedCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := SeedObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &kubermaticv1.Seed{}, false); err != nil {
			return fmt.Errorf("failed to ensure Seed %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}
