# OPA integration

**Author**: Lovro Sviben (@lsviben)

**Status**: Draft proposal; prototype in progress.

## Goals

The goal is to integrate [OPA](https://www.openpolicyagent.org/) into Kubermatic so that we can enable users to
manage their OPA policies through the Kubermatic platform.

## Prerequisites

For OPA integration we would use the OPA with [Gatekeeper](https://github.com/open-policy-agent/gatekeeper) which means Gatekeeper needs to be installed on the user cluster.
We already have gatekeeper as an optional addon, but we can change it to default. We would be using Gatekeeper v3.

## How OPA works

In a nutshell, OPA is used to create and enforce policies. Example:

- All images must be from approved repositories
- All ingress hostnames must be globally unique
- All pods must have resource limits
- All namespaces must have a label that lists a point-of-contact

More info about what it does and how it works is in the OPA docs linked above, as well as in the
 [Kubernetes OPA blog post](https://kubernetes.io/blog/2019/08/06/opa-gatekeeper-policy-and-governance-for-kubernetes/).
 
For Kubermatic, what is important is that OPA defines 2 CRDs:
- ConstraintTemplate - Defines the policy template with REGO(policy language)
- Config - sets sync config for Gatekeeper, to allow for evaluating or accessing resources

Each ConstraintTemplate which is created Gatekeeper dynamically creates a new CRD for the Constraint with the CRD name being the ConstraintTemplate name.
By using these Constraint CRDs we can set the parameters for the ConstraintTemplates and thus enforce the policy.

## Implementation

High level overview on how it would work is that a seed(master) cluster has a list of some default ConstraintTemplates deployed,
similar to RBACs. Admins can add more and these are shared across all user clusters. When a user cluster is created it's deployed 
with Gatekeeper. A controller reconciles all the seed ConstraintTemplates to the user cluster, so that its gatekeeper has them. 
The user can then manage Configs and Constraints for its cluster using Kubermatic dashboard or API. 


To integrate OPA with Kubermatic we will need to:
- deploy Gatekeeper by default
- implement a default list of ConstraintTemplates that is deployed with Kubermatic
- implement API endpoints and API structs for ConstraintTemplates, Configs and Constraints
- implement a CRD for Constraints, which will be in the user-cluster namespace and a controller which will sync it to the user cluster
- implement a controller for managing Gatekeeper ConstraintTemplates and Configs on the user cluster
- implement the dashboard for OPA integration

### Implementing API and points and Constraints CRDs

Kubermatic needs to implement CRUD API endpoints for ConstraintTemplates, Configs and Constraints. Also 
the Constraint CRD need to be implemented.

Possible endpoints:
- Constraints `/projects/{project_id}/clusters/{cluster_id}/constraints`
- Configs `/projects/{project_id}/clusters/{cluster_id}/configs` - maybe configs is too general?
- ConstraintTemplates `/constrainttemplates`

Constraint CRD
```
//+genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Constraint struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	ConstraintSpec ConstraintSpec `json:"spec,omitempty"`
}

type ConstraintSpec struct {
	// type of gatekeeper constraint that the constraint applies to
	ConstraintType string `json:"constraintType"`
	Match      `json:"match,omitempty"`
	Parameters interface{} `json:"parameters,omitempty"`
}

type Match struct {
	Kinds             []Kind               `json:"kinds,omitempty"`
	Scope             string               `json:"scope,omitempty"`
	Namespaces        []string             `json:"namespaces,omitempty"`
	LabelSelector     metav1.LabelSelector `json:"labelSelector,omitempty"`
	NamespaceSelector metav1.LabelSelector `json:"namespaceSelector,omitempty"`
}

type Kind struct {
	Kinds     string `json:"kinds,omitempty"`
	ApiGroups string `json:"apiGroups,omitempty"`
}

// ConstraintList specifies a list of constraints
type ConstraintList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Config `json:"items"`
}

```

### Possible extension - Offering the user example constraints based on cluster, project

As a possible future extension, constraint can be offered to users based on their access to other projects and clusters.

## Outstanding questions and possible issues

1. Should OPA integration be implemented in user clusters by default or should users have a choice?

2. Is it ok that we just support Gatekeeper v3?

3. When rolling out the new Kubermatic version with gatekeeper, the user clusters which already exist will get the gatekeeper deployed.
But there is a possibility that they already have their managed gatekeeper, what to do then?




