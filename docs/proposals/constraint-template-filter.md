# Constraint Template filter

**Author**: Lovro Sviben (@lsviben)

**Status**: Draft proposal; Prototype in progress.

## Motivation and Background

The goal is to allow admins to decide to which OPA-integration enabled clusters they want the Constraint Templates to be synced to. This would allow them greater flexibility when managing cluster constraints, and will help with default constraints as well. 

## How it currently works

Currently all the Kubermatic Constraint Templates present on the seed cluster are reconciled as Gatekeeper Constraint Templates on the OPA-integration enabled user clusters through a controller in the `seed controller`.


## Implementation proposal

The idea is to add a series of filters in the Kubermatic Constraint Template CRD that would be executed on the `seed controller`. For now the plan is to have 2 filters, but it could be easily extended to more:

1. A `label selector` which would just use the Kubernetes label selector that would choose the clusters based on their labels

2. `Provider filter` which would filter clusters based on the provider


Changes needed to Constraint Template CRD:

```
apiVersion: kubermatic.k8s.io/v1
kind: ConstraintTemplate
metadata:
  name: k8srequiredlabels
spec:
  selector:
    labelsSelector:
      matchLabels:
        component: redis
      matchExpressions:
        - {key: tier, operator: In, values: [cache]}
        - {key: environment, operator: NotIn, values: [dev]}
    provider:
      - aws
      - gcp
  crd:
    spec:
      names:
        kind: K8sRequiredLabels
      validation:
        # Schema for the `parameters` field
        openAPIV3Schema:
          properties:
            labels:
              type: array
              items: 
                type: string
  targets:
    - target: admission.k8s.gatekeeper.sh
      rego: |
        package k8srequiredlabels
        violation[{"msg": msg, "details": {"missing_labels": missing}}] {
          provided := {label | input.review.object.metadata.labels[label]}
          required := {label | label := input.parameters.labels[_]}
          missing := required - provided
          count(missing) > 0
          msg := sprintf("you must provide labels: %v", [missing])
        }
```

Also changes will be needed in the Cluster management to allow the management of Cluster object labels.


## Task & effort

* Extend Constraint Templates with filters - 3d
* Extend Cluster management to include labels - 2d
* Implement Constraint Template filtering in the constraint template controller - 4d
* UI changes - 4-5d?

