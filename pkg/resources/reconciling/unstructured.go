package reconciling

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// UnstructuredCreator defines an interface to create/update Unstructureds
type UnstructuredCreator = func(existing *unstructured.Unstructured) (*unstructured.Unstructured, error)

// NamedUnstructuredCreatorGetter returns the name of the resource and the corresponding creator function
type NamedUnstructuredCreatorGetter = func() (name, kind, apiVersion string, create UnstructuredCreator)

// UnstructuredObjectWrapper adds a wrapper so the UnstructuredCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func UnstructuredObjectWrapper(create UnstructuredCreator, emptyObject *unstructured.Unstructured) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*unstructured.Unstructured))
		}
		return create(emptyObject)
	}
}

// ReconcileUnstructureds will create and update the Unstructureds coming from the passed UnstructuredCreator slice
func ReconcileUnstructureds(ctx context.Context, namedGetters []NamedUnstructuredCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, kind, apiVersion, create := get()
		if kind == "" || apiVersion == "" {
			return fmt.Errorf("both Kind(%q) and apiVersion(%q) must be set", kind, apiVersion)
		}

		emptyObject := &unstructured.Unstructured{}
		emptyObject.SetKind(kind)
		emptyObject.SetAPIVersion(apiVersion)

		createObject := UnstructuredObjectWrapper(create, emptyObject)
		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, emptyObject, false); err != nil {
			return fmt.Errorf("failed to ensure Unstructured %s.%s %s/%s: %v", kind, apiVersion, namespace, name, err)
		}
	}

	return nil
}
