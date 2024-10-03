# User Cluster RBACs
**Author**: Lukasz Zajaczkowski(@zreigz)

**Status**: Draft proposal.

## Introduction
Share cluster feature allows to share access to the user cluster with other users.
The owner of the cluster must grant other user some permissions. To do so, configure `kubectl` to point to the cluster and
create a `rolebinding` or `clusterrolebinding`, using the email address of the user the kubeconfig was shared.
We would like simplify this process.

## Core concept
The cluster owner will be able to create roles which could restrict the user from accessing specific resources.
For example, a role can be created which allows the user to view and update clusters but does not allow deleting them.
Roles can be created in cluster scope (impacts all namespaces) or namespace scope (impacts a specific namespace in which it is created).
In order to link the user to a role, the cluster owner has to create a binding.
We expose API endpoints for all the operation which are needed for managing Role/ClusterRole and bindings.
End-user will get UI interface to interact with RBACs on the cluster detail page.

## Implementation
First we have to start with managing Role/ClusterRole in the user cluster. A various endpoints will be created.

For Roles:

Create
```
POST /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles
```

List:
```
GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles
```

Get:
```
GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles/{namespace}/{role_id}
```

Update:
```
PATCH /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles/{namespace}/{role_id}
```

Delete:
```
DELETE /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/roles/{namespace}/{role_id}
```

For the ClusterRoles we have similar endpoints but without `{namespace}` path parameter.
