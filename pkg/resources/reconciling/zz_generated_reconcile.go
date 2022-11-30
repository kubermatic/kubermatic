// This file is generated. DO NOT EDIT.
package reconciling

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	gatekeeperv1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1"
	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	autoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	kvinstancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

// NamespaceCreator defines an interface to create/update Namespaces
type NamespaceCreator = func(existing *corev1.Namespace) (*corev1.Namespace, error)

// NamedNamespaceCreatorGetter returns the name of the resource and the corresponding creator function
type NamedNamespaceCreatorGetter = func() (name string, create NamespaceCreator)

// NamespaceObjectWrapper adds a wrapper so the NamespaceCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func NamespaceObjectWrapper(create NamespaceCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
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
			return fmt.Errorf("failed to ensure Namespace %s/%s: %w", namespace, name, err)
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
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
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
			return fmt.Errorf("failed to ensure Service %s/%s: %w", namespace, name, err)
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
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
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
			return fmt.Errorf("failed to ensure Secret %s/%s: %w", namespace, name, err)
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
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
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
			return fmt.Errorf("failed to ensure ConfigMap %s/%s: %w", namespace, name, err)
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
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
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
			return fmt.Errorf("failed to ensure ServiceAccount %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// EndpointsCreator defines an interface to create/update Endpointss
type EndpointsCreator = func(existing *corev1.Endpoints) (*corev1.Endpoints, error)

// NamedEndpointsCreatorGetter returns the name of the resource and the corresponding creator function
type NamedEndpointsCreatorGetter = func() (name string, create EndpointsCreator)

// EndpointsObjectWrapper adds a wrapper so the EndpointsCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func EndpointsObjectWrapper(create EndpointsCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*corev1.Endpoints))
		}
		return create(&corev1.Endpoints{})
	}
}

// ReconcileEndpoints will create and update the Endpoints coming from the passed EndpointsCreator slice
func ReconcileEndpoints(ctx context.Context, namedGetters []NamedEndpointsCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := EndpointsObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &corev1.Endpoints{}, false); err != nil {
			return fmt.Errorf("failed to ensure Endpoints %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// EndpointSliceCreator defines an interface to create/update EndpointSlices
type EndpointSliceCreator = func(existing *discovery.EndpointSlice) (*discovery.EndpointSlice, error)

// NamedEndpointSliceCreatorGetter returns the name of the resource and the corresponding creator function
type NamedEndpointSliceCreatorGetter = func() (name string, create EndpointSliceCreator)

// EndpointSliceObjectWrapper adds a wrapper so the EndpointSliceCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func EndpointSliceObjectWrapper(create EndpointSliceCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*discovery.EndpointSlice))
		}
		return create(&discovery.EndpointSlice{})
	}
}

// ReconcileEndpointSlices will create and update the EndpointSlices coming from the passed EndpointSliceCreator slice
func ReconcileEndpointSlices(ctx context.Context, namedGetters []NamedEndpointSliceCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := EndpointSliceObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &discovery.EndpointSlice{}, false); err != nil {
			return fmt.Errorf("failed to ensure EndpointSlice %s/%s: %w", namespace, name, err)
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
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
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
			return fmt.Errorf("failed to ensure StatefulSet %s/%s: %w", namespace, name, err)
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
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
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
			return fmt.Errorf("failed to ensure Deployment %s/%s: %w", namespace, name, err)
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
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
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
			return fmt.Errorf("failed to ensure DaemonSet %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// PodDisruptionBudgetCreator defines an interface to create/update PodDisruptionBudgets
type PodDisruptionBudgetCreator = func(existing *policyv1.PodDisruptionBudget) (*policyv1.PodDisruptionBudget, error)

// NamedPodDisruptionBudgetCreatorGetter returns the name of the resource and the corresponding creator function
type NamedPodDisruptionBudgetCreatorGetter = func() (name string, create PodDisruptionBudgetCreator)

// PodDisruptionBudgetObjectWrapper adds a wrapper so the PodDisruptionBudgetCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func PodDisruptionBudgetObjectWrapper(create PodDisruptionBudgetCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*policyv1.PodDisruptionBudget))
		}
		return create(&policyv1.PodDisruptionBudget{})
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

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &policyv1.PodDisruptionBudget{}, true); err != nil {
			return fmt.Errorf("failed to ensure PodDisruptionBudget %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// VerticalPodAutoscalerCreator defines an interface to create/update VerticalPodAutoscalers
type VerticalPodAutoscalerCreator = func(existing *autoscalingv1.VerticalPodAutoscaler) (*autoscalingv1.VerticalPodAutoscaler, error)

// NamedVerticalPodAutoscalerCreatorGetter returns the name of the resource and the corresponding creator function
type NamedVerticalPodAutoscalerCreatorGetter = func() (name string, create VerticalPodAutoscalerCreator)

// VerticalPodAutoscalerObjectWrapper adds a wrapper so the VerticalPodAutoscalerCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func VerticalPodAutoscalerObjectWrapper(create VerticalPodAutoscalerCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*autoscalingv1.VerticalPodAutoscaler))
		}
		return create(&autoscalingv1.VerticalPodAutoscaler{})
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

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &autoscalingv1.VerticalPodAutoscaler{}, false); err != nil {
			return fmt.Errorf("failed to ensure VerticalPodAutoscaler %s/%s: %w", namespace, name, err)
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
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
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
			return fmt.Errorf("failed to ensure ClusterRoleBinding %s/%s: %w", namespace, name, err)
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
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
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
			return fmt.Errorf("failed to ensure ClusterRole %s/%s: %w", namespace, name, err)
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
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
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
			return fmt.Errorf("failed to ensure Role %s/%s: %w", namespace, name, err)
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
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
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
			return fmt.Errorf("failed to ensure RoleBinding %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// CustomResourceDefinitionCreator defines an interface to create/update CustomResourceDefinitions
type CustomResourceDefinitionCreator = func(existing *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error)

// NamedCustomResourceDefinitionCreatorGetter returns the name of the resource and the corresponding creator function
type NamedCustomResourceDefinitionCreatorGetter = func() (name string, create CustomResourceDefinitionCreator)

// CustomResourceDefinitionObjectWrapper adds a wrapper so the CustomResourceDefinitionCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func CustomResourceDefinitionObjectWrapper(create CustomResourceDefinitionCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*apiextensionsv1.CustomResourceDefinition))
		}
		return create(&apiextensionsv1.CustomResourceDefinition{})
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

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &apiextensionsv1.CustomResourceDefinition{}, false); err != nil {
			return fmt.Errorf("failed to ensure CustomResourceDefinition %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// CronJobCreator defines an interface to create/update CronJobs
type CronJobCreator = func(existing *batchv1.CronJob) (*batchv1.CronJob, error)

// NamedCronJobCreatorGetter returns the name of the resource and the corresponding creator function
type NamedCronJobCreatorGetter = func() (name string, create CronJobCreator)

// CronJobObjectWrapper adds a wrapper so the CronJobCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func CronJobObjectWrapper(create CronJobCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*batchv1.CronJob))
		}
		return create(&batchv1.CronJob{})
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

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &batchv1.CronJob{}, false); err != nil {
			return fmt.Errorf("failed to ensure CronJob %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// MutatingWebhookConfigurationCreator defines an interface to create/update MutatingWebhookConfigurations
type MutatingWebhookConfigurationCreator = func(existing *admissionregistrationv1.MutatingWebhookConfiguration) (*admissionregistrationv1.MutatingWebhookConfiguration, error)

// NamedMutatingWebhookConfigurationCreatorGetter returns the name of the resource and the corresponding creator function
type NamedMutatingWebhookConfigurationCreatorGetter = func() (name string, create MutatingWebhookConfigurationCreator)

// MutatingWebhookConfigurationObjectWrapper adds a wrapper so the MutatingWebhookConfigurationCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func MutatingWebhookConfigurationObjectWrapper(create MutatingWebhookConfigurationCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*admissionregistrationv1.MutatingWebhookConfiguration))
		}
		return create(&admissionregistrationv1.MutatingWebhookConfiguration{})
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

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &admissionregistrationv1.MutatingWebhookConfiguration{}, false); err != nil {
			return fmt.Errorf("failed to ensure MutatingWebhookConfiguration %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// ValidatingWebhookConfigurationCreator defines an interface to create/update ValidatingWebhookConfigurations
type ValidatingWebhookConfigurationCreator = func(existing *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error)

// NamedValidatingWebhookConfigurationCreatorGetter returns the name of the resource and the corresponding creator function
type NamedValidatingWebhookConfigurationCreatorGetter = func() (name string, create ValidatingWebhookConfigurationCreator)

// ValidatingWebhookConfigurationObjectWrapper adds a wrapper so the ValidatingWebhookConfigurationCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func ValidatingWebhookConfigurationObjectWrapper(create ValidatingWebhookConfigurationCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*admissionregistrationv1.ValidatingWebhookConfiguration))
		}
		return create(&admissionregistrationv1.ValidatingWebhookConfiguration{})
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

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &admissionregistrationv1.ValidatingWebhookConfiguration{}, false); err != nil {
			return fmt.Errorf("failed to ensure ValidatingWebhookConfiguration %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// APIServiceCreator defines an interface to create/update APIServices
type APIServiceCreator = func(existing *apiregistrationv1.APIService) (*apiregistrationv1.APIService, error)

// NamedAPIServiceCreatorGetter returns the name of the resource and the corresponding creator function
type NamedAPIServiceCreatorGetter = func() (name string, create APIServiceCreator)

// APIServiceObjectWrapper adds a wrapper so the APIServiceCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func APIServiceObjectWrapper(create APIServiceCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*apiregistrationv1.APIService))
		}
		return create(&apiregistrationv1.APIService{})
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

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &apiregistrationv1.APIService{}, false); err != nil {
			return fmt.Errorf("failed to ensure APIService %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// IngressCreator defines an interface to create/update Ingresss
type IngressCreator = func(existing *networkingv1.Ingress) (*networkingv1.Ingress, error)

// NamedIngressCreatorGetter returns the name of the resource and the corresponding creator function
type NamedIngressCreatorGetter = func() (name string, create IngressCreator)

// IngressObjectWrapper adds a wrapper so the IngressCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func IngressObjectWrapper(create IngressCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*networkingv1.Ingress))
		}
		return create(&networkingv1.Ingress{})
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

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &networkingv1.Ingress{}, false); err != nil {
			return fmt.Errorf("failed to ensure Ingress %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// KubermaticConfigurationCreator defines an interface to create/update KubermaticConfigurations
type KubermaticConfigurationCreator = func(existing *kubermaticv1.KubermaticConfiguration) (*kubermaticv1.KubermaticConfiguration, error)

// NamedKubermaticConfigurationCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticConfigurationCreatorGetter = func() (name string, create KubermaticConfigurationCreator)

// KubermaticConfigurationObjectWrapper adds a wrapper so the KubermaticConfigurationCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func KubermaticConfigurationObjectWrapper(create KubermaticConfigurationCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*kubermaticv1.KubermaticConfiguration))
		}
		return create(&kubermaticv1.KubermaticConfiguration{})
	}
}

// ReconcileKubermaticConfigurations will create and update the KubermaticConfigurations coming from the passed KubermaticConfigurationCreator slice
func ReconcileKubermaticConfigurations(ctx context.Context, namedGetters []NamedKubermaticConfigurationCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticConfigurationObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &kubermaticv1.KubermaticConfiguration{}, false); err != nil {
			return fmt.Errorf("failed to ensure KubermaticConfiguration %s/%s: %w", namespace, name, err)
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
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
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
			return fmt.Errorf("failed to ensure Seed %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// EtcdBackupConfigCreator defines an interface to create/update EtcdBackupConfigs
type EtcdBackupConfigCreator = func(existing *kubermaticv1.EtcdBackupConfig) (*kubermaticv1.EtcdBackupConfig, error)

// NamedEtcdBackupConfigCreatorGetter returns the name of the resource and the corresponding creator function
type NamedEtcdBackupConfigCreatorGetter = func() (name string, create EtcdBackupConfigCreator)

// EtcdBackupConfigObjectWrapper adds a wrapper so the EtcdBackupConfigCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func EtcdBackupConfigObjectWrapper(create EtcdBackupConfigCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*kubermaticv1.EtcdBackupConfig))
		}
		return create(&kubermaticv1.EtcdBackupConfig{})
	}
}

// ReconcileEtcdBackupConfigs will create and update the EtcdBackupConfigs coming from the passed EtcdBackupConfigCreator slice
func ReconcileEtcdBackupConfigs(ctx context.Context, namedGetters []NamedEtcdBackupConfigCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := EtcdBackupConfigObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &kubermaticv1.EtcdBackupConfig{}, false); err != nil {
			return fmt.Errorf("failed to ensure EtcdBackupConfig %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// ConstraintTemplateCreator defines an interface to create/update ConstraintTemplates
type ConstraintTemplateCreator = func(existing *gatekeeperv1.ConstraintTemplate) (*gatekeeperv1.ConstraintTemplate, error)

// NamedConstraintTemplateCreatorGetter returns the name of the resource and the corresponding creator function
type NamedConstraintTemplateCreatorGetter = func() (name string, create ConstraintTemplateCreator)

// ConstraintTemplateObjectWrapper adds a wrapper so the ConstraintTemplateCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func ConstraintTemplateObjectWrapper(create ConstraintTemplateCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*gatekeeperv1.ConstraintTemplate))
		}
		return create(&gatekeeperv1.ConstraintTemplate{})
	}
}

// ReconcileConstraintTemplates will create and update the ConstraintTemplates coming from the passed ConstraintTemplateCreator slice
func ReconcileConstraintTemplates(ctx context.Context, namedGetters []NamedConstraintTemplateCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := ConstraintTemplateObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &gatekeeperv1.ConstraintTemplate{}, false); err != nil {
			return fmt.Errorf("failed to ensure ConstraintTemplate %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// OperatingSystemProfileCreator defines an interface to create/update OperatingSystemProfiles
type OperatingSystemProfileCreator = func(existing *osmv1alpha1.OperatingSystemProfile) (*osmv1alpha1.OperatingSystemProfile, error)

// NamedOperatingSystemProfileCreatorGetter returns the name of the resource and the corresponding creator function
type NamedOperatingSystemProfileCreatorGetter = func() (name string, create OperatingSystemProfileCreator)

// OperatingSystemProfileObjectWrapper adds a wrapper so the OperatingSystemProfileCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func OperatingSystemProfileObjectWrapper(create OperatingSystemProfileCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*osmv1alpha1.OperatingSystemProfile))
		}
		return create(&osmv1alpha1.OperatingSystemProfile{})
	}
}

// ReconcileOperatingSystemProfiles will create and update the OperatingSystemProfiles coming from the passed OperatingSystemProfileCreator slice
func ReconcileOperatingSystemProfiles(ctx context.Context, namedGetters []NamedOperatingSystemProfileCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := OperatingSystemProfileObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &osmv1alpha1.OperatingSystemProfile{}, false); err != nil {
			return fmt.Errorf("failed to ensure OperatingSystemProfile %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// KubermaticV1ConstraintTemplateCreator defines an interface to create/update ConstraintTemplates
type KubermaticV1ConstraintTemplateCreator = func(existing *kubermaticv1.ConstraintTemplate) (*kubermaticv1.ConstraintTemplate, error)

// NamedKubermaticV1ConstraintTemplateCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1ConstraintTemplateCreatorGetter = func() (name string, create KubermaticV1ConstraintTemplateCreator)

// KubermaticV1ConstraintTemplateObjectWrapper adds a wrapper so the KubermaticV1ConstraintTemplateCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func KubermaticV1ConstraintTemplateObjectWrapper(create KubermaticV1ConstraintTemplateCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*kubermaticv1.ConstraintTemplate))
		}
		return create(&kubermaticv1.ConstraintTemplate{})
	}
}

// ReconcileKubermaticV1ConstraintTemplates will create and update the KubermaticV1ConstraintTemplates coming from the passed KubermaticV1ConstraintTemplateCreator slice
func ReconcileKubermaticV1ConstraintTemplates(ctx context.Context, namedGetters []NamedKubermaticV1ConstraintTemplateCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticV1ConstraintTemplateObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &kubermaticv1.ConstraintTemplate{}, false); err != nil {
			return fmt.Errorf("failed to ensure ConstraintTemplate %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// KubermaticV1ProjectCreator defines an interface to create/update Projects
type KubermaticV1ProjectCreator = func(existing *kubermaticv1.Project) (*kubermaticv1.Project, error)

// NamedKubermaticV1ProjectCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1ProjectCreatorGetter = func() (name string, create KubermaticV1ProjectCreator)

// KubermaticV1ProjectObjectWrapper adds a wrapper so the KubermaticV1ProjectCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func KubermaticV1ProjectObjectWrapper(create KubermaticV1ProjectCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*kubermaticv1.Project))
		}
		return create(&kubermaticv1.Project{})
	}
}

// ReconcileKubermaticV1Projects will create and update the KubermaticV1Projects coming from the passed KubermaticV1ProjectCreator slice
func ReconcileKubermaticV1Projects(ctx context.Context, namedGetters []NamedKubermaticV1ProjectCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticV1ProjectObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &kubermaticv1.Project{}, false); err != nil {
			return fmt.Errorf("failed to ensure Project %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// KubermaticV1UserProjectBindingCreator defines an interface to create/update UserProjectBindings
type KubermaticV1UserProjectBindingCreator = func(existing *kubermaticv1.UserProjectBinding) (*kubermaticv1.UserProjectBinding, error)

// NamedKubermaticV1UserProjectBindingCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1UserProjectBindingCreatorGetter = func() (name string, create KubermaticV1UserProjectBindingCreator)

// KubermaticV1UserProjectBindingObjectWrapper adds a wrapper so the KubermaticV1UserProjectBindingCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func KubermaticV1UserProjectBindingObjectWrapper(create KubermaticV1UserProjectBindingCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*kubermaticv1.UserProjectBinding))
		}
		return create(&kubermaticv1.UserProjectBinding{})
	}
}

// ReconcileKubermaticV1UserProjectBindings will create and update the KubermaticV1UserProjectBindings coming from the passed KubermaticV1UserProjectBindingCreator slice
func ReconcileKubermaticV1UserProjectBindings(ctx context.Context, namedGetters []NamedKubermaticV1UserProjectBindingCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticV1UserProjectBindingObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &kubermaticv1.UserProjectBinding{}, false); err != nil {
			return fmt.Errorf("failed to ensure UserProjectBinding %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// KubermaticV1GroupProjectBindingCreator defines an interface to create/update GroupProjectBindings
type KubermaticV1GroupProjectBindingCreator = func(existing *kubermaticv1.GroupProjectBinding) (*kubermaticv1.GroupProjectBinding, error)

// NamedKubermaticV1GroupProjectBindingCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1GroupProjectBindingCreatorGetter = func() (name string, create KubermaticV1GroupProjectBindingCreator)

// KubermaticV1GroupProjectBindingObjectWrapper adds a wrapper so the KubermaticV1GroupProjectBindingCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func KubermaticV1GroupProjectBindingObjectWrapper(create KubermaticV1GroupProjectBindingCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*kubermaticv1.GroupProjectBinding))
		}
		return create(&kubermaticv1.GroupProjectBinding{})
	}
}

// ReconcileKubermaticV1GroupProjectBindings will create and update the KubermaticV1GroupProjectBindings coming from the passed KubermaticV1GroupProjectBindingCreator slice
func ReconcileKubermaticV1GroupProjectBindings(ctx context.Context, namedGetters []NamedKubermaticV1GroupProjectBindingCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticV1GroupProjectBindingObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &kubermaticv1.GroupProjectBinding{}, false); err != nil {
			return fmt.Errorf("failed to ensure GroupProjectBinding %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// KubermaticV1ConstraintCreator defines an interface to create/update Constraints
type KubermaticV1ConstraintCreator = func(existing *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error)

// NamedKubermaticV1ConstraintCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1ConstraintCreatorGetter = func() (name string, create KubermaticV1ConstraintCreator)

// KubermaticV1ConstraintObjectWrapper adds a wrapper so the KubermaticV1ConstraintCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func KubermaticV1ConstraintObjectWrapper(create KubermaticV1ConstraintCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*kubermaticv1.Constraint))
		}
		return create(&kubermaticv1.Constraint{})
	}
}

// ReconcileKubermaticV1Constraints will create and update the KubermaticV1Constraints coming from the passed KubermaticV1ConstraintCreator slice
func ReconcileKubermaticV1Constraints(ctx context.Context, namedGetters []NamedKubermaticV1ConstraintCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticV1ConstraintObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &kubermaticv1.Constraint{}, false); err != nil {
			return fmt.Errorf("failed to ensure Constraint %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// KubermaticV1UserCreator defines an interface to create/update Users
type KubermaticV1UserCreator = func(existing *kubermaticv1.User) (*kubermaticv1.User, error)

// NamedKubermaticV1UserCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1UserCreatorGetter = func() (name string, create KubermaticV1UserCreator)

// KubermaticV1UserObjectWrapper adds a wrapper so the KubermaticV1UserCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func KubermaticV1UserObjectWrapper(create KubermaticV1UserCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*kubermaticv1.User))
		}
		return create(&kubermaticv1.User{})
	}
}

// ReconcileKubermaticV1Users will create and update the KubermaticV1Users coming from the passed KubermaticV1UserCreator slice
func ReconcileKubermaticV1Users(ctx context.Context, namedGetters []NamedKubermaticV1UserCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticV1UserObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &kubermaticv1.User{}, false); err != nil {
			return fmt.Errorf("failed to ensure User %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// KubermaticV1ClusterCreator defines an interface to create/update Clusters
type KubermaticV1ClusterCreator = func(existing *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error)

// NamedKubermaticV1ClusterCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1ClusterCreatorGetter = func() (name string, create KubermaticV1ClusterCreator)

// KubermaticV1ClusterObjectWrapper adds a wrapper so the KubermaticV1ClusterCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func KubermaticV1ClusterObjectWrapper(create KubermaticV1ClusterCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*kubermaticv1.Cluster))
		}
		return create(&kubermaticv1.Cluster{})
	}
}

// ReconcileKubermaticV1Clusters will create and update the KubermaticV1Clusters coming from the passed KubermaticV1ClusterCreator slice
func ReconcileKubermaticV1Clusters(ctx context.Context, namedGetters []NamedKubermaticV1ClusterCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticV1ClusterObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &kubermaticv1.Cluster{}, false); err != nil {
			return fmt.Errorf("failed to ensure Cluster %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// KubermaticV1ClusterTemplateCreator defines an interface to create/update ClusterTemplates
type KubermaticV1ClusterTemplateCreator = func(existing *kubermaticv1.ClusterTemplate) (*kubermaticv1.ClusterTemplate, error)

// NamedKubermaticV1ClusterTemplateCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1ClusterTemplateCreatorGetter = func() (name string, create KubermaticV1ClusterTemplateCreator)

// KubermaticV1ClusterTemplateObjectWrapper adds a wrapper so the KubermaticV1ClusterTemplateCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func KubermaticV1ClusterTemplateObjectWrapper(create KubermaticV1ClusterTemplateCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*kubermaticv1.ClusterTemplate))
		}
		return create(&kubermaticv1.ClusterTemplate{})
	}
}

// ReconcileKubermaticV1ClusterTemplates will create and update the KubermaticV1ClusterTemplates coming from the passed KubermaticV1ClusterTemplateCreator slice
func ReconcileKubermaticV1ClusterTemplates(ctx context.Context, namedGetters []NamedKubermaticV1ClusterTemplateCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticV1ClusterTemplateObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &kubermaticv1.ClusterTemplate{}, false); err != nil {
			return fmt.Errorf("failed to ensure ClusterTemplate %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// NetworkPolicyCreator defines an interface to create/update NetworkPolicys
type NetworkPolicyCreator = func(existing *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error)

// NamedNetworkPolicyCreatorGetter returns the name of the resource and the corresponding creator function
type NamedNetworkPolicyCreatorGetter = func() (name string, create NetworkPolicyCreator)

// NetworkPolicyObjectWrapper adds a wrapper so the NetworkPolicyCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func NetworkPolicyObjectWrapper(create NetworkPolicyCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*networkingv1.NetworkPolicy))
		}
		return create(&networkingv1.NetworkPolicy{})
	}
}

// ReconcileNetworkPolicies will create and update the NetworkPolicies coming from the passed NetworkPolicyCreator slice
func ReconcileNetworkPolicies(ctx context.Context, namedGetters []NamedNetworkPolicyCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := NetworkPolicyObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &networkingv1.NetworkPolicy{}, false); err != nil {
			return fmt.Errorf("failed to ensure NetworkPolicy %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// KubermaticV1RuleGroupCreator defines an interface to create/update RuleGroups
type KubermaticV1RuleGroupCreator = func(existing *kubermaticv1.RuleGroup) (*kubermaticv1.RuleGroup, error)

// NamedKubermaticV1RuleGroupCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1RuleGroupCreatorGetter = func() (name string, create KubermaticV1RuleGroupCreator)

// KubermaticV1RuleGroupObjectWrapper adds a wrapper so the KubermaticV1RuleGroupCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func KubermaticV1RuleGroupObjectWrapper(create KubermaticV1RuleGroupCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*kubermaticv1.RuleGroup))
		}
		return create(&kubermaticv1.RuleGroup{})
	}
}

// ReconcileKubermaticV1RuleGroups will create and update the KubermaticV1RuleGroups coming from the passed KubermaticV1RuleGroupCreator slice
func ReconcileKubermaticV1RuleGroups(ctx context.Context, namedGetters []NamedKubermaticV1RuleGroupCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticV1RuleGroupObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &kubermaticv1.RuleGroup{}, false); err != nil {
			return fmt.Errorf("failed to ensure RuleGroup %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// AppsKubermaticV1ApplicationDefinitionCreator defines an interface to create/update ApplicationDefinitions
type AppsKubermaticV1ApplicationDefinitionCreator = func(existing *appskubermaticv1.ApplicationDefinition) (*appskubermaticv1.ApplicationDefinition, error)

// NamedAppsKubermaticV1ApplicationDefinitionCreatorGetter returns the name of the resource and the corresponding creator function
type NamedAppsKubermaticV1ApplicationDefinitionCreatorGetter = func() (name string, create AppsKubermaticV1ApplicationDefinitionCreator)

// AppsKubermaticV1ApplicationDefinitionObjectWrapper adds a wrapper so the AppsKubermaticV1ApplicationDefinitionCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func AppsKubermaticV1ApplicationDefinitionObjectWrapper(create AppsKubermaticV1ApplicationDefinitionCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*appskubermaticv1.ApplicationDefinition))
		}
		return create(&appskubermaticv1.ApplicationDefinition{})
	}
}

// ReconcileAppsKubermaticV1ApplicationDefinitions will create and update the AppsKubermaticV1ApplicationDefinitions coming from the passed AppsKubermaticV1ApplicationDefinitionCreator slice
func ReconcileAppsKubermaticV1ApplicationDefinitions(ctx context.Context, namedGetters []NamedAppsKubermaticV1ApplicationDefinitionCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := AppsKubermaticV1ApplicationDefinitionObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &appskubermaticv1.ApplicationDefinition{}, false); err != nil {
			return fmt.Errorf("failed to ensure ApplicationDefinition %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// AppsKubermaticV1ApplicationInstallationCreator defines an interface to create/update ApplicationInstallations
type AppsKubermaticV1ApplicationInstallationCreator = func(existing *appskubermaticv1.ApplicationInstallation) (*appskubermaticv1.ApplicationInstallation, error)

// NamedAppsKubermaticV1ApplicationInstallationCreatorGetter returns the name of the resource and the corresponding creator function
type NamedAppsKubermaticV1ApplicationInstallationCreatorGetter = func() (name string, create AppsKubermaticV1ApplicationInstallationCreator)

// AppsKubermaticV1ApplicationInstallationObjectWrapper adds a wrapper so the AppsKubermaticV1ApplicationInstallationCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func AppsKubermaticV1ApplicationInstallationObjectWrapper(create AppsKubermaticV1ApplicationInstallationCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*appskubermaticv1.ApplicationInstallation))
		}
		return create(&appskubermaticv1.ApplicationInstallation{})
	}
}

// ReconcileAppsKubermaticV1ApplicationInstallations will create and update the AppsKubermaticV1ApplicationInstallations coming from the passed AppsKubermaticV1ApplicationInstallationCreator slice
func ReconcileAppsKubermaticV1ApplicationInstallations(ctx context.Context, namedGetters []NamedAppsKubermaticV1ApplicationInstallationCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := AppsKubermaticV1ApplicationInstallationObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &appskubermaticv1.ApplicationInstallation{}, false); err != nil {
			return fmt.Errorf("failed to ensure ApplicationInstallation %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// KvInstancetypeV1alpha1VirtualMachineInstancetypeCreator defines an interface to create/update VirtualMachineInstancetypes
type KvInstancetypeV1alpha1VirtualMachineInstancetypeCreator = func(existing *kvinstancetypev1alpha1.VirtualMachineInstancetype) (*kvinstancetypev1alpha1.VirtualMachineInstancetype, error)

// NamedKvInstancetypeV1alpha1VirtualMachineInstancetypeCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKvInstancetypeV1alpha1VirtualMachineInstancetypeCreatorGetter = func() (name string, create KvInstancetypeV1alpha1VirtualMachineInstancetypeCreator)

// KvInstancetypeV1alpha1VirtualMachineInstancetypeObjectWrapper adds a wrapper so the KvInstancetypeV1alpha1VirtualMachineInstancetypeCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func KvInstancetypeV1alpha1VirtualMachineInstancetypeObjectWrapper(create KvInstancetypeV1alpha1VirtualMachineInstancetypeCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*kvinstancetypev1alpha1.VirtualMachineInstancetype))
		}
		return create(&kvinstancetypev1alpha1.VirtualMachineInstancetype{})
	}
}

// ReconcileKvInstancetypeV1alpha1VirtualMachineInstancetypes will create and update the KvInstancetypeV1alpha1VirtualMachineInstancetypes coming from the passed KvInstancetypeV1alpha1VirtualMachineInstancetypeCreator slice
func ReconcileKvInstancetypeV1alpha1VirtualMachineInstancetypes(ctx context.Context, namedGetters []NamedKvInstancetypeV1alpha1VirtualMachineInstancetypeCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := KvInstancetypeV1alpha1VirtualMachineInstancetypeObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &kvinstancetypev1alpha1.VirtualMachineInstancetype{}, false); err != nil {
			return fmt.Errorf("failed to ensure VirtualMachineInstancetype %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// KvInstancetypeV1alpha1VirtualMachinePreferenceCreator defines an interface to create/update VirtualMachinePreferences
type KvInstancetypeV1alpha1VirtualMachinePreferenceCreator = func(existing *kvinstancetypev1alpha1.VirtualMachinePreference) (*kvinstancetypev1alpha1.VirtualMachinePreference, error)

// NamedKvInstancetypeV1alpha1VirtualMachinePreferenceCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKvInstancetypeV1alpha1VirtualMachinePreferenceCreatorGetter = func() (name string, create KvInstancetypeV1alpha1VirtualMachinePreferenceCreator)

// KvInstancetypeV1alpha1VirtualMachinePreferenceObjectWrapper adds a wrapper so the KvInstancetypeV1alpha1VirtualMachinePreferenceCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func KvInstancetypeV1alpha1VirtualMachinePreferenceObjectWrapper(create KvInstancetypeV1alpha1VirtualMachinePreferenceCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*kvinstancetypev1alpha1.VirtualMachinePreference))
		}
		return create(&kvinstancetypev1alpha1.VirtualMachinePreference{})
	}
}

// ReconcileKvInstancetypeV1alpha1VirtualMachinePreferences will create and update the KvInstancetypeV1alpha1VirtualMachinePreferences coming from the passed KvInstancetypeV1alpha1VirtualMachinePreferenceCreator slice
func ReconcileKvInstancetypeV1alpha1VirtualMachinePreferences(ctx context.Context, namedGetters []NamedKvInstancetypeV1alpha1VirtualMachinePreferenceCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := KvInstancetypeV1alpha1VirtualMachinePreferenceObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &kvinstancetypev1alpha1.VirtualMachinePreference{}, false); err != nil {
			return fmt.Errorf("failed to ensure VirtualMachinePreference %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// KubermaticV1PresetCreator defines an interface to create/update Presets
type KubermaticV1PresetCreator = func(existing *kubermaticv1.Preset) (*kubermaticv1.Preset, error)

// NamedKubermaticV1PresetCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1PresetCreatorGetter = func() (name string, create KubermaticV1PresetCreator)

// KubermaticV1PresetObjectWrapper adds a wrapper so the KubermaticV1PresetCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func KubermaticV1PresetObjectWrapper(create KubermaticV1PresetCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*kubermaticv1.Preset))
		}
		return create(&kubermaticv1.Preset{})
	}
}

// ReconcileKubermaticV1Presets will create and update the KubermaticV1Presets coming from the passed KubermaticV1PresetCreator slice
func ReconcileKubermaticV1Presets(ctx context.Context, namedGetters []NamedKubermaticV1PresetCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticV1PresetObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &kubermaticv1.Preset{}, false); err != nil {
			return fmt.Errorf("failed to ensure Preset %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// CDIv1beta1DataVolumeCreator defines an interface to create/update DataVolumes
type CDIv1beta1DataVolumeCreator = func(existing *cdiv1beta1.DataVolume) (*cdiv1beta1.DataVolume, error)

// NamedCDIv1beta1DataVolumeCreatorGetter returns the name of the resource and the corresponding creator function
type NamedCDIv1beta1DataVolumeCreatorGetter = func() (name string, create CDIv1beta1DataVolumeCreator)

// CDIv1beta1DataVolumeObjectWrapper adds a wrapper so the CDIv1beta1DataVolumeCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func CDIv1beta1DataVolumeObjectWrapper(create CDIv1beta1DataVolumeCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*cdiv1beta1.DataVolume))
		}
		return create(&cdiv1beta1.DataVolume{})
	}
}

// ReconcileCDIv1beta1DataVolumes will create and update the CDIv1beta1DataVolumes coming from the passed CDIv1beta1DataVolumeCreator slice
func ReconcileCDIv1beta1DataVolumes(ctx context.Context, namedGetters []NamedCDIv1beta1DataVolumeCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := CDIv1beta1DataVolumeObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &cdiv1beta1.DataVolume{}, false); err != nil {
			return fmt.Errorf("failed to ensure DataVolume %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// KubermaticV1ResourceQuotaCreator defines an interface to create/update ResourceQuotas
type KubermaticV1ResourceQuotaCreator = func(existing *kubermaticv1.ResourceQuota) (*kubermaticv1.ResourceQuota, error)

// NamedKubermaticV1ResourceQuotaCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1ResourceQuotaCreatorGetter = func() (name string, create KubermaticV1ResourceQuotaCreator)

// KubermaticV1ResourceQuotaObjectWrapper adds a wrapper so the KubermaticV1ResourceQuotaCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func KubermaticV1ResourceQuotaObjectWrapper(create KubermaticV1ResourceQuotaCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*kubermaticv1.ResourceQuota))
		}
		return create(&kubermaticv1.ResourceQuota{})
	}
}

// ReconcileKubermaticV1ResourceQuotas will create and update the KubermaticV1ResourceQuotas coming from the passed KubermaticV1ResourceQuotaCreator slice
func ReconcileKubermaticV1ResourceQuotas(ctx context.Context, namedGetters []NamedKubermaticV1ResourceQuotaCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticV1ResourceQuotaObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &kubermaticv1.ResourceQuota{}, false); err != nil {
			return fmt.Errorf("failed to ensure ResourceQuota %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// KubermaticV1UserSSHKeyCreator defines an interface to create/update UserSSHKeys
type KubermaticV1UserSSHKeyCreator = func(existing *kubermaticv1.UserSSHKey) (*kubermaticv1.UserSSHKey, error)

// NamedKubermaticV1UserSSHKeyCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1UserSSHKeyCreatorGetter = func() (name string, create KubermaticV1UserSSHKeyCreator)

// KubermaticV1UserSSHKeyObjectWrapper adds a wrapper so the KubermaticV1UserSSHKeyCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func KubermaticV1UserSSHKeyObjectWrapper(create KubermaticV1UserSSHKeyCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*kubermaticv1.UserSSHKey))
		}
		return create(&kubermaticv1.UserSSHKey{})
	}
}

// ReconcileKubermaticV1UserSSHKeys will create and update the KubermaticV1UserSSHKeys coming from the passed KubermaticV1UserSSHKeyCreator slice
func ReconcileKubermaticV1UserSSHKeys(ctx context.Context, namedGetters []NamedKubermaticV1UserSSHKeyCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticV1UserSSHKeyObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &kubermaticv1.UserSSHKey{}, false); err != nil {
			return fmt.Errorf("failed to ensure UserSSHKey %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// KubermaticV1AddonCreator defines an interface to create/update Addons
type KubermaticV1AddonCreator = func(existing *kubermaticv1.Addon) (*kubermaticv1.Addon, error)

// NamedKubermaticV1AddonCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1AddonCreatorGetter = func() (name string, create KubermaticV1AddonCreator)

// KubermaticV1AddonObjectWrapper adds a wrapper so the KubermaticV1AddonCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func KubermaticV1AddonObjectWrapper(create KubermaticV1AddonCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*kubermaticv1.Addon))
		}
		return create(&kubermaticv1.Addon{})
	}
}

// ReconcileKubermaticV1Addons will create and update the KubermaticV1Addons coming from the passed KubermaticV1AddonCreator slice
func ReconcileKubermaticV1Addons(ctx context.Context, namedGetters []NamedKubermaticV1AddonCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticV1AddonObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &kubermaticv1.Addon{}, false); err != nil {
			return fmt.Errorf("failed to ensure Addon %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// KubermaticV1AddonConfigCreator defines an interface to create/update AddonConfigs
type KubermaticV1AddonConfigCreator = func(existing *kubermaticv1.AddonConfig) (*kubermaticv1.AddonConfig, error)

// NamedKubermaticV1AddonConfigCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1AddonConfigCreatorGetter = func() (name string, create KubermaticV1AddonConfigCreator)

// KubermaticV1AddonConfigObjectWrapper adds a wrapper so the KubermaticV1AddonConfigCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func KubermaticV1AddonConfigObjectWrapper(create KubermaticV1AddonConfigCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*kubermaticv1.AddonConfig))
		}
		return create(&kubermaticv1.AddonConfig{})
	}
}

// ReconcileKubermaticV1AddonConfigs will create and update the KubermaticV1AddonConfigs coming from the passed KubermaticV1AddonConfigCreator slice
func ReconcileKubermaticV1AddonConfigs(ctx context.Context, namedGetters []NamedKubermaticV1AddonConfigCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticV1AddonConfigObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &kubermaticv1.AddonConfig{}, false); err != nil {
			return fmt.Errorf("failed to ensure AddonConfig %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}
