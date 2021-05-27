## Introduction

A cluster template is a reusable cluster template object. The Kubernetes CRD will be used for this purpose.
A cluster template allows you to specify a provider, node layout, and configurations to materialize a cluster instance
via Kubermatic API or UI.

During cluster creation or after successful creation the user can store cluster definition as a template.

### Scope

The cluster templates should be accessible from different levels. Should have global (admin), project or user scope.
Template management will be available from project level.
We also need a central template management place for admins.

### Credentials

For security reasons, the cluster template should contain a preset name to retrieve credentials. The user should be able
to use their own credentials. In this case, the preset logic will be used. Credentials will copy to internal preset with a special
label and unique name. The label will indicate if the preset is attached to the cluster templates or it's a global preset.

### Update

We will reuse the existing cluster wizard for templates.


### Creating and Using Templates
During the cluster creation process the user should be able to pick the desired template and specify number of instances
and nodes or use fixed sizes (S, M, L defined in global settings).

The cluster templates should be filtered out by a provider, scope, name.

It can be easily achieved by using k8s labels. Templates can include a set of labels. Defining a label in this way makes
it easy for users to find and manage templates. The API should have some mechanism for labeling cluster templates.

```
type ClusterTemplate struct {
	ObjectMeta      `json:",inline"`
	Labels          map[string]string `json:"labels,omitempty"`
	Credential      string            `json:"credential"`
	Spec            ClusterSpec       `json:"spec"`
}

```




