package reconciling

import (
	"context"
	"fmt"
	"time"

	"k8s.io/api/apps/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate go run ../../../codegen/reconcile/main.go

// ObjectCreator defines an interface to create/update a runtime.Object
type ObjectCreator = func(existing runtime.Object) (runtime.Object, error)

// ObjectModifier is a wrapper function which modifies the object which gets returned by the passed in ObjectCreator
type ObjectModifier func(create ObjectCreator) ObjectCreator

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

// EnsureNamedObject will generate the Object with the passed create function & create or update it in Kubernetes if necessary.
func EnsureNamedObject(ctx context.Context, namespacedName types.NamespacedName, rawcreate ObjectCreator, client ctrlruntimeclient.Client, emptyObject runtime.Object, requiresRecreate bool) error {
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
		// Wait until the object exists in the cache
		createdObjectIsInCache := waitUntilObjectExistsInCacheConditionFunc(ctx, client, namespacedName, obj)
		err = wait.PollImmediate(10*time.Millisecond, 10*time.Second, createdObjectIsInCache)
		if err != nil {
			return fmt.Errorf("failed waiting for the cache to contain our newly created object: %v", err)
		}

		klog.V(2).Infof("Created %T %s in Namespace %q", obj, obj.(metav1.Object).GetName(), obj.(metav1.Object).GetNamespace())
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

	if !requiresRecreate {
		// We keep resetting the status here to avoid working on any outdated object
		// and all objects are up-to-date once a reconcile process starts.
		switch v := obj.(type) {
		case *v1.StatefulSet:
			v.Status.Reset()
		case *v1.Deployment:
			v.Status.Reset()
		}

		if err := client.Update(ctx, obj); err != nil {
			return fmt.Errorf("failed to update object %T '%s': %v", obj, namespacedName.String(), err)
		}
	} else {
		if err := client.Delete(ctx, obj.DeepCopyObject()); err != nil {
			return fmt.Errorf("failed to delete object %T %q: %v", obj, namespacedName.String(), err)
		}
		if err := client.Create(ctx, obj); err != nil {
			return fmt.Errorf("failed to create object %T %q: %v", obj, namespacedName.String(), err)
		}
	}

	// Wait until the object we retrieve via "client.Get" has a different ResourceVersion than the old object
	updatedObjectIsInCache := waitUntilUpdateIsInCacheConditionFunc(ctx, client, namespacedName, existingObject)
	err = wait.PollImmediate(10*time.Millisecond, 10*time.Second, updatedObjectIsInCache)
	if err != nil {
		return fmt.Errorf("failed waiting for the cache to contain our latest changes: %v", err)
	}

	klog.V(2).Infof("Updated %T %s in Namespace %q", obj, obj.(metav1.Object).GetName(), obj.(metav1.Object).GetNamespace())

	return nil
}

func waitUntilUpdateIsInCacheConditionFunc(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	namespacedName types.NamespacedName,
	oldObj runtime.Object,
) wait.ConditionFunc {
	return func() (bool, error) {
		// Create a copy to have something which we can pass into the client
		currentObj := oldObj.DeepCopyObject()

		if err := client.Get(ctx, namespacedName, currentObj); err != nil {
			klog.Errorf("failed retrieving object %T %s while waiting for the cache to contain our latest changes: %v", currentObj, namespacedName, err)
			return false, nil
		}
		// Check if the object from the store differs the old object
		if !DeepEqual(currentObj.(metav1.Object), oldObj.(metav1.Object)) {
			return true, nil
		}
		return false, nil
	}
}

func waitUntilObjectExistsInCacheConditionFunc(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	namespacedName types.NamespacedName,
	obj runtime.Object,
) wait.ConditionFunc {
	return func() (bool, error) {
		newObj := obj.DeepCopyObject()
		if err := client.Get(ctx, namespacedName, newObj); err != nil {
			if kubeerrors.IsNotFound(err) {
				return false, nil
			}
			klog.Errorf("failed retrieving object %T %s while waiting for the cache to contain our newly created object: %v", newObj, namespacedName, err)
			return false, nil
		}
		return true, nil
	}
}
