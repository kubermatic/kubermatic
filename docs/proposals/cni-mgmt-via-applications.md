# CNI Management via Applications Infra

**Authors**: Authors: Rastislav Szabo (@rastislavs), Simon Bein (@SimonTheLeg), Vincent Gramer (@vgramer)

**Status**: Draft proposal; Prototype in progress.

**Related Issues:**
- https://github.com/kubermatic/kubermatic/issues/8540
- https://github.com/kubermatic/kubermatic/issues/10851
- https://github.com/kubermatic/kubermatic/issues/10449

## Goals

The main goals of this proposal are:

- Provide more flexibility for configuring CNIs in KKP user clusters for experienced KKP cluster administrators, such as enabling/disabling advanced CNI features (e.g. bandwidth monitoring, built-in ingress, built-in service-mesh features etc.)
- Allow for tighter integration of CNI features with KKP to achieve more automation, e.g. CIlium cluster-mesh integration, service-mesh integration etc.
- Reduce the complexity of the manual upgrade process when upgrading CNIs in KKP.

Same concepts may be later used for managing other resources in the user cluster, which contain many configurable options as well - such as CSI plugins.

## Non-Goals
- The user experience of CNIs should stay the same at least for inexperienced users. More flexibility should not make the basic use-cases any harder.
- This proposal does not focus on managing all networking resources in the user-cluster, e.g. kube-proxy or CoreDNS management would not change for now.

## Motivation and Background
Modern CNIs nowadays provide much more functionality than just the basic pod networking. Those may include for example: service mesh features, advanced load-balancing / ingress features, transparent encryption, bandwidth limiting, etc. The features like these are most often disabled by default and cannot be easily used in KKP user clusters at the moment.

Since CNIs in KKP are deployed as KKP addons - templated yaml manifests from the `addons` folder, it is not easy to provide many configuration options for them. A limited set of network configuration options is available via KKP’s cluster `spec. clusterNetwork`, but these options are by design generic (not CNI-specific) and expanding them would just bring more complexity to the cluster API and addon templates.

At the same time, maintenance of CNIs in KKP is already becoming more and more complex as we keep adding more networking features. For example, adding a new version of CIlium CNI is a tedious error-prone process, where we use the official Cilium Helm chart of Cilium to render the manifest yaml for the KKP addon, which we in turn need to manually template again to allow for configurations that KKP supports. However, most of the manually templated parts were already templated in the upstream Helm chart, so the whole error-prone process is unnecessary and could be avoided if we just used the Helm chart to install CNI directly.

From the KKP code perspective, the KKP addon mechanism itself is a bit limiting due to the promise of keeping the KKP addons simple. Nevertheless, it already contains some hacks around CNI management (e.g. for triggering a CNI upgrade). Those should be ideally removed and the addon mechanism logic should be kept simple and generic.

The recent addition of the Applications feature into KKP provides a solid infrastructure for installing third-party applications into KKP user clusters leveraging established Kubernetes projects (e.g. Helm) for templating manifests. Since most of the CNIs nowadays provide officially maintained Helm charts, this proposal suggests to reuse the existing Applications code in KKP to manage CNIs internally. That would remove the need for manual maintenance of the CNI manifests, as well as allow the users for more customization of the CNIs via user-provided Helm values.

## Implementation

The main goals of this proposal are:
- Manage CNIs as KKP Applications via Helm charts (ideally 1:1 synced with upstream), so that CNI maintenance becomes simpler and less error-prone.
- Provide the user an easy way to modify the Helm values used to render the CNI, so that they can easily configure certain CNI features as they need.

At the same time, we would like to satisfy the following:
- We still want to control the versions and manifests of CNIs that can be deployed for the given KKP / k8s version, to give KKP users compatibility guarantees.
- We don’t want to make the basic CNI deployment any harder - the user experience should stay the same.
- We don’t want users to break the cluster by incorrectly changing the important parts of the CNI configuration - mostly the configuration that needs to be in sync with the control plane configuration, such as pod / service CIDRs etc.
- It must be possible to deploy CNIs easily in offline scenarios (without Internet connectivity) as well.

Note that in the future, Helm may not be the only way for installing the CNIs - we may use different packaging mechanisms as well. However, at the moment of writing, Helm is the only straightforward packaging mechanism directly supported by CNIs that we use.

### KKP-Managed / KKP-Internal Applications

The idea is to reuse the existing Applications infra in KKP and introduce a new flavor of Applications, that would be considered as “system” / “internal” and managed by KKP. The users may not see them in the KKP UI by default when installing / editing standard applications in their user cluster (but potentially they still could be shown e.g. after clicking on some kind of “show internal” toggle). Internally, they will be just labeled with specific labels, e.g.:

- `apps.kubermatic.k8c.io/managed-by` == “kkp” - will denote that the application is internal & managed by KKP,
- `apps.kubermatic.k8c.io/type` - will specify type of the internal application, e.g. “cni”.

The `ApplicationDefinition` of the internal Applications will be managed by the KKP operator. That way, in the case of CNIs, the versions of CNIs available in each KKP version and the manifest sources will be fixed and managed by Kubermatic. KKP admins will not be able to manage them, besides the `defaultValues` section, to which changes may be allowed (but would need to be validated by KKP).

KKP-managed Applications will be internally handled a bit differently in comparison to the “standard” Applications:

- Applications are not periodically reconciled. We may need to periodically reconcile internal Applications, e.g. once in 60 minutes, similarly as we do for Addons.
- Helm charts for the applications should be applied with `--wait` and `--atomic` flags (which waits for workload to be ready and rolls back otherwise), to prevent breaking the cluster with incorrect values etc.
- The result of the reconciliation of these Applications should be updated in the cluster status (conditions).
- Regular applications may be installed only after the system Applications are installed (to achieve faster cluster bring-up time) based on the above mentioned status.

### Managing ApplicationInstallations for User Clusters

A new KKP seed-cluster-manager level controller will be in charge of creating `ApplicationInstallation` CR of the CNI for the given user cluster based on the networking configuration in KKP Cluster API, that would reflect into `spec.values` of the `ApplicationInstallation` CR in the user cluster.

To allow specifying initial user-specified Helm values during cluster creation (e.g. on the same step of the Cluster creation wizard as for the “regular” Applications), we would use mechanisms similar to the existing `initial-application-installations-request` annotation (but with the different name, e.g. `initial-cni-application-installations-request`). That approach would be compatible also with KKP Cluster templates.

KKP-managed Helm values would be always applied on top of the user-provided initial values, to make sure that necessary CNI configuration is always applied correctly.

Apart from cluster creation, the controller would reconcile the `ApplicationInstallation` in the user cluster only when related fields like `spec.cniPlugin` changes in the cluster (to support CNI version upgrade). Also in this case, it will override only the necessary parts of the Helm values, and keep the user-provided parts unchanged.

### Validation of ApplicationInstallations

As KKP cluster admins will be able to access and modify the CNI `ApplicationInstallation` CR in the user cluster (which was the original intention - to give them the ability to tweak CNI’s config), we need to make sure that they cannot change values that are immutable from the control plane perspective (e.g. pods/services CIDR etc), or that they cannot change certain config to an invalid value. For this, we will implement CNI-specific webhook logic in the existing validation webhook for the ApplicationInstallation CRs.

### UI Integration

As already mentioned, the existing UI experience of the CNIs and networking configuration would not change. That includes the initial cluster networking configuration as well as CNI upgrade options.

On top of that, we can provide additional way to optionally customize the CNI configuration further from the KKP UI via existing Application’s integration:
- During cluster creation, users will be able to provide initial CNI Helm values under the “Advanced Network Configuration” on the step 2 (cluster details), e.g. via a modal window that could open after clicking on a button, and potentially will be able to see provided values in the step 6 (cluster creation summary)
- After cluster creation, we can allow modifying the CNI configuration e.g. on the Applications tab of the cluster (e.g. after clicking on some kind of “show internal” toggle).

Note that even if this extension of the UI integration is not implemented, the users can still manage the CNI configuration via the ApplicationInstallation CR in the user cluster.


### Helm Chart Source & Chart Maintenance

To provide a stable & trusted source of the Helm charts to install CNIs, we propose to maintain them in a KKP-managed public OCI repository. Most of the time it will be just a mirror of the upstream chart 1:1, but if the chart needs to be patched in rare cases, it still will be possible. For the air-gapped and semi-air-gapped scenarios, customers will be able to use their private OCI registry to host the charts and install them even without access to the original upstream chart source.

The exact process of maintaining the charts will be defined in more detail after further discussions with sig-infra and/or sig-app-management. In any case, it will be automated as much as possible.

The existing OverwriteRegistry configuration KKP needs to be taken into account when rendering the Helm chart. While no generic approach for Applications is provided, this can be done on the level of the controller responsible for creating `ApplicationInstallation` CR of the CNI based on the networking configuration in KKP Cluster API. Also, CNI images need to be included in the images selection for the `kubermatic-installer mirror-images` command.

### Backward Compatibility & Migration

**For Canal CNI:**
In the initial release, the new CNI management will be used only for the Cilium CNI, Canal CNI will still remain as a KKP Addon. The reasons are:

- To have a fallback option if the new CNI management strategy contains a bug.
- Canal does not provide many configuration options anyways.
- There is no official upstream Helm chart for the Canal CNI.

At some later point we may consider deprecating Canal CNI and introducing vanilla Calico CNI (which does have an upstream Helm chart) instead, as nowadays the difference between the two in the basic VXLAN mode is very small, and in general Calico is better maintained.

**For Cilium CNI:**
- The Applications-based CNI management will be in place as of a specific minor version of Cilium, both for new and existing clusters. For older Cilium releases, Cilium will be still managed as an Addon. The migration between the old and the new management strategy will happen as part of the CNI upgrade.
- When Cilium is managed by Applications, Hubble will not be deployable as an Addon anymore, only by enabling it in the Helm values.

## Alternatives Considered

Within sig-networking, we also considered 2 other approaches to address the goals of this proposal:

1) Introducing a new CRD for all cluster network configuration, factoring out the cluster network configuration from the KKP cluster API and introducing a specific controller managing all user-cluster networking components:
   - This turned out to be a too complex change and just by itself it would not serve all goals - we would still need to replace manifest-based CNI deployment with e.g. Helm-based CNI deployment.
   - As we already have the Applications infra capable of doing it, it is much easier to just reuse it.
   - If more complex networking use-cases come in the future, we can still proceed with this approach even later.

2) Generic multi-CNI operator, running in the user cluster independently (similar to machine-controller):
   - If it provided a common high-level  CNI API, it would again become hard to use any arbitrary CNI features + it would be hard to maintain.
   - If the API was generic (e.g. possible to provide any Helm values), it would eventually be just a Helm operator providing little value - after all, from KKP perspective it would be similar to just reusing the Applications infra, but without the benefit of having it already integrated with the KKP UI etc.

Considering the fact that KKP users are already familiar with the Applications infra in KKP, it would be also easier for them to interact with the CNI management in a similar way.
