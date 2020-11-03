# User management
**Author**: Lukasz Szaszkiewicz(@p0lyn0mial)

**Status**: Draft proposal.

## Introduction

User management describes how to manage user access to various resources like devices, applications, networks and many more. It plays a core part in any system and it is a basic security essential for any organisation. At this moment `kubermatic-server` doesn’t allow for defining and assigning permissions to resources. In fact, the current situation is simple, users have unlimited access to `Resources` and cannot share them with others. One of the goals of this document is to change the steady state but at the same time we would like to introduce the concept of a `Project`. A `Project` is a new entity that will hold various `Resources`.  All `Resources` in a `Project` are equal in terms of the `Groups` attached to them. The `Groups` can be arbitrary but as a good starting point we will start off with `Owner`, `Editor` and `Reader` types. Affiliation of a `User` to one of the `Groups` give them certain powers they are allowed to use within a `Project`.  For example if Bob belongs to `Editor` he has write access to all `Resources` in a `Project`, in other words Bob can create `clusters` in a `Project` he belongs to.


## Goals
1. Describe how `kubermatic-server` is going to drive authorization decisions.
2. Utilise kubernetes `RBAC` mechanism as much as possible.
3. Describe how `RBAC` roles are generated and attached to project’s resources.

## Non-Goal
1. Describes how user management will work inside consumer clusters.


## Core concept

Since we are going to use kubernete’s `RBAC` as an underlying authorisation mechanism, we can bind roles to certain subjects. At this moment subjects can be groups, users or service accounts. We decided to bind to groups not only because it will be simpler (we don’t have to generate a set of roles for each `User`) but it also aligns with our own concept of `Groups`. In the `Introduction` section we said that “all `Resources` are equal in terms of the `Groups` attached to them.” What we mean by that is that we are going to create and maintain a set of fixed `RBAC Roles` for each `Resource` that belongs to a `Project`.  The number of `Roles` and their definitions springs directly from the number and the type of the `Groups`. For example the following `Role` was generated for `editors Group` and describes access to `cluster` `Resource`

```
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  namespace: kubermatic
  name: kubermatic:cluster:editors-htwhln8jnb
rules:
- apiGroups: ["kubermatic.k8s.io"]
  resources: [“clusters”]
  resourceNames: ["my-powerfull-cluster“]
  verbs: ["get", “update”, "create", "delete", "patch"]
```

If we were to generate a `Role` for a different `Group` let’s say `readers` the only difference would be in the `verbs` field and the `name`

```
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  namespace: kubermatic
  name: kubermatic:cluster:readers-htwhln8jnb
rules:
- apiGroups: ["kubermatic.k8s.io"]
  resources: [“clusters”]
  resourceNames: ["my-powerfull-cluster“]
  verbs: ["get"]
```

The next step is to create a connection between the `Groups` and the `Roles`. This essentially gives the `Groups` certain permissions to the `Resources`. In our case the number of `RoleBindings`  we have to create also stems directly from the types and the number of the `Groups` we have in our system. For example the following binding assigns `kubermatic:cluster:editors-htwhln8jnb` `Role` to `editors-projectIdentity` `Group`

```
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: kubermatic:cluster:editors-htwhln8jnb
  namespace: kubermatic
subjects:
- kind: Group
  name: editor-projectIdentity
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: cluster-editor-role-projectIdentity
  apiGroup: rbac.authorization.k8s.io

```

**Possible Optimization**:

It may turn out that the `verb` list for `Editor`, `Owner`, `Reader` is exactly the same in such case we could generate only one `Role` to rule them all.
Does it mean we don't need more than two Groups ? Not necessarly.
We could say that the `Editor`group is special in a way that allows them to add other users to the `Project`.
Similarly the `Owner` can be special in a way that would allow them to delete an instance of  `Project`.

This complexity could be enlosed in `ProjectProvider` and since `RBACGenerator` is parameter driven expressing the above statement will be possible.

One thing worth mentioning is that the names of the `Groups`, `Roles` and `RoleBindings` are unique, for more details please refer to `RBACGenerator` that contains more details about creating the `Roles` and assigning them to the `Groups`


## Implementation

Having a proper set of permissions in place that are attached to all `Resources` for each `Group` we have in our system is a foundation we will build our user management functionality on. In reality it boils down to assigning a `Group` (think unique name) to an authenticated `User`. From that moment we will query our system as Bob that belongs  to `Editors` group and let Kubernetes take care of enforcing authorisation decisions.

One way of doing this we would like to propose is to create a new type namely `Project`. For exact type definitions please refer to `Types` section. An instance of `Project` represents a unique project
and gives us an identity we will attach to project’s `resources`. At the same time we would like to extend existing `apiv1.User` type in a way that would allow us to store a list of `Projects` along with a `Group` a user belongs to. We will also provide a new implementation of `ClusterProvider` that will be different in two ways. Firstly, it will use a built-in admin account to get the list of `clusters` associated with the given project. Secondly, all the other operations like cluster creation, deletion and retrieval will use `Impersonation` and will accept the name of the group the user belongs to.

To recap when a user logs-in to our system this is what happens:
- a user is authenticated
- we retrieve a list of projects the user belongs to along with groups names from `apiv1.User`
- we map the user to a group.
- all the future queries will use `Impersonation` to authorise. To make this part more concrete imagine that we want to list clusters a user has access to. Since all `Resources` that belong to the `Project` will have the project's identity attached to them (be it `projectName` or `projectID`).
To find corresponding clusters we will use a built-in admin account to look up the data from the cache (`Informers`). Once we have the list of interesting resources as a sanity check we impersonate as Bob and try to get each cluster directly from the api-server. This logic will be enclosed in `ProjectProvider`.


To hide complexity from developers and to present a more hierarchical view we will create `ProjectProvider` which will accept a modified version of `ClusterProvider`, the new component will also rely on `RBACGenerator`.  The new component will not only make sure that queries are executed in the context of a user but it will also encapsulate resources and will provide convenient methods to operate on them. In the event of a resource creation or deletion the `ProjectProvider` will rely on `RBACGenerator` to create necessary `RBAC` roles.

## RBACGenerator
`RBACGenerator` is a separate component responsible for creating `Role` and `RoleBindings` for every `Group` we wish to have. It can be implemented as an active
reconciliation loop that constantly watches for project's resources. The controller pattern guarantees that required `RBAC` are in place
even in the event of failure, otherwise enforcing I want to "create a resource AND attach `RBAC`" semantic could be hard.

Initially the component could maintain the hardcoded list of `Groups`. By convention the names of groups are `owners-projectIdentity`, `editors-projectIdentity` etc.
The names of `Role` and `RoleBindings` are `prefix:resourceKind:groupName` and `prefix:resourceKind:groupName` respectively.
When the controller detects a new resource it will generate a pair of `Role` and `RoleBindings`. In order to do this it needs to determine
the list of valid `verbs` for each `Group`. This part could also be hardcoded and implemented as a `mapper` that accepts the `groupName` and spits out the `verbs`.
The next step requires `apiGroup`, `resourceKind`, `resourceName` and `verbs` to generate valid `Role`. The last step is to create `RoleBindings` this part
needs the name of the `Role` and the subject which in our case is the `groupName`.


## Types
TBD
