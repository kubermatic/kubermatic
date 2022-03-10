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
  * [Custom Resources](#custom-resources)
  * [Master Cluster](#master-cluster)
  * [Seed Cluster](#seed-cluster)
  * [User Cluster](#user-cluster)
  * [Agent](#agent)
  * [UI](#ui)
* [Alternatives considered](#alternatives-considered)
  * [Reconciliation of all Pods running on user clusters](#reconciliation-of-all-pods-running-on-user-clusters)
  * [Setting imagePullSecrets on ServiceAccounts](#setting-imagepullsecrets-on-serviceaccounts)
  * [Extension of addon templates with imagePullSecrets](#extension-of-addon-templates-with-imagepullsecrets)
* [Task & effort](#task-&-effort)

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

Similar to the [SSH key feature][ssh key agent] container registry credentials can be managed in the KKP
dashboard.

* **KKP admins** can create credentials to be made available for KKP projects
* **KKP admins** can define default fallback credentials for KKP projects
* **Project Owners/Editors** can create or select credentials to be made available for user clusters
* **Project Owners/Editors** can define default fallback credentials for user clusters
* **User cluster admins** can create or select credentials to be used in all kinds of Pods on the
user cluster

There is a possibility to define a default fallback set of credentials in the
`KubermaticConfiguration` at `spec.userCluster.imagePullSecret`. If there are no credentials
selected on any elvel, this will be used as a fallback.

#### Limitations

1. For now only containerd is supported. Because of the [dockershim deprecation][] currently there
   is no support planned for nodes using docker as container runtime.
2. Only one credential set per registry can be chosen. Validation will fail if multiple credentials
   for the same registry are chosen to avoid unexpected behavior.

## Implementation

There are quite some similarities to the implementation of the [SSH key feature][ssh key agent].
Writing the containerd-config on every worker node of a user cluster based on credentials managed
and selected in the dashboard sounds quite familiar. This opens up some potential for refactoring
the current SSH key agent into a more general Agent controlling files on worker node's file system.

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
            "username": "<<QUAY_USERNAME>>",
            "password": "<<QUAY_PASSWORD>>",
          }
        }
      }
```

### Custom Resources

Similar to user ssh keys the `RegistryCredentialSet` is tied to a project, so credentials created in
one project can be used in all user clusters created in that same project.

The credentials themselves are being stored in `kubernetes.io/dockerconfigjson` typed secrets which
are referenced by `RegistryCredentialSet` resources.

#### Example

```yaml
apiVersion: kubermatic.k8c.io/v1
kind: RegistryCredentialSet
metadata:
  name: my-quay-credentials
  uid: ..
  ownerReferences:
  - apiVersion: kubermatic.k8c.io/v1
    kind: Project
    name: <<PROJECT_ID>>
    uid: ..
spec:
  secretRef:
    name: my-quay-credentials
    key: .dockerconfigjson
```

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-quay-credentials
type: kubernetes.io/dockerconfigjson
stringData:
  .dockerconfigjson: |
    {
      "auths": {
        "quay.io": {
          "auth": "<<base64(username:password)>>"
        }
      }
    }
```

### Master Cluster

* A controller that synchronizes selected credentials into the cluster namespace on the seed
cluster

### Seed Cluster

* A controller within the user-cluster-controller-manager to set up the DaemonSet running the Agent
on every user cluster node. (similar to usersshkeys)
* The user-cluster-controller synchronizes selected credentials from the cluster namespace to the
user cluster. If none is selected, it uses the default fallback credentials from the config (if
set).

### User Cluster

* The DaemonSet running the Agent on every user cluster node (see above)

### Agent

Similar to the SSH key agent this agent watches a named Secret containing registry credentials and
also the containerd-config on the node the agent is running on. It makes sure that the containerd-
config always contain the selected credentials.

#### Refactoring potential

The general functionality of the user registry credentials agent is very similar to the SSH key
agent: They watch some kubernetes resource and local files and have a reconciliation process that
makes sure the content of the local files always match the content of the resources.

This agent could just mirror the behavior of the SSH key agent for different resources and files, so
a good share of code might be reusable. Another direction could be to refactor the whole SSH key
agent to become a more general "node-file-agent" that can manage all kinds of files based on
resources.

### UI

* Similar to SSH key management. Details to be discussed with `#sig-ui`

## Alternatives considered

### Reconciliation of all Pods running on user clusters

Similar to the reconciliation of master- and seed-component Deployments custom credentials can be
injected into podspecs of Deployments and DaemonSets. In this approach the current concept of
creators and modifiers can be used, but it only affects KKP internal components and neither KKP
addons nor user deployed applications. This means if we decide to go with this approach, we would
need to add handling for the not covered cases.

### Setting imagePullSecrets on ServiceAccounts

Instead of setting `imagePullSecret` explicitly on podspecs, it's possible to set `imagePullSecret`
on ServiceAccounts. Every Pod created using a ServiceAccount with an imagePullSecret attached will
automatically have this imagePullSecret set in its spec. This approach would require less
reconciliation than reconciling Pods directly but doesn't cover anything created using
ServiceAccounts that have been created outside of the user cluster's reconciler.

### Extension of addon templates with imagePullSecrets

Pods or ServiceAccounts in addon templates could be templated to use an imagePullSecret, but this
does not cover reconciled KKP components nor user apps. So this approach would need to be combined
with at least one of the 2 above.

## Task & effort
* tbd.

[#6231]: https://github.com/kubermatic/kubermatic/issues/6231
[dockershim deprecation]: https://kubernetes.io/blog/2020/12/02/dockershim-faq/
[ssh key agent]: https://docs.kubermatic.com/kubermatic/master/tutorials_howtos/administration/user_settings/user_ssh_key_agent/
