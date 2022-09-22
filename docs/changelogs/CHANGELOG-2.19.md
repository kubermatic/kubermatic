# Kubermatic 2.19

- [v2.19.0](#v2190)
- [v2.19.1](#v2191)
- [v2.19.2](#v2192)
- [v2.19.3](#v2193)
- [v2.19.4](#v2194)
- [v2.19.5](#v2195)
- [v2.19.6](#v2196)
- [v2.19.7](#v2197)
- [v2.19.8](#v2198)
- [v2.19.9](#v2199)

## [v2.19.9](https://github.com/kubermatic/kubermatic/releases/tag/v2.19.9)

### Bugfixes

- Add additional header to prevent KKP dashboard from being shown in an iframe ([#4796](https://github.com/kubermatic/dashboard/pull/4796))
- Allow proxy mode change by CNI migration ([#10717](https://github.com/kubermatic/kubermatic/pull/10717))
- Fix an issue with vSphere CSI driver using improved-csi-idempotency that's currently not supported by KKP ([#10793](https://github.com/kubermatic/kubermatic/pull/10793))
- Update KubeVirt logo to mark technology preview ([#4810](https://github.com/kubermatic/dashboard/pull/4810))

### Chore

- Add support for Kubernetes 1.22.15; existing clusters using these Kubernetes releases will be automatically updated as any previous version is affected by CVEs ([#11029](https://github.com/kubermatic/kubermatic/pull/11029))


## [v2.19.8](https://github.com/kubermatic/kubermatic/releases/tag/v2.19.8)

### Updates

- Update Canal 3.21 version to v3.21.6 ([#10491](https://github.com/kubermatic/kubermatic/issues/10491))
- Update machine-controller to v1.42.7 ([#10626](https://github.com/kubermatic/kubermatic/issues/10626))


## [v2.19.7](https://github.com/kubermatic/kubermatic/releases/tag/v2.19.7)

### Updates

- Update Canal 3.21 to v3.21.5 ([#10271](https://github.com/kubermatic/kubermatic/issues/10271))
- Update Konnectivity to v0.0.31 ([#10112](https://github.com/kubermatic/kubermatic/issues/10112))
- Update machine-controller to v1.42.6 ([#10401](https://github.com/kubermatic/kubermatic/issues/10401))

### Misc

- Add `--skip-dependencies` flag to kubermatic-installer that skips downloading Helm chart dependencies (requires chart dependencies to be downloaded already) ([#10348](https://github.com/kubermatic/kubermatic/issues/10348))
- Use quay.io as the default registry for Canal CNI images, bump Canal v3.20 version to v3.20.5 ([#10305](https://github.com/kubermatic/kubermatic/issues/10305))
- etcd backup files are named differently (`foo-YYYY-MM-DDThh:mm:ss` to `foo-YYYY-MM-DDThhmmss.db`) to improve compatibility with different storage solutions ([#10143](https://github.com/kubermatic/kubermatic/issues/10143))


## [v2.19.6](https://github.com/kubermatic/kubermatic/releases/tag/v2.19.6)

- Fix not referencing a custom CA bundle in vSphere CSI driver (regression from 2.18) ([#9989](https://github.com/kubermatic/kubermatic/issues/9989))
- Fix probes, resources and allow overriding resource requests/limits for Konnectivity proxy via components override in the Cluster resource ([#9911](https://github.com/kubermatic/kubermatic/issues/9911))


## [v2.19.5](https://github.com/kubermatic/kubermatic/releases/tag/v2.19.5)

With this patch release, etcd for Kubernetes 1.22+ is upgraded to etcd 3.5.3. Data consistency issues as reported in previous release notes are fixed. Warnings and recommendations related to that can be considered withdrawn for this release.

- Add `vsphereCSIClusterID` feature flag for the cluster object. This feature flag changes the cluster-id in the vSphere CSI config to the cluster name instead of the vSphere Compute Cluster name provided via Datacenter config. Migrating the cluster-id requires manual steps ([#9202](https://github.com/kubermatic/kubermatic/issues/9202))
- Enable the "vsphereCSIClusterID" feature flag when running the CCM/CSI migration ([#9557](https://github.com/kubermatic/kubermatic/issues/9557))
- For Kubernetes 1.22 and higher, etcd is updated to v3.5.3 to fix data consistency issues as reported by upstream developers ([#9610](https://github.com/kubermatic/kubermatic/issues/9610))
- The flag `--kubelet-certificate-authority` (introduced in KKP 2.19) is not set for "kubeadm" / "bringyourown" user clusters anymore ([#9674](https://github.com/kubermatic/kubermatic/issues/9674))


## [v2.19.4](https://github.com/kubermatic/kubermatic/releases/tag/v2.19.4)

This patch release enables etcd corruption checks on every etcd ring that is running etcd 3.5 (which applies to all user clusters with Kubernetes 1.22). This change is a [recommendation from the etcd maintainers](https://groups.google.com/a/kubernetes.io/g/dev/c/B7gJs88XtQc/m/rSgNOzV2BwAJ) due to issues in etcd 3.5 that can cause data consistency issues. The changes in this patch release will prevent corrupted etcd members from joining or staying in the etcd ring.

To replace a member in case of data consistency issues, please:

- Follow our documentation for [replacing an etcd member](https://docs.kubermatic.com/kubermatic/v2.19/cheat_sheets/etcd/replace_a_member/) if you are **not running etcd-launcher**.
- Delete the `PersistentVolume` that backs the corrupted etcd member to trigger the [automated recovery procedure](https://docs.kubermatic.com/kubermatic/v2.19/cheat_sheets/etcd/etcd-launcher/#automated-persistent-volume-recovery) if you **are using etcd-launcher**.

Please be aware we do not recommend enabling `etcd-launcher` on existing Kubernetes 1.22 environments at the time. This is due to the fact that the migration to `etcd-launcher` requires several etcd restarts and we currently recommend keeping the etcd ring as stable as possible (apart from the restarts triggered by this patch release to roll out the consistency checks).

### Misc

- For user clusters that use etcd 3.5 (Kubernetes 1.22 clusters), etcd corruption checks are turned on to detect [etcd data consistency issues](https://github.com/etcd-io/etcd/issues/13766). Checks run at etcd startup and every 4 hours ([#9477](https://github.com/kubermatic/kubermatic/issues/9477))


## [v2.19.3](https://github.com/kubermatic/kubermatic/releases/tag/v2.19.3)

### Breaking Changes
- ACTION REQUIRED: Secret name for S3 credentials updated to `kubermatic-s3-credentials`. If the secret `s3-credentials` was manually created instead of using the `minio` helm chart, new secret `kubermatic-s3-credentials` must be created. ([#9230](https://github.com/kubermatic/kubermatic/pull/9230))

### Bugfixes
- Fix LoadBalancer expose strategy for LBs with external DNS names instead of IPs ([#9105](https://github.com/kubermatic/kubermatic/pull/9105))
- `image-loader` parses custom versions in KubermaticConfiguration configuration files correctly ([#9154](https://github.com/kubermatic/kubermatic/pull/9154))
- Fix OpenVPNServerDown alerting rule to work as expected and not fire if Konnectivity is enabled. ([#9216](https://github.com/kubermatic/kubermatic/pull/9216))
- Fixes Preset API Body for preset creation and update API calls. ([#9300](https://github.com/kubermatic/kubermatic/pull/9300))
- Fix wrong CPU configs for the KubeVirt Virtual Machine Instances ([#1203](https://github.com/kubermatic/machine-controller/pull/1203))
- Fix vSphere VolumeAttachment cleanups during cluster deletion or node draining ([#1190](https://github.com/kubermatic/machine-controller/pull/1190))

### Misc
- Upgrade machine controller to v1.42.5 ([#9440](https://github.com/kubermatic/kubermatic/pull/9440))
- Support for `network:ha_router_replicated_interface` ports when discovering existing subnet router in Openstack ([#9176](https://github.com/kubermatic/kubermatic/pull/9176))


## [v2.19.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.19.2)

### Breaking Changes

- ACTION REQUIRED: Restore correct labels on nodeport-proxy-envoy Deployment. Deleting the existing Deployment for each cluster with the `LoadBalancer` expose strategy if upgrading from affected versions (v2.19.1 or v2.18.6) is necessary ([#9060](https://github.com/kubermatic/kubermatic/issues/9060))

### Bugfixes

- Fix applying resource requirements when using incomplete overrides (e.g. specifying only limits, but no request for a container) ([#9045](https://github.com/kubermatic/kubermatic/issues/9045))


## [v2.19.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.19.1)

### Bugfixes

- ICMP rules migration only runs on Azure NSGs created by KKP ([#8843](https://github.com/kubermatic/kubermatic/issues/8843))
- Fix apiserver network policy: allow all egress DNS traffic from the apiserver ([#8788](https://github.com/kubermatic/kubermatic/issues/8788))
- Automatic upgrades from 1.19.\* to 1.20.13 work as intended now ([#8821](https://github.com/kubermatic/kubermatic/issues/8821))
- Fixes installation issue with charts/logging/loki (error evaluating symlink) ([#8823](https://github.com/kubermatic/kubermatic/issues/8823))
- Update example values and KubermaticConfiguration to respect OIDC settings ([#8851](https://github.com/kubermatic/kubermatic/issues/8851))
- Fix bad owner references for ClusterRoleBindings ([#8858](https://github.com/kubermatic/kubermatic/issues/8858))
- If a network is set in the Hetzner cluster spec, it is now correctly applied to generated machines ([#8872](https://github.com/kubermatic/kubermatic/issues/8872))
- fix AWS cloud provider cleanup sometimes getting stuck when cleaning up tags ([#8879](https://github.com/kubermatic/kubermatic/issues/8879))
- Helm charts using dependencies (loki, promtail, nginx, cert-manager) now have specified apiVersion v2 ([#9038](https://github.com/kubermatic/kubermatic/issues/9038))

### Misc

- Add endpoints to v2 KKP API to query Nutanix clusters, projects and subnets ([#8736](https://github.com/kubermatic/kubermatic/issues/8736))
- Do not reference Nutanix cluster in KKP API endpoint path for subnets ([#8906](https://github.com/kubermatic/kubermatic/issues/8906))
- Support custom pod resources for NodePort-Proxy pod for the user cluster ([#9018](https://github.com/kubermatic/kubermatic/issues/9018))

### Known Issues

- Upgrading to this version is not recommended if the [`LoadBalancer` expose strategy](https://docs.kubermatic.com/kubermatic/v2.19/tutorials_howtos/networking/expose_strategies/) is used due to a bug in the `nodeport-proxy-envoy` Deployment template (#9059).
- Defining a component override that only gives partial resource configuration (only limits or requests) triggers an exception in the `seed-controller-manager`. If resources are defined in a component override, always define a full set of resource limits and requests to prevent this.


## [v2.19.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.19.0)

Before upgrading, make sure to read the [general upgrade guidelines](https://docs.kubermatic.com/kubermatic/v2.19/tutorials_howtos/upgrading/). Consider tweaking `seedControllerManager.maximumParallelReconciles` to ensure usercluster reconciliations will not cause resource exhaustion on seed clusters.

Several vulnerabilities were identified in Kubernetes (CVE-2021-44716, CVE-2021-44717, CVE-2021-3711, CVE-2021-3712, CVE-2021-33910) which have been fixed in Kubernetes 1.20.13, 1.21.8, 1.22.5.
Because of these updates, this KKP release includes automatic update rules for all 1.20/1.21/1.22 clusters older than these patch releases. This release also removes all affected Kubernetes versions from the list of supported versions. Once the automated controlplane updates have completed, an administrator must manually patch all vulnerable `MachineDeployment`s in all affected userclusters.

To lower the resource consumption on the seed clusters during the reconciliation / node rotation, it's recommended to adjust the `spec.seedControllerManager.maximumParallelReconciles` option in the `KubermaticConfiguration` to restrict the number of parallel updates.

The automatic update rules can, if needed, be overwritten using the `spec.versions.kubernetes.updates` field in the `KubermaticConfiguration`. See [#7825](https://github.com/kubermatic/kubermatic/issues/7824) for how the versions and updates are configured. It is however not recommended to deviate from the default and leave userclusters vulnerable.

### Highlights

- **All Your Clusters Under One Roof With External Cluster Support:** Import your existing AKS, EKS and GKE clusters and manage their entire lifecycle including upgrades and replica counts from our intuitive UI.
- **More Control Over Your Hybrid and Edge Deployments With OSM Support (Experimental):** KKP 2.19 adds experimental support for the [Operating System Manager](https://github.com/kubermatic/operating-system-manager) (OSM) to extend the functionality of the [Kubermatic Machine-Controller](https://github.com/kubermatic/machine-controller). This gives you better control over your OS in hybrid cloud and edge environments.
- **Best-in-Class Networking with Cilium Support:** With Cilium CNI support, KKP users can now choose between the two most popular CNIs Canal and Cilium or simply add and manage a CNI of their choice; regardless of whether it’s supported by KKP or not.
- **Enhanced Control Plane Networking with Konnectivity Support:** To harness the power of Cilium with eBPF, KKP 2.19 comes with added Konnectivity support. It provides TCP level proxy for the control plane (seed cluster) to worker nodes (user cluster) communication. It is based on the upstream [apiserver-network-proxy project](https://github.com/kubernetes-sigs/apiserver-network-proxy) and replaces the older KKP-specific solution based on OpenVPN and network address translation.
- **Multiple Backup Locations for Improved Business Continuity:** As KKP Admin, you can now configure multiple backup destinations from the UI Admin Panel. Users can select daily, weekly, monthly or customized backup options.

### Breaking Changes

- ACTION REQUIRED: When upgrading from older KKP versions (before v2.19.0), additional flags:
  - `--migrate-upstream-nginx-ingress` is required to perform upgrade of nginx-ingress-controller. Ingress will be briefly unavailable during this process ([#8333](https://github.com/kubermatic/kubermatic/issues/8333))
  - `--migrate-upstream-cert-manager` is required to perform the migration of the `cert-manager`. During the upgrade, the chart is uninstalled completely so there is a short time when certificates will not be renewed.([#8392](https://github.com/kubermatic/kubermatic/issues/8392))
- ACTION REQUIRED: Set the default Service Accounts `automountServiceAccountToken` property to false for all KKP provided namespaces and kube-system ([#8344](https://github.com/kubermatic/kubermatic/issues/8344))
- cert-manager chart has been updated to use upstream chart from https://charts.jetstack.io as a dependency, updated to version 1.6.1
  - ACTION REQUIRED: configuration for cert-manager should now be included under key `cert-manager` in values.yaml file (changed from `certManager`)
  - ACTION REQUIRED: The chart will no longer configure ClusterIssuers. Refer to the documentation of cert-manager for configuration guide: https://cert-manager.io/docs/configuration/ ([#8392](https://github.com/kubermatic/kubermatic/issues/8392))
- The old Docker repository at `quay.io/kubermatic/api` will not receive new images anymore, please use `quay.io/kubermatic/kubermatic` (CE) or `quay.io/kubermatic/kubermatic-ee` (EE) instead. This should only affect EE users who were still using the deprecated Helm chart ([#7922](https://github.com/kubermatic/kubermatic/issues/7922))
- BREAKING: `-dynamic-datacenters` was effectively already always true when using the KKP Operator and has now been removed as a flag. Support for handling a `datacenters.yaml` has been removed ([#7779](https://github.com/kubermatic/kubermatic/issues/7779))

#### Dashboard:

- Add multiple backup destinations.
  ACTION REQUIRED: KKP version 2.19 makes it possible to configure multiple destinations, therefore the current implementation of "backup buckets" will be deprecated soon. Migrate the configuration to a destination to keep using it in the future. It is reachable via the UI Admin Panel -> Backup Destinations ([#3911](https://github.com/kubermatic/dashboard/issues/3911))

### Supported Kubernetes Versions

- 1.20.13
- 1.20.14
- 1.21.8
- 1.22.5


### New and Enhanced Features

#### External Cluster
- New endpoint to get external cluster upgrades `GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/upgrades` ([#8115](https://github.com/kubermatic/kubermatic/issues/8115))
- New endpoint to list external cluster machine deployments `GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments` ([#8200](https://github.com/kubermatic/kubermatic/issues/8200))
- Add PATCH endpoint for external cluster `PATCH /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}`([#8135](https://github.com/kubermatic/kubermatic/issues/8135))
- New endpoint to list External MD Metrics: `/api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/nodes/metrics` ([#8239](https://github.com/kubermatic/kubermatic/issues/8239))
- Manage AKS Clusters natively. Extend external cluster functionality to import existing AKS cluster ([#8244](https://github.com/kubermatic/kubermatic/issues/8244))
- Create AKS NodePool ([#8399](https://github.com/kubermatic/kubermatic/issues/8399))
- Upgrade AKS NodePool Kubernetes version  ([#8349](https://github.com/kubermatic/kubermatic/issues/8349))
- Scale AKS NodePool ([#8349](https://github.com/kubermatic/kubermatic/issues/8349))
- Get AKS NodePool machineDeployments and nodes([#8349](https://github.com/kubermatic/kubermatic/issues/8349))
- Delete AKS NodePool ([#8349](https://github.com/kubermatic/kubermatic/issues/8349))
- Modified GET `/providers/aks/clusters` => `/projects/{project_id}/providers/aks/clusters` ([#8423](https://github.com/kubermatic/kubermatic/issues/8423))
- GET `/providers/eks/clusters` => `/projects/{project_id}/providers/eks/clusters` ([#8423](https://github.com/kubermatic/kubermatic/issues/8423))
- Add GET Endpoint `/api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/kubeconfig` ([#8122](https://github.com/kubermatic/kubermatic/issues/8122))


#### Operating System Management (OSM)
- Integrating Operating System Manager in KKP ([#8473](https://github.com/kubermatic/kubermatic/issues/8473))
- Validating webhooks for OSM resources, OperatingSystemProfile and OperatingSystemConfig ([#8503](https://github.com/kubermatic/kubermatic/issues/8503))
- Deploy OSM CRDs in KKP seed clusters ([#8528](https://github.com/kubermatic/kubermatic/issues/8528))
- Introduce new field enableOperatingSystemManager in Cluster and ClusterTemplate CRD specification:
  - use-osm flag is passed to machine-controller and machine-controller webhooks in case OSM is enabled
  - API integration for OSM ([#8523](https://github.com/kubermatic/kubermatic/issues/8523))
- CRDs for OSP and OSC are updated in kubermatic-operator and kubermatic-operator chart bumped to v0.3.37 ([#8573](https://github.com/kubermatic/kubermatic/issues/8573))


#### Open Policy Agent Integration
- Add option to set custom Gatekeeper controller and audit resource limits as part of OPAIntegration settings in the Cluster object ([#8397](https://github.com/kubermatic/kubermatic/issues/8397))
- Changed KKP Constraint parameter type to `map[string]json.RawMessage` ([#8050](https://github.com/kubermatic/kubermatic/issues/8050))

####  Backup & Restore
- seed.backupRestore is now deprecated in favor of seed.etcdBackupRestore which offers the option to manage multiple etcd backup destinations. The deprecated seed.backupRestore is still supported for now ([#8021](https://github.com/kubermatic/kubermatic/issues/8021))
- Add endpoint for listing possible backup destinations name for a cluster: GET `/api/v2/projects/{project_id}/clusters/{cluster_id}/backupdestinations` ([#8341](https://github.com/kubermatic/kubermatic/issues/8341))
- Add API endpoint for deleting Seed Backup Destinations ([#8512](https://github.com/kubermatic/kubermatic/issues/8512))
- Backup credentials API endpoint now supports multiple Seed backup destinations ([#8242](https://github.com/kubermatic/kubermatic/issues/8242))
- Extended Seed and EtcdBackupConfig API with multiple etcd backup destinations support ([#8228](https://github.com/kubermatic/kubermatic/issues/8228))
- EtcdRestore now supports multiple backup destinations. As there are multiple destinations, destination name needs to be provided alongside the usual backup name ([#8316](https://github.com/kubermatic/kubermatic/issues/8316))
- Etcd backups now support multiple destinations, which can be configured per Seed. If a destination is set for an EtcdBackupConfig, it will be used instead of the legacy `backup-s3` credentials secret and `s3-settings` backup bucket details ([#8283](https://github.com/kubermatic/kubermatic/issues/8283))
- Add support for setting a default etcd backup destination when using the new multiple etcd backup destinations feature. The default destination is used for the default EtcdBackupConfig which is created for each user cluster when automatic etcd backups are configured for a Seed ([#8628](https://github.com/kubermatic/kubermatic/issues/8628))

#### User Cluster Monitoring, Logging and Alerting
- Add Prometheus scraping if minio is running with TLS ([#8467](https://github.com/kubermatic/kubermatic/issues/8467))
- Promtail chart has been updated to use upstream chart from https://grafana.github.io/helm-charts as a dependency ([#8044](https://github.com/kubermatic/kubermatic/issues/8044))
- Upgrade User Cluster MLA components ([#8178](https://github.com/kubermatic/kubermatic/issues/8178)):
  | Component  | Image                                          | From   | To     |
  |------------|------------------------------------------------|--------|--------|
  | prometheus | prometheus/prometheus                          | 2.29.2 | 2.31.1 |
  | prometheus | prometheus-operator/prometheus-config-reloader | 0.49.0 | 0.52.0 |
  | promtail   | grafana/promtail                               | 2.2.1  | 2.4.1  |
  | promtail   | busybox                                        | 1.33   | 1.34   |

- Alertmanager groups alerts by alertname, namespace, seed_cluster, cluster labels ([#8193](https://github.com/kubermatic/kubermatic/issues/8193))
- Make user cluster Prometheus replicas configurable via KKP Cluster API ([#8223](https://github.com/kubermatic/kubermatic/issues/8223))
- Add MLA Alertmanager health status in the API /health endpoint (whether the alert manager configuration was successfully applied or not):- . extendedHealth. alertmanagerConfig ([#8128](https://github.com/kubermatic/kubermatic/issues/8128))
- Add MLAGateway health status in the /health API-endpoint:- . extendedHealth. mlaGateway ([#8153](https://github.com/kubermatic/kubermatic/issues/8153))
- Add MLA Health status in the API /health endpoint:- . extendedHealth. monitoring- . extendedHealth. logging ([#8082](https://github.com/kubermatic/kubermatic/issues/8082))
- Add v2 endpoints for KKP admins to manage rule group template:
  - GET `/api/v2/seeds/{seed_name}/rulegroups`
  - GET `/api/v2/seeds/{seed_name}/rulegroups/{rulegroup_id}`
  - POST `/api/v2/seeds/{seed_name}/rulegroups`
  - PUT  `/api/v2/seeds/{seed_name}/rulegroups/{rulegroup_id}`
  - DELETE `/api/v2/seeds/{seed_name}/rulegroups/{rulegroup_id}` ([#8158](https://github.com/kubermatic/kubermatic/issues/8158))
- Loki chart has been updated to use upstream chart from https://grafana.github.io/helm-charts as a dependency ([#8252](https://github.com/kubermatic/kubermatic/issues/8252))
- Add configurable root_url option for grafana ([#7927](https://github.com/kubermatic/kubermatic/issues/7927))
- Prometheus alert KubeAPILatencyHIGH has been replaced with KubeAPITerminatedRequests, which is more reliable ([#8727](https://github.com/kubermatic/kubermatic/issues/8727))


### Network
- Add support for CNI type "none" ([#8107](https://github.com/kubermatic/kubermatic/issues/8107))
- Allow CNI version upgrades ([#8150](https://github.com/kubermatic/kubermatic/issues/8150))
- Deploy network policies with default deny to kube-system namespace in user cluster. Adding network policies for custom deployments in kube-system namespace maybe required ([#8282](https://github.com/kubermatic/kubermatic/issues/8282))
- Add endpoint for listing supported CNI versions for a cluster. Add endpoint for listing supported versions for a CNI type ([#8483](https://github.com/kubermatic/kubermatic/issues/8483))
- nginx-ingress-controller chart has been updated to use upstream chart from https://kubernetes.github.io/ingress-nginx as a dependencyACTIONS REQUIRED:* entire nginx-ingresss-controller configuration is moved to a subkey in values file: `nginx.controller`* option to run as a daemonset removed (`nginx.asDaemonSet`)* option to schedule on master nodes removed (`nginx.ignoreMasterTaint`) and a way to reconfigure it has been added to the `values.yaml` file ([#8277](https://github.com/kubermatic/kubermatic/issues/8277))
- Enhance apiserver-to-node throughput by relaxing OpenVPN & envoy-agent CPU limits ([#8102](https://github.com/kubermatic/kubermatic/issues/8102))
- Change KonnectivityEnabled flag to a pointer in ClusterNetworkingConfig to make defaulting easier ([#8103](https://github.com/kubermatic/kubermatic/issues/8103))

#### Cilium
- Add Cilium CNI addon ([#7853](https://github.com/kubermatic/kubermatic/issues/7853))
- Add ebpf proxy mode support for Cilium CNI ([#7861](https://github.com/kubermatic/kubermatic/issues/7861))
- Add Cilium Hubble addon for CNI network observability ([#8548](https://github.com/kubermatic/kubermatic/issues/8548))

#### Canal
- Add Canal CNI addon v3.20 and make it the default CNI for new clusters ([#8081](https://github.com/kubermatic/kubermatic/issues/8081))
- Bump Flannel version in Canal to v0.15.1 to prevent segfaults in iptables ([#8478](https://github.com/kubermatic/kubermatic/issues/8478))
- Add Canal CNI v3.21 make it the default CNI ([#8510](https://github.com/kubermatic/kubermatic/issues/8510))

### Cloud Providers

#### Amazon Web Services (AWS)
- Introduce periodic reconciling for cloud resources used by userclusters; currently implemented for AWS-based userclusters ([#8101](https://github.com/kubermatic/kubermatic/issues/8101))
- Removed list aws regions endpoint ([#8529](https://github.com/kubermatic/kubermatic/issues/8529))
- Add API support for assuming AWS IAM roles. This allows running user clusters in e.g. external AWS accounts ([#8038](https://github.com/kubermatic/kubermatic/issues/8038))

#### Microsoft Azure
- Open Azure Network Security Group for external traffic to NodePort port ranges ([#7966](https://github.com/kubermatic/kubermatic/issues/7966))
- Azure cloud resources are periodically reconciled ([#8213](https://github.com/kubermatic/kubermatic/issues/8213))

#### OpenStack
- Add support for Openstack authentication with `project name` and  `project id`. `tenant name`  and `tenant id`  are deprecated ([#8211](https://github.com/kubermatic/kubermatic/issues/8211))

#### VMware vSphere
- Support latest vSphere Cloud Controller Manager and  CSI driver for Kubernetes 1.22 ([#8505](https://github.com/kubermatic/kubermatic/issues/8505))

#### Google Cloud Platform (GCP)
- Manage GKE Clusters natively. Extend external cluster functionality to import existing GKE cluster ([#8046](https://github.com/kubermatic/kubermatic/issues/8046))

#### KubeVirt
- Add API endpoints for Kubevirt VirtualMachineInstancePresets and StorageClasses ([#8441](https://github.com/kubermatic/kubermatic/issues/8441))
- Support for KubeVirt CSI driver ([#8416](https://github.com/kubermatic/kubermatic/issues/8416))

### Nutanix
- Add experimental/alpha cloud provider support for Nutanix ([#8448](https://github.com/kubermatic/kubermatic/issues/8448))

### Misc
- Add tolerations to user cluster system daemonsets ([#8425](https://github.com/kubermatic/kubermatic/issues/8425))
- Components running on user clusters are set up with the runtime's default seccomp profile ([#8326](https://github.com/kubermatic/kubermatic/issues/8326))
- Support for configuring the EventRateLimit admission plugin ([#8291](https://github.com/kubermatic/kubermatic/issues/8291))
- Add support to remove a preset provider via the REST API ([#8093](https://github.com/kubermatic/kubermatic/issues/8093))
- Add support to update the requiredEmails of a preset via REST-API and Admins will see all presets within KKP independent of their email ([#8087](https://github.com/kubermatic/kubermatic/issues/8087))
- Add an endpoint to list users and extend the user object to allow checking the last seen date ([#7989](https://github.com/kubermatic/kubermatic/issues/7989))
- etcd-launcher uses TLS peer connections for new etcd clusters and automatically upgrades existing etcd clusters- etcd-launcher cannot be disabled as a feature on a cluster to go back to plain etcd ([#8065](https://github.com/kubermatic/kubermatic/issues/8065))
- Delete flatcar update operator if flatcar nodes are not schedulable ([#7316](https://github.com/kubermatic/kubermatic/issues/7316))
- OpenAPI v3 CRD schema generation ([#8027](https://github.com/kubermatic/kubermatic/issues/8027))
- Cluster resources are now defaulted and validated a lot more thoroughly by the webhooks ([#8346](https://github.com/kubermatic/kubermatic/issues/8346))
- `kubectl get clusters` now shows the cloud provider and datacenter name ([#8352](https://github.com/kubermatic/kubermatic/issues/8352))
- User clusters now expose `spec.auditLogging.policyPreset`, which can be set to `metadata`, `minimal` or `recommended` to configure audit logging policy ([#8161](https://github.com/kubermatic/kubermatic/issues/8161))
- Provide functionality to define a default cluster template for new clusters per seed.DefaultComponentSettings of the Seed object is deprecated in favor of DefaultClusterTemplate ([#8089](https://github.com/kubermatic/kubermatic/issues/8089))
- Child cluster etcd services now use `spec.publishNotReadyAddresses` instead of the `service.alpha.kubernetes.io/tolerate-unready-endpoints` annotation (deprecated in Kubernetes v1.11) to ensure compatibility with `EndpointSlice` consumers ([#7968](https://github.com/kubermatic/kubermatic/issues/7968))
- Add endpoints for machine deployment:
  - GET `/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}`
  - PATCH `/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}` ([#8224](https://github.com/kubermatic/kubermatic/issues/8224))
- Profiling endpoints on control plane components have been disabled ([#8110](https://github.com/kubermatic/kubermatic/issues/8110))
- cert-manager's upgrade will reinstall the chart cleanly, existing ClusterIssuers created by a helm chart will be recreated ([#8555](https://github.com/kubermatic/kubermatic/issues/8555))
- KKP Installer will now check that all userclusters match the given versioning configuration prior to upgrading KKP. This ensures no userclusters are suddenly "orphaned" and incompatible with KKP ([#8670](https://github.com/kubermatic/kubermatic/issues/8670))
- Add `spec.debugLog` field to `Cluster` objects to toggle the verbose log on the usercluster-controller-manager ([#8735](https://github.com/kubermatic/kubermatic/issues/8735))

### Bugfixes

#### Open Policy Agent Integration
- Fix for AllowedRegistry Constraint parameters being badly parsed, causing the Constraint not to work ([#8722](https://github.com/kubermatic/kubermatic/issues/8722))

####  Backup & Restore
- Fix for issue with EtcdBackupConfigs from multiple clusters having the same ID, which was causing a bug in the UI ([#7896](https://github.com/kubermatic/kubermatic/issues/7896))
- Fix minio chart template to correctly reference 'certificateSecret' value for TLS setup ([#8234](https://github.com/kubermatic/kubermatic/issues/8234))

#### User Cluster Monitoring, Logging and Alerting
- Fixed user cluster MLA certificate issue by LoadBalancer expose strategy ([#7877](https://github.com/kubermatic/kubermatic/issues/7877))
- Fix Grafana dashboard for MinIO to display the correct Prometheus metrics ([#8687](https://github.com/kubermatic/kubermatic/issues/8687))
- Fix Grafana dashboards using legacy kube-state-metrics metrics for CPU/memory limits and requests ([#8749](https://github.com/kubermatic/kubermatic/issues/8749))

### Metering
- Fix some hardcoded Docker images for seed-proxy and metering components ([#8615](https://github.com/kubermatic/kubermatic/issues/8615))
- Fix disabling metering not having any effect ([#8673](https://github.com/kubermatic/kubermatic/issues/8673))
- Fixed high resource usage during metering startup ([#8270](https://github.com/kubermatic/kubermatic/issues/8270))

### Network
- Fix setting of nodeport-proxy resource requests/limits, relax default nodeport-proxy envoy limits ([#8169](https://github.com/kubermatic/kubermatic/issues/8169))
- Fix apiserver NetworkPolicy for OIDC issuer - allow local path to KKP ingress-controller & IP address in the OIDC issuer URL ([#8419](https://github.com/kubermatic/kubermatic/issues/8419))
- Fix Delegated OIDC Authentication feature by allowing appiserver to oidc-issuer communication in apiserver network policies ([#8264](https://github.com/kubermatic/kubermatic/issues/8264))
- Fix LoadBalancer expose strategy for LBs with external DNS names instead of IPs ([#7933](https://github.com/kubermatic/kubermatic/issues/7933))
- Fix seed-proxy forbidding all traffic, breaking Karma dashboards ([#8016](https://github.com/kubermatic/kubermatic/issues/8016))
- Fix nginx-ingress role to allow update of leader configmap ([#7942](https://github.com/kubermatic/kubermatic/issues/7942))


#### Canal
- Fix CNI reconciliation issue after upgrade to k8s 1.22. For clusters running Canal v3.8, automatically upgrade Canal CNI version to latest supported version upon k8s upgrade to version >= 1.22 ([#8396](https://github.com/kubermatic/kubermatic/issues/8396))

### Cloud Providers

#### Google Cloud Platform (GCP)
- Fix missing validation for GCE datacenter zone_suffixes ([#8525](https://github.com/kubermatic/kubermatic/issues/8525))

#### VMware vSphere
- Fix vSphere clusters getting stuck after CSI migration due to bad ValidatingWebhookConfiguration reconciling ([#8738](https://github.com/kubermatic/kubermatic/issues/8738))

### Misc
- The field `userCluster.overwriteRegistry` in the KubermaticConfiguration is now properly respectedby all provided addons and by more user cluster controllers: core-dns, envoy-agent, gatekeeper,konnectivity, kubernetes-dashboard, mla-prometheus, mla-promtail, node-local-dns ([#8055](https://github.com/kubermatic/kubermatic/issues/8055))
- Fix IDP icons in Dex theme ([#8319](https://github.com/kubermatic/kubermatic/issues/8319))
- Create missing ClusterRole 'cert-manager-edit' for cert-manager ([#8554](https://github.com/kubermatic/kubermatic/issues/8554))
- Fixes a bug where `$$` in the environment-variables for machine-controller was interpreted in the Kubernetes Manifest and caused machine-controller to be unable to deploy resources, when for e.g. the password contains two consecutive `$` signs ([#7984](https://github.com/kubermatic/kubermatic/issues/7984))
- kubeadm-config is updated based on Kubernetes version ([#8149](https://github.com/kubermatic/kubermatic/issues/8149))
- Fix kubermatic-installer not properly updating CRDs ([#8552](https://github.com/kubermatic/kubermatic/issues/8552))
- Fix PodDisruptionBudgets for master/seed-controller-manager blocking node rotations ([#8672](https://github.com/kubermatic/kubermatic/issues/8672))
- Fix clusters being occasionally stuck in deletion because seed-level RBAC resources were deleted too early ([#8744](https://github.com/kubermatic/kubermatic/issues/8744))

### Updates
- Update nginx-ingress-controller to 1.0.0 to support Kubernetes 1.22 master/seed clusters ([#7845](https://github.com/kubermatic/kubermatic/issues/7845))
- Update nginx-ingress-controller to 1.0.2 ([#7875](https://github.com/kubermatic/kubermatic/issues/7875))
- Update Kubernetes Dashboard to v2.4.0 ([#8172](https://github.com/kubermatic/kubermatic/issues/8172))
- Update Dex to 2.30.0 ([#7846](https://github.com/kubermatic/kubermatic/issues/7846))
- Update to controller-runtime 0.10 / k8s 1.22 ([#7671](https://github.com/kubermatic/kubermatic/issues/7671))
- Update controller-runtime to 0.10.1 ([#7857](https://github.com/kubermatic/kubermatic/issues/7857))
- Update to controller-runtime 0.11.0 ([#8551](https://github.com/kubermatic/kubermatic/issues/8551))
- Update to Go 1.17.1 ([#7873](https://github.com/kubermatic/kubermatic/issues/7873))
- Upgrade etcd to 3.5.1 for Kubernetes 1.22 (and higher) ([#8559](https://github.com/kubermatic/kubermatic/issues/8559))
- Update kube-state-metrics to 2.2.3 ([#8049](https://github.com/kubermatic/kubermatic/issues/8049))
- Bump machine controller v1.36.1 ([#8099](https://github.com/kubermatic/kubermatic/issues/8099))
- Bump operating-system-manager to v0.3.2 ([#8570](https://github.com/kubermatic/kubermatic/issues/8570))
- Upgrade machine controller to 1.36.2 ([#8233](https://github.com/kubermatic/kubermatic/issues/8233))
- machine-controller is updated to v1.41.0 ([#8590](https://github.com/kubermatic/kubermatic/issues/8590))
- machine-controller updated to v1.41.1 ([#8693](https://github.com/kubermatic/kubermatic/issues/8693))
- operating-system-manager updated to v0.3.6 ([#8693](https://github.com/kubermatic/kubermatic/issues/8693))
- Update machine-controller to v1.42.0 ([#8731](https://github.com/kubermatic/kubermatic/issues/8731))
- Update machine controller to 1.42.1 ([#8757](https://github.com/kubermatic/kubermatic/issues/8757))
- Removed support for Kubernetes 1.19 ([#8167](https://github.com/kubermatic/kubermatic/issues/8167))
- Add support for Kubernetes 1.22.5, 1.21.8, and 1.20.14  ([#8472](https://github.com/kubermatic/kubermatic/issues/8472))
- Automatically upgrade clusters running Kubernetes 1.21 to 1.21.8 to include fixes for CVE-2021-44716 and CVE-2021-44717  ([#8472](https://github.com/kubermatic/kubermatic/issues/8472))
- Automatically upgrade clusters running Kubernetes 1.22 to 1.22.5 to include fixes for CVE-2021-44716 and CVE-2021-44717 ([#8472](https://github.com/kubermatic/kubermatic/issues/8472))
- Update to Go 1.17.5 ([#8472](https://github.com/kubermatic/kubermatic/issues/8472))
- Add support for Kubernetes version v1.20.13 and automatically upgrading clusters with version < v1.20.13 (fixes CVE-2021-3711, CVE-2021-3712, CVE-2021-33910) ([#8251](https://github.com/kubermatic/kubermatic/issues/8251))
- Add support for Kubernetes version v1.21.7 and automatically upgrading clusters with version < v1.21.7 (fixes CVE-2021-3711, CVE-2021-3712, CVE-2021-33910) ([#8251](https://github.com/kubermatic/kubermatic/issues/8251))
- Add support for Kubernetes version v1.22.4 and automatically upgrading clusters with version < v1.22.4 (fixes CVE-2021-3711, CVE-2021-3712, CVE-2021-33910) ([#8251](https://github.com/kubermatic/kubermatic/issues/8251))


### Dashboard
- Allow users to select one of the two supported CNIs - Canal or Cilium ([#3709](https://github.com/kubermatic/dashboard/issues/3709))
- Remove type column from cluster list ([#3751](https://github.com/kubermatic/dashboard/issues/3751))
- Add account audits support to the admin settings ([#3782](https://github.com/kubermatic/dashboard/issues/3782))
- Allow configuring AWS AssumeRole credentials for user clusters ([#3811](https://github.com/kubermatic/dashboard/issues/3811))
- Add Konnectivity support ([#3822](https://github.com/kubermatic/dashboard/issues/3822))
- Add support for RHEL and Flatcar to the VSphere provider. Make RHEL subscription manager fields optional ([#3824](https://github.com/kubermatic/dashboard/issues/3824))
- Disable dialog select form until existing template is selected ([#3836](https://github.com/kubermatic/dashboard/issues/3836))
- Fix a bug where OPA data was not retrieved when there were no machine deployments in the cluster ([#3851](https://github.com/kubermatic/dashboard/issues/3851))
- Add support for Openstack authentication with `project name` and  `project id`.  `tenant name`  and `tenant id`  are deprecated and have been renamed to `project name` and  `project id` in the UI ([#3854](https://github.com/kubermatic/dashboard/issues/3854))
- Add audit policy preset picker to the wizard ([#3862](https://github.com/kubermatic/dashboard/issues/3862))
- Add support for importing GKE clusters ([#3892](https://github.com/kubermatic/dashboard/issues/3892))
- Allow audit policy preset changes during cluster edit ([#3908](https://github.com/kubermatic/dashboard/issues/3908))
- Hidden edit provider settings action for kubeAdm as it is not supported ([#3910](https://github.com/kubermatic/dashboard/issues/3910))
- Add support for external providers presets ([#3916](https://github.com/kubermatic/dashboard/issues/3916))
- Add support for importing AKS and EKS clusters ([#3922](https://github.com/kubermatic/dashboard/issues/3922))
- Allow CNI Version Upgrade from the UI ([#3923](https://github.com/kubermatic/dashboard/issues/3923))
- Support for EventRateLimit admission plugin ([#3948](https://github.com/kubermatic/dashboard/issues/3948))
- Cluster and MachineDeployment names are validated to be lowercase, alphanumerical and with dashes in between ([#3963](https://github.com/kubermatic/dashboard/issues/3963))
- Support to enable experimental feature OSM ([#3980](https://github.com/kubermatic/dashboard/issues/3980))
- Redesigned the cluster details additional information section ([#4078](https://github.com/kubermatic/dashboard/issues/4078))
- Add option for administrators to specify default destination in "Backup Destinations" view ([#4080](https://github.com/kubermatic/dashboard/issues/4080))

### Known issues
- CNI: service LoadBalancer not working on Azure RHEL ([8768](https://github.com/kubermatic/kubermatic/issues/8768))
- Incompatibilty between Kubevirt csi driver and Kubevirt ccm (namespace)([8772](https://github.com/kubermatic/kubermatic/issues/8772))
