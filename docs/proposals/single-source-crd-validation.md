# Single Source of Truth for CR Validation

**Author**: @SimonTheLeg

**Status**: Draft proposal; prototype in progress

## Goals

Have a single source of truth from which CustomResources can be validated for our CustomResourceDefinitions.

## Motivation and Background

We have done a good job of unifying our CustomResource validation into a single package, but there is another challenge: Keeping consistency between our `+kubebuilder:validation` markers and the validation package. Because we want to directly call the validate func, both markers and the func must contain the same logic. In practice this can be quite difficult and work-intensive to achieve.

As an example, consider the  `spec.cniPlugin.type` field of our `cluster` object: For this field only the values “canal”, “cilium”, and “none” are valid. In order to achieve this we are using a [kubebuilder enum](https://github.com/kubermatic/kubermatic/blob/e911620f75a98106545a8b0cce40add8fd044987/pkg/apis/kubermatic/v1/cluster.go#L210). However as you can also call the validate func directly, we need the same validation logic in that function as well. Currently we have basically duplicated the logic ([see example for enum](https://github.com/kubermatic/kubermatic/blob/0eeeeedac1712e68dfc4f71a612ebcd29fd5ff7a/pkg/validation/cluster.go#L75)). As a result, every time you touch either the kubebuilder markers or the validation func, you need to make sure that both are in sync.

## Implementation

So what if we could use the kubebuilder (or to be more precise OpenAPI v2 and v3) validations  in our custom validate funcs directly?

This can be achieved by using `kube-openapis` validate package. We can use it to create SchemaValidators directly from CRDs. This can be done fully client-side, without requiring any kubernetes-api-server.

A full example for validating CRs can be found at [https://github.com/SimonTheLeg/kubermatic/blob/crd-validation-single-source/validationtest/validation_test.go](https://github.com/SimonTheLeg/kubermatic/blob/crd-validation-single-source/validationtest/validation_test.go). It provides a generic `validatorFromCRD` func, which can create OpenAPI validators based on a CRD. The validators then can be used to validate any CR. Additionally an example to validate sub-fields (e.g. `.spec`) is included.

The aforementioned logic should fit quite well into our current model, as `validation.ValidateCustomResource` returns a `field.Errorlist` which we can simply append to our `allErrs` convention field. Additionally it also supports custom parent fields. We only have to be aware that it works on the CRD itself, so we need to make the yaml/json representation of the CRD available to our validate funcs. Using something like GOs file embedding or pulling the CRD directly from kubernetes-api would work.

**Advantages**

- validation between the crd and our validate func will always be in sync ⟹ our validate func will be the single source of truth
- we save a lot of custom validation code

## Alternatives considered

1. don’t use `+kubebuilder:validation` markers at all and only validate using the validation func ⟹ While this gives you single source of validation, I feel like it would take a lot of boilerplate code
2. keeping the markers, but generate the validation logic directly from the kubebuilder annotations ⇒ While I think this might be possible, it is a lot more difficult to implement. As it basically builds on top of this proposal, I wanted to get this proposal underway first, so I don’t spend unnecessary effort on building something more complicated

## Task & effort

Will depend on the decided approach
