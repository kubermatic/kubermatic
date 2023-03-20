# Cluster Defaulting & Validation

This document describes how KKP applies default values to Clusters and how validation works. This
document has been extracted from https://github.com/kubermatic/kubermatic/pull/8346.

## Basics

At the most basic, **all Cluster validation happens in `pkg/validation`**, just as **all defaulting
happens in `pkg/defaulting`**. The webhooks and the KKP API are simply calling the functions in
those packages.

Defaulting for `Seed` and `KubermaticConfiguration` objects lives in `pkg/controller/operator/defaults`,
but should probably be moved to the packages mentioned above, for consistency.

## Webhook Behaviour

It is vital to understand that Kubernetes will always call the mutating (defaulting) webhooks first
and call the validating webhooks last. This means the validation can always assume that defaulting
has already happened, i.e. `pkg/validation.ValidateClusterSpec()` will throw an error if no NodeportRange
is set in the Cluster. The NodeportRange is optional to the user, but the mutating webhook will default
it and the validating webhook will then ensure it.

TL:DR; First mutation, then validation.

## Defaulting

KKP offers 5 different sources of default values for a Cluster, in order of importance:

* the default `ClusterTemplate`, configured on the Seed object (importantly, the ClusterTemplate is **optional**)
* the `Seed` object's `spec.defaultComponentSettings`
* the `KubermaticConfiguration` object's `spec.userCluster`
* the chosen Cloud Provider can also default (though none of our providers actually do anything)
* Go constants in `pkg/controller/operator/defaults`

This means that if you want to apply default values, you need all 5 of these. For example if the
`ClusterTemplate` does not contain the number of APIserver replicas, we check the `Seed`, then the
`KubermaticConfiguration`, then fallback to the constants.

However, in our codebase we invert this flow: Whenever someone uses a `KubermaticConfigurationGetter`,
that thing will already apply the Go constants to the `KubermaticConfiguration`. So the config returned
by the getter is already defaulted.

### Important to know:
When the `Cluster.spec` is extended with new values that have defaults assigned, all existing Clusters are updated to set the default value.
If no initial defaults are set, keep in mind that a change later will again trigger an update to all existing Clusters.

Default values in `Cluster` objects are persisted, so that changed defaults **do not**
affect existing clusters. This is different to the `KubermaticConfiguration`/`Seed`, where defaulting
happens only at runtime, because we do want new defaults (like new `spec.versions` in the `KubermaticConfiguration`)
to also apply to existing KKP installations.

## Validation

Validation now means that we have a single set of validation rules that we use from everywhere in the
codebase. We do make a distinction between "validation NEW clusters" and "validate cluster updates", so
we can ensure immutability for certain fields.

### Open-API Validation

In order to combine your custom validation code with [kubebuilder markers](https://book.kubebuilder.io/reference/markers/crd-validation.html) from the CRD, the generic `openapi` package can be used. It will automatically create a validator for your object against the corresponding CRD.

When creating a new CRD, it is important to run `update-codegen.sh` once before developing. This is required as the openapi package needs a generated version of your CRD before it supports that type.

#### Validating Whole Objects

The openapi package can be easily integrated into your custom validation funcs. Here is an example for the ApplicationDefinition Type:

```Go
import (
	appskubermaticv1 "k8c.io/api/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/validation/openapi"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateApplicationDefinition(ad *appskubermaticv1.ApplicationDefinition) field.ErrorList {
	allErrs := field.ErrorList{}

	v, _ := openapi.NewValidatorForObject(ad)
	allErrs = append(allErrs, validation.ValidateCustomResource(nil, ad, v)...)

	// your custom validation code
	// ...

	return allErrs
```

#### Validating Child Elements

Validating child elements is a bit more tricky, because openapi can only validate starting from the root element. In these cases we need to wrap the child first and later strip out the results that do not match. Here is an example for `Cluster.Spec`:

```Go
func ValidateClusterSpec(spec *kubermaticv1.ClusterSpec, ...) field.ErrorList {
	allErrs := field.ErrorList{}
	filter := "spec" // this can be stacked using dot notation (e.g. "spec.cloud")

	cwrap := &kubermaticv1.Cluster{}
	cwrap.Spec = *spec
	v, _ := openapi.NewValidatorForObject(cwrap)

	res := validation.ValidateCustomResource(nil, cwrap, v)
	for _, e := range res {
		if strings.HasPrefix(e.Field, filter) {
			allErrs = append(allErrs, e)
		}
	}

	// your custom validation code
	// ...

	return allErrs
```

## KKP API

The KKP API does something funky: During cluster creation, it will create a `ClusterSpec` and then run the
validation logic on it. Any error there is then reported back up to the user. **This** is the one reason why
we had separate validation logics: One was only for a few fields that are relevant in the KKP dashboard, the
other was, well, other fields. However now that we only have 1 function for it, this means that the KKP API
also has to do defaulting before it can validate. And this leads to the new situation that the KKP API will
always call `seedClient.Create(ctx, cluster)` with a fully defaulted `Cluster` object. This has no visible
or noticeable side effects, it's just something "good to know".

Technically the API could just call `seedClient.Create()`, get the answer (from the webhook) and take any
errors from there and send them to the user, but this might require more response parsing and error handling.
So for now, inside the KKP API we still manually run the defaulting/validation.

## Unit Tests

The refactoring has shown lots and lots of unit tests that were technically broken. For example the tests for
the `PatchEndpoint()` started with incomplete clusters and then asserted incomplete patches. Now, because way
more validation rules apply everywhere, a bunch of tests had to extended to provide a `Seed`, a
`KubermaticConfiguration` etc.

Importantly, the tests for the webhooks break the mentioned "first mutate, then validate" rule: The validation
webhook's tests sometimes test clusters and situations that can be misleading if not understood in the full
context. For example there is a test that says "reject empty nodeport range", which is true: A `Cluster` must
have a nodeport range. But in reality, the mutating webhook will always ensure that it is set. But just from
reading the unit tests it could seem the opposite. I left a big fat comment to warn future devs.
