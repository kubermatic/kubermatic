# Resource handling in the Kubermatic codebase

One of Kubermatic's main tasks is to reconcile Kubernetes resources.
The following will describe how we structure the code in Kubermatic to achieve a unified handling of various Kubernetes resources from different controllers.

## Table of contents
1. [Introduction](#introduction)
2. [Generic reconcile functions](#general-function-definitions)
    1. [ObjectCreator](#objectcreator)
        1. [Example](#example-implementation)
    2. [EnsureNamedObject](#ensurenamedobject)
3. [Typed reconcile functions](#reconcilesecrets-aka-reduce-the-type-casting)
    1. [SecretCreator](#secretcreator)
    2. [NamedSecretCreatorGetter](#namedsecretcreatorgetter)
    3. [Example](#example-namedsecretcreator-implementation)
    4. [Example with data](#template-data)
    5. [Reconcile creators inside a controller](#reconcile-the-resources-in-a-controller)
    6. [Wrap/Modify existing objects](#wrapmodify-existing-objects)
        1. [ObjectModifier](#objectmodifier)
        2. [Wrap the typed creator](#wrap-the-typed-creator)
    7. [Extend the codegen / add new typed reconcile functions](#extend-the-codegen)

## Introduction

Our reconciling is designed in a way, that we get notified whenever an object changes, compare it to a desired state and if there is a diff, update it.
The desired state comes from the [ObjectCreators](#objectcreator) we define.

Kubermatic is currently going through some heavy refactoring which moves all resource handling to the `Named*Creator` functions.
Additionally we're migrating our code to use the [controller-runtime libraries](https://github.com/kubernetes-sigs/controller-runtime).

## General function definitions

### ObjectCreator

ObjectCreator is a function definition to handle the create/update of a runtime.Object.
It uses `runtime.Object` as that is the minimal interface which gets satisfied by all Kubernetes objects.
That way we can reconcile all Objects within a single function.
`existing` will be the already existing object coming from the cache. If no object is found in the cache `existing` is nil.
```go
type ObjectCreator = func(existing runtime.Object) (runtime.Object, error)
```

### EnsureNamedObject

`EnsureNamedObject` is a generic "reconcile" function which will update the existing object or create it.
If the existing object does not differ from the "wanted" object, `EnsureNamedObject` will not issue any API call.

```go
func EnsureNamedObject(ctx context.Context, namespacedName types.NamespacedName, rawcreate ObjectCreator, client ctrlruntimeclient.Client, emptyObject runtime.Object) error
```

Though its to be noted that it's not recommended to use `EnsureNamedObject` directly. Instead prefer the [Reconcile* functions](#reconcilesecrets-aka-reduce-the-type-casting)


#### Example implementation

The following will demonstrate a potential `ObjectCreator` implementation.
```go
func MyWonderfulConfigMap(existing runtime.Object) (runtime.Object, error) {
	var cm *corev1.ConfigMap
	if existing == nil {
		// In case there is no already existing object to update
		cm = &corev1.ConfigMap{}
	} else {
		// We case for better usability here
		cm = existing.(*corev1.ConfigMap)
	}

	cm.Name = "wonderful-config"
	cm.Data = map[string]string{
		"config.yaml": "some-config",
	}

	return cm, nil
}
```

When having to maintain multiple resources the first 7 lines start to become an error prone & annoying boilerplate code.
For this we created the typed `Reconcile*` functions.

## ReconcileSecrets aka "Reduce the type casting"

To avoid certain boilerplate code we introduced typed `Reconcile*` functions.
Those functions are being generated using `go generate`: https://github.com/kubermatic/kubermatic/tree/master/api/codegen/reconcile
The result is written into: https://github.com/kubermatic/kubermatic/blob/master/pkg/resources/zz_generated_reconcile.go

`ReconcileSecrets` is a typed convenience function around `EnsureNamedObject`.
It offers:
- Typed creator functions `SecretCreator` (`func(existing *corev1.Secret) (*corev1.Secret, error)`)
- Automatic nil checks & struct initialization
- Informer allocation from a passed in `InformerFactory`
- Unified modifier functions to allow to apply certain modifications to all passed in `SecretCreator` functions
- Setting resource name based on the name coming from the `NamedSecretCreatorGetter`


```go
func ReconcileSecrets(ctx context.Context, namedGetters []NamedSecretCreatorGetter, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error
```

### SecretCreator

A typed creator function. Prefer the [NamedSecretCreatorGetter](#namedsecretcreatorgetter) instead.
```go
type SecretCreator = func(existing *corev1.Secret) (*corev1.Secret, error)
```

### NamedSecretCreatorGetter

The `NamedSecretCreatorGetter` is a simple function definition to combine the name of the resource + the creator function.
This avoids the need to call the creator function twice (1st time to get the objects name + second time to get the actual creator)

```go
type NamedSecretCreatorGetter = func() (name string, create SecretCreator)
```

### Example NamedSecretCreator implementation

```go
func MyWonderfulSecretCreator() reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return "my-wonderful-secret", func(existing *corev1.Secret) (*corev1.Secret, error) {

			// We don't need to set the object's name, as that's being done by the typed Reconcile function.
			existing.Data = map[string][]byte{
				"user":     []byte("foo"),
				"password": []byte("bar"),
			}

			return existing, nil
		}
	}
}
```

The namespace of your object should not be set in the Creator, but rather passed in as an argument to the typed Reconcile func (in our example `ReconcileSecrets`)

### Template data

Some resources require some dynamic data during reconciling (Such as other Services, Secrets, Configmaps, etc.).
As the creator function does not allow passing it arbitrary data, data must be injected by using a closure.
This avoids to have controller specific creator functions.

```go
func MyWonderfulSecretCreator(data dataProvider) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return "my-wonderful-secret", func(existing *corev1.Secret) (*corev1.Secret, error) {

			// We don't need to set the object's name, as that's being done by the typed Reconcile function.
			existing.Data = map[string][]byte{
				"user":     []byte(data.GetClusterUsername()),
				"password": []byte(data.GetClusterPassword()),
			}

			return existing, nil
		}
	}
}
```

### Reconcile the resources in a controller

```go
creators := []NamedSecretCreatorGetter{
	MyWonderfulSecretCreator(data)
}

// controller-runtime controller:
client := mgr.GetClient()

if err := reconciling.EnsureNamedObjects(ctx, client, "some-namespace", creators); err != nil {
	return fmt.Errorf("failed to reconcile Secrets: %w", err)
}
```

### Wrap/Modify existing objects

For wrapping/modifying existing resources we have 2 options:
- Pass `ObjectModifier` functions to the typed `Reconcile*` functions
  Good if you want to modify all resources of a specific type
- Wrap the typed `*Creator` function.
  Good if you want to modify a single resource

#### ObjectModifier

Every `Reconcile*` functions has a variadic parameter called `objectModifiers`.
All passed in `*Creator` functions are being wrapped by the passed in `objectModifiers`.

Example:

```go
// ClusterRefWrapper is responsible for wrapping a ObjectCreator function, solely to set the OwnerReference to the cluster object
func ClusterRefWrapper(c *kubermaticv1.Cluster) ObjectModifier {
	return func(create ObjectCreator) ObjectCreator {
		return func(existing runtime.Object) (runtime.Object, error) {
			obj, err := create(existing)
			if err != nil {
				return obj, err
			}

			obj.(metav1.Object).SetOwnerReferences([]metav1.OwnerReference{GetClusterRef(c)})
			return obj, nil
		}
	}
}
```

#### Wrap the typed creator

To apply a modification only to single a resource function it can be wrapped:

```go
func MyWonderfulMyNewTypeCreator(data dataProvider) MyNewTypeCreator {
	return func(existing *myapiv1.MyNewType) (*myapiv1.MyNewType, error) {
		existing.Foo = "bar"
		return existing, nil
	}
}

func WrappedMyWonderfulMyNewTypeCreator(create MyNewTypeCreator) MyNewTypeCreator {
	return func(existing *myapiv1.MyNewType) (*myapiv1.MyNewType, error) {
    existing, err := create(existing)
    if err != nil {
      return nil, err
    }

    existing.Foo = "baz"
    return existing, nil
	}
}
```

### Extend the codegen

As mentioned [above](#reconcilesecrets-aka-reduce-the-type-casting), all typed reconcile functions are being created using code generation.
To add a new type, the type must be added to the code generation first.

Extend the `Resources` slice with the additional item in the [code](https://github.com/kubermatic/kubermatic/blob/master/codegen/reconcile/main.go):
```go
		Resources: []reconcileFunctionData{
			{
				ResourceName:       "MyNewType",
				ImportAlias:        "myapiv1",
				// ResourceImportPath must only be defined once inside the Resources slice.
				// If it has already been defined, just omit it here.
				ResourceImportPath: "my.io/api/v1",
				// New resources should use Named*Creator's
				UseNamedObject: true,
			},
			//...
```

Run the code generation from the repository root:
```bash
go generate api/pkg/resources/reconciling/ensure.go
```

See [here](#example-namedsecretcreator-implementation) on how to use the generated functions.
