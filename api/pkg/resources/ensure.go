package resources

import (
	"context"
	"fmt"

	batchv1beta1 "k8s.io/api/batch/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	batchv1beta1client "k8s.io/client-go/kubernetes/typed/batch/v1beta1"
	rbacv1client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	batchv1beta1lister "k8s.io/client-go/listers/batch/v1beta1"
	rbacv1lister "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"

	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate go run ../../codegen/reconcile/main.go

// EnsureRole will create the role with the passed create function & create or update it if necessary.
// To check if it's necessary it will do a lookup of the resource at the lister & compare the existing Role with the created one
func EnsureRole(data RoleDataProvider, create RoleCreator, lister rbacv1lister.RoleNamespaceLister, client rbacv1client.RoleInterface) error {
	var existing *rbacv1.Role
	role, err := create(data, nil)
	if err != nil {
		return fmt.Errorf("failed to build Role: %v", err)
	}

	if existing, err = lister.Get(role.Name); err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}

		if _, err = client.Create(role); err != nil {
			return fmt.Errorf("failed to create Role %s: %v", role.Name, err)
		}
		return nil
	}
	existing = existing.DeepCopy()

	role, err = create(data, existing.DeepCopy())
	if err != nil {
		return fmt.Errorf("failed to build Role: %v", err)
	}

	if DeepEqual(role, existing) {
		return nil
	}

	if _, err = client.Update(role); err != nil {
		return fmt.Errorf("failed to update Role %s: %v", role.Name, err)
	}

	return nil
}

// EnsureRoleBinding will create the RoleBinding with the passed create function & create or update it if necessary.
// To check if it's necessary it will do a lookup of the resource at the lister & compare the existing RoleBinding with the created one
func EnsureRoleBinding(data RoleBindingDataProvider, create RoleBindingCreator, lister rbacv1lister.RoleBindingNamespaceLister, client rbacv1client.RoleBindingInterface) error {
	var existing *rbacv1.RoleBinding
	rb, err := create(data, nil)
	if err != nil {
		return fmt.Errorf("failed to build RoleBinding: %v", err)
	}

	if existing, err = lister.Get(rb.Name); err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}

		if _, err = client.Create(rb); err != nil {
			return fmt.Errorf("failed to create RoleBinding %s: %v", rb.Name, err)
		}
		return nil
	}
	existing = existing.DeepCopy()

	rb, err = create(data, existing.DeepCopy())
	if err != nil {
		return fmt.Errorf("failed to build RoleBinding: %v", err)
	}

	if DeepEqual(rb, existing) {
		return nil
	}

	if _, err = client.Update(rb); err != nil {
		return fmt.Errorf("failed to update RoleBinding %s: %v", rb.Name, err)
	}

	return nil
}

// EnsureClusterRoleBinding will create the RoleBinding with the passed create function & create or update it if necessary.
// To check if it's necessary it will do a lookup of the resource at the lister & compare the existing RoleBinding with the created one
func EnsureClusterRoleBinding(data ClusterRoleBindingDataProvider, create ClusterRoleBindingCreatorDeprecated, lister rbacv1lister.ClusterRoleBindingLister, client rbacv1client.ClusterRoleBindingInterface) error {
	var existing *rbacv1.ClusterRoleBinding
	crb, err := create(data, nil)
	if err != nil {
		return fmt.Errorf("failed to build ClusterRoleBinding: %v", err)
	}

	if existing, err = lister.Get(crb.Name); err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}

		if _, err = client.Create(crb); err != nil {
			return fmt.Errorf("failed to create ClusterRoleBinding %s: %v", crb.Name, err)
		}
		return nil
	}
	existing = existing.DeepCopy()

	crb, err = create(data, existing.DeepCopy())
	if err != nil {
		return fmt.Errorf("failed to build ClusterRoleBinding: %v", err)
	}

	if DeepEqual(crb, existing) {
		return nil
	}

	if _, err = client.Update(crb); err != nil {
		return fmt.Errorf("failed to update ClusterRoleBinding %s: %v", crb.Name, err)
	}

	return nil
}

// EnsureCronJob will create the CronJob with the passed create function & create or update it if necessary.
// To check if it's necessary it will do a lookup of the resource at the lister & compare the existing CronJob with the created one
func EnsureCronJob(data *TemplateData, create CronJobCreator, cronJobLister batchv1beta1lister.CronJobNamespaceLister, cronJobClient batchv1beta1client.CronJobInterface) error {
	var existing *batchv1beta1.CronJob
	cronjob, err := create(data, nil)
	if err != nil {
		return fmt.Errorf("failed to build CronJob: %v", err)
	}

	if existing, err = cronJobLister.Get(cronjob.Name); err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}

		if _, err = cronJobClient.Create(cronjob); err != nil {
			return fmt.Errorf("failed to create CronJob %s: %v", cronjob.Name, err)
		}
		return nil
	}
	existing = existing.DeepCopy()

	cronjob, err = create(data, existing.DeepCopy())
	if err != nil {
		return fmt.Errorf("failed to build CronJob: %v", err)
	}

	if DeepEqual(cronjob, existing) {
		return nil
	}

	if _, err = cronJobClient.Update(cronjob); err != nil {
		return fmt.Errorf("failed to update CronJob %s: %v", cronjob.Name, err)
	}

	return nil
}

func createWithNamespace(rawcreate ObjectCreator, namespace string) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		obj, err := rawcreate(existing)
		if err != nil {
			return nil, err
		}
		obj.(metav1.Object).SetNamespace(namespace)
		return obj, nil
	}
}

func createWithName(rawcreate ObjectCreator, name string) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		obj, err := rawcreate(existing)
		if err != nil {
			return nil, err
		}
		obj.(metav1.Object).SetName(name)
		return obj, nil
	}
}

// informerStore is the minimal informer interface we need.
// We're using to ease testing
type informerStore interface {
	GetByKey(key string) (item interface{}, exists bool, err error)
}

// EnsureObject will generate the Object with the passed create function & create or update it in Kubernetes if necessary.
// Deprecated, use EnsureNamedObject instead, it doesn't require to call the creator twice
func EnsureObject(namespace string, rawcreate ObjectCreator, store informerStore, client ctrlruntimeclient.Client) error {
	ctx := context.Background()

	// A wrapper to ensure we always set the Namespace. This is useful as we call create twice
	create := createWithNamespace(rawcreate, namespace)

	obj, err := create(nil)
	if err != nil {
		return fmt.Errorf("failed to build Object(%T): %v", obj, err)
	}

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return fmt.Errorf("failed to get key for Object(%T): %v", obj, err)
	}

	iobj, exists, err := store.GetByKey(key)
	if err != nil {
		return err
	}

	// Object does not exist in lister -> Create the Object
	if !exists {
		if err := client.Create(ctx, obj); err != nil {
			return fmt.Errorf("failed to create %T '%s': %v", obj, key, err)
		}
		return nil
	}

	// Object does exist in lister -> Update it
	existing, ok := iobj.(runtime.Object)
	if !ok {
		return fmt.Errorf("failed case Object from lister to metav1.Object. Object is %T", iobj)
	}

	// Create a copy to ensure we don't modify any lister state
	existing = existing.DeepCopyObject()

	obj, err = create(existing.DeepCopyObject())
	if err != nil {
		return fmt.Errorf("failed to build Object(%T) '%s': %v", existing, key, err)
	}

	if DeepEqual(obj.(metav1.Object), existing.(metav1.Object)) {
		return nil
	}

	if err = client.Update(ctx, obj); err != nil {
		return fmt.Errorf("failed to update object %T '%s': %v", obj, key, err)
	}

	return nil
}

// EnsureNamedObjectV2 will generate the Object with the passed create function & create or update it in Kubernetes if necessary.
// Different to EnsureObject, EnsureNamedObjectV2 requires the name of the resource being passed so the generation just for the name gets avoided.
// It differs from EnsureNamedObject mainly in its signature and is intended to be used from a controller-runtime based controller
func EnsureNamedObjectV2(ctx context.Context, namespacedName types.NamespacedName, rawcreate ObjectCreator, client ctrlruntimeclient.Client, emptyObject runtime.Object) error {
	// A wrapper to ensure we always set the Namespace and Name. This is useful as we call create twice
	create := createWithNamespace(rawcreate, namespacedName.Namespace)
	create = createWithName(create, namespacedName.Name)

	exists := true
	existingObject := emptyObject.DeepCopyObject()
	if err := client.Get(ctx, namespacedName, existingObject); err != nil {
		if !kubeerrors.IsNotFound(err) {
			return fmt.Errorf("failed to get Object(%T): %v", existingObject, err)
		}
		exists = false
	}

	// Object does not exist in lister -> Create the Object
	if !exists {
		obj, err := create(emptyObject)
		if err != nil {
			return fmt.Errorf("failed to generate object: %v", err)
		}
		if err := client.Create(ctx, obj); err != nil {
			return fmt.Errorf("failed to create %T '%s': %v", obj, namespacedName.String(), err)
		}
		return nil
	}

	// Create a copy to make sure we don't compare the object onto itself
	// in case the creator returns the same pointer it got passed in
	obj, err := create(existingObject.DeepCopyObject())
	if err != nil {
		return fmt.Errorf("failed to build Object(%T) '%s': %v", existingObject, namespacedName.String(), err)
	}

	if DeepEqual(obj.(metav1.Object), existingObject.(metav1.Object)) {
		return nil
	}

	if err = client.Update(ctx, obj); err != nil {
		return fmt.Errorf("failed to update object %T '%s': %v", obj, namespacedName.String(), err)
	}

	return nil
}

// EnsureNamedObject will generate the Object with the passed create function & create or update it in Kubernetes if necessary.
// Different to EnsureObject, EnsureNamedObject requires the name of the resource being passed so the generation just for the name gets avoided.
// If you are trying to use this from a controller-runtime-based controller, check out EnsureNamedObjectV2 instead
func EnsureNamedObject(name string, namespace string, rawcreate ObjectCreator, store informerStore, client ctrlruntimeclient.Client) error {
	ctx := context.Background()

	// A wrapper to ensure we always set the Namespace. This is useful as we call create twice
	create := createWithNamespace(rawcreate, namespace)
	create = createWithName(create, name)

	// Create the name for the object in the lister
	key := name
	if len(namespace) > 0 {
		key = fmt.Sprintf("%s/%s", namespace, name)
	}

	iobj, exists, err := store.GetByKey(key)
	if err != nil {
		return err
	}

	// Object does not exist in lister -> Create the Object
	if !exists {
		obj, err := create(nil)
		if err != nil {
			return fmt.Errorf("failed to generate object: %v", err)
		}
		if err := client.Create(ctx, obj); err != nil {
			return fmt.Errorf("failed to create %T '%s': %v", obj, key, err)
		}
		return nil
	}

	// Object does exist in lister -> Update it
	existing, ok := iobj.(runtime.Object)
	if !ok {
		return fmt.Errorf("failed to cast object from lister to metav1.Object. Object is %T", iobj)
	}

	// Create a copy to ensure we don't modify any lister state
	existing = existing.DeepCopyObject()
	obj, err := create(existing.DeepCopyObject())
	if err != nil {
		return fmt.Errorf("failed to build Object(%T) '%s': %v", existing, key, err)
	}

	if DeepEqual(obj.(metav1.Object), existing.(metav1.Object)) {
		return nil
	}

	if err = client.Update(ctx, obj); err != nil {
		return fmt.Errorf("failed to update object %T '%s': %v", obj, key, err)
	}

	return nil
}
