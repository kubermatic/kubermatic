// This file is generated. DO NOT EDIT.
package reconciling

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	gatekeeperv1beta1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
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
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*corev1.Namespace))
		}
		return create(&corev1.Namespace{})
	}
}

// ReconcileNamespaces will create and update the Namespaces coming from the passed NamespaceCreator slice
func ReconcileNamespaces(ctx context.Context, namedGetters []NamedNamespaceCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := NamespaceObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &corev1.Namespace{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := ServiceObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &corev1.Service{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := SecretObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &corev1.Secret{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := ConfigMapObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &corev1.ConfigMap{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := ServiceAccountObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &corev1.ServiceAccount{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		create = DefaultStatefulSet(create)
		createObject := StatefulSetObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &appsv1.StatefulSet{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		create = DefaultDeployment(create)
		createObject := DeploymentObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &appsv1.Deployment{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		create = DefaultDaemonSet(create)
		createObject := DaemonSetObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &appsv1.DaemonSet{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
}

// PodDisruptionBudgetCreator defines an interface to create/update PodDisruptionBudgets
type PodDisruptionBudgetCreator = func(existing *policyv1beta1.PodDisruptionBudget) (*policyv1beta1.PodDisruptionBudget, error)

// NamedPodDisruptionBudgetCreatorGetter returns the name of the resource and the corresponding creator function
type NamedPodDisruptionBudgetCreatorGetter = func() (name string, create PodDisruptionBudgetCreator)

// PodDisruptionBudgetObjectWrapper adds a wrapper so the PodDisruptionBudgetCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func PodDisruptionBudgetObjectWrapper(create PodDisruptionBudgetCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*policyv1beta1.PodDisruptionBudget))
		}
		return create(&policyv1beta1.PodDisruptionBudget{})
	}
}

// ReconcilePodDisruptionBudgets will create and update the PodDisruptionBudgets coming from the passed PodDisruptionBudgetCreator slice
func ReconcilePodDisruptionBudgets(ctx context.Context, namedGetters []NamedPodDisruptionBudgetCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := PodDisruptionBudgetObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &policyv1beta1.PodDisruptionBudget{},
			RequiresRecreate: true,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
}

// VerticalPodAutoscalerCreator defines an interface to create/update VerticalPodAutoscalers
type VerticalPodAutoscalerCreator = func(existing *autoscalingv1beta2.VerticalPodAutoscaler) (*autoscalingv1beta2.VerticalPodAutoscaler, error)

// NamedVerticalPodAutoscalerCreatorGetter returns the name of the resource and the corresponding creator function
type NamedVerticalPodAutoscalerCreatorGetter = func() (name string, create VerticalPodAutoscalerCreator)

// VerticalPodAutoscalerObjectWrapper adds a wrapper so the VerticalPodAutoscalerCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func VerticalPodAutoscalerObjectWrapper(create VerticalPodAutoscalerCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*autoscalingv1beta2.VerticalPodAutoscaler))
		}
		return create(&autoscalingv1beta2.VerticalPodAutoscaler{})
	}
}

// ReconcileVerticalPodAutoscalers will create and update the VerticalPodAutoscalers coming from the passed VerticalPodAutoscalerCreator slice
func ReconcileVerticalPodAutoscalers(ctx context.Context, namedGetters []NamedVerticalPodAutoscalerCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := VerticalPodAutoscalerObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &autoscalingv1beta2.VerticalPodAutoscaler{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := ClusterRoleBindingObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &rbacv1.ClusterRoleBinding{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := ClusterRoleObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &rbacv1.ClusterRole{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := RoleObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &rbacv1.Role{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := RoleBindingObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &rbacv1.RoleBinding{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
}

// CustomResourceDefinitionCreator defines an interface to create/update CustomResourceDefinitions
type CustomResourceDefinitionCreator = func(existing *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error)

// NamedCustomResourceDefinitionCreatorGetter returns the name of the resource and the corresponding creator function
type NamedCustomResourceDefinitionCreatorGetter = func() (name string, create CustomResourceDefinitionCreator)

// CustomResourceDefinitionObjectWrapper adds a wrapper so the CustomResourceDefinitionCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func CustomResourceDefinitionObjectWrapper(create CustomResourceDefinitionCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*apiextensionsv1beta1.CustomResourceDefinition))
		}
		return create(&apiextensionsv1beta1.CustomResourceDefinition{})
	}
}

// ReconcileCustomResourceDefinitions will create and update the CustomResourceDefinitions coming from the passed CustomResourceDefinitionCreator slice
func ReconcileCustomResourceDefinitions(ctx context.Context, namedGetters []NamedCustomResourceDefinitionCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := CustomResourceDefinitionObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &apiextensionsv1beta1.CustomResourceDefinition{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
}

// CronJobCreator defines an interface to create/update CronJobs
type CronJobCreator = func(existing *batchv1beta1.CronJob) (*batchv1beta1.CronJob, error)

// NamedCronJobCreatorGetter returns the name of the resource and the corresponding creator function
type NamedCronJobCreatorGetter = func() (name string, create CronJobCreator)

// CronJobObjectWrapper adds a wrapper so the CronJobCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func CronJobObjectWrapper(create CronJobCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*batchv1beta1.CronJob))
		}
		return create(&batchv1beta1.CronJob{})
	}
}

// ReconcileCronJobs will create and update the CronJobs coming from the passed CronJobCreator slice
func ReconcileCronJobs(ctx context.Context, namedGetters []NamedCronJobCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		create = DefaultCronJob(create)
		createObject := CronJobObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &batchv1beta1.CronJob{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := MutatingWebhookConfigurationObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &admissionregistrationv1.MutatingWebhookConfiguration{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := ValidatingWebhookConfigurationObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &admissionregistrationv1.ValidatingWebhookConfiguration{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
}

// APIServiceCreator defines an interface to create/update APIServices
type APIServiceCreator = func(existing *apiregistrationv1beta1.APIService) (*apiregistrationv1beta1.APIService, error)

// NamedAPIServiceCreatorGetter returns the name of the resource and the corresponding creator function
type NamedAPIServiceCreatorGetter = func() (name string, create APIServiceCreator)

// APIServiceObjectWrapper adds a wrapper so the APIServiceCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func APIServiceObjectWrapper(create APIServiceCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*apiregistrationv1beta1.APIService))
		}
		return create(&apiregistrationv1beta1.APIService{})
	}
}

// ReconcileAPIServices will create and update the APIServices coming from the passed APIServiceCreator slice
func ReconcileAPIServices(ctx context.Context, namedGetters []NamedAPIServiceCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := APIServiceObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &apiregistrationv1beta1.APIService{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
}

// IngressCreator defines an interface to create/update Ingresss
type IngressCreator = func(existing *networkingv1beta1.Ingress) (*networkingv1beta1.Ingress, error)

// NamedIngressCreatorGetter returns the name of the resource and the corresponding creator function
type NamedIngressCreatorGetter = func() (name string, create IngressCreator)

// IngressObjectWrapper adds a wrapper so the IngressCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func IngressObjectWrapper(create IngressCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*networkingv1beta1.Ingress))
		}
		return create(&networkingv1beta1.Ingress{})
	}
}

// ReconcileIngresses will create and update the Ingresses coming from the passed IngressCreator slice
func ReconcileIngresses(ctx context.Context, namedGetters []NamedIngressCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := IngressObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &networkingv1beta1.Ingress{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := SeedObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &kubermaticv1.Seed{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
}

// CertificateCreator defines an interface to create/update Certificates
type CertificateCreator = func(existing *certmanagerv1alpha2.Certificate) (*certmanagerv1alpha2.Certificate, error)

// NamedCertificateCreatorGetter returns the name of the resource and the corresponding creator function
type NamedCertificateCreatorGetter = func() (name string, create CertificateCreator)

// CertificateObjectWrapper adds a wrapper so the CertificateCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func CertificateObjectWrapper(create CertificateCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*certmanagerv1alpha2.Certificate))
		}
		return create(&certmanagerv1alpha2.Certificate{})
	}
}

// ReconcileCertificates will create and update the Certificates coming from the passed CertificateCreator slice
func ReconcileCertificates(ctx context.Context, namedGetters []NamedCertificateCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := CertificateObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &certmanagerv1alpha2.Certificate{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := EtcdBackupConfigObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &kubermaticv1.EtcdBackupConfig{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
}

// ConstraintTemplateCreator defines an interface to create/update ConstraintTemplates
type ConstraintTemplateCreator = func(existing *gatekeeperv1beta1.ConstraintTemplate) (*gatekeeperv1beta1.ConstraintTemplate, error)

// NamedConstraintTemplateCreatorGetter returns the name of the resource and the corresponding creator function
type NamedConstraintTemplateCreatorGetter = func() (name string, create ConstraintTemplateCreator)

// ConstraintTemplateObjectWrapper adds a wrapper so the ConstraintTemplateCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func ConstraintTemplateObjectWrapper(create ConstraintTemplateCreator) ObjectCreator {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return create(existing.(*gatekeeperv1beta1.ConstraintTemplate))
		}
		return create(&gatekeeperv1beta1.ConstraintTemplate{})
	}
}

// ReconcileConstraintTemplates will create and update the ConstraintTemplates coming from the passed ConstraintTemplateCreator slice
func ReconcileConstraintTemplates(ctx context.Context, namedGetters []NamedConstraintTemplateCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := ConstraintTemplateObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &gatekeeperv1beta1.ConstraintTemplate{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticV1ConstraintTemplateObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &kubermaticv1.ConstraintTemplate{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticV1ProjectObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &kubermaticv1.Project{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticV1UserProjectBindingObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &kubermaticv1.UserProjectBinding{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticV1ConstraintObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &kubermaticv1.Constraint{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticV1UserObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &kubermaticv1.User{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
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
	var objs []EnsureObject
	for _, get := range namedGetters {
		name, create := get()
		createObject := KubermaticV1ClusterTemplateObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		objs = append(objs, EnsureObject{
			Name:             types.NamespacedName{Namespace: namespace, Name: name},
			Creator:          createObject,
			EmptyObj:         &kubermaticv1.ClusterTemplate{},
			RequiresRecreate: false,
		})
	}

	return EnsureNamedObjectsConcurrent(ctx, client, objs)
}
