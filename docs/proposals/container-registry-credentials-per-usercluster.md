# Container Registry Credentials per User Cluster

**Author**: Moritz Bracht (@dermorz)

**Status**: Draft proposal; prototype in progress.


* [Goals](#goals)
* [Non-Goals](#non-goals)
* [Motivation and Background](#motivation-and-background)
* [Implementation](#implementation)
* [Alternatives considered](#alternatives-considered)
* [Looking ahead](#looking-ahead)
* [Tasks and effort](#tasks-and-effort)

## Goals

The goal of this proposal is to add the ability to define default container registry credentials per
user cluster. Cluster owners can for example make use of their Docker Hub credentials to avoid image
pull limits or to access private images.

## Non-Goals

Setting local mirrors or private container registries for KKP components or Addons.

## Motivation and Background

Based on the user story [#6231][] users want to be able to use their Docker Hub paid accounts to
mitigate image pull limits. Setting this up cluster wide is currently not possible.

### Current state

Currently, it's possible to set `spec.imagePullSecret` in the `KubermaticConfiguration`. The
dockerconfig-json defined on this key is used in the Kubermatic Operator where the master- and the
seed-controller create a `dockerconfigjson`-secret respectively. When creating the deployments for
KKP master and seed components this secret gets referenced as `imagePullSecret` in all of their
pod specs.

This dockerconfig has no effect on anything running on the user clusters, so currently KKP does not
offer a way to manage container registry credentials to be used by KKP internal components, KKP
Addons nor user deployed applications by default.

### Proposed feature

`Cluster` resources get a new field `imagePullSecrets` that resembles `imagePullSecrets` on other
resources like Deployments or Pods. The secrets referenced here are being passed to the
machine-controller, which writes registry auth configs for the used container runtimes on the worker
nodes of the user clusters. All pods created on worker nodes can use any registries configured this
way.

There is also the possibility to define KKP wide default credentials in the
`KubermaticConfiguration` at `spec.userCluster.imagePullSecrets`. If defined, these secrets will be
used for any user cluster that has no explicit `imagePullSecrets` set.

## Implementation

### KubermaticConfiguration

A new configuration field to set default fallback registry credentials for any user cluster where no
credentials are explicitly selected. `spec.imagePullSecret`:

```yaml
apiVersion: kubermatic.k8c.io/v1
kind: KubermaticConfiguration
metadata:
  name: <<mykubermatic>>
  namespace: kubermatic
spec:
  userCluster:
    imagePullSecrets:
    - name: default-registry-auth
```

### CRD changes

#### `Cluster`

* New field `imagePullSecrets` on `ClusterSpec` analog to `imagePullSecrets` on `PodSpec`, but with
namespace as additional selector.

```
spec:
  imagePullSecrets:
  - name: my-dockerhub-creds
    namespace: kubermatic
```

### machine-controller

The machine-controller is already capable to set registry credentials on node level, but currently
only for containerd, and it is only possible to pass one secret reference.

* Add registry credential config on node level for docker
* Extend the `-node-registry-credentials-secret` to be able to take a list of secret references

### seed-controller-manager

* Add flag `-node-registry-credentials-secret` to machine-controller DeploymentCreator
* Set credentials flag with secrets referenced in `Cluster`

## Alternatives considered

### Set imagePullSecrets through an Addon

Solving this problem in an Addon would isolate the feature to everyone who really needs it and keep
the KKP codebase mostly clean of it. The problem is, that doing this in an Addon is just too late.
In the case where this secret needs to be set before any image is pulled, because the external IP of
the cluster has already hit the free tier limit of image pulls on docker hub, the user cluster would
be broken from the get go and could not even install this Addon to fix itself.

### Reconciliation of all Pods running on user clusters

Similar to the reconciliation of master- and seed-component Deployments custom credentials can be
injected into pod specs of Deployments and DaemonSets. In this approach the current concept of
creators and modifiers can be used, but it only affects KKP internal components and neither KKP
Addons nor user deployed applications. This means if we decide to go with this approach, we would
need to add handling for the not covered cases.

### Extension of Addon templates with imagePullSecrets

Pods or ServiceAccounts in Addon templates could be templated to use an imagePullSecret, but this
does not cover reconciled KKP components nor user apps.

### A webhook that adds imagePullSecrets to Pods

A simple webhook for Pods created on the user cluster. It merges given imagePullSecrets into the
PodSpec of any created Pods.

### Setting imagePullSecrets on user cluster ServiceAccounts

It's possible to set `imagePullSecrets` on `ServiceAccount` resources. All pods created with such a
ServiceAccount "inherits" its imagePullSecrets implicitly. So instead of having a Webhook the
reconciling mechanics can be used to set `imagePullSecrets` on all ServiceAccounts in the user
cluster.

See:
https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#add-imagepullsecrets-to-a-service-account

## Looking ahead

To make this more accessible to KKP admins we could build on top of this and extend the KKP dashboard
with an interface to manage this. Similar to the [SSH key feature][ssh key agent] container registry
credentials could be managed from there.

### Admin Penal

* **KKP admins** could create credentials to be made available for selection in KKP projects
* **KKP admins** could define default fallback credentials for KKP projects

### Project overview

* **Project Owners/Editors** could create or select credentials to be made available for selection
in user clusters
* **Project Owners/Editors** could define default fallback credentials for user clusters

### Cluster Creation Wizard

* Similar to SSH key selection it could be possible to select credentials available to the project
the cluster is created in.
* It could be possible to add credentials to the project and use then in the cluster, similar to the
"+ Add SSH Key" button.

### Cluster Management

* **User cluster admins** can create or select credentials to be used in all kinds of Pods on the
user cluster

## Tasks and effort

* Extend the `KubermaticConfiguration` with `spec.userCluster.imagePullSecrets`
* Extend the `Cluster` CRD with `imagePullSecrets` with default value from config
* Extend machine-controller
  * Add ability to take multiple secret references via flag
  * Add docker support for writing node level registry credentials (analog to containerd)

[#6231]: https://github.com/kubermatic/kubermatic/issues/6231
[ssh key agent]: https://docs.kubermatic.com/kubermatic/main/tutorials-howtos/administration/user-settings/user-ssh-key-agent/
