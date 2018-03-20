# User Management

**Author**: Luk Burchard (@realfake)
**Status**: Draft proposal

User managemet allows us to group and protect resources between users. We introduce projects as a new kind of grouping resources, most resource ownage will move from a single users to projects managed by other users, with e.g more privillages (It introduces privilage hirachies) which is wanted in an organisation.

## Motivation and Background

# Terminology
* `group`: _(*Kubernetes*)_ object that can be bound to a set of rules. (currently not in use)
* `rule`: _(*Kubernetes*)_ a manifest to specify actions a ServiceAcoount/User/Group is allowed to perform.
* `user`: a person that interacts with the system. A user can have different `groups`.
* `project`:  An abstraction to group different users and resources.
* `role`: Controls a set of actions the user can perform.
* `resource`: A _thing_ that will be owned by a `user` or a `project`. i.e A customer cluster

# Log from our discussion, user flow
* We use [dex/(oauth)](https://github.com/coreos/dex) as AuthN source.
  + If the user is removed from the auth provider he shouldn't be able to login into our product.
  + We wan't to get AuthZ information from the auth provider (In a later stage) i.e Map LDAP groups to our permission system
* A `user` can create a project, a project will own multiple `resources`.
* A `user` can create `resource` that belong to the `user`.
* A `user` in a `project` can create a `resource` (i.e customer cluster / SSH key) that belongs to the `project`.
* A `user` is restricted in a project by it's `roles`.
* `roles` are additive.
* A `user` can add other `user`s to a`project`.
* A `user` can add other `user`s to an `role` in the `project` (only if he is permitted to do so).
* We wan't to have 4 different `role`s:
  + Owner (can do everything)
  + Admin (can add users...)
  + Editor (can create/update/delete resources...)
  + Viewer (can only view resources...)
* `role` maps to the`project` resources.
* Outlook: A `role` maps to the internal customer cluser i.e An Admin-`role` can see/create/edit/... every Kubernetes resource in the cluster.
* A resource always has to belong to a `project`.
* When a user logs in for the first time a default project is created.

---
# Technical design
The general idea is to map `project`/`role` to Kubernetes `group` which are bound to `rule`s
![untitled drawing](https://user-images.githubusercontent.com/7387703/34309206-2c49e604-e751-11e7-8264-16ed5bca7ee1.jpg)
When a project gets created we generate all of it's rules in Kubernetes. For this we will have a predefinded set of Roles which are used to template kubernetes `rules`. The roles will act as a template to allow subpath restrictions i.e `/path/.../created-resource/subpath` to allow this we need simple templating (only of simple strings not objects).
We generate a `rule` + RoleBinding(to group) for each rendered `role`
We do apply the templates to every cluster, in the Future we can route the rules to the corret seed-clusters.
We use [User Impersonation](https://kubernetes.io/docs/admin/authentication/#user-impersonation) to perform user actions such as creating/deleting/editing `resources`.
List users: When setting `roles` for a specific path in kubernets it won't be filtered when you list them. As a solution we label each resource with the project id (this also helps dev's to find resources manually). When listing resources we query with the project label and filter with the [SelfSubjectAccessReview](https://github.com/kubernetes/client-go/blob/42a124578af9e61f5c6902fa7b6b2cb6538f17d2/kubernetes/typed/authorization/v1/selfsubjectaccessreview_expansion.go#L24) call on each object.

Sample code to generate user _login_ config:

```
active_groups=[]
foreach g in group with labels project=projectid:
  if user in g:
    active_groups += id of g
```

Sample code to list resources:
```
allowed_resources=[]
foraech r in list all resources with labels project=projectid:
  if through impersonation -> SelfSubjectRulesReview r:
    allowed_resources += r 
```

Example `Group`:
```yaml
apiVersion: kubermatic.k8s.io/v1
kind: Group
metadata:
  labels:
    project: some-id
  name: editor
spec:
  roleBindings:
  - role-id1
  - role-id2
  userBindings:
  - user-id1
  - user-id2
  # Have this in all clusters usefull for single cluster testing (debuggable).
  # default: true
  propergate: true
```

Example `Role`:
```yaml
apiVersion: kubermatic.k8s.io/v1
kind: Role
metadata:
  labels:
    project: some-id
  name: role-name1
spec:
  # We use the PolicyRules here: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.9/#policyrule-v1beta1-rbac
  policyies:
  - apiGroups: ["kubermatic.k8s.io"]
    resources: ["clusters"]
    verbs: ["get", "delete"]
    resourceNames: [] # WILL be TEMPLATED/AUTOMATICALLY FILLED
```

Example `User`:
```yaml
apiVersion: kubermatic.k8s.io/v1
kind: User
metadata:
  name: name123
spec:
  ...
```

Out of this we generate the appropriate K8s resources that reflect the state described (Case cluster is added):
```yaml
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  labels:
    project: some-id
  namespace: default
  name: xxxxxxx
rules:
- apiGroups: ["kubermatic.k8s.io"]
  resources: ["clusters"]
  verbs: ["get", "delete"]
  resourceNames: ["cluster-just-created"]
---

kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  labels:
    project: some-id
  namespace: default
  name: xxxxxxx
subjects:
- kind: Group
  name: editor
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: generated-role-id1
  apiGroup: rbac.authorization.k8s.io
```



## Task & effort
* [ ] Write new API endpoints reflecting the new structure (mocks).
  * [ ] Write swagger annotations.
* [ ] Rewrite old endpoints to reflect project ownage (mocks).
  * [ ] Write swagger annotations.
* [ ] Write Project, User, Role CRD.
* [ ] Write first Implementation (use SSHKeys).
  * [ ] When a user logs in Create a User Object.
  * [ ] When Creating a SSHKey create/update a corresponding K8s Role + Role bindign to the User.
  * [ ] When querrying use User Impersonation to get a list of the keys.
  * [ ] Set the ownerRef to the user when the user gets deleted, the key also gets deleted. 
* [ ] Evaluation of Impl (revisit). 
* [ ] Adpot other resources (TBD<After Evalutation>)...
* [ ] Set owner references. 
* [ ] Migration scipts.
