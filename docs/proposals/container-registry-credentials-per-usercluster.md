# Container Registry Credentials per User Cluster

**Author**: Moritz Bracht (@dermorz)

**Status**: Draft proposal; prototype in progress.

* [Goals](#goals)
* [Non-Goals](#non-goals)
* [Motivation and Background](#motivation-and-background)
  * [Current state](#current-state)
  * [Proposed feature](#proposed-feature)
* [Implementation](#implementation)
  * [KubermaticConfiguration](#kubermaticconfiguration)
  * [CRD changes](#crd-changes)
  * [Master Cluster](#master-cluster)
  * [Seed Cluster](#seed-cluster)
* [Alternative implementation](#alternative-implementation)
  * [Set imagePullSecrets on user cluster ServiceAccounts](#set-imagepullsecrets-on-user-cluster-serviceaccounts)
* [Alternatives considered](#alternatives-considered)
  * [Set imagePullSecrets through an Addon](#set-imagepullsecrets-through-an-addon)
  * [Reconciliation of all Pods running on user clusters](#reconciliation-of-all-pods-running-on-user-clusters)
  * [Extension of addon templates with imagePullSecrets](#extension-of-addon-templates-with-imagepullsecrets)
  * [A look ahead](#a-look-ahead)
* [Task and effort](#task-and-effort)

## Goals

Defining separate container registry credentials per user cluster so cluster owners can make use of
their dockerhub credentials to avoid image pull limits or to access private images.

## Non-Goals

Setting local mirrors or private container registries for KKP components or addons.

## Motivation and Background

Based on the user story [#6231][] users want to be able to use their Docker Hub paid accounts to
mitigate image pull limits. Setting this up cluster wide is currently not possible.

### Current state

Currently it's possible to set `spec.imagePullSecret` in the `KubermaticConfiguration`. The
dockerconfig-json defined on this key is used in the Kubermatic Operator where the master- and the
seed-controller create a `dockerconfigjson`-secret respectively. When creating the deployments for
KKP master and seed components this secret gets referenced as `imagePullSecret` in all of their
podspecs.

This dockerconfig has no effect on anything running on the user clusters, so currently KKP does not
offer a way to manage container registry credentials to be used by KKP internal components, KKP
addons nor user deployed applications by default.

### Proposed feature

`Cluster` resources get a new field `imagePullSecrets` that resembles `imagePullSecrets` on other
resources like Deployments or Pods. The secrets referenced here get synchronized into the user
cluster and a mutating admission webhook injects them into every Pod that is created on that user
cluster.

For this first iteration the secrets referenced on `Cluster` resources need to be created manually.

There is a also the possibility to define default credentlals in the `KubermaticConfiguration` at
`spec.userCluster.imagePullSecret`. If defined, this will be used for any user cluster that has no
explicit `imagePullSecrets` set.

## Implementation

### KubermaticConfiguration

A new configuration field to set default fallback registry credentials for any user cluster where no
credentials are explicitly selected. For consistency this could be in the format of
`spec.imagePullSecret`:

```yaml
apiVersion: kubermatic.k8c.io/v1
kind: KubermaticConfiguration
metadata:
  name: <<mykubermatic>>
  namespace: kubermatic
spec:
  userCluster:
    imagePullSecret: |-
      {
        "auths": {
          "quay.io": {
            "username": "<<DOCKERHUB_USERNAME>>",
            "password": "<<DOCKERHUB_PASSWORD>>",
          }
        }
      }
```

### CRD changes

#### `Cluster`

* New field `imagePullSecrets` on `ClusterSpec`: Analog to `imagePullSecrets` on `PodSpec`, but with
namespace as additional selector.

```
spec:
  imagePullSecrets:
  - name: my-dockerhub-creds
    namespace: kubermatic
```

### Master Cluster
 
* The cluster-controller in the seed-controller-user-manager creates all resources (Deployment,
Service, TLSServingCertificate) for the Webhook in the cluster-namespace on the seed cluster

### Seed Cluster

* The user-cluster-controller synchronizes imagePullSecrets of the Cluster resource into all
namespaces on the user cluster. This means the Namespace type needs to be added to the typed that
are being watched.
* The user-cluster-controller does the same for the default imagePullSecret if it's defined in the
KubermaticConfiguration.
* The user-cluster-controller adds the MutatingWebhookConfiguration for our Webhook.

#### Webhook

A simple webhook for Pods that merges imagePullSecrets from the Cluster and from the PodSpec. This
way users can still specify their own secrets for private registries but also have the "global"
secrets as fallback.

## Alternative implementation

### Set imagePullSecrets on user cluster ServiceAccounts

It's possible to set `imagePullSecrets` on `ServiceAccount` resources. All pods created with such a
ServiceAccount "inherits" its imagePullSecrets implicitly. So instead of having a Webhook we could
use our reconciling mechanics to set `imagePullSecrets` on all ServiceAccounts in the user cluster.

See:
https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#add-imagepullsecrets-to-a-service-account

It's not fully decided if we go with the Webhook modifying Pods or with setting imagePullSecrets on
ServiceAccounts.

## Alternatives considered

### Set imagePullSecrets through an Addon

Solving this problem in an Addon would isolate the feature to everyone who really needs it and keep
the KKP codebase mostly clean of it. The problem is, that doing this in an Addon is just too late.
In the case where this secret needs to be set before any image is pulled, because the external IP of
the cluster has already hit the free tier limit of image pulls on dockerhub, the user cluster would
be broken from the get go and could not even install this addon to fix itself.

### Reconciliation of all Pods running on user clusters

Similar to the reconciliation of master- and seed-component Deployments custom credentials can be
injected into podspecs of Deployments and DaemonSets. In this approach the current concept of
creators and modifiers can be used, but it only affects KKP internal components and neither KKP
addons nor user deployed applications. This means if we decide to go with this approach, we would
need to add handling for the not covered cases.

### Extension of addon templates with imagePullSecrets

Pods or ServiceAccounts in addon templates could be templated to use an imagePullSecret, but this
does not cover reconciled KKP components nor user apps.

### A look ahead

To make this more accessible to customers we could build ontop of this and extend the KKP dashboard
with an interface to manage this. Similar to the [SSH key feature][ssh key agent] container registry
credentials could be managed from there.

For example:

#### Admin Penal

* **KKP admins** could create credentials to be made available for selection in KKP projects
* **KKP admins** could define default fallback credentials for KKP projects

#### Project overview

* **Project Owners/Editors** could create or select credentials to be made available for selection
in user clusters
* **Project Owners/Editors** could define default fallback credentials for user clusters

#### Cluster Creation Wizard

* Similar to SSH key selection it could be possible to select credentials available to the project
the cluster is created in.
* It could be possible to add credentials to the project and use then in the cluster, similar to the
"+ Add SSH Key" button.

#### Cluster Management

* **User cluster admins** can create or select credentials to be used in all kinds of Pods on the
user cluster

## Task and effort

- tbd.

[#6231]: https://github.com/kubermatic/kubermatic/issues/6231
[ssh key agent]: https://docs.kubermatic.com/kubermatic/master/tutorials_howtos/administration/user_settings/user_ssh_key_agent/
