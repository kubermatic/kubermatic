# Enforce allowed registries with OPA

**Author**: Lovro Sviben (@lsviben)

**Status**: Draft proposal;

## Motivation and Background

The current OPA integration in KKP allows for the creation of any kind of OPA policies. But we would like to make it more
user-friendly and accessible for users then just making them learn REGO or look for a solution online.
One of the more useful policies that could be enforced is the enforcement for allowed registries in all user clusters.

## Overview

Currently OPA integration in KKP supports:
- managing OPA Constraint Templates across user clusters - as admin
- managing OPA Constraints for user clusters - as cluster admin
- managing Default OPA Constraints for all user clusters - as admin (in progress)
- Constraint Template and Constraint filtering - as admin, in EE version

With this, and especially Default Constraints, we can allow admins to easily enforce policies across all or selected user clusters.

## Goals

Add a feature for admins to add a list of allowed registries and enforce them across user clusters

## Prerequisites

Default Constraints (in progress)

## Implementation

The idea is to add a AllowedRegistries CRD and a controller which will handle the CRDs and create appropriate Constraint Templates and
Constraints.

### AllowedRegistries CRD and controller

The CRD would look something like this:

```yaml
apiVersion: kubermatic.k8s.io/v1
kind: AllowedRegistry
metadata:
  name: allowed-registy
spec:
  registries: ["myharborinstance.com/", "quay.com"]
  selector:
    labelSelector:
      matchLabels:
        filtered: "true"
    providers:
      - azure
      - aws

```

And the controller would run in the master cluster and handle the Constraint Template and Constraint management.

### Enforcement through OPA

The underlying enforcement of the allowing registries would be done through the creation of a AllowedRegistry Constraint Template and corresponding
Default Constraint. The Constraint Template would be something along the lines of this REGO:

```
package kubernetes.admission                                              

deny[msg] {                                                                
  input.request.kind.kind == "Pod"                                        
  image := input.request.object.spec.containers[_].image                   
  not startswith(image, "hooli.com/")                                       
  msg := sprintf("image '%v' comes from untrusted registry", [image])       
}
```

But with parametrized registry name list through Constraints. Creation of the proper REGO will be one of the tasks that need to be done.

The allowed registry controller will ensure the Constraint Template and the Default Constraint, and the CT and Default Constraint Controllers will take care of the rest.

### Additional options

We could also add Cluster filtering to the allowed registry, which would be done through the Default Constraint filtering, to allow
admins to fine tune which clusters they want to target. 


### API/UI 

The allowed registry feature will be available for KKP admins through a new endpoint and through the UI. The UI team will need to decide where to 
place the new feature.

## Tasks

1. AllowedRegistry CRD
2. AllowedRegistry controller
3. AllowedRegistry REGO for the Constraint Template
4. AllowedRegistry API
5. AllowedRegistry UI

