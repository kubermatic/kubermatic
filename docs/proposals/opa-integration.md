# OPA integration

**Author**: Lovro Sviben (@lsviben)

**Status**: Draft proposal; prototype in progress.

## Goals

The goal is to integrate [OPA](https://www.openpolicyagent.org/) into Kubermatic so that we can enable users to
manage their OPA policies through the Kubermatic platform. Furthermore, the idea is to give users policy proposals 
for clusters based on what policies are present on other clusters owned by the user.

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

To integrate OPA with Kubermatic we will need to:
- implement API endpoints and API CRDs for ConstraintTemplates, Configs and Constraints
- set Gatekeeper as a default addon - if we want OPA integration by default
- implement a client for managing Gatekeeper ConstraintTemplates, Configs and Constraints on the user cluster
- implement a controller for syncing API CRDs to the user cluster Gatekeeper CRDs
- implement the dashboard for OPA integration
- implement policies proposal to users based on their other policies

High level overview on how it would work is that when a user cluster is created its deployed with Gatekeeper. The user can then
use Kubermatic dashboard or API to manage the ConstraintTemplates, Configs and Constraints for its cluster. The user is offered 
ConstraintTemplates and Constraints based on the ones he has in different clusters, or some example Kubermatic ones.
The API creates the Kubermatic CRDs for a user cluster in the user clusters namespace. From there a controller reconciles them
 to the user cluster as the real Gatekeeper objects.

### Implementing API and points and API CRDs

Kubermatic needs to implement CRUD API endpoints for ConstraintTemplates, Configs and Constraints. Also 
the API CRDs need to be implemented:

```
//+genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ConstraintTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   opa.ConstraintTemplateSpec   `json:"spec"`
	Status opa.ConstraintTemplateStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConstraintTemplateList specifies a list of constraintTemplates
type ConstraintTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ConstraintTemplate `json:"items"`
}

//+genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Config struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec opaconf.ConfigSpec `json:"spec,omitempty"`
}

// ConfigList specifies a list of configs
type ConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Config `json:"items"`
}

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

// ConfigList specifies a list of constraints
type ConstraintList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Config `json:"items"`
}

```

### Offering the user example policies based on cluster, project

Users will be offered policies(constraintTemplates) based on the cluster they creating them for:
- based on other clusters in the project
- based on other projects they have access to

this can be achieved using project labels on the ClusterTemplates. Similar can be done for the config, if needed.

For Constraints, they can be offered based on the ConstraintTemplate used in the cluster. Meaning if a ConstraintTemplate is used in multiple clusters,
the Constraints used in those clusters can be offered as an option to users.

## Outstanding questions and possible issues

1. Does a user need to have ConstraintTemplates, Constraint and Configs offered based on activity on other clusters,
or is it enough to offer just some example clusters. This is important because it is the reason we need local Kubermatic CRDs, to make it easy to search. 
Otherwise we could just have the endpoints directly manage the Gatekeeper CRDs on the user cluster.

2. Should OPA integration be implemented in user clusters by default or should users have a choice?

3. Is it ok that we just support Gatekeeper v3?

4. If we use the controller option, what to do if the user modifies the Gatekeeper CRDs on the user cluster. The Kubermatic ones on the seed cluster will get out of sync.





