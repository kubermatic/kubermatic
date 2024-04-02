# Kubermatic 2.25

- [v2.25.0](#v2250)

## [v2.25.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.25.0)

Before upgrading, make sure to read the [general upgrade guidelines](https://docs.kubermatic.com/kubermatic/v2.25/installation/upgrading/). Consider tweaking `seedControllerManager.maximumParallelReconciles` to ensure user cluster reconciliations will not cause resource exhaustion on seed clusters. A [full upgrade guide is available from the official documentation](https://docs.kubermatic.com/kubermatic/v2.25/installation/upgrading/upgrade-from-2.24-to-2.25/).

### Action Required

- **ACTION REQUIRED:** VMware Cloud Director: Support for attaching multiple networks to a vApp
    - The field `ovdcNetwork` in `cluster` and `preset` CRDs is considered deprecated for VMware Cloud Director and `ovdcNetworks` should be used instead ([#12996](https://github.com/kubermatic/kubermatic/pull/12996))
- **ACTION REQUIRED:** KubeLB(EE): The prefix for the tenant namespaces created in the management cluster has been updated from `cluster-` to `tenant-`. The tenants will be migrated automatically to the new namespace, load balancers, and services. The load balancer IPs need to be rotated and previous namespace cleaned up ([#13093](https://github.com/kubermatic/kubermatic/pull/13093))
- **ACTION REQUIRED:** For velero helm chart upgrade related change. If you use `velero.restic.deploy: true`, you will see new daemonset `node-agent` running in `velero` namespace. You might need to remove existing daemonset named `restic` manually ([#12998](https://github.com/kubermatic/kubermatic/pull/12998))
- **ACTION REQUIRED:** For velero helm chart upgrade. If running node-agent daemonset along with velero, then following replacement should be made in the velero's values.yaml before proceeding with upgrade ([#13118](https://github.com/kubermatic/kubermatic/pull/13118))
    - `velero.restic.deploy` with `velero.deployNodeAgent`
    - `velero.restic.resources` with `velero.nodeAgent.resources`
    - `velero.restic.nodeSelector` with `velero.nodeAgent.nodeSelector`
    - `velero.restic.affinity` with `velero.nodeAgent.affinity`
    - `velero.restic.tolerations` with `velero.nodeAgent.tolerations`
- **ACTION REQUIRED:** [User MLA] If you had copied `values.yaml` of loki-distributed chart to further customize it, then please cleanup your copy of `values.yaml` for user-mla to retain your customization only ([#12967](https://github.com/kubermatic/kubermatic/pull/12967))
- **ACTION REQUIRED:** [User MLA] Cortex chart upgraded to resolve issues for cortex-compactor and improve stability of user-cluster MLA feature. Few actions are required to be taken to use new upgraded charts ([#12935](https://github.com/kubermatic/kubermatic/pull/12935)):
    - Refer to [Upstream helm chart values](https://github.com/cortexproject/cortex-helm-chart/blob/v2.1.0/values.yaml) to see the latest default values.
    - Some of the values from earlier `values.yaml` are now incompatible with latest version. They are removed in the `values.yaml` in the current chart. But if you had copied the original values.yaml to customize it further, you may see that `kubermatic-installer` will detect such incompatible options and churn out errors and explain that action that needs to be taken.
    - The memcached-* charts are now subcharts of cortex chart so if you provided configuration for `memcached-*` blocks in your `values.yaml` for user-mla, you must move them under `cortex:` block.
- **ACTION REQUIRED:** [User MLA] minio has been updated to RELEASE.2023-04-28T18-11-17Z.
    - Before upgrading from older versions please refer to [the upgrade notes](https://docs.kubermatic.com/kubermatic/v2.25/installation/upgrading/upgrade-from-2.24-to-2.25/) to verify if you're affected and how to move forward.

### Highlights
- EE: Add KubeVirt to the Default Applications Catalog ([#12851](https://github.com/kubermatic/kubermatic/pull/12851))
- Upstream Documentation and SourceURLs can be added to ApplicationDefinitions ([#13019](https://github.com/kubermatic/kubermatic/pull/13019))
- EE: Add k8sgpt operator to the Default Applications Catalog ([#13025](https://github.com/kubermatic/kubermatic/pull/13025))
- EE: Add nvidia-gpu-operator to the Default Applications Catalog ([#13147](https://github.com/kubermatic/kubermatic/pull/13147))
- Add K8sGPT to the Webshell ([#6501](https://github.com/kubermatic/dashboard/pull/6501))
- Add new feature to create, restore and schedule backups for user cluster namespaces ([#6296](https://github.com/kubermatic/dashboard/pull/6296))
- Add new page to manage backup storage location for the cluster backup feature ([#6478](https://github.com/kubermatic/dashboard/pull/6478))
- Support for downloading backups from the UI ([#6521](https://github.com/kubermatic/dashboard/pull/6521))
- Add support for Edge provider ([#6502](https://github.com/kubermatic/dashboard/pull/6502))
- Display comments in application values ([#6510](https://github.com/kubermatic/dashboard/pull/6510))
- Add Support for Kubernetes 1.29 ([#12936](https://github.com/kubermatic/kubermatic/pull/12936))


### API Changes

- Add the edge cloud provider ([#13018](https://github.com/kubermatic/kubermatic/pull/13018))
- EtcdStatefulSetSettings: Add nodeSelector option to let etcd pods only run on specific nodes ([#12838](https://github.com/kubermatic/kubermatic/pull/12838))

### Supported Kubernetes Versions

- Add Support for Kubernetes 1.29 ([#12936](https://github.com/kubermatic/kubermatic/pull/12936))
- Add support for Kubernetes v1.26.13, v1.27.10, v1.28.6, v1.29.1 ([#12981](https://github.com/kubermatic/kubermatic/pull/12981))
- Update supported kubernetes versions ([#13079](https://github.com/kubermatic/kubermatic/pull/13079)):
    - Add 1.29.2/1.28.7/1.27.11 to the list of supported Kubernetes releases. 
    - Add 1.29 to the list of supported EKS versions. 
    - Add 1.29 / remove 1.26 from the list of supported AKS versions
- Remove support for Kubernetes 1.26 ([#13032](https://github.com/kubermatic/kubermatic/pull/13032))
- Remove 1.25 from list of supported versions on AKS (EOL on January 14th) ([#12962](https://github.com/kubermatic/kubermatic/pull/12962))

#### Supported Versions

- 1.27.10
- 1.27.11
- 1.28.6
- 1.28.7 (default for k8s)
- 1.29.1
- 1.29.2


### Cloud Providers

#### Anexia

- Update Anexia CCM (cloud-controller-manager) to version 1.5.5 ([#12909](https://github.com/kubermatic/kubermatic/pull/12909))
    - Fixes leaking LoadBalancer reconciliation metric
    - Updates various dependencies 

#### GCP/GCE

- Add support for GCP/GCE cloud-controller-manager (CCM) ([#12955](https://github.com/kubermatic/kubermatic/pull/12955))
    - Existing user clusters can be migrated to the external CCM by setting the `externalCloudProvider` feature gate or using the KKP Dashboard.

#### OpenStack
- Allow configuring Cinder CSI topology support either on `Cluster` or `Seed` resource field `cinderTopologyEnabled` ([#12878](https://github.com/kubermatic/kubermatic/pull/12878))

#### VMware Cloud Director

- Move CSI controller to seed cluster ([#13020](https://github.com/kubermatic/kubermatic/pull/13020))
- Add support for configuring allowed IP allocation modes for VMware Cloud Director in KubermaticSettings ([#13002](https://github.com/kubermatic/kubermatic/pull/13002))

### Applications Catalog

- EE: Add KubeVirt to the Default Applications Catalog ([#12851](https://github.com/kubermatic/kubermatic/pull/12851))
- Upstream Documentation and SourceURLs can be added to ApplicationDefinitions ([#13019](https://github.com/kubermatic/kubermatic/pull/13019))
- EE: Add k8sgpt operator to the Default Applications Catalog ([#13025](https://github.com/kubermatic/kubermatic/pull/13025))
- A logo can now be added to Applications for better visibility ([#13044](https://github.com/kubermatic/kubermatic/pull/13044))
- Add documentation link, source code link and logo to the default applications ([#13054](https://github.com/kubermatic/kubermatic/pull/13054))
- EE: Update default application definitions with latest helm chart version ([#13058](https://github.com/kubermatic/kubermatic/pull/13058))
- Comments are now persisted in the values section of ApplicationDefinitions and ApplicationInstallations when using the new defaultValuesBlock and valuesBlock fields respectively ([#13075](https://github.com/kubermatic/kubermatic/pull/13075))
- EE: Add nvidia-gpu-operator to the Default Applications Catalog ([#13147](https://github.com/kubermatic/kubermatic/pull/13147))


### Kubermatic-installer

- Update local KubeVirt chart to v1.1.1 and CDI to 1.58.1 ([#13088](https://github.com/kubermatic/kubermatic/pull/13088))
- The Kubermatic installer will now detect DNS settings based on the Ingress instead of the nginx-ingress LoadBalancer, allowing for other ingress solutions to be properly detected ([#12934](https://github.com/kubermatic/kubermatic/pull/12934))
- Fix `mirror-images` command in installer not being able to extract the addons ([#12868](https://github.com/kubermatic/kubermatic/pull/12868))


### User Cluster MLA

- Grafana has been updated to v10.2.2 ([#12956](https://github.com/kubermatic/kubermatic/pull/12956))
- Minio has been updated to RELEASE.2023-04-28T18-11-17Z ([#13008](https://github.com/kubermatic/kubermatic/pull/13008))


### New Features

- Add `Seed.spec.metering.retentionDays` to configure the Prometheus retention; fix missing defaulting for `Seed.spec.metering.storageSize` ([#12843](https://github.com/kubermatic/kubermatic/pull/12843))
- Add new admin option to enable/disable user cluster backups ([#12888](https://github.com/kubermatic/kubermatic/pull/12888))
- Charts/kubermatic-operator: Ability to configure environment variables for the kubermatic-operator pod ([#12973](https://github.com/kubermatic/kubermatic/pull/12973))
- Add `ClusterBackupStorageLocation` and related user cluster configurations ([#12929](https://github.com/kubermatic/kubermatic/pull/12929))
    - Add a new field `backupConfig` to the Cluster Spec.
    - Add a new API type `ClusterBackupStorageLocation` for cluster backup integration
- Deploy all Velero components on the user cluster, when backup is enabled ([#13010](https://github.com/kubermatic/kubermatic/pull/13010))
- IAP ingresses can be configured to use an existing TLS secret instead of the one generated by the cert-manager ([#13061](https://github.com/kubermatic/kubermatic/pull/13061))
- EE: Update KubeLB to v0.7.0 ([#13169](https://github.com/kubermatic/kubermatic/pull/13169))
- Update KubeOne to [v1.7.2](https://github.com/kubermatic/kubeone/releases/tag/v1.7.2) ([#13076](https://github.com/kubermatic/kubermatic/pull/13076))
- We maintain now a dedicated docker image for the conformance tester, mainly for internal use ([#13113](https://github.com/kubermatic/kubermatic/pull/13113))


### Bugfixes

- Fix insufficient RBAC permission for VPA recommender pod caused by [upstream release issue](https://github.com/kubernetes/autoscaler/issues/5982) ([#12872](https://github.com/kubermatic/kubermatic/pull/12872))
- Fix cert-manager values block. cert-manager deployment will get updated as part of upgrade ([#12854](https://github.com/kubermatic/kubermatic/pull/12854))
- Fix a bug where resources deployed in the user cluster namespace on seed, for CSI drivers, were not being removed when the CSI driver was disabled ([#13045](https://github.com/kubermatic/kubermatic/pull/13045))
- Fix cases where, when using dedicated infra- and ccm-credentials, infra-credentials were always overwritten by ccm-credentials ([#12421](https://github.com/kubermatic/kubermatic/pull/12421))
- Fix missing image registry override for hubble-ui components if Cilium is deployed as System Application ([#13139](https://github.com/kubermatic/kubermatic/pull/13139))
- Fix OIDC network policy, by allowing to set NamespaceOverride to specify where the ingress controller is deployed ([#13135](https://github.com/kubermatic/kubermatic/pull/13135))
- Fix panic, if no KubeVirt DNS config was set in the datacenter ([#12933](https://github.com/kubermatic/kubermatic/pull/12933))
- Fix the issue with blocked cluster provisioning, when selected initial applications that conflicted with Cilium system application and user-cluster-controller-manager was stuck ([#12997](https://github.com/kubermatic/kubermatic/pull/12997))
- Fix the panic of the seed controller manager while checking CSI addon usage for user clusters, when a user cluster has PVs which were migrated from the in-tree provisioner to the CSI provisioner ([#13122](https://github.com/kubermatic/kubermatic/pull/13122))
- No longer fail constructing vSphere endpoint when a `/` suffix is present in the datacenter configuration ([#12861](https://github.com/kubermatic/kubermatic/pull/12861))
- Stop constantly re-deploying operating-system-manager when registry mirrors are configured ([#12972](https://github.com/kubermatic/kubermatic/pull/12972))
- If the seed cluster is using Cilium as CNI, create CiliumClusterwideNetworkPolicy for api-server connectivity ([#12924](https://github.com/kubermatic/kubermatic/pull/12924))
- Resolved an issue where logs were duplicated when multiple pods from the same service were deployed on the same Kubernetes node ([#13109](https://github.com/kubermatic/kubermatic/pull/13109))
- Exclude `test` folders which contain symlinks that break once the archive is untarred. ([#13151](https://github.com/kubermatic/kubermatic/pull/13151))
- Fix to allow IPv6 IPs for etcd-launcher Pods ([#13160](https://github.com/kubermatic/kubermatic/pull/13160))
- Raise memory limit on envoy-agent from 64Mi to 512Mi to support larger user clusters. ([#13161](https://github.com/kubermatic/kubermatic/pull/13161))
- Fix the usercluster-controller-manager failure to reconcile cluster with disable CSI drivers ([#13167](https://github.com/kubermatic/kubermatic/pull/13167))


### Updates

- Update KKP images to Alpine 3.18; auxiliary single-binary images (alertmanager-authorization-server, network-interface-manager, s3-exporter and user-ssh-keys-agent) have been changed to use `gcr.io/distroless/static-debian12` as the base image ([#12870](https://github.com/kubermatic/kubermatic/pull/12870))
- Update metering to v1.1.2, fixing an error when a custom CA bundle is used ([#13013](https://github.com/kubermatic/kubermatic/pull/13013))
- Update metrics-server to v0.7.0 ([#13056](https://github.com/kubermatic/kubermatic/pull/13056))
- Update telemetry to v0.5.0, now shipping with a distroless image ([#13055](https://github.com/kubermatic/kubermatic/pull/13055))
- Update to Go 1.22.1 ([#13152](https://github.com/kubermatic/kubermatic/pull/13152))
- Update to Kubernetes 1.29 / controller-runtime 0.17.1 ([#13066](https://github.com/kubermatic/kubermatic/pull/13066))
- Update Vertical Pod Autoscaler to 1.0 ([#12863](https://github.com/kubermatic/kubermatic/pull/12863))
- Increase the default resources for VPA components to prevent OOMs ([#12887](https://github.com/kubermatic/kubermatic/pull/12887))
- Update OSM and MC ([#13175](https://github.com/kubermatic/kubermatic/pull/13175))
    - Update operating-system-manager to [v1.5.0](https://github.com/kubermatic/operating-system-manager/releases/tag/v1.5.0)
    - Update machine-controller to [v1.59.0](https://github.com/kubermatic/machine-controller/releases/tag/v1.59.0)



### Cleanup

- Remove `CloudControllerReconcilledSuccessfully` (double L) Cluster condition, which was deprecated in KKP 2.21 and has since been replaced with `CloudControllerReconciledSuccessfully` (single L) ([#12867](https://github.com/kubermatic/kubermatic/pull/12867))
- Remove CriticalAddonsOnly toleration from node-local-dns DaemonSet as it has more general tolerations configured ([#12957](https://github.com/kubermatic/kubermatic/pull/12957))
- Some of high cardinality metrics were dropped from the User Cluster MLA prometheus. If your KKP installation was using some of those metrics for the custom Grafana dashboards for the user clusters, your dashboards might stop showing some of the charts ([#12756](https://github.com/kubermatic/kubermatic/pull/12756))
- Deprecate v1.11 and v1.12 Cilium and Hubble KKP Addons, as Cilium CNI is managed by Applications from version 1.13 ([#12848](https://github.com/kubermatic/kubermatic/pull/12848))


### Documentation

- Examples now include command to generate secrets that works on vanilla macOS ([#12974](https://github.com/kubermatic/kubermatic/pull/12974))

### Miscellaneous

- Addon manifests are now loaded once upon startup of the seed-controller-manager instead of during every reconciliation. Invalid addons will now send the seed-controller-manager into a crash loop ([#12684](https://github.com/kubermatic/kubermatic/pull/12684))
- Kube state metrics can be configured to get metrics for custom kubernetes resources ([#13027](https://github.com/kubermatic/kubermatic/pull/13027))

### Dashboard & API

#### Cloud Providers

##### Anexia

- API change: Update MachineDeployment form for Anexia provider ([#6460](https://github.com/kubermatic/dashboard/pull/6460))
    - Add configuration support for named templates
    - Add configuration support for multiple disks- diskSize attribute gets automatically migrated to the disks attribute when saved- Fix error occurring when listing MachineDeployments which have named templates configured

##### Azure

- Set LoadBalancerSKU on Azure clusters if the field is set in the preset ([#6506](https://github.com/kubermatic/dashboard/pull/6506))

##### GCE

- Flatcar is now supported on GCE ([#6399](https://github.com/kubermatic/dashboard/pull/6399))

##### Nutanix

- Fix invalid project ID in API requests for Nutanix provider ([#6572](https://github.com/kubermatic/dashboard/pull/6572))

##### vSphere

- Fix a bug where dedicated credentials were incorrectly being required as mandatory input when editing provider settings for a cluster ([#6567](https://github.com/kubermatic/dashboard/pull/6567))
- No longer fail constructing vSphere endpoint when a `/` suffix is present in the datacenter configuration ([#6403](https://github.com/kubermatic/dashboard/pull/6403))


##### VMware Cloud Director

- Support for attaching multiple networks to a vApp ([#6480](https://github.com/kubermatic/dashboard/pull/6480))
- Added Flatcar as supported OS ([#6391](https://github.com/kubermatic/dashboard/pull/6391))
- Add support for configuring allowed IP allocation modes for VMware Cloud Director ([#6482](https://github.com/kubermatic/dashboard/pull/6482))
- Fix a bug where OSPs were not being listed for VMware Cloud Director ([#6592](https://github.com/kubermatic/dashboard/pull/6592))

#### API Changes

- Support for edge provider in KKP API ([#6525](https://github.com/kubermatic/dashboard/pull/6525))
- ValuesBlock and defaultValuesBlock fields are now available via the API ([#6562](https://github.com/kubermatic/dashboard/pull/6562))

#### New Features

- Add an option to enable/disable the cluster backup feature for user clusters ([#6493](https://github.com/kubermatic/dashboard/pull/6493))
- Add K8sGPT to the Webshell ([#6501](https://github.com/kubermatic/dashboard/pull/6501))
- Add new feature to create, restore and schedule backups for user cluster namespaces ([#6296](https://github.com/kubermatic/dashboard/pull/6296))
- Add new page to manage backup storage location for the cluster backup feature ([#6478](https://github.com/kubermatic/dashboard/pull/6478))
- Add support for Edge provider ([#6502](https://github.com/kubermatic/dashboard/pull/6502))
- Display source URL, documentation URL and logo of applications ([#6504](https://github.com/kubermatic/dashboard/pull/6504))
- Display comments in application values ([#6510](https://github.com/kubermatic/dashboard/pull/6510))
- Support for configuring static network for flatcar machines ([#6446](https://github.com/kubermatic/dashboard/pull/6446))
- Support for disabling CSI driver for user clusters ([#6395](https://github.com/kubermatic/dashboard/pull/6395))
- Support to enable/disable the cluster backup feature from the admin panel ([#6433](https://github.com/kubermatic/dashboard/pull/6433))
- Support for downloading backups from the UI ([#6521](https://github.com/kubermatic/dashboard/pull/6521))
- Edge provider support in the node deployment spec ([#6545](https://github.com/kubermatic/dashboard/pull/6545))
- Option to enable Cilium Ingress capabilities for user clusters ([#6490](https://github.com/kubermatic/dashboard/pull/6490))

#### Bugfixes

- Fix a bug where Operating System Profiles were not being listed for GCP ([#6453](https://github.com/kubermatic/dashboard/pull/6453))
- Fix issue in editing and updating applications of cluster template ([#6415](https://github.com/kubermatic/dashboard/pull/6415))
- Fix issue with cursor position inside YAML editor ([#6419](https://github.com/kubermatic/dashboard/pull/6419))
- Enable web terminal button when at least one MD replica is ready ([#6602](https://github.com/kubermatic/dashboard/pull/6602))


#### Updates

- Update to alpine 3.19(latest available) version for container images ([#6503](https://github.com/kubermatic/dashboard/pull/6503))
- KKP Dashboard is now built with Go 1.22.0 ([#6505](https://github.com/kubermatic/dashboard/pull/6505))
