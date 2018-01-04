# Terminology
* `group`: _(*Kubernetes*)_ object that can be bound to a set of rules. (currently not in use)
* `rule`: _(*Kubernetes*)_ a manifest to specify actions a ServiceAcoount/User/Group is allowed to perform.
* `user`: a person that interacts with the system. A user can have different `groups`.
* `project`:  An abstraction to group different users and resources.
* `role`: Controls a set of actions the user can perform.
* `resource`: A _thing_ that will be owned by a `user` or a `project`. i.e A customer cluster

# Log from our discussion, user flow
* We use dex/(oauth) as AuthN source.
  + If the use is removed from the auth provider he shouldn't be able to login into our product.
  + We wan't to get AuthZ information from the auth provider (In a later stage) i.e Map LDAP groups to our permission system
* A `user` can create a project, a project will own multiple `resources`.
* A `user` can create `resource` that belong to the `user`.
* A `user` in a `project` can create a `resource` (i.e customer cluster / SSH key) that belongs to the `project`.
* A `user` is restricted in a project it's `roles`.
* `roles` are additive.
* A `user` can add other `users` to a`project`.
* A `user` can add other `users` to an `role` in the `project`.
* We wan't to have 4 different `role`s:
  + Owner (can do everything)
  + Admin (can add users...)
  + Editor (can create/delete resources...)
  + Viewer (can view resources...)
* `role` maps to the`project` resources.
* `role` maps to the internal customer cluser i.e A Viewer-`role` can see every Kubernetes resource in the cluster.
* A resource always has to belong to a `project`.
* When a user logs in for the first time a default project is created.

---
# Technical design
The general idea is to map `project`/`role` to Kubernetes `group` which are bound to `rule`s
![untitled drawing](https://user-images.githubusercontent.com/7387703/34309206-2c49e604-e751-11e7-8264-16ed5bca7ee1.jpg)
When a project gets created we generate all of it's rules in Kubernetes. When a `user` joins an `role` the rules get bound to the user. This also happens in all client cluster. The client cluster will also have the `role`s as rules but differently generated.
We use [User Impersonation](https://kubernetes.io/docs/admin/authentication/#user-impersonation) to perform user actions such as listing/editing `resources`. For that we can't use the K8s indexer which normally would heavily improve request times. We have to call the API server with every user request.
  
