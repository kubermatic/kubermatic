# Kubermatic 2.18

- [v2.18.0](#v2180)
- [v2.18.1](#v2181)
- [v2.18.2](#v2182)
- [v2.18.3](#v2183)
- [v2.18.4](#v2184)
- [v2.18.5](#v2185)
- [v2.18.6](#v2186)
- [v2.18.7](#v2187)
- [v2.18.8](#v2188)
- [v2.18.9](#v2189)
- [v2.18.10](#v21810)

## [v2.18.10](https://github.com/kubermatic/kubermatic/releases/tag/v2.18.10)

With this patch release, etcd for Kubernetes 1.22+ is upgraded to etcd 3.5.3. Data consistency issues as reported in previous release notes are fixed. Warnings and recommendations related to that can be considered withdrawn for this release.

- Add `vsphereCSIClusterID` feature flag for the cluster object. This feature flag changes the cluster-id in the vSphere CSI config to the cluster name instead of the vSphere Compute Cluster name provided via Datacenter config. Migrating the cluster-id requires manual steps ([#9202](https://github.com/kubermatic/kubermatic/issues/9202))
- Enable the "vsphereCSIClusterID" feature flag when running the CCM/CSI migration ([#9557](https://github.com/kubermatic/kubermatic/issues/9557))
- For Kubernetes 1.22 and higher, etcd is updated to v3.5.3 to fix data consistency issues as reported by upstream developers ([#9611](https://github.com/kubermatic/kubermatic/issues/9611))


## [v2.18.9](https://github.com/kubermatic/kubermatic/releases/tag/v2.18.9)

This patch release enables etcd corruption checks on every etcd ring that is running etcd 3.5 (which applies to all user clusters with Kubernetes 1.22). This change is a [recommendation from the etcd maintainers](https://groups.google.com/a/kubernetes.io/g/dev/c/B7gJs88XtQc/m/rSgNOzV2BwAJ) due to issues in etcd 3.5 that can cause data consistency issues. The changes in this patch release will prevent corrupted etcd members from joining or staying in the etcd ring.

To replace a member in case of data consistency issues, please:

- Follow our documentation for [replacing an etcd member](https://docs.kubermatic.com/kubermatic/v2.18/cheat_sheets/etcd/replace_a_member/) if you are **not running etcd-launcher**.
- Delete the `PersistentVolume` that backs the corrupted etcd member to trigger the [automated recovery procedure](https://docs.kubermatic.com/kubermatic/v2.18/cheat_sheets/etcd/etcd-launcher/#automated-persistent-volume-recovery) if you **are using etcd-launcher**.

Please be aware we do not recommend enabling `etcd-launcher` on existing Kubernetes 1.22 environments at the time. This is due to the fact that the migration to `etcd-launcher` requires several etcd restarts and we currently recommend keeping the etcd ring as stable as possible (apart from the restarts triggered by this patch release to roll out the consistency checks).

### Misc

- For user clusters that use etcd 3.5 (Kubernetes 1.22 clusters), etcd corruption checks are turned on to detect [etcd data consistency issues](https://github.com/etcd-io/etcd/issues/13766). Checks run at etcd startup and every 4 hours ([#9477](https://github.com/kubermatic/kubermatic/issues/9477))


## [v2.18.8](https://github.com/kubermatic/kubermatic/releases/tag/v2.18.8)

### Bugfixes
- Fix LoadBalancer expose strategy for LBs with external DNS names instead of IPs ([#9105](https://github.com/kubermatic/kubermatic/pull/9105))
- `image-loader` parses custom versions in KubermaticConfiguration configuration files correctly ([#9154](https://github.com/kubermatic/kubermatic/pull/9154))
- Fix wrong CPU configs for the KubeVirt Virtual Machine Instances ([#1203](https://github.com/kubermatic/machine-controller/pull/1203))
- Fix vSphere VolumeAttachment cleanups during cluster deletion or node draining ([#1190](https://github.com/kubermatic/machine-controller/pull/1190))

### Misc
- Upgrade machine controller to v1.36.4
- Support for `network:ha_router_replicated_interface` ports when discovering existing subnet router in Openstack ([#9176](https://github.com/kubermatic/kubermatic/pull/9176))


## [v2.18.7](https://github.com/kubermatic/kubermatic/releases/tag/v2.18.7)

### Breaking Changes

- ACTION REQUIRED: Restore correct labels on nodeport-proxy-envoy Deployment. Deleting the existing Deployment for each cluster with the `LoadBalancer` expose strategy if upgrading from affected version (v2.18.6) is necessary ([#9060](https://github.com/kubermatic/kubermatic/issues/9060))

### Misc

- Fix applying resource requirements when using incomplete overrides (e.g. specifying only limits, but no request for a container) ([#9045](https://github.com/kubermatic/kubermatic/issues/9045))


## [v2.18.6](https://github.com/kubermatic/kubermatic/releases/tag/v2.18.6)

### Bugfixes

- ICMP rules migration only runs on Azure NSGs created by KKP ([#8843](https://github.com/kubermatic/kubermatic/issues/8843))
- Add Prometheus scraping if minio is running with TLS ([#8467](https://github.com/kubermatic/kubermatic/issues/8467))
- Fix Grafana dashboards using legacy kube-state-metrics metrics for CPU/memory limits and requests ([#8749](https://github.com/kubermatic/kubermatic/issues/8749))
- Fix apiserver network policy: allow all egress DNS traffic from the apiserver ([#8852](https://github.com/kubermatic/kubermatic/issues/8852))

### Misc

- Support custom pod resources for NodePort-Proxy pod for the user cluster ([#9028](https://github.com/kubermatic/kubermatic/issues/9028))

### Known Issues

- Upgrading to this version is not recommended if the [`LoadBalancer` expose strategy](https://docs.kubermatic.com/kubermatic/v2.18/guides/kkp_networking/expose_strategies/) is used due to a bug in the `nodeport-proxy-envoy` Deployment template (#9059).
- Defining a component override that only gives partial resource configuration (only limits or requests) triggers an exception in the `seed-controller-manager`. If resources are defined in a component override, always define a full set of resource limits and requests to prevent this.


## [v2.18.5](https://github.com/kubermatic/kubermatic/releases/tag/v2.18.5)

### Bugfixes

- Fix PodDisruptionBudgets for master/seed-controller-manager blocking node rotations ([#8672](https://github.com/kubermatic/kubermatic/issues/8672))
- Fix vSphere clusters getting stuck after CSI migration due to bad ValidatingWebhookConfiguration reconciling ([#8738](https://github.com/kubermatic/kubermatic/issues/8738))

### Misc

- Upgrade etcd to 3.5.1 for Kubernetes 1.22 (and higher) ([#8563](https://github.com/kubermatic/kubermatic/issues/8563))
- Add `spec.debugLog` field to `Cluster` objects to toggle the verbose log on the usercluster-controller-manager ([#8735](https://github.com/kubermatic/kubermatic/issues/8735))


## [v2.18.4](https://github.com/kubermatic/kubermatic/releases/tag/v2.18.4)

### Bugfixes

- Fix CNI reconciliation issue after upgrade to k8s 1.22. For clusters running Canal v3.8, automatically upgrade Canal CNI version to v3.19 upon k8s upgrade to version >= 1.22 ([#8394](https://github.com/kubermatic/kubermatic/issues/8394))
- Fix apiserver NetworkPolicy for OIDC issuer - allow local path to KKP ingress-controller & IP address in the OIDC issuer URL ([#8419](https://github.com/kubermatic/kubermatic/issues/8419))
- Fix minio chart template to correctly reference 'certificateSecret' value for TLS setup ([#8420](https://github.com/kubermatic/kubermatic/issues/8420))

### Updates

- Bump Flannel version in Canal to v0.15.1 ([#8479](https://github.com/kubermatic/kubermatic/issues/8479))
- Add support for Kubernetes 1.22.5, 1.21.8, and 1.20.14 ([#8481](https://github.com/kubermatic/kubermatic/issues/8481))
- Automatically upgrade clusters running Kubernetes 1.21 to 1.21.8 to include fixes for CVE-2021-44716 and CVE-2021-44717 ([#8481](https://github.com/kubermatic/kubermatic/issues/8481))
- Automatically upgrade clusters running Kubernetes 1.22 to 1.22.5 to include fixes for CVE-2021-44716 and CVE-2021-44717 ([#8481](https://github.com/kubermatic/kubermatic/issues/8481))
- Update to Go 1.16.12 ([#8481](https://github.com/kubermatic/kubermatic/issues/8481))

### Misc

- Add tolerations to user cluster system daemonsets ([#8425](https://github.com/kubermatic/kubermatic/issues/8425))
- Add seed dns overwrite option to mla links ([#3912](https://github.com/kubermatic/dashboard/issues/3912))


## [v2.18.3](https://github.com/kubermatic/kubermatic/releases/tag/v2.18.3)

### Bugfixes

- Fix Delegated OIDC Authentication feature by allowing apiserver-to-oidc-issuer communication in apiserver network policies ([#8264](https://github.com/kubermatic/kubermatic/issues/8264))
- Fix IDP icons in Dex theme ([#8319](https://github.com/kubermatic/kubermatic/issues/8319))
- Fix setting of nodeport-proxy resource requests/limits, relax default nodeport-proxy envoy limits ([#8169](https://github.com/kubermatic/kubermatic/issues/8169))

### Misc

- Add support for Kubernetes version v1.20.13 and automatically upgrading clusters with version < v1.20.13 (fixes CVE-2021-3711, CVE-2021-3712, CVE-2021-33910) ([#8268](https://github.com/kubermatic/kubermatic/issues/8268))
- Add support for Kubernetes version v1.21.7 and automatically upgrading clusters with version < v1.21.7 (fixes CVE-2021-3711, CVE-2021-3712, CVE-2021-33910) ([#8268](https://github.com/kubermatic/kubermatic/issues/8268))
- Add support for Kubernetes version v1.22.4 and automatically upgrading clusters with version < v1.22.4 (fixes CVE-2021-3711, CVE-2021-3712, CVE-2021-33910) ([#8268](https://github.com/kubermatic/kubermatic/issues/8268))
- Add support to update the `requiredEmails` of a Preset via REST-API and admins will see all Presets within KKP independent of their email ([#8168](https://github.com/kubermatic/kubermatic/issues/8168))
- Update Kubernetes Dashboard to v2.4.0 ([#8172](https://github.com/kubermatic/kubermatic/issues/8172))
- Update kubeadm-config addon based on Kubernetes version ([#8149](https://github.com/kubermatic/kubermatic/issues/8149))


## [v2.18.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.18.2)

### Bugfixes

- Fix a bug where `$$` in the environment-variables for machine-controller was interpreted in the Kubernetes Manifest and caused machine-controller to be unable to deploy resources, when for e.g. the password contains two consecutive `$` signs ([#7984](https://github.com/kubermatic/kubermatic/issues/7984))
- Fix issue with EtcdBackupConfigs from multiple clusters having the same ID, which was causing a bug in the UI ([#7896](https://github.com/kubermatic/kubermatic/issues/7896))
- Fix nginx-ingress Role to allow update of leader ConfigMap ([#7942](https://github.com/kubermatic/kubermatic/issues/7942))
- Fix seed-proxy forbidding all traffic, breaking Karma dashboards ([#8016](https://github.com/kubermatic/kubermatic/issues/8016))

### Misc

- Add configurable `root_url` option for Grafana Helm chart ([#7930](https://github.com/kubermatic/kubermatic/issues/7930))
- Update machine controller to 1.36.1 ([#8099](https://github.com/kubermatic/kubermatic/issues/8099))
- Usercluster etcd services now use `spec.publishNotReadyAddresses` instead of the `service.alpha.kubernetes.io/tolerate-unready-endpoints` annotation (deprecated in Kubernetes v1.11) to ensure compatibility with `EndpointSlice` consumers ([#7968](https://github.com/kubermatic/kubermatic/issues/7968))


## [v2.18.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.18.1)

This release primarily improves support for Kubernetes 1.22 master/seed clusters.

### Misc

- Update nginx-ingress-controller to 1.0.2 ([#7845](https://github.com/kubermatic/kubermatic/issues/7845), [#7875](https://github.com/kubermatic/kubermatic/issues/7875))
- Update Dex to 2.30.0 ([#7846](https://github.com/kubermatic/kubermatic/issues/7846))
- Fix user cluster MLA certificate issue by LoadBalancer expose strategy ([#7877](https://github.com/kubermatic/kubermatic/issues/7877))
- Fix styling of very long labels ([#3730](https://github.com/kubermatic/dashboard/pull/3730))


## [v2.18.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.18.0)

Before upgrading, make sure to read the [general upgrade guidelines](https://docs.kubermatic.com/kubermatic/v2.18/tutorials_howtos/upgrading/). Consider tweaking `seedControllerManager.maximumParallelReconciles` to ensure usercluster reconciliations will not cause resource exhausting on seed clusters.

Two vulnerabilities were identified in Kubernetes ([CVE-2021-25741](https://github.com/kubernetes/kubernetes/issues/104980) and [CVE-2020-8561](https://github.com/kubernetes/kubernetes/issues/104720)) of which one (CVE-2021-25741) was fixed in Kubernetes 1.19.15 / 1.20.11 / 1.21.5 / 1.22.2. CVE-2020-8561 is mitigated by Kubermatic not allowing users to reconfigure the kube-apiserver.

Because of these updates, this KKP release includes automatic update rules for all 1.19/1.20/1.21/1.22 clusters older than these patch releases. This release also removes all affected Kubernetes versions from the list of supported versions. While CVE-2020-8561 affects the controlplane, CVE-2021-25741 affects the kubelets, which means that updating the controlplane is not enough. Once the automated controlplane updates have completed, an administrator must manually patch all vulnerable `MachineDeployment`s in all affected userclusters.

To lower the resource consumption on the seed clusters during the reconciliation / node rotation, it's recommended to adjust the `spec.seedControllerManager.maximumParallelReconciles` option in the `KubermaticConfiguration` to restrict the number of parallel updates.

The automatic update rules can, if needed, be overwritten using the `spec.versions.kubernetes.updates` field in the `KubermaticConfiguration`. See [#7825](https://github.com/kubermatic/kubermatic/issues/7824) for how the versions and updates are configured. It is however not recommended to deviate from the default and leave userclusters vulnerable.

### Highlights

* User Cluster Monitoring, Logging and Alerting
* Cluster Templates for Deploying Optimal Clusters Instantly
* Metering Tool Integration for Easier Accountability of Resources
* AWS Spot Instances Support to Optimize Workload
* User Cluster Backup & Restore UI
* Enhancements on Open Policy Agent for Allowed Container Registries and Standard Policies
* Kubevirt Cloud Controller Integration
* Add Kubernetes 1.22 Support
* Enable CCM and CSI Migration on distinct Cloud Providers
* Docker to containerd Container Runtime Migration

### Breaking Changes

- Kubernetes 1.19 is now the minimum supported version for master, seed and user clusters. Please upgrade all userclusters to 1.19
  prior to upgrading to KKP 2.18. See the [documentation](https://docs.kubermatic.com/kubermatic/v2.17/tutorials_howtos/upgrading/upgrade_from_2.16_to_2.17/chart_migration/)
  for more details.
- Helm 2 is not supported anymore, please use Helm 3 instead. It might still be possible to install KKP using Helm 2, but it's neither
  supported nor recommended.
- The (in 2.14) deprecated `kubermatic` and `nodeport-proxy` Helm charts have now been removed. If you haven't done so, migrate your KKP installation
  to use the `kubermatic-operator` Helm chart or, even better, use the KKP installer. The KKP operator will automatically manage the
  nodeport-proxy, so the chart is not required anymore.
- The `cert-manager` Helm chart requires admins to set the Let's Encrypt account email explicitly ([#7184](https://github.com/kubermatic/kubermatic/issues/7184))
- The experimental new backup mechanism was updated and available as an opt-in option per Seed. If enabled, it will replace the old backups and is not backwards compatible. Users that are already using the experimental backup mechanism, be aware that to prepare it for regular use, we made some changes, admins please check the [documentation](https://docs.kubermatic.com/kubermatic/v2.18/cheat_sheets/etcd/backup-and-restore/).

### Supported Kubernetes Versions

* 1.19.0
* 1.19.2
* 1.19.3
* 1.19.8
* 1.19.9
* 1.19.13
* 1.20.2
* 1.20.5
* 1.20.9
* 1.21.0
* 1.21.3
* 1.22.1

### New and Enhanced Features

#### User Cluster Monitoring, Logging and Alerting

- Add MLA support ([#3293](https://github.com/kubermatic/dashboard/issues/3293))
- Extend cluster details view by tab User Cluster Alert Rules for MLA ([#3476](https://github.com/kubermatic/dashboard/issues/3476))
- Add MLA options to admin settings ([#7009](https://github.com/kubermatic/kubermatic/issues/7009))
- Add `MLAAdminSetting` CRD ([#7603](https://github.com/kubermatic/kubermatic/issues/7603))
- Add v2 endpoints for KKP admin to manage MLA admin setting ([#7652](https://github.com/kubermatic/kubermatic/issues/7652))
- Expose MLA options in the Seed CRD and API object ([#6967](https://github.com/kubermatic/kubermatic/issues/6967))
- Add option to specify initContainer to override inotify max user instances for promtail chart ([#7388](https://github.com/kubermatic/kubermatic/issues/7388))
- Add Blackbox Exporter configuration, scrape config, dashboard ([#7376](https://github.com/kubermatic/kubermatic/issues/7376))
  - A Blackbox Exporter module that can be used to perform health status checks for HTTPS endpoint with TLS verify skipped.
  - A Blackbox Exporter dashboard in Grafana.
  - A scrape job in Prometheus will be used to check the health status of ClusterIP services of user cluster Kubernetes API servers
- Allow configuring extra args in Prometheus Helm chart ([#7443](https://github.com/kubermatic/kubermatic/issues/7443))
- Allow configuring remote_write in Prometheus Helm chart ([#7288](https://github.com/kubermatic/kubermatic/issues/7288))
- Fix Helm post-rendering problems within monitoring/prometheus chart due to duplicate resource definitions ([#7425](https://github.com/kubermatic/kubermatic/issues/7425))
- Fix dashboard source in the Prometheus Exporter dashboard ([#7640](https://github.com/kubermatic/kubermatic/issues/7640))
- Add admin settings for Alertmanager domain and the link to Alertmanager UI ([#3488](https://github.com/kubermatic/dashboard/issues/3488))
- Extend admin settings by new field MLA alertmanager domain ([#7326](https://github.com/kubermatic/kubermatic/issues/7326))
- Add v2 endpoints to manage alertmanager configuration ([#6943](https://github.com/kubermatic/kubermatic/issues/6943), [#6997](https://github.com/kubermatic/kubermatic/issues/6997))
- Allow to specify sidecars in Alertmanager Helm Chart ([#7329](https://github.com/kubermatic/kubermatic/issues/7329))
- Add link to Grafana to UI, visible if user cluster monitoring or user cluster logging is enabled ([#3642](https://github.com/kubermatic/dashboard/issues/3642))
- Alert `VeleroBackupTakesTooLong` will now reset if the next backup of the same schedule finished successfully ([#7600](https://github.com/kubermatic/kubermatic/issues/7600))
- Add `Logs` as a RuleGroup list option: `GET /api/v2/projects/{project_id}/clusters/{cluster_id}/rulegroups?type=Logs` ([#7202](https://github.com/kubermatic/kubermatic/issues/7202))
- Add kube-state-metrics addon ([#7513](https://github.com/kubermatic/kubermatic/issues/7513))
- Add v2 endpoints for KKP users and admins to manage rule groups ([#7162](https://github.com/kubermatic/kubermatic/issues/7162))

#### Metering Tool Integration

- Metering tool integration ([#7549](https://github.com/kubermatic/kubermatic/issues/7549))
- Support the metering tool configuration in the API ([#7601](https://github.com/kubermatic/kubermatic/issues/7601))
- Add API endpoint for metering reports ([#7449](https://github.com/kubermatic/kubermatic/issues/7449))
- Add metering to KKP Operator ([#7448](https://github.com/kubermatic/kubermatic/issues/7448))

#### User Cluster Backup & Restore UI

- Add API endpoints for cluster backups ([#7395](https://github.com/kubermatic/kubermatic/issues/7395))
- Add API endpoints for Etcd Backup Restore ([#7430](https://github.com/kubermatic/kubermatic/issues/7430))
- Limit number of simultaneously running etcd backup delete jobs ([#6952](https://github.com/kubermatic/kubermatic/issues/6952))
- Add API endpoint for creating/updating S3 backup credentials per Seed ([#7641](https://github.com/kubermatic/kubermatic/issues/7641))
- Move backup and restore configuration to Seed resource to allow to have different s3-settings ([#7428](https://github.com/kubermatic/kubermatic/issues/7428))
- Enable Etcd-launcher by default ([#7821](https://github.com/kubermatic/kubermatic/issues/7821))

#### Enhancements on Open Policy Agent Integration

- Kubermatic OPA Constraints now additionally support using regular yaml `parameters` instead of `rawJSON`. `rawJSON` is still supported so no migration needed ([#7066](https://github.com/kubermatic/kubermatic/issues/7066))
- Remove Gatekeeper from default accessible addon list ([#7510](https://github.com/kubermatic/kubermatic/issues/7510))
- Update Gatekeeper version to v3.5.2 with new CRDs Assign, AssignMetadata, MutatorPodStatus and resources MutatingWebhookConfiguration, PodDisruptionBudget ([#7613](https://github.com/kubermatic/kubermatic/issues/7613))
- Add OPA Default Constraints to UI ([#3543](https://github.com/kubermatic/dashboard/issues/3543))
- Add new endpoints for default constraint creation/deletion: `POST /api/v2/constraints` ([#7256](https://github.com/kubermatic/kubermatic/issues/7256), [#7321](https://github.com/kubermatic/kubermatic/issues/7321))
- Add default constraint get and list endpoints for v2 ([#7307](https://github.com/kubermatic/kubermatic/issues/7307))
- Add label info on Constraint GET/LIST ([#7399](https://github.com/kubermatic/kubermatic/issues/7399))
- Add endpoint to patch constraint: `PATCH /api/v2/constraints/{constraint_name}` ([#7339](https://github.com/kubermatic/kubermatic/issues/7339))
- Add allowlist for Docker registries [EE only], which allows users to set which image registries are allowed so only workloads from those registries can be deployed on user clusters ([#7305](https://github.com/kubermatic/kubermatic/issues/7305), [#3562](https://github.com/kubermatic/dashboard/issues/3562))
- Add API endpoints for Whitelisted Registry [EE] ([#7346](https://github.com/kubermatic/kubermatic/issues/7346))
- Reduce default OPA webhooks timeout to 1s, exempt kube-system namespace from OPA and deploy OPA mutating webhook only when enabled ([#7683](https://github.com/kubermatic/kubermatic/issues/7683))

#### Enhanced KubeVirt Integration

- Add Flatcar support for KubeVirt ([#3561](https://github.com/kubermatic/dashboard/issues/3561))
- Change the default cluster CIDR for KubeVirt cloud provider ([#7238](https://github.com/kubermatic/kubermatic/issues/7238))
- Users are able to expose load balancer from the overkube clusters ([#7543](https://github.com/kubermatic/kubermatic/issues/7543))

### Cloud Providers

#### Amazon Web Services (AWS)

- AWS Spot instances support ([#7073](https://github.com/kubermatic/kubermatic/issues/7073), [#3607](https://github.com/kubermatic/dashboard/issues/3607))
- Support spot instance market options ([#7295](https://github.com/kubermatic/kubermatic/issues/7295))
- Add option to filter AWS instance types by architecture ([#3600](https://github.com/kubermatic/dashboard/issues/3600))
- ARM instances for AWS are now being filtered out from the instance size list as KKP does not support ARM ([#6940](https://github.com/kubermatic/kubermatic/issues/6940))
- Add support for external CCM migration ([#3554](https://github.com/kubermatic/dashboard/issues/3554)) if supported

#### Microsoft Azure

- Add option to select Azure LoadBalancer SKU in cluster creation ([#3455](https://github.com/kubermatic/dashboard/issues/3455))
- Support standard load balancers for Azure ([#7271](https://github.com/kubermatic/kubermatic/issues/7271))
- Add option to set the Load Balancer SKU when creating Azure clusters ([#7208](https://github.com/kubermatic/kubermatic/issues/7208))
- Add vNet resource group field for Azure ([#3275](https://github.com/kubermatic/dashboard/issues/3275))
- Add vNet resource group for Azure ([#6908](https://github.com/kubermatic/kubermatic/issues/6908))
- vNet resource group instead of resource group will be used if it is specified ([#3399](https://github.com/kubermatic/dashboard/issues/3399))
- Add checkbox for assigning Azure availability sets ([#3612](https://github.com/kubermatic/dashboard/issues/3612))
- Create and assign Azure availability sets on demand ([#7445](https://github.com/kubermatic/kubermatic/issues/7445))
- Add support for external CCM migration ([#3554](https://github.com/kubermatic/dashboard/issues/3554)) if supported

#### OpenStack

- Fix OpenStack crashing with Kubernetes 1.20 and 1.21 ([#6923](https://github.com/kubermatic/kubermatic/issues/6923))
- Fix using a custom CA Bundle for OpenStack by authenticating after setting the proper CA bundle ([#7192](https://github.com/kubermatic/kubermatic/issues/7192))
- Open NodePort range in OpenStack ([#7081](https://github.com/kubermatic/kubermatic/issues/7081), [#7121](https://github.com/kubermatic/kubermatic/issues/7121))
- Add support for Application Credentials to the Openstack provider ([#3480](https://github.com/kubermatic/dashboard/issues/3480))
- Redesign OpenStack provider settings step to better support different type of credentials ([#3528](https://github.com/kubermatic/dashboard/issues/3528))
- Add application credentials and OIDC token for OpenStack ([#7221](https://github.com/kubermatic/kubermatic/issues/7221))
- Use OpenStack CCM v1.21.0 for Kubernetes v1.21 clusters, and CCM v1.22.0 for Kubernetes v1.22 clusters ([#7576](https://github.com/kubermatic/kubermatic/issues/7576))
- Add `ClusterFeatureCCMClusterName` feature for OpenStack clusters. This feature adds the `--cluster-name` flag to the OpenStack external CCM deployment. The feature gate is enabled by default for newly created clusters. Enabling this feature gate for existing clusters can cause the external CCM to lose the track of the existing cloud resources, so it's up to the users to clean up any leftover resources ([#7330](https://github.com/kubermatic/kubermatic/issues/7330))
- Add support for external CCM migration ([#3554](https://github.com/kubermatic/dashboard/issues/3554)) if supported

#### Hetzner

- When creating Hetzner Clusters, specifying the network is now mandatory ([#6878](https://github.com/kubermatic/kubermatic/issues/6878), [#6878](https://github.com/kubermatic/kubermatic/issues/6878)
- Add support for external CCM migration ([#3554](https://github.com/kubermatic/dashboard/issues/3554)) if supported

#### VMware vSphere

- Add option to specify vSphere resource pool ([#3471](https://github.com/kubermatic/dashboard/issues/3471))
- Support vSphere resource pool ([#7281](https://github.com/kubermatic/kubermatic/issues/7281))
- `DefaultStoragePolicy` field has been added to the vSphere datacenter spec and storagePolicy to the vSphere CloudSpec at cluster level ([#7423](https://github.com/kubermatic/kubermatic/issues/7423))
- Fix vSphere client not using the provided custom CA bundle ([#6973](https://github.com/kubermatic/kubermatic/issues/6973))
- Add support for external CCM migration ([#3554](https://github.com/kubermatic/dashboard/issues/3554)) if supported

#### Google Cloud Platform (GCP)

- Disable CentOS for GCP ([#3331](https://github.com/kubermatic/dashboard/issues/3331))
- Add support for external CCM migration ([#3554](https://github.com/kubermatic/dashboard/issues/3554)) if supported

### Misc

- Add container runtime selector for clusters ([#3448](https://github.com/kubermatic/dashboard/issues/3448))
- Add container runtime to the cluster spec ([#7225](https://github.com/kubermatic/kubermatic/issues/7225))
- Add network configuration for user clusters ([#3460](https://github.com/kubermatic/dashboard/issues/3460))
- Add network configuration to user cluster API ([#6970](https://github.com/kubermatic/kubermatic/issues/6970))
- Add telemetry chart, use --disable-telemetry in installer to disable it (#7579)
- Add option to restart machine deployments ([#3491](https://github.com/kubermatic/dashboard/issues/3491))
- Add optional TLS support for minio chart. The user can define a TLS secret that minio will use for its server. The TLS certificates should be signed by the global Kubermatic CA documented in https://github.com/kubermatic/docs/pull/524/files ([#7665](https://github.com/kubermatic/kubermatic/issues/7665))
- Add endpoint to restart machine deployments ([#7340](https://github.com/kubermatic/kubermatic/issues/7340))
- Add support for creating MachineDeployment with annotations ([#7447](https://github.com/kubermatic/kubermatic/issues/7447))
- Add Kubernetes 1.22, remove Kubernetes 1.17 and 1.18 ([#7461](https://github.com/kubermatic/kubermatic/issues/7461))
- Add NodeLocal DNS Cache configuration to Cluster API ([#7091](https://github.com/kubermatic/kubermatic/issues/7091))
- Changes to the tolerations on the node-local-dns DaemonSet will now be kept instead of being overwritten ([#7466](https://github.com/kubermatic/kubermatic/issues/7466))
- Re-enable NodeLocal DNS Cache in user clusters ([#7075](https://github.com/kubermatic/kubermatic/issues/7075))
- The Spec for the user-cluster etcd Statefulset was updated; this will cause the etcd pods for user-clusters to be restarted on KKP upgrade ([#6975](https://github.com/kubermatic/kubermatic/issues/6975))
- Users can now enable/disable konnectivity on their clusters ([#7679](https://github.com/kubermatic/kubermatic/issues/7679))
- Add vSphere resource pool to the preset ([#7796](https://github.com/kubermatic/kubermatic/issues/7796))

### Bugfixes

- Do not delete custom links after pressing enter key ([#3266](https://github.com/kubermatic/dashboard/issues/3266))
- Fix a bug that always applies default values to container resources ([#7302](https://github.com/kubermatic/kubermatic/issues/7302))
- Fix cluster list endpoint that was returning errors when user didn't have access to at least one cluster datacenter ([#7440](https://github.com/kubermatic/kubermatic/issues/7440))
- Fix finalizers duplication ([#7135](https://github.com/kubermatic/kubermatic/issues/7135))
- Fix issue where Kubermatic non-admin users were not allowed to manage Kubermatic Constraints ([#6942](https://github.com/kubermatic/kubermatic/issues/6942))
- Fix issue where cluster validation was failing on certificate error because the validation provider was not using the provided custom CA Bundle ([#6907](https://github.com/kubermatic/kubermatic/issues/6907))
- Fix missed cluster-autoscaler resource from the ClusterRole ([#6950](https://github.com/kubermatic/kubermatic/issues/6950))
- Fix user-ssh-keys-agent migration ([#7193](https://github.com/kubermatic/kubermatic/issues/7193))
- Fix for Seed API `PATCH` endpoint which sometimes removed Seed fields unrelated to the PATCH ([#7674](https://github.com/kubermatic/kubermatic/issues/7674))
- Fix the issue where Seed API was using seed clients to update the Seeds on master cluster instead of using the master client. This was causing Seed API not to work on Seeds which were not also the master clusters ([#7744](https://github.com/kubermatic/kubermatic/issues/7744))

### Updates

- Update Alertmanager to 0.22.2 ([#7438](https://github.com/kubermatic/kubermatic/issues/7438))
- Update Go dependencies to controller-runtime 0.9.5 and Kubernetes 1.21.3 ([#7462](https://github.com/kubermatic/kubermatic/issues/7462))
- Update Gatekeeper to v3.5.2 ([#7613](https://github.com/kubermatic/kubermatic/issues/7613))
- Update Grafana to 8.1.2 ([#7561](https://github.com/kubermatic/kubermatic/issues/7561))
- Update Minio to RELEASE.2021-08-20T18-32-01Z ([#7562](https://github.com/kubermatic/kubermatic/issues/7562))
- Update Prometheus to 2.29.1 ([#7437](https://github.com/kubermatic/kubermatic/issues/7437))
- Update Velero to 1.6.3 ([#7496](https://github.com/kubermatic/kubermatic/issues/7496))
- Update cert-manager to 1.5.2 ([#7563](https://github.com/kubermatic/kubermatic/issues/7563))
- Update go-swagger to v0.27.0 ([#7465](https://github.com/kubermatic/kubermatic/issues/7465))
- Update karma to 0.89 ([#7439](https://github.com/kubermatic/kubermatic/issues/7439))
- Update kube-state-metrics to 2.2.0 ([#7571](https://github.com/kubermatic/kubermatic/issues/7571))
- Update machine-controller to 1.35.1 ([#7492](https://github.com/kubermatic/kubermatic/issues/7492))
- Update nginx-ingress-controller to 0.49.0 ([#7560](https://github.com/kubermatic/kubermatic/issues/7560))
- Update node-exporter to v1.2.2 ([#7523](https://github.com/kubermatic/kubermatic/issues/7523))
