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
When a project gets created we generate all of it's rules in Kubernetes. When a `user` joins an `role` the rules get bound to the user. Outlook: "This also happens in all client clusters. The client cluster will also have the `role`s as rules but differently generated."
We use [User Impersonation](https://kubernetes.io/docs/admin/authentication/#user-impersonation) to perform user actions such as listing/editing `resources`. For that we can't use the K8s indexer which normally would heavily improve request times. We have to call the API server with every user request.
  

## Task & effort
* [ ] Write new API endpoints reflecting the new structure (mocks)(4h).
  * [ ] Write swagger annotations (1h).
* [ ] Rewrite old endpoints to reflect project ownage (mocks)(3-4h).
  * [ ] Write swagger annotations (1h).
* [ ] Write Project, User, Role CRD (1h)
* [ ] Write first Implementation (use SSHKeys) (4h)
  * [ ] When a user logs in Create a User Object
  * [ ] When Creating a SSHKey create/update a corresponding K8s Role + Role bindign to the User
  * [ ] When querrying use User Impersonation to get a list of the keys.
  * [ ] Set the ownerRef to the user when the user gets deleted, the key also gets deleted. 
* [ ] Evaluation of Impl (revisit) (2h)
* [ ] Adpot other resources (TBD<After Evalutation>)...
* [ ] Set owner references (2h)
* [ ] Migration scipts (4h)
