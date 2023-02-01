# Kubermatic 2.20

- [v2.20.0](#v2200)
- [v2.20.1](#v2201)
- [v2.20.2](#v2202)
- [v2.20.3](#v2203)
- [v2.20.4](#v2204)
- [v2.20.5](#v2205)
- [v2.20.6](#v2206)
- [v2.20.7](#v2207)
- [v2.20.8](#v2208)
- [v2.20.9](#v2209)
- [v2.20.10](#v22010)
- [v2.20.11](#v22011)
- [v2.20.12](#v22012)
- [v2.20.13](#v22013)

## [v2.20.13](https://github.com/kubermatic/kubermatic/releases/tag/v2.20.13)

### Bugfixes

- Update machine-controller to v1.45.6 and operating-system-manager (OSM) to v0.4.6. This fixes the issue with the new nodes not joining the cluster because of non-existing containerd and Docker packages. containerd is updated to v1.6 (from 1.4) and Docker is updated to 20.10 (from 19.03) ([#11796](https://github.com/kubermatic/kubermatic/pull/11796))

### Updates

- Update Anexia CCM (cloud-controller-manager) to version 1.5.1 ([#11750](https://github.com/kubermatic/kubermatic/pull/11750))

## [v2.20.12](https://github.com/kubermatic/kubermatic/releases/tag/2.20.12)

### Bugfixes

- Fix an issue where creating Clusters through ClusterTemplates failed without leaving a trace (the ClusterTemplateInstance got deleted as if all was good) ([#11601](https://github.com/kubermatic/kubermatic/pull/11601))
- Fix yet another API error in extended disk configuration for provider Anexia ([#11603](https://github.com/kubermatic/kubermatic/pull/11603))
- Use seed proxy configuration for seed-controller-manager ([#11631](https://github.com/kubermatic/kubermatic/pull/11631))
- Add support for kube-dns configmap for NodeLocal DNSCache to allow customization of dns. Fixes an issue with a wrong mounted Corefile in NodeLocal DNSCache ([#11664](https://github.com/kubermatic/kubermatic/pull/11664))

### Misc 

- Stop overriding upstream chart tolerations for logging/promtail by default, adding `node-role.kubernetes.io/control-plane` toleration ([#11592](https://github.com/kubermatic/kubermatic/pull/11592))

## [v2.20.11](https://github.com/kubermatic/kubermatic/releases/tag/v2.20.11)

### Action Required

- ACTION REQUIRED: Use `registry.k8s.io` instead of `k8s.gcr.io` for Kubernetes upstream images. It might be necessary to update firewall rules or mirror registries accordingly ([#11391](https://github.com/kubermatic/kubermatic/pull/11391))

### Bugfixes

- Disable promtail initContainer that was overriding system `fs.inotify.max_user_instances` configuration ([#11382](https://github.com/kubermatic/kubermatic/pull/11382))
- Fix duplicate SourceRange entries for front-loadbalancer Service ([#11369](https://github.com/kubermatic/kubermatic/pull/11369))
- Fix missing etcd metrics in Grafana etcd dashboards and master/seed Prometheus by renaming to: `etcd_mvcc_db_total_size_in_bytes`, `etcd_mvcc_delete_total`, `etcd_mvcc_put_total`, `etcd_mvcc_range_total`, `etcd_mvcc_txn_total` ([#11439](https://github.com/kubermatic/kubermatic/pull/11439))
- Prioritise public IP over private IP in front LoadBalancer service ([#11512](https://github.com/kubermatic/kubermatic/pull/11512))

### Updates

- Update to etcd 3.5.6 for Kubernetes 1.22+ to prevent potential data inconsistency issues during online defragmentation ([#11405](https://github.com/kubermatic/kubermatic/pull/11405))
- Update nginx-ingress to 1.5.1; this raises the minimum supported Kubernetes version for master/seed clusters to 1.21 ([#11417](https://github.com/kubermatic/kubermatic/pull/11417))
- Update Dex to 2.35.3 ([#11420](https://github.com/kubermatic/kubermatic/pull/11420))
- Update OpenStack Cinder CSI to v1.23.4 and v1.22.2 ([#11456](https://github.com/kubermatic/kubermatic/pull/11456))
- Update machine-controller to v1.45.5 ([#11546](https://github.com/kubermatic/kubermatic/pull/11546))
- Add support for Kubernetes v1.22.17 and v1.23.15 ([#11555](https://github.com/kubermatic/kubermatic/pull/11555))
- Add etcd database size alerts `EtcdDatabaseQuotaLowSpace`, `EtcdExcessiveDatabaseGrowth`, `EtcdDatabaseHighFragmentationRatio`([#11559](https://github.com/kubermatic/kubermatic/pull/11559))

## [v2.20.10](https://github.com/kubermatic/kubermatic/releases/tag/v2.20.10)

This release includes updated Kubernetes versions that fix CVE-2022-3162 and CVE-2022-3294. For more information, see below. We strongly recommend upgrading to those Kubernetes patch releases as soon as possible.

### Bugfixes

- Adding finalizer `kubermatic.k8c.io/cleanup-usersshkeys-cluster-ids` to `Cluster` resources can no longer remove other finalizers ([#11323](https://github.com/kubermatic/kubermatic/pull/11323))
- Remove digests from Docker images in addon manifests to fix issues with Docker registry mirrors / local registries. KKP 2.22  will restore the digests and properly support them ([#11240](https://github.com/kubermatic/kubermatic/pull/11240))

### Updates

- Add support for Kubernetes 1.23.14 and 1.22.16 and automatically upgrade existing 1.23.x and 1.22.x clusters ([#11342](https://github.com/kubermatic/kubermatic/pull/11342))
    * Those Kubernetes patch releases fix CVE-2022-3162 and CVE-2022-3294, both in kube-apiserver: [CVE-2022-3162: Unauthorized read of Custom Resources](https://groups.google.com/g/kubernetes-announce/c/oR2PUBiODNA/m/tShPgvpUDQAJ) and [CVE-2022-3294: Node address isn't always verified when proxying](https://groups.google.com/g/kubernetes-announce/c/eR0ghAXy2H8/m/sCuQQZlVDQAJ).

### Upcoming Changes

- For the next series of KKP patch releases, image references will move from `k8s.gcr.io` to `registry.k8s.io`. This will be done to keep up with [latest upstream changes](https://github.com/kubernetes/enhancements/tree/master/keps/sig-release/3000-artifact-distribution). Please ensure that any mirrors you use are going to host `registry.k8s.io` and/or that firewall rules are going to allow access to `registry.k8s.io` to pull images before applying the next KKP patch releases. **This is not included in this patch release but just a notification of future changes.**


## [v2.20.9](https://github.com/kubermatic/kubermatic/releases/tag/v2.20.9)

### Bugfixes

- Fixes Openstack `api/v1/providers/openstack/tenants` API endpoint for some cases where "couldn't get projects: couldn't get tenants for region XX: couldn't get identity endpoint: No suitable endpoint could be found in the service catalog." was wrongly returned ([#10968](https://github.com/kubermatic/kubermatic/issues/10968))
- Fix Seed-Proxy ServiceAccount token not being generated ([#11190](https://github.com/kubermatic/kubermatic/issues/11190))

### Misc

- Set PriorityClassName of konnectivity-agent and openvpn-client to system-cluster-critical ([#11140](https://github.com/kubermatic/kubermatic/issues/11140))

### Updates

- Update konnectivity to v0.0.33 ([#11080](https://github.com/kubermatic/kubermatic/issues/11080))

## [v2.20.8](https://github.com/kubermatic/kubermatic/releases/tag/v2.20.8)

### Bugfixes

- etcd-launcher is now capable of automatically rejoining the etcd ring when a member is removed during the peer TLS migration ([#9322](https://github.com/kubermatic/kubermatic/pull/9322))
- Fix usercluster-controller crashing when `.status.userEmail` on `Cluster` objects is not set ([#11047](https://github.com/kubermatic/kubermatic/pull/11047))
- Fix API error in extended disk configuration for provider Anexia ([#11051](https://github.com/kubermatic/kubermatic/pull/11051))


## [v2.20.7](https://github.com/kubermatic/kubermatic/releases/tag/v2.20.7)

### API Changes

- Extend disk configuration for provider Anexia ([#10915](https://github.com/kubermatic/kubermatic/pull/10915))

### Bugfixes

- Add additional header to prevent KKP dashboard from being shown in an iframe ([#4796](https://github.com/kubermatic/dashboard/pull/4796))
- Allow proxy mode change by CNI migration ([#10717](https://github.com/kubermatic/kubermatic/pull/10717))
- Fix an issue with vSphere CSI driver using improved-csi-idempotency that's currently not supported by KKP ([#10792](https://github.com/kubermatic/kubermatic/pull/10792))
- Fix kubermatic-webhook failing to start on external seed clusters ([#10959](https://github.com/kubermatic/kubermatic/pull/10959))
- Update KubeVirt logo to mark technology preview ([#4810](https://github.com/kubermatic/dashboard/pull/4810))

### Chore

- Add support for Kubernetes 1.22.15, 1.23.12; existing clusters using these Kubernetes releases will be automatically updated as any previous version is affected by CVEs ([#11028](https://github.com/kubermatic/kubermatic/pull/11028))

### Updates

- Update OpenStack version for k8s 1.23 to fix services ports mapping issue ([#11037](https://github.com/kubermatic/kubermatic/pull/11037))
- Update machine-controller to v1.45.4 ([#10915](https://github.com/kubermatic/kubermatic/pull/10915))


## [v2.20.6](https://github.com/kubermatic/kubermatic/releases/tag/v2.20.6)

### Bugfixes

- Fix etcdbackup controller constantly updating the EtcdBackupConfig status ([#10650](https://github.com/kubermatic/kubermatic/issues/10650))
- Fix finalizers on clusters sometimes getting overwritten by the cloud controller or cluster-credentials controller ([#10536](https://github.com/kubermatic/kubermatic/issues/10536))
- Fix handling custom annotations for the front-loadbalancer Service ([#10436](https://github.com/kubermatic/kubermatic/issues/10436))
- Fix reconcile loop in AllowedRegistry controller ([#10644](https://github.com/kubermatic/kubermatic/issues/10644))

### Updates

- Update Canal 3.21 version to v3.21.6 ([#10491](https://github.com/kubermatic/kubermatic/issues/10491))
- Update Canal 3.22 version to v3.22.4 ([#10499](https://github.com/kubermatic/kubermatic/issues/10499))
- Update ec2-instances-info to a newer version to include newer AWS EC2 instances ([#10653](https://github.com/kubermatic/kubermatic/issues/10653))
- Update machine-controller to v1.45.3 ([#10628](https://github.com/kubermatic/kubermatic/issues/10628))


## [v2.20.5](https://github.com/kubermatic/kubermatic/releases/tag/v2.20.5)

### Updates

- Update Canal 3.21 to v3.21.5 ([#10271](https://github.com/kubermatic/kubermatic/issues/10271))
- Update Canal 3.22 to v3.22.3 ([#10272](https://github.com/kubermatic/kubermatic/issues/10272))
- Update Konnectivity to v0.0.31 ([#10112](https://github.com/kubermatic/kubermatic/issues/10112))
- Update OSM to v0.4.5 ([#10273](https://github.com/kubermatic/kubermatic/issues/10273))
- Update machine-controller to v1.45.2 ([#10399](https://github.com/kubermatic/kubermatic/issues/10399))

### Misc

- Add `--skip-dependencies` flag to kubermatic-installer that skips downloading Helm chart dependencies (requires chart dependencies to be downloaded already) ([#10348](https://github.com/kubermatic/kubermatic/issues/10348))
- Fix automatic Canal version upgrade for clusters with k8s 1.23+ ([#10308](https://github.com/kubermatic/kubermatic/issues/10308))
- Making telemetry UUID field optional ([#9900](https://github.com/kubermatic/kubermatic/issues/9900))
- OSM deployment image repo and tag override ([#10123](https://github.com/kubermatic/kubermatic/issues/10123))
- Use quay.io as the default registry for Canal CNI images, bump Canal v3.20 version to v3.20.5 ([#10305](https://github.com/kubermatic/kubermatic/issues/10305))
- etcd backup files are named differently (`foo-YYYY-MM-DDThh:mm:ss` to `foo-YYYY-MM-DDThhmmss.db`) to improve compatibility with different storage solutions ([#10143](https://github.com/kubermatic/kubermatic/issues/10143))


## [v2.20.4](https://github.com/kubermatic/kubermatic/releases/tag/v2.20.4)

### Bugfixes

- Fix addon variables not being persisted ([#10010](https://github.com/kubermatic/kubermatic/issues/10010)). During the KKP 2.20.0 upgrade, addon variables were removed by accident (i.e. `spec.variables` is set to `null` for all `Addon` resources) and need to be restored from the pre-migration backup.
- Fix deprecated nodePortProxy annotations (in `spec.nodePortProxy.annotations` in a Seed object) being ignored ([#10008](https://github.com/kubermatic/kubermatic/issues/10008))
- Fix probes, resources and allow overriding resource requests/limits for Konnectivity proxy via components override in the cluster resource ([#9911](https://github.com/kubermatic/kubermatic/issues/9911))

### Misc

- Add External Snapshotter for Openstack and vSphere CSI ([#10066](https://github.com/kubermatic/kubermatic/issues/10066))
- Containerd container runtime mirror registries support for OSM ([#10134](https://github.com/kubermatic/kubermatic/issues/10134))
- Update ingress-nginx to 1.2.1 ([#10036](https://github.com/kubermatic/kubermatic/issues/10036))


## [v2.20.3](https://github.com/kubermatic/kubermatic/releases/tag/v2.20.3)

### Misc

- Add support for Kubernetes 1.23 ([#9836](https://github.com/kubermatic/kubermatic/issues/9836))
- Add support for Kubernetes 1.21.12 and 1.22.9 (default Kubernetes version is now 1.21.12) ([#9884](https://github.com/kubermatic/kubermatic/issues/9884))
- Fix Mutating webhook for None CNI ([#9737](https://github.com/kubermatic/kubermatic/issues/9737))
- Fix an issue where helm invocations by the kubermatic-installer ignored most environment variables ([#9876](https://github.com/kubermatic/kubermatic/issues/9876))
- Fix telemetry CronJob not producing data ([#9740](https://github.com/kubermatic/kubermatic/issues/9740))
- Fix kubermatic-installer: improve error handling when building helm chart dependencies ([#9851](https://github.com/kubermatic/kubermatic/issues/9851))
- Update cluster-autoscaler (1.20 to 1.20.2, 1.21 to 1.21.2, 1.22 to 1.22.2) ([#9836](https://github.com/kubermatic/kubermatic/issues/9836))
- `image-loader` loads more images that were missing before (OpenStack CSI, user-ssh-keys-agent, operatingsystem-manager) ([#9871](https://github.com/kubermatic/kubermatic/issues/9871))


## [v2.20.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.20.2)

With this patch release, etcd for Kubernetes 1.22+ is upgraded to etcd 3.5.3. Data consistency issues as reported in previous release notes are fixed. Warnings and recommendations related to that can be considered withdrawn for this release.

- Add support for configuration annotations and loadBalancerSourceRanges for front-loadbalancer service of node port proxy ([#9486](https://github.com/kubermatic/kubermatic/issues/9486))
- For the Seed CR, `spec.nodeportProxy.annotations` is deprecated and `spec.nodeportProxy.envoy.loadBalancerService.annotations` should be used instead ([#9486](https://github.com/kubermatic/kubermatic/issues/9486))
- The Image-loader utility now includes all the images used by KKP, considering also the provider-specific ones (e.g., CCM, CSI) ([#9518](https://github.com/kubermatic/kubermatic/issues/9518))
- Enable the "vsphereCSIClusterID" feature flag when running the CCM/CSI migration ([#9557](https://github.com/kubermatic/kubermatic/issues/9557))
- For Kubernetes 1.22 and higher, etcd is updated to v3.5.3 to fix data consistency issues as reported by upstream developers ([#9606](https://github.com/kubermatic/kubermatic/issues/9606))
- The flag `--kubelet-certificate-authority` (introduced in KKP 2.19) is not set for "kubeadm" / "bringyourown" user clusters anymore ([#9674](https://github.com/kubermatic/kubermatic/issues/9674))


## [v2.20.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.20.1)

This patch release enables etcd corruption checks on every etcd ring that is running etcd 3.5 (which applies to all user clusters with Kubernetes 1.22). This change is a [recommendation from the etcd maintainers](https://groups.google.com/a/kubernetes.io/g/dev/c/B7gJs88XtQc/m/rSgNOzV2BwAJ) due to issues in etcd 3.5 that can cause data consistency issues. The changes in this patch release will prevent corrupted etcd members from joining or staying in the etcd ring.

To replace a member in case of data consistency issues, please:

- Follow our documentation for [replacing an etcd member](https://docs.kubermatic.com/kubermatic/v2.20/cheat_sheets/etcd/replace_a_member/) if you are **not running etcd-launcher**.
- Delete the `PersistentVolume` that backs the corrupted etcd member to trigger the [automated recovery procedure](https://docs.kubermatic.com/kubermatic/v2.20/cheat_sheets/etcd/etcd-launcher/#automated-persistent-volume-recovery) if you **are using etcd-launcher**.

Please be aware we do not recommend enabling `etcd-launcher` on existing Kubernetes 1.22 environments at the time. This is due to the fact that the migration to `etcd-launcher` requires several etcd restarts and we currently recommend keeping the etcd ring as stable as possible (apart from the restarts triggered by this patch release to roll out the consistency checks).

### Misc

- For user clusters that use etcd 3.5 (Kubernetes 1.22 clusters), etcd corruption checks are turned on to detect [etcd data consistency issues](https://github.com/etcd-io/etcd/issues/13766). Checks run at etcd startup and every 4 hours ([#9480](https://github.com/kubermatic/kubermatic/issues/9480))


## [v2.20.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.20.0)

Before upgrading, make sure to read the [general upgrade guidelines](https://docs.kubermatic.com/kubermatic/v2.20/tutorials_howtos/upgrading/). Consider tweaking `seedControllerManager.maximumParallelReconciles` to ensure usercluster reconciliations will not cause resource exhaustion on seed clusters.

### Highlights

- Migrate API group `kubermatic.k8s.io` to `kubermatic.k8c.io` ([#8783](https://github.com/kubermatic/kubermatic/issues/8783)). This change requires a migration of all KKP master/seed clusters. Please consult the [upgrade documentation](https://docs.kubermatic.com/kubermatic/v2.20/tutorials_howtos/upgrading/upgrade_from_2.19_to_2.20/) for more information.
- Full Nutanix support([#8428](https://github.com/kubermatic/kubermatic/issues/8428))

### Supported Kubernetes Versions

- 1.20.13
- 1.20.14
- 1.21.8
- 1.22.5

### Breaking Changes

- Add Canal CNI v3.22 support & make it the default CNI. NOTE: Automatically upgrades Canal to v3.22 in clusters with k8s v1.23 and higher and older Canal version ([#9258](https://github.com/kubermatic/kubermatic/issues/9258))
- Restore correct labels on nodeport-proxy-envoy Deployment. Deleting the existing Deployment for each cluster with the `LoadBalancer` expose strategy if upgrading from affected version is necessary ([#9060](https://github.com/kubermatic/kubermatic/issues/9060))
- Secret name for S3 credentials updated to `kubermatic-s3-credentials`. If the secret `s3-credentials` was manually created instead of using the `minio` helm chart, new secret `kubermatic-s3-credentials` must be created ([#9242](https://github.com/kubermatic/kubermatic/issues/9242))
- The etcd-backup-related containers are now loaded dynamically from the KubermaticConfiguration, the relevant CLI flags like `-backup-container=<file>` have been removed. The deprecated configuration options `KubermaticConfiguration.spec.seedController.backupRestore` and `Seed.spec.backupRestore` have been removed. Please migrate to `Seed.spec.etcdBackupRestore` ([#9003](https://github.com/kubermatic/kubermatic/issues/9003))
- etcd backup API now requires destination to be set for EtcdBackupConfig, EtcdRestore and BackupCredentials endpoints ([#9139](https://github.com/kubermatic/kubermatic/issues/9139))

### Bugfixes

- Automatic upgrades from Kubernetes 1.19.* to 1.20.13 work as intended now ([#8821](https://github.com/kubermatic/kubermatic/issues/8821))
- Correctly handle the 'default' Nutanix project in API calls ([#9336](https://github.com/kubermatic/kubermatic/issues/9336))
- Fix AWS cloud provider cleanup sometimes getting stuck when cleaning up tags ([#8879](https://github.com/kubermatic/kubermatic/issues/8879))
- Fix Konnectivity authentication issue in some scenarios by fixing cluster-external-addr-allow apiserver network policy ([#9224](https://github.com/kubermatic/kubermatic/issues/9224))
- Fix Preset API Body for preset creation and update API calls ([#9298](https://github.com/kubermatic/kubermatic/issues/9298))
- Fix `OpenVPNServerDown` alerting rule to work as expected and not fire if Konnectivity is enabled ([#9216](https://github.com/kubermatic/kubermatic/issues/9216))
- Fix apiserver network policy: allow all egress DNS traffic from the apiserver ([#8788](https://github.com/kubermatic/kubermatic/issues/8788))
- Fix applying resource requirements when using incomplete overrides (e.g. specifying only limits, but no request for a container) ([#9045](https://github.com/kubermatic/kubermatic/issues/9045))
- Fix bad owner references for ClusterRoleBindings ([#8858](https://github.com/kubermatic/kubermatic/issues/8858))
- Fix installation issue with charts/logging/loki (error evaluating symlink) ([#8823](https://github.com/kubermatic/kubermatic/issues/8823))
- Fix missing snapshot CRD's for cinder CSI ([#9042](https://github.com/kubermatic/kubermatic/issues/9042))
- ICMP rules migration only runs on Azure NSGs created by KKP ([#8843](https://github.com/kubermatic/kubermatic/issues/8843))
- If a network is set in the Hetzner cluster spec, it is now correctly applied to generated machines ([#8872](https://github.com/kubermatic/kubermatic/issues/8872))

### Dashboard

- Add allowed IP range override support to the GCP, Azure, AWS, and OpenStack providers ([#4318](https://github.com/kubermatic/dashboard/issues/4318))
- Add option to get kubeconfig for external clusters ([#4164](https://github.com/kubermatic/dashboard/issues/4164))
- Add support for Nutanix provider ([#4145](https://github.com/kubermatic/dashboard/issues/4145))
- Admins can define default Rule Groups in Admin Settings ([#3971](https://github.com/kubermatic/dashboard/issues/3971))
- Redesign cluster summary step. Update error notifications and event colors styling ([#4141](https://github.com/kubermatic/dashboard/issues/4141))
- Support for application credentials in OpenStack preset ([#4192](https://github.com/kubermatic/dashboard/issues/4192))
- Update OS default disk size to 64GB for the Azure provider when RHEL OS is selected ([#4318](https://github.com/kubermatic/dashboard/issues/4318))

### Misc

- Add Nutanix CSI support ([#9104](https://github.com/kubermatic/kubermatic/issues/9104), [#4251](https://github.com/kubermatic/dashboard/issues/4251))
- Add `vsphereCSIClusterID` feature flag for the cluster object. This feature flag changes the cluster-id in the vSphere CSI config to the cluster name instead of the vSphere Compute Cluster name provided via Datacenter config. Migrating the cluster-id requires manual steps (docs link to be added) ([#9265](https://github.com/kubermatic/kubermatic/issues/9265))
- Add credential validation for Hetzner and Equinixmetal ([#9051](https://github.com/kubermatic/kubermatic/issues/9051))
- Add endpoints to v2 KKP API to query Nutanix clusters, projects and subnets ([#8736](https://github.com/kubermatic/kubermatic/issues/8736))
- Do not reference Nutanix cluster in KKP API endpoint path for subnets ([#8906](https://github.com/kubermatic/kubermatic/issues/8906))
- Ensure existing cluster have `.Spec.CNIPlugin` initialized ([#8829](https://github.com/kubermatic/kubermatic/issues/8829))
- Metric `kubermatic_api_init_node_deployment_failures` was renamed to `kubermatic_api_failed_init_node_deployment_total`. Metric `kubermatic_cloud_controller_provider_reconciliations_successful` was renamed to `kubermatic_cloud_controller_provider_successful_reconciliations_total` ([#8763](https://github.com/kubermatic/kubermatic/issues/8763))
- Remove deprecated fields from CRD types ([#8961](https://github.com/kubermatic/kubermatic/issues/8961))
- Support custom Pod resources for NodePort-Proxy pod for the user cluster ([#9015](https://github.com/kubermatic/kubermatic/issues/9015))
- Support for `network:ha_router_replicated_interface` ports when discovering existing subnet router in Openstack ([#9164](https://github.com/kubermatic/kubermatic/issues/9164))
- Unused flatcar update resources are removed when no Flatcar `Nodes` are in a user cluster ([#8745](https://github.com/kubermatic/kubermatic/issues/8745))
- Update example values and KubermaticConfiguration to respect OIDC settings ([#8851](https://github.com/kubermatic/kubermatic/issues/8851))
- Update machine-controller to v1.45.0 ([#9293](https://github.com/kubermatic/kubermatic/issues/9293))
- Update OSM to 0.4.1 ([#9329](https://github.com/kubermatic/kubermatic/issues/9329))


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
- [v2.19.10](#v21910)
- [v2.19.11](#v21911)
- [v2.19.12](#v21912)
- [v2.19.13](#v21913)
- [v2.19.14](#v21914)
- [v2.19.15](#v21915)

## [v2.19.15](https://github.com/kubermatic/kubermatic/releases/tag/v2.19.15)

### Bugfixes

- Update machine-controller to v1.42.9 and operating-system-manager (OSM) to v0.3.10. This fixes the issue with the new nodes not joining the cluster because of non-existing containerd and Docker packages. containerd is updated to v1.6 (from 1.4) and Docker is updated to 20.10 (from 19.03) ([#11797](https://github.com/kubermatic/kubermatic/pull/11797))

## [v2.19.14](https://github.com/kubermatic/kubermatic/releases/tag/2.19.14)

### Bugfixes

- Use seed proxy configuration for seed-controller-manager ([#11669](https://github.com/kubermatic/kubermatic/pull/11669))
- Add support for kube-dns configmap for NodeLocal DNSCache to allow customization of dns. Fixes an issue with a wrong mounted Corefile in NodeLocal DNSCache ([#11664](https://github.com/kubermatic/kubermatic/pull/11664))

### Misc

- Stop overriding upstream chart tolerations for logging/promtail by default, adding `node-role.kubernetes.io/control-plane` toleration ([#11592](https://github.com/kubermatic/kubermatic/pull/11592))

## [v2.19.13](https://github.com/kubermatic/kubermatic/releases/tag/v2.19.13)

### Action Required

- ACTION REQUIRED: Use `registry.k8s.io` instead of `k8s.gcr.io` for Kubernetes upstream images. It might be necessary to update firewall rules or mirror registries accordingly ([#11510](https://github.com/kubermatic/kubermatic/pull/11510))

### Bugfixes

- Fix missing etcd metrics in Grafana etcd dashboards and master/seed Prometheus by renaming to: `etcd_mvcc_db_total_size_in_bytes`, `etcd_mvcc_delete_total`, `etcd_mvcc_put_total`, `etcd_mvcc_range_total`, `etcd_mvcc_txn_total` ([#11440](https://github.com/kubermatic/kubermatic/pull/11440))
- Prioritise public IP over private IP in front LoadBalancer service ([#11512](https://github.com/kubermatic/kubermatic/pull/11512))

### Updates

- Update to etcd 3.5.6 for Kubernetes 1.22 to prevent potential data inconsistency issues during online defragmentation ([#11406](https://github.com/kubermatic/kubermatic/pull/11406))
- Update Dex to 2.35.3 ([#11421](https://github.com/kubermatic/kubermatic/pull/11421))
- Update OpenStack Cinder CSI to v1.22.2 ([#11457](https://github.com/kubermatic/kubermatic/pull/11457))
- Update machine-controller to v1.42.8 ([#11547](https://github.com/kubermatic/kubermatic/pull/11547))
- Add support for Kubernetes v1.22.17 ([#11556](https://github.com/kubermatic/kubermatic/pull/11556))
- Add etcd database size alerts `EtcdDatabaseQuotaLowSpace`, `EtcdExcessiveDatabaseGrowth`, `EtcdDatabaseHighFragmentationRatio`([#11558](https://github.com/kubermatic/kubermatic/pull/11558))

## [v2.19.12](https://github.com/kubermatic/kubermatic/releases/tag/v2.19.12)

This release includes updated Kubernetes versions that fix CVE-2022-3162 and CVE-2022-3294. For more information, see below. We strongly recommend upgrading to those Kubernetes patch releases as soon as possible.

### Bugfixes

- Fix regression that no longer created `Role` `kubermatic:secret:sa-etcd-launcher` for etcd-launcher-based restores ([#11241](https://github.com/kubermatic/kubermatic/pull/11241))

### Updates

- Add support for Kubernetes 1.22.16 and automatically upgrade existing 1.22.x clusters ([#11343](https://github.com/kubermatic/kubermatic/pull/11343))
    * Those Kubernetes patch releases fix CVE-2022-3162 and CVE-2022-3294, both in kube-apiserver: [CVE-2022-3162: Unauthorized read of Custom Resources](https://groups.google.com/g/kubernetes-announce/c/oR2PUBiODNA/m/tShPgvpUDQAJ) and [CVE-2022-3294: Node address isn't always verified when proxying](https://groups.google.com/g/kubernetes-announce/c/eR0ghAXy2H8/m/sCuQQZlVDQAJ).

### Upcoming Changes

- For the next series of KKP patch releases, image references will move from `k8s.gcr.io` to `registry.k8s.io`. This will be done to keep up with [latest upstream changes](https://github.com/kubernetes/enhancements/tree/master/keps/sig-release/3000-artifact-distribution). Please ensure that any mirrors you use are going to host `registry.k8s.io` and/or that firewall rules are going to allow access to `registry.k8s.io` to pull images before applying the next KKP patch releases. **This is not included in this patch release but just a notification of future changes.**

## [v2.19.11](https://github.com/kubermatic/kubermatic/releases/tag/v2.19.11)

### Misc

- Set PriorityClassName of konnectivity-agent and openvpn-client to system-cluster-critical ([#11140](https://github.com/kubermatic/kubermatic/issues/11140))

### Updates

- Upgrade konnectivity version to v0.0.33 ([#11080](https://github.com/kubermatic/kubermatic/issues/11080))

## [v2.19.10](https://github.com/kubermatic/kubermatic/releases/tag/v2.19.10)

### Bugfixes

- etcd-launcher is now capable of automatically rejoining the etcd ring when a member is removed during the peer TLS migration ([#9322](https://github.com/kubermatic/kubermatic/pull/9322))

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

Before upgrading, make sure to read the [general upgrade guidelines](https://docs.kubermatic.com/kubermatic/v2.19/upgrading/guidelines/). Consider tweaking `seedControllerManager.maximumParallelReconciles` to ensure usercluster reconciliations will not cause resource exhaustion on seed clusters.

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

**Dashboard:**
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

# Kubermatic 2.18

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

Before upgrading, make sure to read the [general upgrade guidelines](https://docs.kubermatic.com/kubermatic/v2.18/upgrading/guidelines/). Consider tweaking `seedControllerManager.maximumParallelReconciles` to ensure usercluster reconciliations will not cause resource exhausting on seed clusters.

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
- The experimental new backup mechanism was updated and available as an opt-in option per Seed. If enabled, it will replace the old backups and is not backwards compatible. Users that are already using the experimental backup mechanism, be aware that to prepare it for regular use, we made some changes, admins please check the [documentation](https://docs.kubermatic.com/kubermatic/master/cheat_sheets/etcd/backup-and-restore/).

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




# Kubermatic 2.17

## [v2.17.5](https://github.com/kubermatic/kubermatic/releases/tag/v2.17.5)

### Bugfixes

- Fix a bug where `$$` in the environment-variables for machine-controller was interpreted in the Kubernetes Manifest and caused machine-controller to be unable to deploy resources, when for e.g. the password contains two consecutive `$` signs ([#7984](https://github.com/kubermatic/kubermatic/issues/7984))
- Fix for Seed API PATCH endpoint which sometimes removed Seed fields unrelated to the PATCH. Fixes the issue where Seed API was using seed clients to update the Seeds on master cluster instead of using the master client. This was causing Seed API not to work on Seeds which were not also the master clusters ([#7925](https://github.com/kubermatic/kubermatic/issues/7925))
- Fix setting of nodeport-proxy resource requests/limits, relax default nodeport-proxy envoy limits ([#8169](https://github.com/kubermatic/kubermatic/issues/8169))




## [v2.17.4](https://github.com/kubermatic/kubermatic/releases/tag/v2.17.4)

### Security

Two vulnerabilities were identified in Kubernetes ([CVE-2021-25741](https://github.com/kubernetes/kubernetes/issues/104980) and [CVE-2020-8561](https://github.com/kubernetes/kubernetes/issues/104720)) of which one (CVE-2021-25741) was fixed in Kubernetes 1.19.15 / 1.20.11 / 1.21.5. CVE-2020-8561 is mitigated by Kubermatic not allowing users to reconfigure the kube-apiserver.

Because of these updates, this KKP release includes automatic update rules for all 1.19/1.20/1.21 clusters older than these patch releases. This release also removes all affected Kubernetes versions from the list of supported versions. While CVE-2020-8561 affects the controlplane, CVE-2021-25741 affects the kubelets, which means that updating the controlplane is not enough. Once the automated controlplane updates have completed, an administrator must manually patch all vulnerable `MachineDeployment`s in all affected userclusters.

To lower the resource consumption on the seed clusters during the reconciliation / node rotation, it's recommended to adjust the `spec.seedControllerManager.maximumParallelReconciles` option in the `KubermaticConfiguration` to restrict the number of parallel updates. Users of the legacy `kubermatic` Helm chart need to update `kubermatic.maxParallelReconcile` in their `values.yaml` to achieve the same effect.

The automatic update rules can, if needed, be overwritten using the `spec.versions.kubernetes.updates` field in the `KubermaticConfiguration` or updating the `updates.yaml` if using the legacy `kubermatic` Helm chart. See [#7825](https://github.com/kubermatic/kubermatic/issues/7824) for how the versions and updates are configured. It is however not recommended to deviate from the default and leave userclusters vulnerable.

### Misc

- Add support of Kubernetes 1.20 and 1.21 in cluster-autoscaler addon ([#7511](https://github.com/kubermatic/kubermatic/issues/7511))
- Remove Gatekeeper from the default accessible addon list ([#7533](https://github.com/kubermatic/kubermatic/issues/7533))
- Fix dashboard source in the Prometheus Exporter dashboard ([#7640](https://github.com/kubermatic/kubermatic/issues/7640))




## [v2.17.3](https://github.com/kubermatic/kubermatic/releases/tag/v2.17.3)

### Bugfixes

- Prometheus and Promtail resources are not mistakenly deleted from userclusters anymore ([#6881](https://github.com/kubermatic/kubermatic/issues/6881))
- Paused userclusters do not reconcile in-cluster resources via the usercluster-controller-manager anymore ([#7470](https://github.com/kubermatic/kubermatic/issues/7470))

### Misc

- Redesign Openstack provider settings step to better support different types of credentials ([#3531](https://github.com/kubermatic/dashboard/issues/3531))
- Changes to the tolerations on the node-local-dns DaemonSet will now be kept instead of being overwritten ([#7466](https://github.com/kubermatic/kubermatic/issues/7466))




## [v2.17.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.17.2)

### Bugfixes

- Kubermatic API, etcd-launcher, and dnat-controller images are defaulted to the docker.io registry only if the provided custom image has less than 3 parts ([#7287](https://github.com/kubermatic/kubermatic/issues/7287))
- Fix a bug that always applies default values to container resources ([#7302](https://github.com/kubermatic/kubermatic/issues/7302))
- Add `ClusterFeatureCCMClusterName` feature for OpenStack clusters. This feature adds the `--cluster-name` flag to the OpenStack external CCM deployment. The feature gate is enabled by default for newly created clusters. Enabling this feature gate for existing clusters will cause the external CCM to lose the track of the existing cloud resources (such as Load Balancers), so it's up to the users to manually clean up any leftover resources ([#7330](https://github.com/kubermatic/kubermatic/issues/7330))
- Explicitly set the namespace for Dex pods in the oauth chart. This fixes the problem with KKP installation failing on Kubernetes 1.21 clusters ([#7348](https://github.com/kubermatic/kubermatic/issues/7348))

### Misc

- allow service account to create projects when belongs to projectmanagers group ([#7043](https://github.com/kubermatic/kubermatic/issues/7043))
- Added option to set the Load Balancer SKU when creating Azure clusters ([#7208](https://github.com/kubermatic/kubermatic/issues/7208))
- add application credentials and OIDC token for OpenStack ([#7221](https://github.com/kubermatic/kubermatic/issues/7221))
- Add `projectmanagers` group for RBAC controller. The new group will be assigned to service accounts ([#7263](https://github.com/kubermatic/kubermatic/issues/7263))
- Allow configuring remote_write in Prometheus Helm chart ([#7288](https://github.com/kubermatic/kubermatic/issues/7288))
- Support standard load balancers for azure in KKP ([#7308](https://github.com/kubermatic/kubermatic/issues/7308))
- Upgrade machine-controller to v1.27.11 ([#7347](https://github.com/kubermatic/kubermatic/issues/7347))

### Dashboard

- Add project managers group as an option for service accounts ([#3375](https://github.com/kubermatic/dashboard/issues/3375))
- Added option to select Azure LoadBalancer SKU in cluster creation ([#3463](https://github.com/kubermatic/dashboard/issues/3463))
- Added support for Application Credentials to the Openstack provider ([#3489](https://github.com/kubermatic/dashboard/issues/3489))




## [v2.17.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.17.1)

### Security

- Upgrade machine-controller to v1.27.8 to address [runC vulnerability CVE-2021-30465](https://github.com/opencontainers/runc/security/advisories/GHSA-c3xm-pvg7-gh7r) ([#7209](https://github.com/kubermatic/kubermatic/pull/7166))

### Bugfixes

- Fixed using a custom CA Bundle for Openstack by authenticating after setting the proper CA bundle ([#7192](https://github.com/kubermatic/kubermatic/issues/7192))
- Fix user ssh key agent migration ([#7193](https://github.com/kubermatic/kubermatic/issues/7193))
- Fix issue where Kubermatic non-admin users were not allowed to manage Kubermatic Constraints ([#6942](https://github.com/kubermatic/kubermatic/issues/6942))
- Fix KKP vSphere client not using the provided custom CA bundle ([#6973](https://github.com/kubermatic/kubermatic/issues/6973))
- Use optimistic lock when adding finalizers to prevent lost updates, and avoiding resource leaks ([#7153](https://github.com/kubermatic/kubermatic/issues/6759))

### Misc

- Use the systemd cgroup driver for newly-created Kubernetes 1.19+ clusters using the kubeadm provider. Since the kubelet-configmap addon is not reconciled, this change will not affect existing clusters, only newly-created clusters ([#7065](https://github.com/kubermatic/kubermatic/issues/7065))
- Re-enable NodeLocal DNS Cache in user clusters ([#7075](https://github.com/kubermatic/kubermatic/issues/7075))
- Open NodePort range in openstack ([#7131](https://github.com/kubermatic/kubermatic/issues/7131))
- Upgrade machine controller to v1.27.8 ([#7209](https://github.com/kubermatic/kubermatic/issues/7209))




## [v2.17.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.17.0)

### Supported Kubernetes Versions

* 1.18.6
* 1.18.8
* 1.18.10
* 1.18.14
* 1.18.17
* 1.19.0
* 1.19.2
* 1.19.3
* 1.19.8
* 1.19.9
* 1.20.2
* 1.20.5
* 1.21.0

### Highlights

- Add support for Kubernetes 1.21 ([#6778](https://github.com/kubermatic/kubermatic/issues/6778))
- [ACTION REQUIRED] Overhaul CA handling, allow to configure a global CA bundle for every component. The OIDC CA file has been removed, manual updates can be necessary. ([#6538](https://github.com/kubermatic/kubermatic/issues/6538))

### Breaking Changes

- Update cert-manager to 1.2.0 ([#6739](https://github.com/kubermatic/kubermatic/issues/6739))
- Helm Chart installations are not supported any longer at KKP 2.17, hence KKP 2.16 chart-based installations have to be imperatively migrated.

### Misc

- New etcd backup and restore controllers ([#5668](https://github.com/kubermatic/kubermatic/issues/5668))
- Add `kubermatic-seed` stack target to the Installer ([#6435](https://github.com/kubermatic/kubermatic/issues/6435))
- Add an endpoint to list Vsphere datastores: GET /api/v2/providers/vsphere/datastores ([#6442](https://github.com/kubermatic/kubermatic/issues/6442))
- KKP Installer will update Helm releases automatically if the values have changed (no need for `--force` in most cases). ([#6449](https://github.com/kubermatic/kubermatic/issues/6449))
- Introduce resource quota for Alibaba provider ([#6458](https://github.com/kubermatic/kubermatic/issues/6458))
- Add Gatekeeper health to cluster health status ([#6461](https://github.com/kubermatic/kubermatic/issues/6461))
- Remove CoreOS Support ([#6465](https://github.com/kubermatic/kubermatic/issues/6465))
- Relax memory limit for openvpn container ([#6467](https://github.com/kubermatic/kubermatic/issues/6467))
- Enable VM resource quota for AWS provider ([#6468](https://github.com/kubermatic/kubermatic/issues/6468))
- Multus has been added as an addon.  ([#6477](https://github.com/kubermatic/kubermatic/issues/6477))
- CoreDNS version now based on Kubernetes version ([#6501](https://github.com/kubermatic/kubermatic/issues/6501))
- Add support for "use-octavia" setting in Openstack provider specs. It defaults to "true" but leaves the possibility to set it to "false" if your provider doesn't support Octavia yet but Neutron LBaaSv2
  ([#6529](https://github.com/kubermatic/kubermatic/issues/6529))
- Add components override field to set nodeportrange for apiserver ([#6533](https://github.com/kubermatic/kubermatic/issues/6533))
- OpenShift support is removed.  ([#6539](https://github.com/kubermatic/kubermatic/issues/6539))
- OpenStack: Add support for "use-octavia" setting in Cluster Openstack cloud specs ([#6565](https://github.com/kubermatic/kubermatic/issues/6565))
- Add support for Hetzner CCM ([#6588](https://github.com/kubermatic/kubermatic/issues/6588))
- Change default gatekeeper webhook timeout to 3 sec, and added option in cluster settings to configure it. ([#6709](https://github.com/kubermatic/kubermatic/issues/6709))
- Add support in Openstack datacenters to explicitly enable certain flavor types. ([#6612](https://github.com/kubermatic/kubermatic/issues/6612))
- Provide the possibility of configuring leader election parameters for user cluster components. ([#6641](https://github.com/kubermatic/kubermatic/issues/6641))
- Remove unused deprecated `certs` chart ([#6656](https://github.com/kubermatic/kubermatic/issues/6656))
- Add `registry_mirrors` to Seed node settings ([#6667](https://github.com/kubermatic/kubermatic/issues/6667))
- Upgrad Gatekeeper from 3.1.0-beta-9 to 3.1.3. NOTICE: this change also moves the Gatekeeper deployment from the Seed to the User clusters. This means that the user clusters will need some additional resources to run the Gatekeeper Pods. Admins please refer to the upgrade guidelines in the documentation. ([#6706](https://github.)com/kubermatic/kubermatic/issues/6706)
- Add spot instances as an option for the aws machines in the API  ([#6726](https://github.com/kubermatic/kubermatic/issues/6726))
- Add Multus-CNI to accessible addons. ([#6731](https://github.com/kubermatic/kubermatic/issues/6731))
- Allow to disable the s3-credentials Secret in the Minio chart ([#6760](https://github.com/kubermatic/kubermatic/issues/6760))
- Add `enable` and `enforce` OPA options to Admin Settings ([#6787](https://github.com/kubermatic/kubermatic/issues/6787))
- Installer does not listen on port 8080 anymore ([#6788](https://github.com/kubermatic/kubermatic/issues/6788))
- Node-local-dns is now using UDP for external queries ([#6796](https://github.com/kubermatic/kubermatic/issues/6796))
- Add validation for Kubermatic Constraint Template API. ([#6841](https://github.com/kubermatic/kubermatic/issues/6841))
- Fetch the provisioning cloud-init over the api-server  ([#6843](https://github.com/kubermatic/kubermatic/issues/6843))
- Add `FELIX_IGNORELOOSERPF=true` to `calico-node` container env to allow running on nodes with `net.ipv4.conf.*.rp_filter = 2` set. ([#6865](https://github.com/kubermatic/kubermatic/issues/6865))
- Hetzner AMD Cloud Server (CPX) now selectable when creating a user cluster ([#6872](https://github.com/kubermatic/kubermatic/issues/6872))
- Add GPU support for Azure provider ([#6605](https://github.com/kubermatic/kubermatic/issues/6605))

### Bugfixes

- Fix kube-system/coredns PodDisruptionBudget matchLabels in user clusters ([#6398](https://github.com/kubermatic/kubermatic/issues/6398))
- Fix S3 storage uploader CA bundle option flag ([#6732](https://github.com/kubermatic/kubermatic/issues/6732))
- Fix cases where GET and LIST endpoints for Kubermatic Constraints failed or didn't return all results because there were no related synced Gatekeeper Constraints on the user cluster by just taking the Status from the Gatekeeper Constraints and setting the Synced status to false if the Gatekeeper Constraint is missing. ([#6800])(https://github.com/kubermatic/kubermatic/issues/6800)
- Fix KAS service port in Tunneling agent configuration. ([#6569](https://github.com/kubermatic/kubermatic/issues/6569))
- Fix a bug in OPA-integration where deleting a Constraint Template in the seed cluster, when the user cluster Constraint Template is already deleted caused the deletion to get stuck.Fixed a bug in OPA-integration where creating a cluster with OPA-integration enabled didn't trigger the Constraint Template reconcile loop. ([#6580])(https://github.com/kubermatic/kubermatic/issues/6580)
- Fix issue with gatekeeper not recognizing the AdmissionReview v1 version by changing the webhook to use v1beta1 ([#6550](https://github.com/kubermatic/kubermatic/issues/6550))
- Fix a bug with kubermatic constraints delete getting stuck when corresponding user cluster constraint is missing ([#6598](https://github.com/kubermatic/kubermatic/issues/6598))
- Fix CE installer binary in EE downloads ([#6673](https://github.com/kubermatic/kubermatic/issues/6673))
- Fix nodeport-proxy role used with LoadBalancer expose strategy. ([#6646](https://github.com/kubermatic/kubermatic/issues/6646))
- Fix the operator failing to reconcile the ValidatingWebhookConfiguration object for the cluster validation webhook ([#6639](https://github.com/kubermatic/kubermatic/issues/6639))
- Fix installer trying an invalid certificate to test cert-manager ([#6761](https://github.com/kubermatic/kubermatic/issues/6761))

### Updates

- controller-runtime 0.8.1 ([#6450](https://github.com/kubermatic/kubermatic/issues/6450))
- CSI drivers ([#6594](https://github.com/kubermatic/kubermatic/issues/6594))
- Hetzner CSI, move to `csi` addon ([#6615](https://github.com/kubermatic/kubermatic/issues/6615))
- Prometheus to 0.25.0 ([#6647](https://github.com/kubermatic/kubermatic/issues/6647))
- Dex to 2.27.0 ([#6648](https://github.com/kubermatic/kubermatic/issues/6648))
- Minio to RELEASE.2021-03-04T00-53-13Z ([#6649](https://github.com/kubermatic/kubermatic/issues/6649))
- Loki to 2.1, use boltdb-shipper starting June 1st ([#6650](https://github.com/kubermatic/kubermatic/issues/6650))
- nginx-ingress-controller to 0.44.0 ([#6651](https://github.com/kubermatic/kubermatic/issues/6651))
- blackbox-exporter to 0.18 ([#6652](https://github.com/kubermatic/kubermatic/issues/6652))
- node-exporter to 1.1.2 ([#6653](https://github.com/kubermatic/kubermatic/issues/6653))
- Karma to 0.80 ([#6654](https://github.com/kubermatic/kubermatic/issues/6654))
- Grafana to 7.4.3 ([#6655](https://github.com/kubermatic/kubermatic/issues/6655))
- oauth2-proxy to 7.0.1 ([#6657](https://github.com/kubermatic/kubermatic/issues/6657))
- Go 1.16.1 ([#6684](https://github.com/kubermatic/kubermatic/issues/6684))
- machine-controller to v1.27.1 ([#6695](https://github.com/kubermatic/kubermatic/issues/6695))
- OpenVPN image to version v2.5.2-r0. ([#6697](https://github.com/kubermatic/kubermatic/issues/6697))
- Velero to 1.5.3. ([#6701](https://github.com/kubermatic/kubermatic/issues/6701))

### Dashboard

- Add resource quota settings to the admin panel. ([#3019](https://github.com/kubermatic/dashboard/issues/3019))
- Add autocompletions for vSphere datastores. ([#3020](https://github.com/kubermatic/dashboard/issues/3020))
- Add option to disable User SSH Key Agent from the cluster wizard. ([#3025](https://github.com/kubermatic/dashboard/issues/3025))
- Remove CoreOS ([#3027](https://github.com/kubermatic/dashboard/issues/3027))
- AWS node sizes in the wizard now provide GPU information. ([#3038](https://github.com/kubermatic/dashboard/issues/3038))
- Filter external openstack networks during cluster creation ([#3053](https://github.com/kubermatic/dashboard/issues/3053))
- Add changelog support ([#3081](https://github.com/kubermatic/dashboard/issues/3081))
- Remove OpenShift support. ([#3100](https://github.com/kubermatic/dashboard/issues/3100))
- Redesign add/edit member dialog ([#3104](https://github.com/kubermatic/dashboard/issues/3104))
- Add GPU count display for Alibaba instance types. ([#3113](https://github.com/kubermatic/dashboard/issues/3113))
- Remove duplicated KubeAdm hints from cluster page. ([#3114](https://github.com/kubermatic/dashboard/issues/3114))
- Redesign manage SSH keys dialog on cluster details to improve user experience. ([#3120](https://github.com/kubermatic/dashboard/issues/3120))
- Change VSPhere's diskSizeGB option from optional to required. ([#3121](https://github.com/kubermatic/dashboard/issues/3121))
- Redesign autocomplete inputs. Right now spinner will be displayed next to the input that loads autocompletions in the background. ([#3122](https://github.com/kubermatic/dashboard/issues/3122))
- Add info about GPU count for Azure instances. ([#3140](https://github.com/kubermatic/dashboard/issues/3140))
- Allow custom links to be placed in the help and support panel ([#3141](https://github.com/kubermatic/dashboard/issues/3141))
- Add support for OPA to UI ([#3147](https://github.com/kubermatic/dashboard/issues/3147))
- Add network to Hetzner ([#3158](https://github.com/kubermatic/dashboard/issues/3158))
- Add `enable` and `enforce` OPA options to Admin Settings ([#3206](https://github.com/kubermatic/dashboard/issues/3206))

### Bugfixes

- Fix bug with changing the theme based on the color scheme if enforced_theme was set. ([#3163](https://github.com/kubermatic/dashboard/issues/3163))




# Kubermatic 2.16

## [v2.16.12](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.12)

### Security

Two vulnerabilities were identified in Kubernetes ([CVE-2021-25741](https://github.com/kubernetes/kubernetes/issues/104980) and [CVE-2020-8561](https://github.com/kubernetes/kubernetes/issues/104720)), of which one (CVE-2021-25741) was fixed in Kubernetes 1.19.15 / 1.20.11. CVE-2020-8561 is mitigated by Kubermatic not allowing users to reconfigure the kube-apiserver.

Because of these updates, this KKP release includes automatic update rules for all 1.19/1.20 clusters older than 1.19.15 / 1.20.11. This release also removes all affected Kubernetes versions from the list of supported versions. While CVE-2020-8561 affects the controlplane, CVE-2021-25741 affects the kubelets, which means that updating the controlplane is not enough. Once the automated controlplane updates have completed, an administrator must manually patch all vulnerable `MachineDeployment`s in all affected userclusters.

To lower the resource consumption on the seed clusters during the reconciliation / node rotation, it's recommended to adjust the `spec.seedControllerManager.maximumParallelReconciles` option in the `KubermaticConfiguration` to restrict the number of parallel updates. Users of the legacy `kubermatic` Helm chart need to update `kubermatic.maxParallelReconcile` in their `values.yaml` to achieve the same effect.

The automatic update rules can, if needed, be overwritten using the `spec.versions.kubernetes.updates` field in the `KubermaticConfiguration` or updating the `updates.yaml` if using the legacy `kubermatic` Helm chart. See [#7824](https://github.com/kubermatic/kubermatic/issues/7824) for how the versions and updates are configured. It is however not recommended to deviate from the default and leave userclusters vulnerable.

### Misc

- Add support of Kubernetes 1.20 in cluster-autoscaler addon ([#7521](https://github.com/kubermatic/kubermatic/issues/7521))
- Remove Gatekeeper from default accessible addon list ([#7532](https://github.com/kubermatic/kubermatic/issues/7532))
- Fix dashboard source in the Prometheus Exporter dashboard ([#7640](https://github.com/kubermatic/kubermatic/issues/7640))




## [v2.16.11](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.11)

### Security

- Upgrade machine-controller to v1.24.5 to address [runC vulnerability CVE-2021-30465](https://github.com/opencontainers/runc/security/advisories/GHSA-c3xm-pvg7-gh7r) ([#7165](https://github.com/kubermatic/kubermatic/issues/7165))

### Bugfixes

- Fix a bug that always applies default values to container resources ([#7302](https://github.com/kubermatic/kubermatic/issues/7302))
- Add `ClusterFeatureCCMClusterName` feature for OpenStack clusters. This feature adds the `--cluster-name` flag to the OpenStack external CCM deployment. The feature gate is enabled by default for newly created clusters. Enabling this feature gate for existing clusters will cause the external CCM to lose the track of the existing cloud resources (such as Load Balancers), so it's up to the users to manually clean up any leftover resources. ([#7330](https://github.com/kubermatic/kubermatic/issues/7330))

### Misc

- Add support for `use-octavia` setting in Openstack provider specs. It defaults to `true` but leaves the possibility to set it to `false` if your provider doesn't support Octavia yet but Neutron LBaaSv2 ([#6529](https://github.com/kubermatic/kubermatic/issues/6529))




## [v2.16.10](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.10)

### Misc

- Fix CA certificates not being mounted on Seed clusters using CentOS/Fedora ([#6680](https://github.com/kubermatic/kubermatic/issues/6680))
- Use the systemd cgroup driver for newly-created Kubernetes 1.19+ clusters using the kubeadm provider. Since the kubelet-configmap addon is not reconciled, this change will not affect existing clusters, only newly-created clusters. ([#7065](https://github.com/kubermatic/kubermatic/issues/7065))
- Re-enable NodeLocal DNS Cache in user clusters ([#7075](https://github.com/kubermatic/kubermatic/issues/7075))




## [v2.16.9](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.9)

### Misc

- Adds `FELIX_IGNORELOOSERPF=true` to `calico-node` container env to allow running on nodes with `net.ipv4.conf.*.rp_filter = 2` set ([#6855](https://github.com/kubermatic/kubermatic/issues/6855))
- Fix default version configuration to have automatic upgrade from Kubernetes 1.16 to 1.17 ([#6899](https://github.com/kubermatic/kubermatic/issues/6899))
- Fix OpenStack crashing with Kubernetes 1.20 and 1.21 ([#6924](https://github.com/kubermatic/kubermatic/issues/6924))
- Update machine-controller to 1.24.4. Fixed double instance creation in us-east1 AWS ([#6962](https://github.com/kubermatic/kubermatic/issues/6962))




## [v2.16.8](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.8)

### Misc

- Installer does not listen on port 8080 anymore ([#6788](https://github.com/kubermatic/kubermatic/issues/6788))
- Node-local-dns is now using UDP for external queries ([#6796](https://github.com/kubermatic/kubermatic/issues/6796))




## [v2.16.7](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.7)

### Bugfixes

- Fix deployment of Openstack CCM ([#6750](https://github.com/kubermatic/kubermatic/issues/6750))
- Projects are now synced from the Master cluster to all Seed clusters. Fixes issue where user clusters could not be created properly on multi seed clusters, when the seed is not also the master cluster ([#6754](https://github.com/kubermatic/kubermatic/issues/6754))
- Fix installer trying an invalid certificate to test cert-manager ([#6761](https://github.com/kubermatic/kubermatic/issues/6761))

### Misc

- Allow to disable the s3-credentials Secret in the Minio chart ([#6760](https://github.com/kubermatic/kubermatic/issues/6760))




## [v2.16.6](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.6)

### Bugfixes

- Fix cert-manager validating webhook ([#6741](https://github.com/kubermatic/kubermatic/issues/6741))
- Fix the operator failing to reconcile the ValidatingWebhookConfiguration object for the cluster validation webhook ([#6639](https://github.com/kubermatic/kubermatic/issues/6639))

### Misc

- Change default gatekeeper webhook timeout to 3 sec ([#6709](https://github.com/kubermatic/kubermatic/issues/6709))
- Update Velero to 1.5.3 ([#6701](https://github.com/kubermatic/kubermatic/issues/6701))



## [v2.16.5](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.5)

### Misc

- Update nginx-ingress-controller to 0.44.0 ([#6651](https://github.com/kubermatic/kubermatic/issues/6651))

### Bugfixes

- Fix CE installer binary in EE downloads ([#6673](https://github.com/kubermatic/kubermatic/issues/6673))




## [v2.16.4](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.4)

### Misc

- Provide the possibility of configuring leader election parameters for user cluster components ([#6641](https://github.com/kubermatic/kubermatic/issues/6641))
- Add `registry_mirrors` to Seed node settings ([#6667](https://github.com/kubermatic/kubermatic/issues/6667))

### Bugfixes

- Fix nodeport-proxy role used with LoadBalancer expose strategy ([#6646](https://github.com/kubermatic/kubermatic/issues/6646))




## [v2.16.3](https://github.com/kubermatic/dashboard/releases/tag/v2.16.3)

This version includes significant improvements to Hetzner userclusters. Please refer to the amended [2.16 upgrade notes](https://docs.kubermatic.com/kubermatic/v2.16/upgrading/2.15_to_2.16/) for more information.

### Misc

- Add support for Hetzner CCM ([#6588](https://github.com/kubermatic/kubermatic/issues/6588))
- Update Hetzner CSI ([#6615](https://github.com/kubermatic/kubermatic/issues/6615))
- Update CSI drivers ([#6594](https://github.com/kubermatic/kubermatic/issues/6594))
- Increase default gatekeeper webhook timeout from 2 to 10 seconds, and add option in cluster settings to configure it ([#6603](https://github.com/kubermatic/kubermatic/issues/6603))
- Remove duplicate Kubeadm hints from cluster page ([#3114](https://github.com/kubermatic/dashboard/issues/3114))
- Change vSphere's diskSizeGB option from optional to required ([#3121](https://github.com/kubermatic/dashboard/issues/3121))

### Bugfixes

- Fix a bug in OPA integration where deleting a Constraint Template in the seed cluster, when the user cluster Constraint Template is already deleted, caused the deletion to get stuck. ([#6582](https://github.com/kubermatic/kubermatic/issues/6582))
- Fix a bug in OPA integration where creating a cluster with OPA integration enabled didn't trigger the Constraint Template reconcile loop ([#6582](https://github.com/kubermatic/kubermatic/issues/6582))
- Fix a bug with Kubermatic constraints delete getting stuck when corresponding user cluster constraint is missing ([#6598](https://github.com/kubermatic/kubermatic/issues/6598))




## [v2.16.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.2)

### Misc

- Fix KAS service port in Tunneling agent configuration ([#6569](https://github.com/kubermatic/kubermatic/issues/6569))




## [v2.16.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.1)

**Note:** No Docker images have been published for this release. Please use 2.16.2 instead.

### Misc

- Fix issue with gatekeeper not recognizing the AdmissionReview v1 version by changing the webhook to use v1beta1 ([#6550](https://github.com/kubermatic/kubermatic/issues/6550))




## [v2.16.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.0)

Before upgrading, make sure to read the [general upgrade guidelines](https://docs.kubermatic.com/kubermatic/v2.16/upgrading/guidelines/)
as well as the [2.16 upgrade notes](https://docs.kubermatic.com/kubermatic/v2.16/upgrading/2.15_to_2.16/).

### Supported Kubernetes Versions

* 1.17.9
* 1.17.11
* 1.17.12
* 1.17.13
* 1.17.16
* 1.18.6
* 1.18.8
* 1.18.10
* 1.18.14
* 1.19.0
* 1.19.2
* 1.19.3
* 1.20.2

### Highlights

- Add Kubernetes 1.20, remove Kubernetes 1.16 ([#6032](https://github.com/kubermatic/kubermatic/issues/6032), [#6122](https://github.com/kubermatic/kubermatic/issues/6122))
- Add Tunneling expose strategy (tech preview) ([#6445](https://github.com/kubermatic/kubermatic/issues/6445))
- First parts of the revamped V2 API are available as a preview (see API section below)
- cert-manager is not a hard dependency for KKP anymore; certificates are acquired using Ingress annotations instead ([#5962](https://github.com/kubermatic/kubermatic/issues/5962), [#5969](https://github.com/kubermatic/kubermatic/issues/5969), [#6069](https://github.com/kubermatic/kubermatic/issues/6069), [#6282](https://github.com/kubermatic/kubermatic/issues/6282), [#6119](https://github.com/kubermatic/kubermatic/issues/6119))

### Breaking Changes

- This is the last release for which the legacy Helm chart is available. Users are encouraged to migrate to the KKP Operator.

### Cloud Provider

- Add Anexia provider ([#6101](https://github.com/kubermatic/kubermatic/issues/6101), [#6128](https://github.com/kubermatic/kubermatic/issues/6128))
- Make `cluster` field in vSphere datacenter spec optional ([#5886](https://github.com/kubermatic/kubermatic/issues/5886))
- Fix creation of RHEL8 machines ([#5950](https://github.com/kubermatic/kubermatic/issues/5950))
- Default to the latest available OpenStack CCM version. This fixes a bug where newly-created OpenStack clusters running Kubernetes 1.19 were using the in-tree cloud provider instead of the external CCM. Those clusters will remain to use the in-tree cloud provider until the CCM migration mechanism is not implemented. The OpenStack clusters running Kubernetes 1.19+ and created with KKP 2.15.6+ will use the external CCM ([#6272](https://github.com/kubermatic/kubermatic/issues/6272))
- Use CoreOS-Cloud-Config for Flatcar machines on AWS ([#6405](https://github.com/kubermatic/kubermatic/issues/6405))

### Misc

- Add Kubernetes 1.17.12, 1.19.2 ([#5927](https://github.com/kubermatic/kubermatic/issues/5927))
- Add Kubernetes 1.17.13, 1.18.10, 1.19.3 ([#6032](https://github.com/kubermatic/kubermatic/issues/6032))
- Add `DNSDomain` variable to addon TemplateData ([#6160](https://github.com/kubermatic/kubermatic/issues/6160))
- Add `MaximumParallelReconciles` option to KubermaticConfiguration ([#6002](https://github.com/kubermatic/kubermatic/issues/6002))
- Add `operator.kubermatic.io/skip-reconciling` annotation to Seeds to allow step-by-step seed cluster upgrades ([#5883](https://github.com/kubermatic/kubermatic/issues/5883))
- Add a controller which syncs Constraints from the seed cluster user cluster namespace to the corresponding user clusters when OPA integration is enabled ([#6224](https://github.com/kubermatic/kubermatic/issues/6224))
- Add a new feature gate to the seed-controller to enable etcd-launcher for all user clusters ([#5997](https://github.com/kubermatic/kubermatic/issues/5997), [#5973](https://github.com/kubermatic/kubermatic/issues/5973))
- Add admission control configuration for the user cluster API deployment ([#6308](https://github.com/kubermatic/kubermatic/issues/6308))
- Add new cluster-autoscaler addon ([#5869](https://github.com/kubermatic/kubermatic/issues/5869))
- Add service account token volume projection options for user clusters ([#6382](https://github.com/kubermatic/kubermatic/issues/6382))
- Add support for KubermaticConfiguration in `image-loader` utility ([#6063](https://github.com/kubermatic/kubermatic/issues/6063))
- Add support for `InstanceReadyCheckPeriod` and `InstanceReadyCheckTimeout` to Openstack provider ([#6139](https://github.com/kubermatic/kubermatic/issues/6139))
- Allow controlling external cluster functionality with global settings ([#5912](https://github.com/kubermatic/kubermatic/issues/5912))
- Allow to customize the Docker image tag for Cluster Addons ([#6102](https://github.com/kubermatic/kubermatic/issues/6102))
- Always mount CABundle for Dex into the kubermatic-api Pod, even when `OIDCKubeCfgEndpoint` is disabled ([#5968](https://github.com/kubermatic/kubermatic/issues/5968))
- Avoid forcing cleanup of failed backup job pods, so that cluster administrators can still look at the pod's logs ([#5913](https://github.com/kubermatic/kubermatic/issues/5913))
- Create an RBAC role to allow kubeadm to get nodes. This fixes nodes failing to join kubeadm clusters running Kubernetes 1.18+ ([#6241](https://github.com/kubermatic/kubermatic/issues/6241))
- Dex configuration does not support `staticPasswordLogins` anymore, use `staticPasswords` instead ([#6289](https://github.com/kubermatic/kubermatic/issues/6289))
- Expose `ServiceAccountSettings` in cluster API object ([#6423](https://github.com/kubermatic/kubermatic/issues/6423))
- Extend Cluster CRD with `PodNodeSelectorAdmissionPluginConfig` ([#6305](https://github.com/kubermatic/kubermatic/issues/6305))
- Extend global settings for resource quota ([#6448](https://github.com/kubermatic/kubermatic/issues/6448))
- Fix Kubermatic Operator getting stuck in Kubernetes 1.18 clusters when reconciling Ingresses ([#5915](https://github.com/kubermatic/kubermatic/issues/5915))
- Fix Prometheus alerts misfiring about absent KKP components ([#6167](https://github.com/kubermatic/kubermatic/issues/6167))
- Fix Prometheus `cluster_info` metric having the wrong `type` label ([#6138](https://github.com/kubermatic/kubermatic/issues/6138))
- Fix cert-manager webhook Service naming ([#6150](https://github.com/kubermatic/kubermatic/issues/6150))
- Fix installer not being able to probe for Certificate support ([#6135](https://github.com/kubermatic/kubermatic/issues/6135))
- Fix master-controller-manager being too verbose ([#5889](https://github.com/kubermatic/kubermatic/issues/5889))
- Fix missing logo in Dex login screens ([#6019](https://github.com/kubermatic/kubermatic/issues/6019))
- Fix orphaned apiserver-is-running initContainers in usercluster controlplane. This can cause a short reconciliation burst to bring older usercluster resources in all Seed clusters up to date. Tune the maxReconcileLimit if needed ([#6329](https://github.com/kubermatic/kubermatic/issues/6329))
- Fix overflowing `kubermatic.io/cleaned-up-loadbalancers` annotation on Cluster objects ([#6229](https://github.com/kubermatic/kubermatic/issues/6229))
- Fix user-cluster Grafana dashboard showing inflated numbers under certain circumstances ([#6026](https://github.com/kubermatic/kubermatic/issues/6026))
- Gatekeeper is now deployed automatically for the user clusters as part of Kubermatic OPA integration ([#5814](https://github.com/kubermatic/kubermatic/issues/5814))
- Improve Helm error handling in KKP Installer ([#6076](https://github.com/kubermatic/kubermatic/issues/6076))
- Improved initial node deployment creation process. Right now cluster annotation is used to save the node deployment object before it is created to improve stability. ([#6064](https://github.com/kubermatic/kubermatic/issues/6064))
- Make etcd-launcher repository configurable in `KubermaticConfiguration` CRD ([#5880](https://github.com/kubermatic/kubermatic/issues/5880))
- Make imagePullSecret optional for Kubermatic Operator ([#5874](https://github.com/kubermatic/kubermatic/issues/5874))
- Makefile: better support for compiling with debug symbols ([#5933](https://github.com/kubermatic/kubermatic/issues/5933))
- Move to k8s.gcr.io Docker registry for CoreDNS, metrics-server, and NodeLocalDNSCache ([#5963](https://github.com/kubermatic/kubermatic/issues/5963))
- Raise cert-manager resource limits to prevent OOMKills ([#6216](https://github.com/kubermatic/kubermatic/issues/6216))
- Remove Helm charts for deprecated ELK (Elasticsearch, Fluentbit, Kibana) stack ([#6149](https://github.com/kubermatic/kubermatic/issues/6149))
- Remove kubernetes-dashboard Helm chart ([#6108](https://github.com/kubermatic/kubermatic/issues/6108))
- Ship image-loader as part of GitHub releases ([#6092](https://github.com/kubermatic/kubermatic/issues/6092))
- Start as a fresh etcd member if data directory is empty ([#6221](https://github.com/kubermatic/kubermatic/issues/6221))
- The User SSH Key Agent can now be disabled per cluster in order to manage SSH keys manually ([#6443](https://github.com/kubermatic/kubermatic/issues/6443), [#6426](https://github.com/kubermatic/kubermatic/issues/6426), [#6444](https://github.com/kubermatic/kubermatic/issues/6444))
- Update to networking.k8s.io/v1beta1 for managing Ingresses ([#6292](https://github.com/kubermatic/kubermatic/issues/6292))

### UI

- Add Datastore/Datastore Cluster support to the VSphere provider in the wizard ([#2762](https://github.com/kubermatic/dashboard/issues/2762))
- Add Preset management UI to the admin settings ([#2880](https://github.com/kubermatic/dashboard/issues/2880))
- Add flag `continuouslyReconcile` to addons ([#2618](https://github.com/kubermatic/dashboard/issues/2618))
- Add option to enable/disable external cluster import feature from admin settings ([#2644](https://github.com/kubermatic/dashboard/issues/2644))
- Add option to filter clusters ([#2695](https://github.com/kubermatic/dashboard/issues/2695))
- Add option to specify Pod Node Selector Configuration ([#2929](https://github.com/kubermatic/dashboard/issues/2929))
- Add support for Anexia cloud provider ([#2693](https://github.com/kubermatic/dashboard/issues/2693))
- Add support for `instanceReadyCheckPeriod` and `instanceReadyCheckTimeout` to the Openstack provider ([#2781](https://github.com/kubermatic/dashboard/issues/2781))
- Add the option to specify OS/data disk size for Azure clusters and support selection of multiple zones ([#2547](https://github.com/kubermatic/dashboard/issues/2547))
- Allow adding help text for addon forms ([#2770](https://github.com/kubermatic/dashboard/issues/2770))
- Allow specifying help text for addon form controls ([#6117](https://github.com/kubermatic/kubermatic/issues/6117))
- Azure resource groups, security groups and route tables will be now loaded from the API to provide autocompletion ([#2936](https://github.com/kubermatic/dashboard/issues/2936))
- Cluster related resources will be now displayed in tabs ([#2876](https://github.com/kubermatic/dashboard/issues/2876))
- Display deletion state of accessible addons ([#2674](https://github.com/kubermatic/dashboard/issues/2674))
- Distributions in the wizard are now correctly shown based on admin settings ([#2839](https://github.com/kubermatic/dashboard/issues/2839))
- Fix addon variables edit ([#2731](https://github.com/kubermatic/dashboard/issues/2731))
- Fix end of life chip/badge styling ([#2841](https://github.com/kubermatic/dashboard/issues/2841))
- Fix issue with listing all projects if one of them had no owner set ([#2848](https://github.com/kubermatic/dashboard/issues/2848))
- Fix loading of the access rights in the SSH keys view ([#2645](https://github.com/kubermatic/dashboard/issues/2645))
- Fix missing group name on Service Account list ([#2851](https://github.com/kubermatic/dashboard/issues/2851))
- Fix project selector auto-scroll to selected value on refresh ([#2638](https://github.com/kubermatic/dashboard/issues/2638))
- Fix styling of 'Add RBAC Binding' dialog ([#2850](https://github.com/kubermatic/dashboard/issues/2850))
- Fix the bug with labels that were removed from form after pressing enter key ([#2903](https://github.com/kubermatic/dashboard/issues/2903))
- Fix wizard rendering in Safari ([#2661](https://github.com/kubermatic/dashboard/issues/2661))
- Improve browser support ([#2668](https://github.com/kubermatic/dashboard/issues/2668))
- Non-existing default projects will be now unchecked in the settings ([#2630](https://github.com/kubermatic/dashboard/issues/2630))
- Openstack: Fill the form with defaults for username and domain name ([#2928](https://github.com/kubermatic/dashboard/issues/2928))
- Remove `mat-icon` dependency and replace them by own icons ([#2883](https://github.com/kubermatic/dashboard/issues/2883))
- Restore list of cluster nodes ([#2773](https://github.com/kubermatic/dashboard/issues/2773))
- Support User SSH Keys in Kubeadm cloud provider ([#2747](https://github.com/kubermatic/dashboard/issues/2747))
- Switch to new cluster endpoints ([#2641](https://github.com/kubermatic/dashboard/issues/2641))
- The seed datacenter param was removed from the path. It is no longer required thanks to the switch to the new version of API ([#2815](https://github.com/kubermatic/dashboard/issues/2815))
- Update cluster resource loading states ([#2690](https://github.com/kubermatic/dashboard/issues/2690))
- Update login page background ([#2849](https://github.com/kubermatic/dashboard/issues/2849))
- Use an endpoint to get AWS Security Group IDs in the wizard for the cluster creation ([#2909](https://github.com/kubermatic/dashboard/issues/2909))
- Use endpoint to load Azure subnets autocompletions ([#2988](https://github.com/kubermatic/dashboard/issues/2988))

### API

- Add endpoint to list AWS Security Groups: `GET /api/v1/providers/aws/{dc}/securitygroups` ([#6331](https://github.com/kubermatic/kubermatic/issues/6331))
- Add new endpoints to list/create/update presets ([#6208](https://github.com/kubermatic/kubermatic/issues/6208))
- remove deprecated `v1/nodes` endpoints ([#6031](https://github.com/kubermatic/kubermatic/issues/6031))
- change endpoint name from DeleteMachineNode to DeleteMachineDeploymentNode ([#6115](https://github.com/kubermatic/kubermatic/issues/6115))
- first parts of the revamped V2 API are available:
  - manage `ContraintTemplate`s ([#5917](https://github.com/kubermatic/kubermatic/issues/5917), [#5966](https://github.com/kubermatic/kubermatic/issues/5966), [#5885](https://github.com/kubermatic/kubermatic/issues/5885), [#5959](https://github.com/kubermatic/kubermatic/issues/5959))
  - manage `Contraint`s ([#6034](https://github.com/kubermatic/kubermatic/issues/6034), [#6127](https://github.com/kubermatic/kubermatic/issues/6127), [#6116](https://github.com/kubermatic/kubermatic/issues/6116), [#6141](https://github.com/kubermatic/kubermatic/issues/6141))
  - list Azure Subnets, VNets etc. ([#6395](https://github.com/kubermatic/kubermatic/issues/6395), [#6363](https://github.com/kubermatic/kubermatic/issues/6363), [#6340](https://github.com/kubermatic/kubermatic/issues/6340))
  - list provider related resources ([#6228](https://github.com/kubermatic/kubermatic/issues/6228), [#6262](https://github.com/kubermatic/kubermatic/issues/6262), [#6264](https://github.com/kubermatic/kubermatic/issues/6264), [#6287](https://github.com/kubermatic/kubermatic/issues/6287), [#6223](https://github.com/kubermatic/kubermatic/issues/6223), [#6275](https://github.com/kubermatic/kubermatic/issues/6275))
  - manage MachineDeployments ([#6109](https://github.com/kubermatic/kubermatic/issues/6109), [#6111](https://github.com/kubermatic/kubermatic/issues/6111), [#6156](https://github.com/kubermatic/kubermatic/issues/6156), [#6068](https://github.com/kubermatic/kubermatic/issues/6068), [#6157](https://github.com/kubermatic/kubermatic/issues/6157), [#6074](https://github.com/kubermatic/kubermatic/issues/6074), [#6132](https://github.com/kubermatic/kubermatic/issues/6132), [#6107](https://github.com/kubermatic/kubermatic/issues/6107), [#6136](https://github.com/kubermatic/kubermatic/issues/6136))
  - manage cluster addons ([#6215](https://github.com/kubermatic/kubermatic/issues/6215))
  - manage RBAC in clusters ([#6196](https://github.com/kubermatic/kubermatic/issues/6196), [#6187](https://github.com/kubermatic/kubermatic/issues/6187), [#6177](https://github.com/kubermatic/kubermatic/issues/6177), [#6162](https://github.com/kubermatic/kubermatic/issues/6162))
  - manage Gatekeeper (Open Policy Agent, OPA) ([#6306](https://github.com/kubermatic/kubermatic/issues/6306), [#6286](https://github.com/kubermatic/kubermatic/issues/6286))
  - manage cluster nodes ([#6030](https://github.com/kubermatic/kubermatic/issues/6030), [#6130](https://github.com/kubermatic/kubermatic/issues/6130))
  - manage cluster SSH keys ([#6005](https://github.com/kubermatic/kubermatic/issues/6005))
  - list cluster namespaces ([#6004](https://github.com/kubermatic/kubermatic/issues/6004))
  - access dashboard proxy ([#6299](https://github.com/kubermatic/kubermatic/issues/6299))
  - access cluster health and metrics ([#5872](https://github.com/kubermatic/kubermatic/issues/5872), [#5908](https://github.com/kubermatic/kubermatic/issues/5908))
  - manage kubeconfig and tokens ([#5881](https://github.com/kubermatic/kubermatic/issues/5881), [#6238](https://github.com/kubermatic/kubermatic/issues/6238))
  - list cluster updates ([#6021](https://github.com/kubermatic/kubermatic/issues/6021))

### Updates

- Prometheus 2.23.0 ([#6290](https://github.com/kubermatic/kubermatic/issues/6290))
- Thanos 0.17.2 ([#6290](https://github.com/kubermatic/kubermatic/issues/6290))
- Velero 1.5.2 ([#6145](https://github.com/kubermatic/kubermatic/issues/6145))
- machine-controller v1.23.1 ([#6387](https://github.com/kubermatic/kubermatic/issues/6387))


### Changes since v2.16.0-beta.1

- Add option to disable User SSH Key Agent from the cluster wizard ([#3025](https://github.com/kubermatic/dashboard/issues/3025))




# Kubermatic 2.15

## [v2.15.13](https://github.com/kubermatic/kubermatic/releases/tag/v2.15.13)

This is the last planned release for the `release/v2.15` branch. Uses are encouraged to update at least to 2.16 to receive future updates.

### Security

Two vulnerabilities were identified in Kubernetes ([CVE-2021-25741](https://github.com/kubernetes/kubernetes/issues/104980) and [CVE-2020-8561](https://github.com/kubernetes/kubernetes/issues/104720)) of which one (CVE-2021-25741) was fixed in Kubernetes 1.19.15. CVE-2020-8561 is mitigated by Kubermatic not allowing users to reconfigure the kube-apiserver.

Because of these updates, this KKP release includes an automatic update rule for all 1.19 clusters older than 1.19.15. This release also removes all affected Kubernetes versions from the list of supported versions. While CVE-2020-8561 affects the controlplane, CVE-2021-25741 affects the kubelets, which means that updating the controlplane is not enough. Once the automated controlplane updates have completed, an administrator must manually patch all vulnerable `MachineDeployment`s in all affected userclusters.

To lower the resource consumption on the seed clusters during the reconciliation / node rotation, it's recommended to adjust the `spec.seedControllerManager.maximumParallelReconciles` option in the `KubermaticConfiguration` to restrict the number of parallel updates. Users of the legacy `kubermatic` Helm chart need to update `kubermatic.maxParallelReconcile` in their `values.yaml` to achieve the same effect.

The automatic update rules can, if needed, be overwritten using the `spec.versions.kubernetes.updates` field in the `KubermaticConfiguration` or updating the `updates.yaml` if using the legacy `kubermatic` Helm chart. See [#7823](https://github.com/kubermatic/kubermatic/issues/7823) for how the versions and updates are configured. It is however not recommended to deviate from the default and leave userclusters vulnerable.

### Misc

- Upgrade machine-controller to v1.19.2 ([#7164](https://github.com/kubermatic/kubermatic/issues/7164))
- Fix dashboard source in the Prometheus Exporter dashboard ([#7640](https://github.com/kubermatic/kubermatic/issues/7640))




## [v2.15.12](https://github.com/kubermatic/kubermatic/releases/tag/v2.15.12)

### Misc

- Adds `FELIX_IGNORELOOSERPF=true` to `calico-node` container env to allow running on nodes with `net.ipv4.conf.*.rp_filter = 2` set. [#6864](https://github.com/kubermatic/kubermatic/issues/6864) ([moelsayed](https://github.com/moelsayed))
- Update machine-controller to 1.19.1. Fixed double instance creation in us-east1 AWS [#6969](https://github.com/kubermatic/kubermatic/issues/6969) ([kron4eg](https://github.com/kron4eg))

### UI

- Rename every occurrence of "Node Deployment" in the UI to "Machine Deployment" [#3282](https://github.com/kubermatic/dashboard/issues/3282) ([cedi](https://github.com/cedi))




## [v2.15.11](https://github.com/kubermatic/kubermatic/releases/tag/v2.15.11)

### Misc

- Installer does not listen on port 8080 anymore ([#6788](https://github.com/kubermatic/kubermatic/issues/6788))
- Node-local-dns is now using UDP for external queries ([#6796](https://github.com/kubermatic/kubermatic/issues/6796))




## [v2.15.10](https://github.com/kubermatic/kubermatic/releases/tag/v2.15.10)

### Bugfixes

- Fix cert-manager validating webhook ([#6741](https://github.com/kubermatic/kubermatic/issues/6741))




## [v2.15.9](https://github.com/kubermatic/kubermatic/releases/tag/v2.15.9)

### Bugfixes

- Fix CA certificates not being mounted on Seed clusters using CentOS/Fedora ([#6680](https://github.com/kubermatic/kubermatic/issues/6680))




## [v2.15.8](https://github.com/kubermatic/kubermatic/releases/tag/v2.15.8)

### Misc

- Provide the possibility of configuring leader election parameters for user cluster components ([#6641](https://github.com/kubermatic/kubermatic/issues/6641))

### Bugfixes

- Fix CE installer binary in EE downloads ([#6673](https://github.com/kubermatic/kubermatic/issues/6673))




## [v2.15.7](https://github.com/kubermatic/kubermatic/releases/tag/v2.15.7)

### Misc

- [ATTN] Fix orphaned apiserver-is-running initContainers in usercluster controlplane. This can cause a short reconciliation burst to bring older usercluster resources in all Seed clusters up to date. Tune the maxReconcileLimit if needed ([#6336](https://github.com/kubermatic/kubermatic/issues/6336))
- Dex does not require cert-manager CRDs to be installed anymore, certificates are acquired via Ingress annotation ([#6284](https://github.com/kubermatic/kubermatic/issues/6284))
- Add option to specify Pod Node Selector Configuration ([#2957](https://github.com/kubermatic/dashboard/issues/2957))
- Extend Cluster CRD for PodNodeSelectorAdmissionPluginConfig ([#6402](https://github.com/kubermatic/kubermatic/issues/6402))
- Add admission control configuration for the user cluster API deployment ([#6431](https://github.com/kubermatic/kubermatic/issues/6431))
- Default to the latest available OpenStack CCM version. This fixes a bug where newly-created OpenStack clusters running Kubernetes 1.19 were using the in-tree cloud provider instead of the external CCM. Those clusters will remain to use the in-tree cloud provider until the CCM migration mechanism is not implemented. The OpenStack clusters running Kubernetes 1.19+ and created with KKP 2.15.6+ will use the external CCM ([#6300](https://github.com/kubermatic/kubermatic/issues/6300))




## [v2.15.6](https://github.com/kubermatic/kubermatic/releases/tag/v2.15.6)

### Misc

- Dex does not require cert-manager CRDs to be installed anymore, certificates are acquired via Ingress annotation ([#6282](https://github.com/kubermatic/kubermatic/issues/6282))




## [v2.15.5](https://github.com/kubermatic/kubermatic/releases/tag/v2.15.5)

### Bugfixes

- Create an RBAC role to allow kubeadm to get nodes. This fixes nodes failing to join kubeadm clusters running Kubernetes 1.18+ ([#6241](https://github.com/kubermatic/kubermatic/issues/6241))
- Fix installer not being able to probe for Certificate support ([#6135](https://github.com/kubermatic/kubermatic/issues/6135))
- Fix overflowing `kubermatic.io/cleaned-up-loadbalancers` annotation on Cluster objects ([#6229](https://github.com/kubermatic/kubermatic/issues/6229))



## [v2.15.4](https://github.com/kubermatic/kubermatic/releases/tag/v2.15.4)

### Misc

- Operator does not require cert-manager to be installed; certificates are configured via Ingress annotations instead ([#6069](https://github.com/kubermatic/kubermatic/issues/6069))
- Fix Prometheus cluster_info metric having the wrong `type` label ([#6138](https://github.com/kubermatic/kubermatic/issues/6138))
- Fix cert-manager webhook Service naming ([#6150](https://github.com/kubermatic/kubermatic/issues/6150))
- Fix Prometheus alerts misfiring about absent KKP components ([#6167](https://github.com/kubermatic/kubermatic/issues/6167))
- Raise cert-manager resource limits to prevent OOMKills ([#6216](https://github.com/kubermatic/kubermatic/issues/6216))




## [v2.15.3](https://github.com/kubermatic/dashboard/releases/tag/v2.15.3)

### Misc

- Allow to customize the Docker image tag for Cluster Addons ([#6106](https://github.com/kubermatic/kubermatic/pull/6106))
- Allow to disable creation of certificates for IAP deployments ([#6126](https://github.com/kubermatic/kubermatic/pull/6126))
- Restore list of nodes in the cluster view ([#2774](https://github.com/kubermatic/dashboard/issues/2774))




## [v2.15.2](https://github.com/kubermatic/dashboard/releases/tag/v2.15.2)

### Misc

- Support User SSH Keys in KubeAdm cloud provider ([#2752](https://github.com/kubermatic/dashboard/issues/2752))




## [v2.15.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.15.1)

### Misc

- Ship image-loader as part of GitHub releases ([#6063](https://github.com/kubermatic/kubermatic/issues/6063))
- Add support for KubermaticConfiguration in image-loader utility ([#6063](https://github.com/kubermatic/kubermatic/issues/6063))
- Improve Helm error handling in KKP Installer ([#6076](https://github.com/kubermatic/kubermatic/issues/6076))




## [v2.15.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.15.0)

Before upgrading, make sure to read the [general upgrade guidelines](https://docs.kubermatic.com/kubermatic/v2.15/upgrading/guidelines/)
as well as the [2.15 upgrade notes](https://docs.kubermatic.com/kubermatic/v2.15/upgrading/2.14_to_2.15/).

### Supported Kubernetes Versions

* 1.17.9
* 1.17.11
* 1.17.12
* 1.17.13
* 1.18.6
* 1.18.8
* 1.18.10
* 1.19.0
* 1.19.2
* 1.19.3

### Highlights

- Add support for Kubernetes 1.19, drop Kubernetes 1.15 and 1.16 ([#5794](https://github.com/kubermatic/kubermatic/issues/5794), [#6032](https://github.com/kubermatic/kubermatic/issues/6032))
- Add dynamic datacenter configuration to KKP Dashboard ([#2333](https://github.com/kubermatic/dashboard/issues/2333), [#2353](https://github.com/kubermatic/dashboard/issues/2353), [#5501](https://github.com/kubermatic/kubermatic/issues/5501), [#5551](https://github.com/kubermatic/kubermatic/issues/5551))
- Add preliminary support for managing external clusters ([#2608](https://github.com/kubermatic/dashboard/issues/2608), [#5689](https://github.com/kubermatic/kubermatic/issues/5689), [#5720](https://github.com/kubermatic/kubermatic/issues/5720), [#5753](https://github.com/kubermatic/kubermatic/issues/5753), [#5757](https://github.com/kubermatic/kubermatic/issues/5757), [#5772](https://github.com/kubermatic/kubermatic/issues/5772), [#5783](https://github.com/kubermatic/kubermatic/issues/5783), [#5796](https://github.com/kubermatic/kubermatic/issues/5796), [#5798](https://github.com/kubermatic/kubermatic/issues/5798), [#5802](https://github.com/kubermatic/kubermatic/issues/5802), [#5809](https://github.com/kubermatic/kubermatic/issues/5809), [#5819](https://github.com/kubermatic/kubermatic/issues/5819))
- It's now possible to enable PodSecurityPolicy on a datacenter level ([#5351](https://github.com/kubermatic/kubermatic/issues/5351))
- Add Flatcar Linux ([#5368](https://github.com/kubermatic/kubermatic/issues/5368))
- The `kubermatic` Helm chart has been deprecated in favor of the KKP Operator ([#447](https://github.com/kubermatic/docs/pull/447))
- Add `kubermatic-installer` to aid in installation/upgrades of KKP setups (preview) ([#442](https://github.com/kubermatic/docs/pull/442))
- Archives with Installer and Helm charts for KKP releases are published on GitHub ([#5580](https://github.com/kubermatic/kubermatic/issues/5580))
- Add configurable etcd cluster size to user cluster spec ([#5571](https://github.com/kubermatic/kubermatic/issues/5571), [#5710](https://github.com/kubermatic/kubermatic/issues/5710), [#5761](https://github.com/kubermatic/kubermatic/issues/5761))
- Use Go Modules, update to Go 1.15 ([#5723](https://github.com/kubermatic/kubermatic/issues/5723), [#5728](https://github.com/kubermatic/kubermatic/issues/5728), [#5834](https://github.com/kubermatic/kubermatic/issues/5834))

### Breaking Changes

- ACTION REQUIRED: Seed API changes. Seed now contains a map of Datacenters instead of SeedDatacenters to be aligned with the Datacenter API ([#5487](https://github.com/kubermatic/kubermatic/issues/5487))
- ACTION REQUIRED: Change CRD handling for cert-manager, Velero ([#5552](https://github.com/kubermatic/kubermatic/issues/5552), [#5553](https://github.com/kubermatic/kubermatic/issues/5553))
- ACTION REQUIRED: Enable Prometheus WAL compression by default ([#5781](https://github.com/kubermatic/kubermatic/issues/5781))
- ACTION REQUIRED: Promtail labelling has changed, please see upgrade notes for further information ([#5504](https://github.com/kubermatic/kubermatic/issues/5504))
- ACTION REQUIRED: Default credentials for Grafana/Minio have been removed. If you never configured credentials, refer to the upgrade notes ([#5509](https://github.com/kubermatic/kubermatic/issues/5509))
- ACTION REQUIRED: Grafana credentials in Helm values are not base64-encoded anymore ([#5509](https://github.com/kubermatic/kubermatic/issues/5509))
- ACTION REQUIRED: 2.15 EE releases will no longer be published in the github.com/kubermatic/kubermatic-installer repository, but on GitHub.

### Misc

- Add ExternalCuster to RBAC controller to generate a proper set of RBAC Role/Binding ([#5715](https://github.com/kubermatic/kubermatic/issues/5715))
- Add KubermaticAddonTakesTooLongToReconcile alert ([#5705](https://github.com/kubermatic/kubermatic/issues/5705))
- Add NoSchedule and NoExecute tolerations to usersshkey DaemonSet ([#5725](https://github.com/kubermatic/kubermatic/issues/5725))
- Add a websocket user self-watch endpoint ([#5604](https://github.com/kubermatic/kubermatic/issues/5604))
- Add configurable time window for coreos-operator node reboots ([#5318](https://github.com/kubermatic/kubermatic/issues/5318))
- Add custom image property to GCP clusters ([#5315](https://github.com/kubermatic/kubermatic/issues/5315))
- Add endpoint to list openstack availability zones ([#5535](https://github.com/kubermatic/kubermatic/issues/5535))
- Add image ID property to Azure clusters ([#5315](https://github.com/kubermatic/kubermatic/issues/5315))
- Add `MaximumParallelReconciles` option to KubermaticConfiguration ([#6002](https://github.com/kubermatic/kubermatic/issues/6002))
- Add missing CSI DaemonSet on Flatcar ([#5698](https://github.com/kubermatic/kubermatic/issues/5698))
- Add new logout endpoint: `POST /api/v1/me/logout` ([#5540](https://github.com/kubermatic/kubermatic/issues/5540))
- Add new v2 endpoint for cluster creation: `POST /api/v2/projects/{project_id}/clusters` ([#5635](https://github.com/kubermatic/kubermatic/issues/5635))
- Add new v2 endpoint to delete clusters: `DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}` ([#5666](https://github.com/kubermatic/kubermatic/issues/5666))
- Add new v2 endpoint to get cluster events: `GET /api/v2/projects/{project_id}/clusters/{cluster_id}/events` ([#5862](https://github.com/kubermatic/kubermatic/issues/5862))
- Add new v2 endpoint to patch cluster: `PATCH /api/v2/projects/{project_id}/clusters/{cluster_id}` ([#5677](https://github.com/kubermatic/kubermatic/issues/5677))
- Add new v2 endpoints to get/list cluster for the project: `GET /api/v2/projects/{project_id}/clusters`, `GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}` ([#5652](https://github.com/kubermatic/kubermatic/issues/5652))
- Add open-policy-agent gatekeeper as an optional Addon ([#5441](https://github.com/kubermatic/kubermatic/issues/5441))
- Add option to limit project creation ([#5579](https://github.com/kubermatic/kubermatic/issues/5579), [#5590](https://github.com/kubermatic/kubermatic/issues/5590))
- Allow admin to create cluster and node deployment ([#5373](https://github.com/kubermatic/kubermatic/issues/5373))
- Allow admin to manage SSH keys for any project ([#5330](https://github.com/kubermatic/kubermatic/issues/5330))
- Allow admin to manage members for any project ([#5319](https://github.com/kubermatic/kubermatic/issues/5319))
- Allow controlling external cluster functionality with global settings ([#5912](https://github.com/kubermatic/kubermatic/issues/5912))
- Allow custom envvar definitions for dex to be passed via the oauth chart ([#5829](https://github.com/kubermatic/kubermatic/issues/5829))
- Always mount CABundle for Dex into the kubermatic-api, even when `OIDCKubeCfgEndpoint` is disabled ([#5968](https://github.com/kubermatic/kubermatic/issues/5968))
- Bugfix: implement missing annotation syncing from nodeport settings in Seed CRD to the created LoadBalancer service ([#5730](https://github.com/kubermatic/kubermatic/issues/5730))
- Create an hourly schedule Velero backup for all namespaces and cluster resources ([#5327](https://github.com/kubermatic/kubermatic/issues/5327))
- Docker image size was reduced by removing development binaries ([#5586](https://github.com/kubermatic/kubermatic/issues/5586))
- Existing default addons will be updated if their definition is changed ([#5432](https://github.com/kubermatic/kubermatic/issues/5432))
- Fake "seed" datacenters will not be returned in the results of datacenter API lists ([#5562](https://github.com/kubermatic/kubermatic/issues/5562))
- Fallback to in-tree cloud provider when OpenStack provider is used with Open Telekom Cloud ([#5778](https://github.com/kubermatic/kubermatic/issues/5778))
- Fix KKP Operator not to specify unsupported `dynamic-datacenter` flag in CE mode ([#5603](https://github.com/kubermatic/kubermatic/issues/5603))
- Fix Operator not properly reconciling Vertical Pod Autoscaler and Webhooks ([#5853](https://github.com/kubermatic/kubermatic/issues/5853))
- Fix Seed validation for Community Edition ([#5611](https://github.com/kubermatic/kubermatic/issues/5611))
- Fix componentsOverride of a cluster affecting other clusters ([#5702](https://github.com/kubermatic/kubermatic/issues/5702))
- Fix master-controller-manager being too verbose ([#5889](https://github.com/kubermatic/kubermatic/issues/5889))
- Fix missing logo in Dex login screens ([#6019](https://github.com/kubermatic/kubermatic/issues/6019))
- Fix nodes sometimes not having the correct distribution label applied ([#5437](https://github.com/kubermatic/kubermatic/issues/5437))
- Fix overflowing `kubermatic.io/cleaned-up-loadbalancers` annotation on Cluster objects ([#5744](https://github.com/kubermatic/kubermatic/issues/5744))
- Fix the KubeClientCertificateExpiration Prometheus Alert which did not alert in for expiring certificates ([#5737](https://github.com/kubermatic/kubermatic/issues/5737))
- Make imagePullSecret optional for Kubermatic Operator ([#5874](https://github.com/kubermatic/kubermatic/issues/5874))
- Openstack: fixed a bug preventing the usage of pre-existing subnets connected to distributed routers ([#5334](https://github.com/kubermatic/kubermatic/issues/5334))
- Prometheus scraping annotations have been normalized to use `prometheus.io/` as their prefix. Old annotations (`kubermatic/`) are still supported, but deprecated ([#5498](https://github.com/kubermatic/kubermatic/issues/5498))
- Re-enable plugin initContainers for Velero ([#5718](https://github.com/kubermatic/kubermatic/issues/5718))
- Remove unused apiserver internal service ([#5621](https://github.com/kubermatic/kubermatic/issues/5621))
- Replace Hyperkube image with respective k8s.gcr.io images ([#5758](https://github.com/kubermatic/kubermatic/issues/5758))
- Replace deprecated Keycloak-Gatekeeper with OAuth2-Proxy ([#5777](https://github.com/kubermatic/kubermatic/issues/5777))
- Restrict project creation for nonadmin users ([#5613](https://github.com/kubermatic/kubermatic/issues/5613))
- Set clusterName parameter in cinder csi addon to the Kubermatic cluster id ([#5323](https://github.com/kubermatic/kubermatic/issues/5323))
- Support datastore cluster and default datastore ([#5399](https://github.com/kubermatic/kubermatic/issues/5399))
- build scripts: target docker repository configurable ([#5547](https://github.com/kubermatic/kubermatic/issues/5547))
- kube-dns is no longer a manifest addon ([#5370](https://github.com/kubermatic/kubermatic/issues/5370))
- local-node-dns-cache is no longer a manifest addon ([#5387](https://github.com/kubermatic/kubermatic/issues/5387))

### Dashboard

- Add flag "continuouslyReconcile" to addons ([#2618](https://github.com/kubermatic/dashboard/issues/2618))
- Add option to enable/disable external cluster import feature from admin settings ([#2644](https://github.com/kubermatic/dashboard/issues/2644))
- Add option to restrict project creation to admins via the Admin Settings ([#2617](https://github.com/kubermatic/dashboard/issues/2617))
- Add option to set project limit for users from the admin panel ([#2463](https://github.com/kubermatic/dashboard/issues/2463))
- Add possibility to specify Availability Zones for Openstack via UI ([#2402](https://github.com/kubermatic/dashboard/issues/2402))
- Align color of the icons in the footer with the text ([#2459](https://github.com/kubermatic/dashboard/issues/2459))
- Any changes related to the user and his settings will be immediately visible in the app as user data is downloaded through WebSocket ([#2476](https://github.com/kubermatic/dashboard/issues/2476))
- Cheapest/smallest provider instance will be chosen by default in the wizard ([#2541](https://github.com/kubermatic/dashboard/issues/2541))
- Dashboard now supports end-of-life configuration for Kubernetes versions. All versions up to defined version will be marked as deprecated or soon-to-be deprecated in the UI. It can be defined in the config.json file as follows: `..."end_of_life": {"1.17.9":"2020-07-01", "1.18.0":"2020-09-15"}` ([#2520](https://github.com/kubermatic/dashboard/issues/2520))
- Enable Flatcar Linux for Openstack ([#2430](https://github.com/kubermatic/dashboard/issues/2430))
- Exclude enterprise edition files from community edition builds ([#2550](https://github.com/kubermatic/dashboard/issues/2550))
- Fix loading of the access rights in the SSH keys view ([#2645](https://github.com/kubermatic/dashboard/issues/2645))
- Fix notification display in the notification panel ([#2443](https://github.com/kubermatic/dashboard/issues/2443))
- Move "Show All Projects" toggle back to the project list ([#2558](https://github.com/kubermatic/dashboard/issues/2558))
- Non-existing default projects will be now unchecked in the settings ([#2630](https://github.com/kubermatic/dashboard/issues/2630))
- Remove list of orphaned nodes from cluster view as only deployments are used right now ([#2574](https://github.com/kubermatic/dashboard/issues/2574))
- Update user panel styling ([#2399](https://github.com/kubermatic/dashboard/issues/2399))
- Use new field `AdmissionPlugins` in cluster struct, instead of `UsePodSecurityPolicyAdmissionPlugin` & `UsePodNodeSelectorAdmissionPlugin` ([#2405](https://github.com/kubermatic/dashboard/issues/2405))
- Use the new wizard and node data ([#2481](https://github.com/kubermatic/dashboard/issues/2481))

### Updates

- Alertmanager 0.21.0 ([#5786](https://github.com/kubermatic/kubermatic/issues/5786))
- Blackbox Exporter 0.17.0 ([#5787](https://github.com/kubermatic/kubermatic/issues/5787))
- cert-manager 0.16.1 ([#5776](https://github.com/kubermatic/kubermatic/issues/5776))
- Dex v0.24.0 ([#5506](https://github.com/kubermatic/kubermatic/issues/5506))
- Grafana Loki 1.6.1 ([#5779](https://github.com/kubermatic/kubermatic/issues/5779))
- Grafana 7.1.5 ([#5788](https://github.com/kubermatic/kubermatic/issues/5788))
- Karma v0.68 ([#5789](https://github.com/kubermatic/kubermatic/issues/5789))
- kube-state-metrics v1.9.7 ([#5790](https://github.com/kubermatic/kubermatic/issues/5790))
- Kubernetes Dashboard v2.0.4 ([#5820](https://github.com/kubermatic/kubermatic/issues/5820))
- machine-controller 1.19.0 ([#5794](https://github.com/kubermatic/kubermatic/issues/5794))
- nginx-ingress-controller 0.34.1 ([#5780](https://github.com/kubermatic/kubermatic/issues/5780))
- node-exporter 1.0.1 (includes the usercluster addon) ([#5791](https://github.com/kubermatic/kubermatic/issues/5791))
- Minio RELEASE.2020-09-10T22-02-45Z ([#5854](https://github.com/kubermatic/kubermatic/issues/5854))
- Prometheus 2.20.1 ([#5781](https://github.com/kubermatic/kubermatic/issues/5781))
- Velero 1.4.2 ([#5775](https://github.com/kubermatic/kubermatic/issues/5775))

### Changes since v2.15.0-rc.1

- Add Kubernetes 1.16.15, 1.17.12, 1.19.2 ([#5927](https://github.com/kubermatic/kubermatic/issues/5927))
- Add `operator.kubermatic.io/skip-reconciling` annotation to Seeds to allow step-by-step seed cluster upgrades ([#5883](https://github.com/kubermatic/kubermatic/issues/5883))
- Add option to enable/disable external cluster import feature from admin settings in KKP dashboard ([#2644](https://github.com/kubermatic/dashboard/issues/2644))
- Allow controlling external cluster functionality with global settings ([#5912](https://github.com/kubermatic/kubermatic/issues/5912))
- Fix KKP Operator getting stuck in Kubernetes 1.18 clusters when reconciling Ingresses ([#5915](https://github.com/kubermatic/kubermatic/issues/5915))
- Fix creation of RHEL8 machines ([#5950](https://github.com/kubermatic/kubermatic/issues/5950))
- Fix loading of the access rights in the SSH keys view ([#2645](https://github.com/kubermatic/dashboard/issues/2645))
- Fix cluster wizard rendering in Safari ([#2661](https://github.com/kubermatic/dashboard/issues/2661))

### Changes since v2.15.0-rc.2

- Add feature flag `etcdLauncher` around etcd-launcher ([#5973](https://github.com/kubermatic/kubermatic/issues/5973))
- Provide a way of skipping Certificate cert-manager resources ([#5962](https://github.com/kubermatic/kubermatic/issues/5962), [#5969](https://github.com/kubermatic/kubermatic/issues/5969))

### Changes since v2.15.0-rc.3

- Add Kubernetes 1.17.13, 1.18.10, 1.19.3, Remove Kubernetes 1.16 ([#5927](https://github.com/kubermatic/kubermatic/issues/5927))
- Always mount CABundle for Dex into the kubermatic-api, even when `OIDCKubeCfgEndpoint` is disabled ([#5968](https://github.com/kubermatic/kubermatic/issues/5968))
- Add a new feature gate to the seed-controller to enable etcd-launcher for all user clusters ([#5997](https://github.com/kubermatic/kubermatic/issues/5997))
- Add MaximumParallelReconciles option to KubermaticConfiguration ([#6002](https://github.com/kubermatic/kubermatic/issues/6002))
- Fix missing logo in Dex login screens ([#6019](https://github.com/kubermatic/kubermatic/issues/6019))
- Bump machine-controller to v1.19.0 ([#6050](https://github.com/kubermatic/kubermatic/issues/6050))




# Kubermatic 2.14

## [v2.14.13](https://github.com/kubermatic/kubermatic/releases/tag/v2.14.13)

### Bugfixes

- Node-local-dns is now using UDP for external queries ([#6840](https://github.com/kubermatic/kubermatic/issues/6840))




## [v2.14.12](https://github.com/kubermatic/kubermatic/releases/tag/v2.14.12)

### Bugfixes

- Fix cert-manager validating webhook ([#6743](https://github.com/kubermatic/kubermatic/issues/6743))




## [v2.14.11](https://github.com/kubermatic/kubermatic/releases/tag/v2.14.11)

### Misc

- Provide the possibility of configuring leader election parameters for user cluster components ([#6641](https://github.com/kubermatic/kubermatic/pull/6641))




## [v2.14.10](https://github.com/kubermatic/kubermatic/releases/tag/v2.14.10)

### Misc

- [ATTN] Fix orphaned apiserver-is-running initContainers in usercluster controlplane. This can cause a short reconciliation burst to bring older usercluster resources in all Seed clusters up to date. Tune the maxReconcileLimit if needed ([#6335](https://github.com/kubermatic/kubermatic/issues/6335))
- Add option to specify Pod Node Selector Configuration ([#2961](https://github.com/kubermatic/dashboard/issues/2961))
- Extend Cluster CRD for PodNodeSelectorAdmissionPluginConfig ([#6401](https://github.com/kubermatic/kubermatic/issues/6401))
- Add admission control configuration for the user cluster API deployment ([#6418](https://github.com/kubermatic/kubermatic/issues/6418))




## [v2.14.9](https://github.com/kubermatic/kubermatic/releases/tag/v2.14.9)

### Bugfixes

- Create an RBAC role to allow kubeadm to get nodes. This fixes nodes failing to join kubeadm clusters running Kubernetes 1.18+ ([#6241](https://github.com/kubermatic/kubermatic/issues/6241))




## [v2.14.8](https://github.com/kubermatic/kubermatic/releases/tag/v2.14.8)

### Misc

- Ship image-loader as part of GitHub releases ([#6096](https://github.com/kubermatic/kubermatic/issues/6096))
- Add support for KubermaticConfiguration in image-loader utility ([#6071](https://github.com/kubermatic/kubermatic/issues/6071))

## [v2.14.7](https://github.com/kubermatic/kubermatic/releases/tag/v2.14.7)

This release includes an important change to the Docker registry used for fetching the Kubernetes control plane
components. The change will require that all user-clusters reconcile their control planes, which can cause
significant load on the seed clusters. Refer to the
[general upgrade guidelines](https://docs.kubermatic.com/kubermatic/master/upgrading/guidelines/) for more
information on how to limit the impact of such changes during KKP upgrades.

### Misc

- ACTION REQUIRED: Migrate from google_containers to k8s.gcr.io Docker registry ([#5986](https://github.com/kubermatic/kubermatic/issues/5986))




## v2.14.6

### Bugfixes

- Fix creation of RHEL8 machines ([#5951](https://github.com/kubermatic/kubermatic/issues/5951))

### Misc

- Allow custom envvar definitions for Dex to be passed via the `oauth` chart, key in `values.yaml` is `dex.env` ([#5847](https://github.com/kubermatic/kubermatic/issues/5847))
- Provide a way of skipping Certificate cert-manager resources in `oauth` and `kubermatic` charts ([#5972](https://github.com/kubermatic/kubermatic/issues/5972))




## v2.14.5

### Bugfixes

- Fallback to in-tree cloud provider for OTC ([#5778](https://github.com/kubermatic/kubermatic/issues/5778))




## v2.14.4

### Bugfixes

- fix flaky TestGCPDiskTypes ([#5693](https://github.com/kubermatic/kubermatic/issues/5693))
- fix changing user/password for OpenStack cluster credentials ([#5691](https://github.com/kubermatic/kubermatic/issues/5691))
- fix componentsOverride affecting default values when reconciling clusters ([#5704](https://github.com/kubermatic/kubermatic/issues/5704))
- fix typo in prometheus chart ([#5726](https://github.com/kubermatic/kubermatic/issues/5726))

### Misc

- Allow to configure Velero plugin InitContainers ([#5718](https://github.com/kubermatic/kubermatic/issues/5718), [#5719](https://github.com/kubermatic/kubermatic/issues/5719))
- addons/csi: add nodeplugin for flatcar linux ([#5701](https://github.com/kubermatic/kubermatic/issues/5701))




## v2.14.3

- Added Kubernetes v1.16.13, and removed v1.16.2-9 in default version configuration ([#5659](https://github.com/kubermatic/kubermatic/issues/5659))
- Added Kubernetes v1.17.9, and removed v1.17.0-5 in default version configuration ([#5664](https://github.com/kubermatic/kubermatic/issues/5664))
- Added Kubernetes v1.18.6, and removed v1.18.2 in default version configuration ([#5673](https://github.com/kubermatic/kubermatic/issues/5673))




## v2.14.2

### Bugfixes

- Fix Kubermatic operator not to specify unsupported `dynamic-datacenter` flag in CE mode. ([#5615](https://github.com/kubermatic/kubermatic/issues/5615))
- Fix Seed validation for Community Edition. ([#5619](https://github.com/kubermatic/kubermatic/issues/5619))
- Fix Subnetworks for GCP, because the network filtering was wrong. ([#5632](https://github.com/kubermatic/kubermatic/pull/5632))
- Fix label for nodeport-proxy when deployed with the operator. ([#5612](https://github.com/kubermatic/kubermatic/pull/5612))


### Misc

- Change default number of replicas for seed and master controller manager to one. ([#5620](https://github.com/kubermatic/kubermatic/issues/5620))
- Remove empty Docker secret for Kubermatic Operator CE Helm chart. ([#5618](https://github.com/kubermatic/kubermatic/pull/5618))




## v2.14.1

- Added missing Flatcar Linux handling in API ([#5368](https://github.com/kubermatic/kubermatic/issues/5368))
- Fixed nodes sometimes not having the correct distribution label applied. ([#5437](https://github.com/kubermatic/kubermatic/issues/5437))
- Fixed missing Kubermatic Prometheus metrics. ([#5505](https://github.com/kubermatic/kubermatic/issues/5505))




## v2.14.0

### Supported Kubernetes versions

- `1.15.5`
- `1.15.6`
- `1.15.7`
- `1.15.9`
- `1.15.10`
- `1.15.11`
- `1.16.2`
- `1.16.3`
- `1.16.4`
- `1.16.6`
- `1.16.7`
- `1.16.9`
- `1.17.0`
- `1.17.2`
- `1.17.3`
- `1.17.5`
- `1.18.2`

### Misc

- **ACTION REQUIRED:** The most recent backup for user clusters is kept when the cluster is deleted. Adjust the cleanup-container to get the old behaviour (delete all backups) back. ([#5262](https://github.com/kubermatic/kubermatic/pull/5262))
- **ACTION REQUIRED:** Addon manifest templating is now a stable API, but different to the old implicit data. Custom addons might need to be adjusted. ([#5275](https://github.com/kubermatic/kubermatic/issues/5275))
- Added Flatcar Linux as an Operating System option
- Added SLES as an Operating System option ([#5040](https://github.com/kubermatic/kubermatic/issues/5040))
- Audit logging can now be enforced in all clusters within a Datacenter. ([#5045](https://github.com/kubermatic/kubermatic/issues/5045))
- Added support for Kubernetes 1.18, drop support for Kubernetes < 1.15. ([#5325](https://github.com/kubermatic/kubermatic/issues/5325))
- Administrators can now manage all projects and clusters
- Added admission plugins CRD support ([#5047](https://github.com/kubermatic/kubermatic/issues/5047))
- Added configurable time window for coreos-operator node reboots ([#5318](https://github.com/kubermatic/kubermatic/issues/5318))
- Created an hourly schedule Velero backup for all namespaces and cluster resources ([#5327](https://github.com/kubermatic/kubermatic/issues/5327))
- Added support for creating RBAC bindings to group subjects ([#5237](https://github.com/kubermatic/kubermatic/issues/5237))
- Added a configuration flag for seed-controller-manager to enforce default addons on userclusters. Enabled by default. ([#5193](https://github.com/kubermatic/kubermatic/issues/5193))
- TLS certificates for Kubermatic/IAP are now not managed by a shared `certs` chart anymore, but handled individually for each Ingress. ([#5163](https://github.com/kubermatic/kubermatic/issues/5163))
- kubelet sets initial machine taints via --register-with-taints ([#664](https://github.com/kubermatic/machine-controller/issues/664))
- Implement the NodeCSRApprover controller for automatically approving node serving certificates ([#705](https://github.com/kubermatic/machine-controller/issues/705))
- Updated blackbox-exporter to v0.16.0 ([#5083](https://github.com/kubermatic/kubermatic/issues/5083))
- Updated cert-manager to 0.13.0 ([#5068](https://github.com/kubermatic/kubermatic/issues/5068))
- Updated coredns to v1.3.1 ([#5145](https://github.com/kubermatic/kubermatic/issues/5145))
- Updated Dex to v2.22.0 ([#5092](https://github.com/kubermatic/kubermatic/issues/5092))
- Updated Elastic Stack to 6.8.5 and mark it as deprecated. ([#5085](https://github.com/kubermatic/kubermatic/issues/5085))
- Updated Envoy in nodeport-proxy to v1.13.0 ([#5135](https://github.com/kubermatic/kubermatic/issues/5135))
- Updated go-swagger to support go v1.14 ([#5247](https://github.com/kubermatic/kubermatic/issues/5247))
- Updated Grafana to v6.7.1 ([#5254](https://github.com/kubermatic/kubermatic/issues/5254))
- Updated helm-exporter to v0.4.3 ([#5113](https://github.com/kubermatic/kubermatic/issues/5113))
- Updated karma to v0.55 ([#5084](https://github.com/kubermatic/kubermatic/issues/5084))
- Updated Keycloak to v7.0.0 ([#5128](https://github.com/kubermatic/kubermatic/issues/5128))
- Updated Kube-state-metrics to v1.9.5 ([#5139](https://github.com/kubermatic/kubermatic/issues/5139))
- Updated Loki to v1.3.0 ([#5081](https://github.com/kubermatic/kubermatic/issues/5081))
- Updated machine-controller to v1.13.2 ([#5349](https://github.com/kubermatic/kubermatic/issues/5349))
- Updated metrics-server to v0.3.6 ([#5140](https://github.com/kubermatic/kubermatic/issues/5140))
- Updated nginx-ingress-controller to v0.29 ([#5134](https://github.com/kubermatic/kubermatic/issues/5134))
- Updated openvpn to 2.4.8 ([#5144](https://github.com/kubermatic/kubermatic/issues/5144))
- Updated Prometheus to v2.17.1 on user cluster ([#5273](https://github.com/kubermatic/kubermatic/issues/5273))
- Updated Thanos to v0.11.0 ([#5176](https://github.com/kubermatic/kubermatic/issues/5176))
- Updated Velero to v1.3.2 ([#5326](https://github.com/kubermatic/kubermatic/issues/5326))

### Dashboard

- Added a dark theme and a selector to the user settings. ([#1867](https://github.com/kubermatic/dashboard-v2/issues/1867))
- Added possibility to define a default project in user settings. When a default project is chosen, the user will be automatically redirected to this project after login. Attention: One initial log in might be needed for the feature to take effect. ([#1895](https://github.com/kubermatic/dashboard-v2/issues/1895))
- Added UI support for dynamic kubelet config option ([#1923](https://github.com/kubermatic/dashboard-v2/issues/1923))
- Added paginators to all tables ([#1932](https://github.com/kubermatic/dashboard-v2/issues/1932))
- Added cluster metrics. ([#1940](https://github.com/kubermatic/dashboard-v2/issues/1940))
- Increased cpu & memory defaults on vSphere ([#1952](https://github.com/kubermatic/dashboard-v2/issues/1952))
- Custom Presets are filtered by datacenter now ([#1955](https://github.com/kubermatic/dashboard-v2/issues/1955))
- Added notification panel. ([#1957](https://github.com/kubermatic/dashboard-v2/issues/1957))
- Added Pod Node Selector field. ([#1968](https://github.com/kubermatic/dashboard-v2/issues/1968))
- Operation Systems on VSphere for which no template is specified in datacenters are now hidden. ([#1981](https://github.com/kubermatic/dashboard-v2/issues/1981))
- Fixes issue that prevented creating Addons which had no AddonConfig deployed. ([#1985](https://github.com/kubermatic/dashboard-v2/issues/1985))
- Added possibility to collapse the sidenav. ([#2004](https://github.com/kubermatic/dashboard-v2/issues/2004))
- We now use WebSocket to get global settings. ([#2008](https://github.com/kubermatic/dashboard-v2/issues/2008))
- We now use `SameSite=Lax` ([#2046](https://github.com/kubermatic/dashboard-v2/issues/2046))
- AddonConfig's shortDescription field is now used in the accessible addons overview. ([#2050](https://github.com/kubermatic/dashboard-v2/issues/2050))
- Audit Logging will be enforced when specified in the datacenter. ([#2070](https://github.com/kubermatic/dashboard-v2/issues/2070))
- Added the option to use an OIDC provider for the kubeconfig download. ([#2076](https://github.com/kubermatic/dashboard-v2/issues/2076))
- Added support for creating RBAC bindings to group subjects ([#2123](https://github.com/kubermatic/dashboard-v2/issues/2123))
- Fixed custom links display on the frontpage. ([#2134](https://github.com/kubermatic/dashboard-v2/issues/2134))
- Moved project selector to the navigation bar. Redesigned the sidebar menu. ([#2144](https://github.com/kubermatic/dashboard-v2/issues/2144))
- Fixed missing pagination issue in the project list view. ([#2177](https://github.com/kubermatic/dashboard-v2/issues/2177))
- Added possibility to specify imageID for Azure node deployments (required for RHEL).
- Added possibility to specify customImage for GCP node deployments (required for RHEL). ([#2190](https://github.com/kubermatic/dashboard-v2/issues/2190))
- Fixed user settings layout on the smaller screens. ([#2209](https://github.com/kubermatic/dashboard-v2/issues/2209))
- Fixed loading Openstack flavors in add/edit node deployment dialog ([#2222](https://github.com/kubermatic/dashboard-v2/issues/2222))
- Fixed filter in combo dropdown ([#2238](https://github.com/kubermatic/dashboard-v2/issues/2238))
- Fixed node data dialog for vSphere clusters. ([#2251](https://github.com/kubermatic/dashboard-v2/issues/2251))
- Cluster creation time is now visible in the UI. ([#2253](https://github.com/kubermatic/dashboard-v2/issues/2253))
- Added info about end-of-life of Container Linux ([#2264](https://github.com/kubermatic/dashboard-v2/issues/2264))
- Enforcing pod security policy by the datacenter is now allowed. ([#2270](https://github.com/kubermatic/dashboard-v2/issues/2270))
- Introduced a number of responsiveness fixes to improve user experience on the smaller screens. ([#2279](https://github.com/kubermatic/dashboard-v2/issues/2279))

### Cloud providers

- Added Alibaba cloud ([#5107](https://github.com/kubermatic/kubermatic/issues/5107))
- Azure: Added image ID property to clusters. ([#5315](https://github.com/kubermatic/kubermatic/issues/5315))
- Azure: Added multiple availability zones support ([#2280](https://github.com/kubermatic/dashboard-v2/issues/2280))
- Azure: Added support for configurable OS and Data disk sizes ([#5156](https://github.com/kubermatic/kubermatic/issues/5156))
- Digitalocean: Fixed and issue when there are more than 200 droplets in the same account. ([#692](https://github.com/kubermatic/machine-controller/issues/692))
- GCP: Added custom image property to clusters.
- GCP: Subnetworks are now fetched from API ([#1950](https://github.com/kubermatic/dashboard-v2/issues/1950))
- Openstack: fixed a bug preventing the usage of pre-existing subnets connected to distributed routers ([#5334](https://github.com/kubermatic/kubermatic/issues/5334))
- vSphere: datastore clusters can now be specified for VMs instead of singular datastores ([#671](https://github.com/kubermatic/machine-controller/issues/671))
- vSphere: Added ResourcePool support ([#726](https://github.com/kubermatic/machine-controller/issues/726))

### Monitoring

- Grafana Loki replaces the ELK logging stack. ([#5164](https://github.com/kubermatic/kubermatic/issues/5164))

### Bugfixes

- Fix bad apiserver Deployments when no Dex CA was configured. ([#5087](https://github.com/kubermatic/kubermatic/issues/5087))
- Fixed cluster credential Secrets not being reconciled properly. ([#5197](https://github.com/kubermatic/kubermatic/issues/5197))
- Fixed swagger and API client for ssh key creation. ([#5069](https://github.com/kubermatic/kubermatic/issues/5069))
- Fixed seed-proxy controller not being triggered. ([#5101](https://github.com/kubermatic/kubermatic/issues/5101))
- Fixed a bug in Kubernetes 1.17 on CoreOS that prevented the Kubelet from starting ([#658](https://github.com/kubermatic/machine-controller/issues/658))




# Kubermatic 2.13

## [v2.13.10](https://github.com/kubermatic/kubermatic/releases/tag/v2.13.10)

### Misc

- Improve image-loader usability, add support for Helm charts ([#6090](https://github.com/kubermatic/kubermatic/issues/6090))

## [v2.13.9](https://github.com/kubermatic/kubermatic/releases/tag/v2.13.9)

### Misc

- Add a configuration flag for seed-controller-manager to enforce default addons on userclusters. Disabled by default ([#5987](https://github.com/kubermatic/kubermatic/issues/5987))




## v2.13.8

### Bugfixes

- Fix `componentsOverride` of a cluster affecting other clusters ([#5702](https://github.com/kubermatic/kubermatic/issues/5702))




## v2.13.7

- Added Kubernetes v1.16.13, and removed v1.16.2-7 in default version configuration ([#5661](https://github.com/kubermatic/kubermatic/issues/5661))
- Added Kubernetes v1.17.9, and removed v1.17.0-3 in default version configuration ([#5667](https://github.com/kubermatic/kubermatic/issues/5667))




## v2.13.6

- Fixed a bug preventing editing of existing cluster credential secrets ([#5569](https://github.com/kubermatic/kubermatic/issues/5569))




## v2.13.5

- Updated machine-controller to v1.10.4 to address issue in CNI plugins ([#5443](https://github.com/kubermatic/kubermatic/issues/5443))




## v2.13.4

- **ACTION REQUIRED:** The most recent backup for user clusters is kept when the cluster is deleted. Adjust the cleanup-container to get the old behaviour (delete all backups) back. ([#5262](https://github.com/kubermatic/kubermatic/issues/5262))
- Updated machine-controller to v1.10.3 to fix the Docker daemon/CLI version incompatibility ([#5427](https://github.com/kubermatic/kubermatic/issues/5427))




## v2.13.3

This release contains only improvements to the image build process.




## v2.13.2

- Openstack: include distributed routers in existing router search ([#5334](https://github.com/kubermatic/kubermatic/issues/5334))




## v2.13.1

- Fixed swagger and API client for ssh key creation. ([#5069](https://github.com/kubermatic/kubermatic/issues/5069))
- Added Kubernetes v1.15.10, v1.16.7, v1.17.3 ([#5102](https://github.com/kubermatic/kubermatic/issues/5102))
- AddonConfig's shortDescription field is now used in the accessible addons overview. ([#2050](https://github.com/kubermatic/dashboard/issues/2050))




## v2.13.0

### Supported Kubernetes versions

- `1.15.5`
- `1.15.6`
- `1.15.7`
- `1.15.9`
- `1.16.2`
- `1.16.3`
- `1.16.4`
- `1.16.6`
- `1.17.0`
- `1.17.2`
- Openshift `v4.1.18`

### Major changes

- End-of-Life Kubernetes v1.14 is no longer supported. ([#4987](https://github.com/kubermatic/kubermatic/issues/4987))
- The `authorized_keys` files on nodes are now updated whenever the SSH keys for a cluster are changed ([#4531](https://github.com/kubermatic/kubermatic/issues/4531))
- Added support for custom CA for OpenID provider in Kubermatic API. ([#4994](https://github.com/kubermatic/kubermatic/issues/4994))
- Added user settings panel. ([#1738](https://github.com/kubermatic/dashboard/issues/1738))
- Added cluster addon UI
- MachineDeployments can now be configured to enable dynamic kubelet config ([#4946](https://github.com/kubermatic/kubermatic/issues/4946))
- Added RBAC management functionality to UI ([#1815](https://github.com/kubermatic/dashboard/issues/1815))
- Added RedHat Enterprise Linux as an OS option ([#669](https://github.com/kubermatic/machine-controller/issues/669))
- Added SUSE Linux Enterprise Server as an OS option ([#659](https://github.com/kubermatic/machine-controller/issues/659))

### Cloud providers

- Openstack: A bug that caused cluster reconciliation to fail if the controller crashed at the wrong time was fixed ([#4754](https://github.com/kubermatic/kubermatic/issues/4754)
- Openstack: New Kubernetes 1.16+ clusters use the external Cloud Controller Manager and CSI by default ([#4756](https://github.com/kubermatic/kubermatic/issues/4756))
- vSphere: Fixed a bug that resulted in a faulty cloud config when using a non-default port ([#4562](https://github.com/kubermatic/kubermatic/issues/4562))
- vSphere: Fixed a bug which cased custom VM folder paths not to be put in cloud-configs ([#4737](https://github.com/kubermatic/kubermatic/issues/4737))
- vSphere: The robustness of machine reconciliation has been improved. ([#4651](https://github.com/kubermatic/kubermatic/issues/4651))
- vSphere: Added support for datastore clusters (#671)
- Azure: Node sizes are displayed in size dropdown when creating/updating a node deployment ([#1908](https://github.com/kubermatic/dashboard/issues/1908))
- GCP: Networks are fetched from API now ([#1913](https://github.com/kubermatic/dashboard/issues/1913))

### Bugfixes

- Fixed parsing Kibana's logs in Fluent-Bit ([#4544](https://github.com/kubermatic/kubermatic/issues/4544))
- Fixed master-controller failing to create project-label-synchronizer controllers. ([#4577](https://github.com/kubermatic/kubermatic/issues/4577))
- Fixed broken NodePort-Proxy for user clusters with LoadBalancer expose strategy. ([#4590](https://github.com/kubermatic/kubermatic/issues/4590))
- Fixed cluster namespaces being stuck in Terminating state when deleting a cluster. ([#4619](https://github.com/kubermatic/kubermatic/issues/4619))
- Fixed Seed Validation Webhook rejecting new Seeds in certain situations ([#4662](https://github.com/kubermatic/kubermatic/issues/4662))
- A panic that could occur on clusters that lack both credentials and a credentialsSecret was fixed. ([#4742](https://github.com/kubermatic/kubermatic/issues/4742))
- A bug that occasionally resulted in a `Error: no matches for kind "MachineDeployment" in version "cluster.k8s.io/v1alpha1"` visible in the UI was fixed. ([#4870](https://github.com/kubermatic/kubermatic/issues/4870))
- A memory leak in the port-forwarding of the Kubernetes dashboard and Openshift console endpoints was fixed ([#4879](https://github.com/kubermatic/kubermatic/issues/4879))
- Fixed a bug that could result in 403 errors during cluster creation when using the BringYourOwn provider ([#4892](https://github.com/kubermatic/kubermatic/issues/4892))
- Fixed a bug that prevented clusters in working seeds from being listed in the dashboard if any other seed was unreachable. ([#4961](https://github.com/kubermatic/kubermatic/issues/4961))
- Prevented removing system labels during cluster edit ([#4986](https://github.com/kubermatic/kubermatic/issues/4986))
- Fixed FluentbitManyRetries Prometheus alert being too sensitive to harmless backpressure. ([#5011](https://github.com/kubermatic/kubermatic/issues/5011))
- Fixed deleting user-selectable addons from clusters. ([#5022](https://github.com/kubermatic/kubermatic/issues/5022))
- Fixed node name validation while creating clusters and node deployments ([#1783](https://github.com/kubermatic/dashboard/issues/1783))

### UI

- **ACTION REQUIRED:** Added logos and descriptions for the addons. In order to see the logos and descriptions addons have to be configured with AddonConfig CRDs with the same names as addons. ([#1824](https://github.com/kubermatic/dashboard/issues/1824))
- **ACTION REQUIRED:** Added application settings view. Some of the settings were moved from config map to the `KubermaticSettings` CRD. In order to use them in the UI it is required to manually update the CRD or do it from newly added UI. ([#1772](https://github.com/kubermatic/dashboard/issues/1772))
- Fixed label form validator. ([#1710](https://github.com/kubermatic/dashboard/issues/1710))
- Removed `Edit Settings` option from cluster detail view and instead combine everything under `Edit Cluster`. ([#1718](https://github.com/kubermatic/dashboard/issues/1718))
- Enabled edit options for kubeAdm ([#1735](https://github.com/kubermatic/dashboard/issues/1735))
- Switched flag proportions to 4:3. ([#1742](https://github.com/kubermatic/dashboard/issues/1742))
- Added new project view ([#1766](https://github.com/kubermatic/dashboard/issues/1766))
- Added custom links to admin settings. ([#1800](https://github.com/kubermatic/dashboard/issues/1800))
- Blocked option to edit cluster labels inherited from the project. ([#1801](https://github.com/kubermatic/dashboard/issues/1801))
- Moved pod security policy configuration to the edit cluster dialog. ([#1837](https://github.com/kubermatic/dashboard/issues/1837))
- Restyled some elements in the admin panel. ([#1850](https://github.com/kubermatic/dashboard/issues/1850))
- Added separate save indicators for custom links in the admin panel. ([#1862](https://github.com/kubermatic/dashboard/issues/1862))

### Addons

- The dashboard addon was removed as it's now deployed in the seed and can be used via its proxy endpoint ([#4567](https://github.com/kubermatic/kubermatic/issues/4567))
- Added default namespace/cluster roles for addons ([#4695](https://github.com/kubermatic/kubermatic/issues/4695))
- Introduced addon configurations. ([#4702](https://github.com/kubermatic/kubermatic/issues/4702))
- Fixed addon config get and list endpoints. ([#4734](https://github.com/kubermatic/kubermatic/issues/4734))
- Added forms for addon variables. ([#1846](https://github.com/kubermatic/dashboard/issues/1846))

### Misc

- **ACTION REQUIRED:** Updated cert-manager to 0.12.0. This requires a full reinstall of the chart. See https://cert-manager.io/docs/installation/upgrading/upgrading-0.10-0.11/ ([#4857](https://github.com/kubermatic/kubermatic/issues/4857))
- Updated Alertmanager to 0.20.0 ([#4864](https://github.com/kubermatic/kubermatic/issues/4864))
- Update Kubernetes Dashboard to v2.0.0-rc3 ([#5015](https://github.com/kubermatic/kubermatic/issues/5015))
- Updated Dex to v2.12.0 ([#4869](https://github.com/kubermatic/kubermatic/issues/4869))
- The envoy version used by the nodeport-proxy was updated to v1.12.2 ([#4865](https://github.com/kubermatic/kubermatic/issues/4865))
- Etcd was upgraded to 3.4 for 1.17+ clusters ([#4856](https://github.com/kubermatic/kubermatic/issues/4856))
- Updated Grafana to 6.5.2 ([#4858](https://github.com/kubermatic/kubermatic/issues/4858))
- Updated karma to 0.52 ([#4859](https://github.com/kubermatic/kubermatic/issues/4859))
- Updated kube-state-metrics to 1.8.0 ([#4860](https://github.com/kubermatic/kubermatic/issues/4860))
- Updated machine-controller to v1.10.0 ([#5070](https://github.com/kubermatic/kubermatic/issues/5070))
  - Added support for EBS volume encryption ([#663](https://github.com/kubermatic/machine-controller/issues/663))
  - kubelet sets initial machine taints via --register-with-taints ([#664](https://github.com/kubermatic/machine-controller/issues/664))
  - Moved deprecated kubelet flags into config file ([#667](https://github.com/kubermatic/machine-controller/issues/667))
  - Enabled swap accounting for Ubuntu deployments ([#666](https://github.com/kubermatic/machine-controller/issues/666))
- Updated nginx-ingress-controller to v0.28.0 ([#4999](https://github.com/kubermatic/kubermatic/issues/4999))
- Updated Minio to RELEASE.2019-10-12T01-39-57Z ([#4868](https://github.com/kubermatic/kubermatic/issues/4868))
- Updated Prometheus to 2.14 in Seed and User clusters ([#4684](https://github.com/kubermatic/kubermatic/issues/4684))
- Updated Thanos to 0.8.1 ([#4549](https://github.com/kubermatic/kubermatic/issues/4549))
- An email-restricted Datacenter can now have multiple email domains specified. ([#4643](https://github.com/kubermatic/kubermatic/issues/4643))
- Add fluent-bit Grafana dashboard ([#4545](https://github.com/kubermatic/kubermatic/issues/4545))
- Updated Dex page styling. ([#4632](https://github.com/kubermatic/kubermatic/issues/4632))
- Openshift: added metrics-server ([#4671](https://github.com/kubermatic/kubermatic/issues/4671))
- For new clusters, the Kubelet port 12050 is not exposed publicly anymore ([#4703](https://github.com/kubermatic/kubermatic/issues/4703))
- The cert-manager Helm chart now creates global ClusterIssuers for Let's Encrypt. ([#4732](https://github.com/kubermatic/kubermatic/issues/4732))
- Added migration for cluster user labels ([#4744](https://github.com/kubermatic/kubermatic/issues/4744))
- Fixed seed-proxy controller not working in namespaces other than `kubermatic`. ([#4775](https://github.com/kubermatic/kubermatic/issues/4775))
- The docker logs on the nodes now get rotated via the new `logrotate` addon ([#4813](https://github.com/kubermatic/kubermatic/issues/4813))
- Made node-exporter an optional addon. ([#4832](https://github.com/kubermatic/kubermatic/issues/4832))
- Added parent cluster readable name to default worker names. ([#4839](https://github.com/kubermatic/kubermatic/issues/4839))
- The QPS settings of Kubeletes can now be configured per-cluster using addon Variables ([#4854](https://github.com/kubermatic/kubermatic/issues/4854))
- Access to Kubernetes Dashboard can be now enabled/disabled by the global settings. ([#4889](https://github.com/kubermatic/kubermatic/issues/4889))
- Added support for dynamic presets ([#4903](https://github.com/kubermatic/kubermatic/issues/4903))
- Presets can now be filtered by datacenter ([#4991](https://github.com/kubermatic/kubermatic/issues/4991))
- Revoking the viewer token is possible via UI now. ([#1708](https://github.com/kubermatic/dashboard/issues/1708))




# Kubermatic 2.12

## v2.12.9

- Added Kubernetes v1.16.13, and removed v1.16.2-7 ([#5662](https://github.com/kubermatic/kubermatic/issues/5662))




## v2.12.8

- Updated machine-controller to v1.8.4 to address issue in CNI plugins ([#5442](https://github.com/kubermatic/kubermatic/issues/5442))




## v2.12.7

- Openstack: fixed a bug preventing the usage of pre-existing subnets connected to distributed routers ([#5334](https://github.com/kubermatic/kubermatic/issues/5334))
- Update machine-controller to v1.8.2 to fix the Docker daemon/CLI version incompatibility ([#5426](https://github.com/kubermatic/kubermatic/issues/5426))




## v2.12.6

### Misc

- System labels can no longer be removed by the user. ([#4983](https://github.com/kubermatic/kubermatic/issues/4983))
- End-of-Life Kubernetes v1.14 is no longer supported. ([#4988](https://github.com/kubermatic/kubermatic/issues/4988))
- Added Kubernetes v1.15.7, v1.15.9, v1.16.4, v1.16.6 ([#4995](https://github.com/kubermatic/kubermatic/issues/4995))




## v2.12.5

- A bug that occasionally resulted in a `Error: no matches for kind "MachineDeployment" in version "cluster.k8s.io/v1alpha1"` visible in the UI was fixed. ([#4870](https://github.com/kubermatic/kubermatic/issues/4870))
- A memory leak in the port-forwarding of the Kubernetes dashboard and Openshift console endpoints was fixed ([#4879](https://github.com/kubermatic/kubermatic/issues/4879))
- Enabled edit options for kubeAdm ([#1873](https://github.com/kubermatic/dashboard/issues/1873))




## v2.12.4

- Fixed an issue with adding new node deployments on Openstack ([#1836](https://github.com/kubermatic/dashboard/issues/1836))
- Added migration for cluster user labels ([#4744](https://github.com/kubermatic/kubermatic/issues/4744))
- Added Kubernetes v1.14.9, v1.15.6 and v1.16.3 ([#4752](https://github.com/kubermatic/kubermatic/issues/4752))
- Openstack: A bug that caused cluster reconciliation to fail if the controller crashed at the wrong time was fixed ([#4754](https://github.com/kubermatic/kubermatic/issues/4754))




## v2.12.3

- Fixed extended cluster options not being properly applied ([#1812](https://github.com/kubermatic/dashboard/issues/1812))
- A panic that could occur on clusters that lack both credentials and a credentialsSecret was fixed. ([#4742](https://github.com/kubermatic/kubermatic/issues/4742))


## v2.12.2

- The robustness of vSphere machine reconciliation has been improved. ([#4651](https://github.com/kubermatic/kubermatic/issues/4651))
- Fixed Seed Validation Webhook rejecting new Seeds in certain situations ([#4662](https://github.com/kubermatic/kubermatic/issues/4662))
- Rolled nginx-ingress-controller back to 0.25.1 to fix SSL redirect issues. ([#4693](https://github.com/kubermatic/kubermatic/issues/4693))




## v2.12.1

- VSphere: Fixed a bug that resulted in a faulty cloud config when using a non-default port ([#4562](https://github.com/kubermatic/kubermatic/issues/4562))
- Fixed master-controller failing to create project-label-synchronizer controllers. ([#4577](https://github.com/kubermatic/kubermatic/issues/4577))
- Fixed broken NodePort-Proxy for user clusters with LoadBalancer expose strategy. ([#4590](https://github.com/kubermatic/kubermatic/issues/4590))




## v2.12.0

### Supported Kubernetes versions

- `1.14.8`
- `1.15.5`
- `1.16.2`
- Openshift `v4.1.18` preview

### Major new features

- Kubernetes 1.16 support was added ([#4313](https://github.com/kubermatic/kubermatic/issues/4313))
- It is now possible to also configure automatic node updates by setting `automaticNodeUpdate: true` in the `updates.yaml`. This option implies `automatic: true` as node versions must not be newer than the version of the corresponding controlplane. ([#4258](https://github.com/kubermatic/kubermatic/issues/4258))
- Cloud credentials can now be configured as presets ([#3723](https://github.com/kubermatic/kubermatic/issues/3723))
- Access to datacenters can now be restricted based on the user's email domain. ([#4470](https://github.com/kubermatic/kubermatic/issues/4470))
- It is now possible to open the Kubernetes Dashboard from the Kubermatic UI. ([#4460](https://github.com/kubermatic/kubermatic/issues/4460))
- An option to use AWS Route53 DNS validation was added to the `certs` chart. ([#4397](https://github.com/kubermatic/kubermatic/issues/4397))
- Added possibility to add labels to projects and clusters and have these labels inherited by node objects.
- Added support for Kubernetes audit logging ([#4151](https://github.com/kubermatic/kubermatic/issues/4151))
- Connect button on cluster details will now open Kubernetes Dashboard/Openshift Console ([#1667](https://github.com/kubermatic/dashboard/issues/1667))
- Pod Security Policies can now be enabled ([#4062](https://github.com/kubermatic/kubermatic/issues/4062))
- Added support for optional cluster addons ([#1683](https://github.com/kubermatic/dashboard/issues/1683))

### Installation and updating

- **ACTION REQUIRED:** the `zone_character` field must be removed from all AWS datacenters in `datacenters.yaml` ([#3986](https://github.com/kubermatic/kubermatic/issues/3986))
- **ACTION REQUIRED:** The default number of apiserver replicas was increased to 2. You can revert to the old behavior by setting `.Kubermatic.apiserverDefaultReplicas` in the `values.yaml` ([#3885](https://github.com/kubermatic/kubermatic/issues/3885))
- **ACTION REQUIRED:** The literal credentials on the `Cluster` object are being deprecated in favor of storing them in a secret. If you have addons that use credentials, replace `.Cluster.Spec.Cloud` with `.Credentials`. ([#4463](https://github.com/kubermatic/kubermatic/issues/4463))
- **ACTION REQUIRED:** Kubermatic now doesn't accept unknown keys in its config files anymore and will crash if an unknown key is present
- **ACTION REQUIRED:** BYO datacenters now need to be specific in the `datacenters.yaml` with a value of `{}`, e.G `bringyourown: {}` ([#3794](https://github.com/kubermatic/kubermatic/issues/3794))
- **ACTION REQUIRED:** Velero does not backup Prometheus, Elasticsearch and Minio by default anymore. ([#4482](https://github.com/kubermatic/kubermatic/issues/4482))
- **ACTION REQUIRED:** On AWS, the nodeport-proxy will be recreated as NLB. DNS entries must be updated to point to the new LB. ([#3840](https://github.com/kubermatic/kubermatic/pull/3840))
- The deprecated nodePortPoxy key for Helm values has been removed. ([#3830](https://github.com/kubermatic/kubermatic/issues/3830))
- Support setting oidc authentication settings on cluster ([#3751](https://github.com/kubermatic/kubermatic/issues/3751))
- The worker-count of controller-manager and master-controller are now configurable ([#3918](https://github.com/kubermatic/kubermatic/issues/3918))
- master-controller-manager can now be deployed with multiple replicas ([#4307](https://github.com/kubermatic/kubermatic/issues/4307))
- It is now possible to configure an http proxy on a Seed. This will result in the proxy being used for all control plane pods in that seed that talk to a cloudprovider and for all machines in that Seed, unless its overridden on Datacenter level. ([#4459](https://github.com/kubermatic/kubermatic/issues/4459))
- The cert-manager Helm chart now allows configuring extra values for its controllers args and env vars. ([#4398](https://github.com/kubermatic/kubermatic/issues/4398))
- A fix for CVE-2019-11253 for clusters that were created with a Kubernetes version < 1.14 was deployed ([#4520](https://github.com/kubermatic/kubermatic/issues/4520))

### Dashboard

- Added Swagger UI for Kubermatic API ([#1418](https://github.com/kubermatic/dashboard/issues/1418))
- Redesign dialog to manage SSH keys on cluster ([#1353](https://github.com/kubermatic/dashboard/issues/1353))
- GCP zones are now fetched from API. ([#1379](https://github.com/kubermatic/dashboard/issues/1379))
- Redesign Wizard: Summary ([#1409](https://github.com/kubermatic/dashboard/issues/1409))
- Cluster type toggle in wizard is now hidden if only one cluster type is active ([#1425](https://github.com/kubermatic/dashboard/issues/1425))
- Disabled the possibility of adding new node deployments until the cluster is fully ready. ([#1439](https://github.com/kubermatic/dashboard/issues/1439))
- The cluster name is now editable from the dashboard ([#1455](https://github.com/kubermatic/dashboard/issues/1455))
- Added warning about node deployment changes that will recreate all nodes. ([#1479](https://github.com/kubermatic/dashboard/issues/1479))
- OIDC client id is now configurable ([#1505](https://github.com/kubermatic/dashboard/issues/1505))
- Replaced particles with a static background. ([#1578](https://github.com/kubermatic/dashboard/issues/1578))
- Pod Security Policy can now be activated from the wizard. ([#1647](https://github.com/kubermatic/dashboard/issues/1647))
- Redesigned extended options in wizard ([#1609](https://github.com/kubermatic/dashboard/issues/1609))
- Various security improvements in authentication
- Various other visual improvements

### Monitoring and logging

- Alertmanager's inhibition feature is now used to hide consequential alerts. ([#3833](https://github.com/kubermatic/kubermatic/issues/3833))
- Removed cluster owner name and email labels from kubermatic_cluster_info metric to prevent leaking PII ([#3854](https://github.com/kubermatic/kubermatic/issues/3854))
- New Prometheus metrics kubermatic_addon_created kubermatic_addon_deleted
- New alert KubermaticAddonDeletionTakesTooLong ([#3941](https://github.com/kubermatic/kubermatic/issues/3941))
- FluentBit will now collect the journald logs ([#4001](https://github.com/kubermatic/kubermatic/issues/4001))
- FluentBit can now collect the kernel messages ([#4007](https://github.com/kubermatic/kubermatic/issues/4007))
- FluentBit now always sets the node name in logs ([#4010](https://github.com/kubermatic/kubermatic/issues/4010))
- Added new KubermaticClusterPaused alert with "none" severity for inhibiting alerts from paused clusters ([#3846](https://github.com/kubermatic/kubermatic/issues/3846))
- Removed Helm-based templating in Grafana dashboards ([#4475](https://github.com/kubermatic/kubermatic/issues/4475))
- Added type label (kubernetes/openshift) to kubermatic_cluster_info metric. ([#4452](https://github.com/kubermatic/kubermatic/issues/4452))
- Added metrics endpoint for cluster control plane: `GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/metrics` ([#4208](https://github.com/kubermatic/kubermatic/issues/4208))
- Added a new endpoint for node deployment metrics: `GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}/metrics` ([#4176](https://github.com/kubermatic/kubermatic/issues/4176))

### Cloud providers

- Openstack: A bug that could result in many security groups being created when the creation of security group rules failed was fixed ([#3848](https://github.com/kubermatic/kubermatic/issues/3848))
- Openstack: Fixed a bug preventing an interrupted cluster creation from being resumed. ([#4476](https://github.com/kubermatic/kubermatic/issues/4476))
- Openstack: Disk size of nodes is now configurable ([#4153](https://github.com/kubermatic/kubermatic/issues/4153))
- Openstack: Added a security group API compatibility workaround for very old versions of Openstack. ([#4479](https://github.com/kubermatic/kubermatic/issues/4479))
- Openstack: Fixed fetching the list of tenants on some OpenStack configurations with one region ([#4182](https://github.com/kubermatic/kubermatic/issues/4182))
- Openstack: Added support for Project ID to the wizard ([#1386](https://github.com/kubermatic/dashboard/issues/1386))
- Openstack: The project name can now be provided manually ([#1423](https://github.com/kubermatic/dashboard/issues/1423))
- Openstack: Fixed API usage for datacenters with only one region ([#4538](https://github.com/kubermatic/kubermatic/issues/4538))
- Openstack: Fixed a bug that resulted in the router not being attached to the subnet when the subnet was manually created ([#4521](https://github.com/kubermatic/kubermatic/issues/4521))
- AWS: MachineDeployments can now be created in any availability zone of the cluster's region ([#3870](https://github.com/kubermatic/kubermatic/issues/3870))
- AWS: Reduced the role permissions for the control-plane & worker role to the minimum ([#3995](https://github.com/kubermatic/kubermatic/issues/3995))
- AWS: The subnet can now be selected ([#1499](https://github.com/kubermatic/dashboard/issues/1499))
- AWS: Setting `Control plane role (ARN)` now is possible ([#1512](https://github.com/kubermatic/dashboard/issues/1512))
- AWS: VM sizes are fetched from the API now. ([#1513](https://github.com/kubermatic/dashboard/issues/1513))
- AWS: Worker nodes can now be provisioned without a public IP ([#1591](https://github.com/kubermatic/dashboard/issues/1591))
- GCP: machine and disk types are now fetched from GCP.([#1363](https://github.com/kubermatic/dashboard/issues/1363))
- vSphere: the VM folder can now be configured
- Added support for KubeVirt provider ([#1608](https://github.com/kubermatic/dashboard/issues/1608))

### Bugfixes

- A bug that sometimes resulted in the creation of the initial NodeDeployment failing was fixed ([#3894](https://github.com/kubermatic/kubermatic/issues/3894))
- `kubeadm join` has been fixed for v1.15 clusters ([#4161](https://github.com/kubermatic/kubermatic/issues/4161))
- Fixed a bug that could cause intermittent delays when using kubectl logs/exec with `exposeStrategy: LoadBalancer` ([#4278](https://github.com/kubermatic/kubermatic/issues/4278))
- A bug that prevented node Labels, Taints and Annotations from getting applied correctly was fixed. ([#4368](https://github.com/kubermatic/kubermatic/issues/4368))
- Fixed worker nodes provisioning for instances with a Kernel >= 4.19 ([#4178](https://github.com/kubermatic/kubermatic/issues/4178))
- Fixed an issue that kept clusters stuck if their creation didn't succeed and they got deleted with LB and/or PV cleanup enabled ([#3973](https://github.com/kubermatic/kubermatic/issues/3973))
- Fixed an issue where deleted project owners would come back after a while ([#4025](https://github.com/kubermatic/kubermatic/issues/4025))
- Enabling the OIDC feature flag in clusters has been fixed. ([#4127](https://github.com/kubermatic/kubermatic/issues/4127))

### Misc

- The share cluster feature now allows to use groups, if passed by the IDP. All groups are prefixed with `oidc:` ([#4244](https://github.com/kubermatic/kubermatic/issues/4244))
- The kube-proxy mode (ipvs/iptables) can now be configured. If not specified, it defaults to ipvs. ([#4247](https://github.com/kubermatic/kubermatic/issues/4247))
- Addons can now read the AWS region  from the `kubermatic.io/aws-region` annotation on the cluster ([#4434](https://github.com/kubermatic/kubermatic/issues/4434))
- Allow disabling of apiserver endpoint reconciling. ([#4396](https://github.com/kubermatic/kubermatic/issues/4396))
- Allow cluster owner to manage RBACs from Kubermatic API ([#4321](https://github.com/kubermatic/kubermatic/issues/4321))
- The default service CIDR for new clusters was increased and changed from 10.10.10.0/24 to 10.240.16.0/20 ([#4227](https://github.com/kubermatic/kubermatic/issues/4227))
- Retries of the initial node deployment creation do not create an event anymore but continue to be logged at debug level. ([#4226](https://github.com/kubermatic/kubermatic/issues/4226))
- Added option to enforce cluster cleanup in UI ([#3966](https://github.com/kubermatic/kubermatic/issues/3966))
- Support PodSecurityPolicies in addons ([#4174](https://github.com/kubermatic/kubermatic/issues/4174))
- Kubernetes versions affected by CVE-2019-9512 and CVE-2019-9514 have been dropped ([#4113](https://github.com/kubermatic/kubermatic/issues/4113))
- Kubernetes versions affected by CVE-2019-11247 and CVE-2019-11249 have been dropped ([#4066](https://github.com/kubermatic/kubermatic/issues/4066))
- Kubernetes 1.13 which is end-of-life has been removed. ([#4327](https://github.com/kubermatic/kubermatic/issues/4327))
- Updated Alertmanager to 0.19 ([#4340](https://github.com/kubermatic/kubermatic/issues/4340))
- Updated blackbox-exporter to 0.15.1 ([#4341](https://github.com/kubermatic/kubermatic/issues/4341))
- Updated Canal to v3.8 ([#3791](https://github.com/kubermatic/kubermatic/issues/3791))
- Updated cert-manager to 0.10.1 ([#4407](https://github.com/kubermatic/kubermatic/issues/4407))
- Updated Dex to 2.19 ([#4343](https://github.com/kubermatic/kubermatic/issues/4343))
- Updated Envoy to 1.11.1 ([#4075](https://github.com/kubermatic/kubermatic/issues/4075))
- Updated etcd to 3.3.15 ([#4199](https://github.com/kubermatic/kubermatic/issues/4199))
- Updated FluentBit to v1.2.2 ([#4022](https://github.com/kubermatic/kubermatic/issues/4022))
- Updated Grafana to 6.3.5 ([#4342](https://github.com/kubermatic/kubermatic/issues/4342))
- Updated helm-exporter to 0.4.2 ([#4124](https://github.com/kubermatic/kubermatic/issues/4124))
- Updated kube-state-metrics to 1.7.2 ([#4129](https://github.com/kubermatic/kubermatic/issues/4129))
- Updated Minio to 2019-09-18T21-55-05Z ([#4339](https://github.com/kubermatic/kubermatic/issues/4339))
- Updated machine-controller to v1.5.6 ([#4310](https://github.com/kubermatic/kubermatic/issues/4310))
- Updated nginx-ingress-controller to 0.26.1 ([#4400](https://github.com/kubermatic/kubermatic/issues/4400))
- Updated Prometheus to 2.12.0 ([#4131](https://github.com/kubermatic/kubermatic/issues/4131))
- Updated Velero to v1.1.0 ([#4468](https://github.com/kubermatic/kubermatic/issues/4468))




# Kubermatic 2.11

## v2.11.8

- End-of-Life Kubernetes v1.14 is no longer supported. ([#4989](https://github.com/kubermatic/kubermatic/issues/4989))
- Added Kubernetes v1.15.7, v1.15.9 ([#4995](https://github.com/kubermatic/kubermatic/issues/4995))




## v2.11.7

- Kubernetes 1.13 which is end-of-life has been removed. ([#4327](https://github.com/kubermatic/kubermatic/issues/4327))
- Added Kubernetes v1.15.4 ([#4329](https://github.com/kubermatic/kubermatic/issues/4329))
- Added Kubernetes v1.14.7 ([#4330](https://github.com/kubermatic/kubermatic/issues/4330))
- A bug that prevented node Labels, Taints and Annotations from getting applied correctly was fixed. ([#4368](https://github.com/kubermatic/kubermatic/issues/4368))
- Removed K8S releases affected by CVE-2019-11253 ([#4515](https://github.com/kubermatic/kubermatic/issues/4515))
- A fix for CVE-2019-11253 for clusters that were created with a Kubernetes version < 1.14 was deployed ([#4520](https://github.com/kubermatic/kubermatic/issues/4520))
- Openstack: fixed API usage for datacenters with only one region ([#4536](https://github.com/kubermatic/kubermatic/issues/4536))




## v2.11.6

- Fixed a bug that could cause intermittent delays when using kubectl logs/exec with `exposeStrategy: LoadBalancer` ([#4279](https://github.com/kubermatic/kubermatic/issues/4279))




## v2.11.5

- Fix a bug that caused setup on nodes with a Kernel > 4.18 to fail ([#4180](https://github.com/kubermatic/kubermatic/issues/4180))
- Fixed fetching the list of tenants on some OpenStack configurations with one region ([#4185](https://github.com/kubermatic/kubermatic/issues/4185))
- Fixed a bug that could result in the clusterdeletion sometimes getting stuck ([#4202](https://github.com/kubermatic/kubermatic/issues/4202))




## v2.11.4

- `kubeadm join` has been fixed for v1.15 clusters ([#4162](https://github.com/kubermatic/kubermatic/issues/4162))




## v2.11.3

- Kubermatic Swagger API Spec is now exposed over its API server ([#3890](https://github.com/kubermatic/kubermatic/issues/3890))
- updated Envoy to 1.11.1 ([#4075](https://github.com/kubermatic/kubermatic/issues/4075))
- Kubernetes versions affected by CVE-2019-9512 and CVE-2019-9514 have been dropped ([#4118](https://github.com/kubermatic/kubermatic/issues/4118))
- Enabling the OIDC feature flag in clusters has been fixed. ([#4136](https://github.com/kubermatic/kubermatic/issues/4136))




## v2.10.3

- Kubernetes 1.11 which is end-of-life has been removed. ([#4031](https://github.com/kubermatic/kubermatic/issues/4031))
- Kubernetes 1.12 which is end-of-life has been removed. ([#4065](https://github.com/kubermatic/kubermatic/issues/4065))
- Kubernetes versions affected by CVE-2019-11247 and CVE-2019-11249 have been dropped ([#4066](https://github.com/kubermatic/kubermatic/issues/4066))
- Kubernetes versions affected by CVE-2019-9512 and CVE-2019-9514 have been dropped ([#4113](https://github.com/kubermatic/kubermatic/issues/4113))
- updated Envoy to 1.11.1 ([#4075](https://github.com/kubermatic/kubermatic/issues/4075))




## v2.11.2


- Fixed an issue where deleted project owners would come back after a while ([#4020](https://github.com/kubermatic/kubermatic/issues/4020))
- Kubernetes versions affected by CVE-2019-11247 and CVE-2019-11249 have been dropped ([#4066](https://github.com/kubermatic/kubermatic/issues/4066))
- Kubernetes 1.11 which is end-of-life has been removed. ([#4030](https://github.com/kubermatic/kubermatic/issues/4030))
- Kubernetes 1.12 which is end-of-life has been removed. ([#4067](https://github.com/kubermatic/kubermatic/issues/4067))




## v2.11.1

### Misc

- Openstack: A bug that could result in many security groups being created when the creation of security group rules failed was fixed ([#3848](https://github.com/kubermatic/kubermatic/issues/3848))
- Added Kubernetes v1.15.1 ([#3859](https://github.com/kubermatic/kubermatic/issues/3859))
- Updated machine controller to v1.5.1 ([#3883](https://github.com/kubermatic/kubermatic/issues/3883))
- A bug that sometimes resulted in the creation of the initial NodeDeployment failing was fixed ([#3894](https://github.com/kubermatic/kubermatic/issues/3894))
- Fixed an issue that kept clusters stuck if their creation didn't succeed and they got deleted with LB and/or PV cleanup enabled ([#3973](https://github.com/kubermatic/kubermatic/issues/3973))
- Fixed joining nodes to Bring Your Own clusters running Kubernetes 1.14 ([#3976](https://github.com/kubermatic/kubermatic/issues/3976))

### Dashboard

- Fixed an issue with handling resources refresh on error conditions ([#1452](https://github.com/kubermatic/dashboard/issues/1452))
- Openstack: the project name can now be provided manually ([#1426](https://github.com/kubermatic/dashboard/issues/1426))
- JS dependencies have been updated to address potential vulnerabilities in some of them. ([#1388](https://github.com/kubermatic/dashboard/issues/1388))




## v2.11.0

### Supported Kubernetes versions:

- `1.11.5-10`
- `1.12.3-10`
- `1.13.0-5`
- `1.13.7`
- `1.14.0-1`
- `1.14.3-4`
- `1.15.0`


### Cloud providers

- It is now possible to create Kubermatic-managed clusters on Packet. ([#3419](https://github.com/kubermatic/kubermatic/issues/3419))
- It is now possible to create Kubermatic-managed clusters on GCP. ([#3350](https://github.com/kubermatic/kubermatic/issues/3350))
- the API stops creating an initial node deployment for new cluster for KubeAdm providers. ([#3346](https://github.com/kubermatic/kubermatic/issues/3346))
- Openstack: datacenter can be configured with minimum required CPU and memory for nodes ([#3487](https://github.com/kubermatic/kubermatic/issues/3487))
- vsphere: root disk size is now configurable ([#3629](https://github.com/kubermatic/kubermatic/issues/3629))
- Azure: fixed failure to provision on new regions due to lower number of fault domains ([#3584](https://github.com/kubermatic/kubermatic/issues/3584))

### Dashboard

- The project menu has been redesigned. ([#1195](https://github.com/kubermatic/dashboard/issues/1195))
- Fixed changing default OpenStack image on operating system change ([#1215](https://github.com/kubermatic/dashboard/issues/1215))
- `containerRuntimeVersion` and `kernelVersion` are now displayed on NodeDeployment detail page ([#1216](https://github.com/kubermatic/dashboard/issues/1216))
- Custom links can now be added to the footer. ([#1220](https://github.com/kubermatic/dashboard/issues/1220))
- The OIDC provider URL is now configurable via "oidc_provider_url" variable. ([#1222](https://github.com/kubermatic/dashboard/issues/1222))
- The application logo has been changed. ([#1232](https://github.com/kubermatic/dashboard/issues/1232))
- The breadcrumbs component has been removed. The dialogs and buttons have been redesigned. ([#1233](https://github.com/kubermatic/dashboard/issues/1233))
- Packet cloud provider is now supported. ([#1238](https://github.com/kubermatic/dashboard/issues/1238))
- Tables have been redesigned. ([#1240](https://github.com/kubermatic/dashboard/issues/1240))
- Added option to specify taints when creating/updating NodeDeployments ([#1244](https://github.com/kubermatic/dashboard/issues/1244))
- Styling of the cluster details view has been improved. ([#1270](https://github.com/kubermatic/dashboard/issues/1270))
- Missing parameters for OIDC providers have been added. ([#1273](https://github.com/kubermatic/dashboard/issues/1273))
- Dates are now displayed using relative format, i.e. 3 days ago. ([#1303](https://github.com/kubermatic/dashboard/issues/1303))
- Redesigned dialogs and cluster details page. ([#1305](https://github.com/kubermatic/dashboard/issues/1305))
- Add provider GCP to UI ([#1307](https://github.com/kubermatic/dashboard/issues/1307))
- Redesigned notifications. ([#1315](https://github.com/kubermatic/dashboard/issues/1315))
- The Instance Profile Name for AWS could be specified in UI. ([#1317](https://github.com/kubermatic/dashboard/issues/1317))
- Redesigned node deployment view. ([#1320](https://github.com/kubermatic/dashboard/issues/1320))
- Redesigned cluster details page. ([#1345](https://github.com/kubermatic/dashboard/issues/1345))

### Monitoring

- **ACTION REQUIRED:** refactored Alertmanager Helm chart for master-cluster monitoring, see documentation for migration notes ([#3448](https://github.com/kubermatic/kubermatic/issues/3448))
- cAdvisor metrics are now being scraped for user clusters ([#3390](https://github.com/kubermatic/kubermatic/issues/3390))
- fixed kube-state-metrics in user-clusters not being scraped ([#3427](https://github.com/kubermatic/kubermatic/issues/3427))
- Improved debugging of resource leftovers through new etcd Object Count dashboard ([#3508](https://github.com/kubermatic/kubermatic/issues/3508))
- New Grafana dashboards for monitoring Elasticsearch ([#3516](https://github.com/kubermatic/kubermatic/issues/3516))
- Added optional Thanos integration to Prometheus for better long-term metrics storage ([#3531](https://github.com/kubermatic/kubermatic/issues/3531))

### Misc

- **ACTION REQUIRED:** nodePortPoxy Helm values has been renamed to nodePortProxy, old root key is now deprecated; please update your Helm values ([#3418](https://github.com/kubermatic/kubermatic/issues/3418))
- Service accounts have been implemented.
- Support for Kubernetes 1.15 was added ([#3579](https://github.com/kubermatic/kubermatic/issues/3579))
- More details are shown when using `kubectl get machine/machineset/machinedeployment` ([#3364](https://github.com/kubermatic/kubermatic/issues/3364))
- The resiliency of in-cluster DNS was greatly improved by adding the nodelocal-dns-cache addon, which runs a DNS cache on each node, avoiding the need to use NAT for DNS queries ([#3369](https://github.com/kubermatic/kubermatic/issues/3369))
- Added containerRuntimeVersion and kernelVersion to NodeInfo ([#3381](https://github.com/kubermatic/kubermatic/issues/3381))
- It is now possible to configure Kubermatic to create one service of type LoadBalancer per user cluster instead of exposing all of them via the nodeport-proxy on one central LoadBalancer service ([#3387](https://github.com/kubermatic/kubermatic/issues/3387))
- Pod AntiAffinity and PDBs were added to the Kubermatic control plane components,the monitoring stack and the logging stack to spread them out if possible and reduce the chance of unavailability ([#3393](https://github.com/kubermatic/kubermatic/issues/3393))
- Reduced API latency for loading Nodes & NodeDeployments ([#3405](https://github.com/kubermatic/kubermatic/issues/3405))
- replace gambol99/keycloak-proxy 2.3.0 with official keycloak-gatekeeper 6.0.1 ([#3411](https://github.com/kubermatic/kubermatic/issues/3411))
- More additional printer columns for kubermatic crds ([#3542](https://github.com/kubermatic/kubermatic/issues/3542))
- Insecure Kubernetes versions v1.13.6 and v1.14.2 have been disabled. ([#3554](https://github.com/kubermatic/kubermatic/issues/3554))
- Kubermatic now supports running in environments where the Internet can only be accessed via a http proxy ([#3615](https://github.com/kubermatic/kubermatic/issues/3615))
- ICMP traffic to clusters is now always permitted to allow MTU discovery ([#3618](https://github.com/kubermatic/kubermatic/issues/3618))
- A bug that caused errors on very big addon manifests was fixed ([#3366](https://github.com/kubermatic/kubermatic/issues/3366))
- Updated Prometheus to 2.10.0 ([#3612](https://github.com/kubermatic/kubermatic/issues/3612))
- Updated cert-manager to 0.8.0 ([#3525](https://github.com/kubermatic/kubermatic/issues/3525))
- Updated Minio to RELEASE.2019-06-11T00-44-33Z ([#3614](https://github.com/kubermatic/kubermatic/issues/3614))
- Updated Grafana to 6.2.1 ([#3528](https://github.com/kubermatic/kubermatic/issues/3528))
- Updated kube-state-metrics to 1.6.0 ([#3420](https://github.com/kubermatic/kubermatic/issues/3420))
- Updated Dex to 2.16.0 ([#3361](https://github.com/kubermatic/kubermatic/issues/3361))
- Updated Alertmanager to 0.17.0, deprecate version field in favor of image.tag in Helm values.yaml ([#3410](https://github.com/kubermatic/kubermatic/issues/3410))
- Updated machine-controller to v1.4.2 ([#3778](https://github.com/kubermatic/kubermatic/issues/3778))
- Updated node-exporter to 0.18.1 ([#3613](https://github.com/kubermatic/kubermatic/issues/3613))
- Updated fluent-bit to 1.1.2 ([#3561](https://github.com/kubermatic/kubermatic/issues/3561))
- Updated Velero to 1.0 ([#3527](https://github.com/kubermatic/kubermatic/issues/3527))




# Kubermatic 2.10

## v2.10.2

### Misc

- Updated Dashboard to v1.2.2 ([#3553](https://github.com/kubermatic/kubermatic/issues/3553))
  - Missing parameters for OIDC providers have been added. ([#1273](https://github.com/kubermatic/dashboard/issues/1273))
  - `containerRuntimeVersion` and `kernelVersion` are now displayed on NodeDeployment detail page ([#1217](https://github.com/kubermatic/dashboard/issues/1217))
  - Fixed changing default OpenStack image on Operating System change ([#1218](https://github.com/kubermatic/dashboard/issues/1218))
  - The OIDC provider URL is now configurable via "oidc_provider_url" variable. ([#1224](https://github.com/kubermatic/dashboard/issues/1224))
- Insecure Kubernetes versions v1.13.6 and v1.14.2 have been disabled. ([#3554](https://github.com/kubermatic/kubermatic/issues/3554))




## v2.10.1

### Bugfixes

- A bug that caused errors on very big addon manifests was fixed ([#3366](https://github.com/kubermatic/kubermatic/issues/3366))
- fixed kube-state-metrics in user-clusters not being scraped ([#3431](https://github.com/kubermatic/kubermatic/issues/3431))
- Updated the machine-controller to fix the wrong CentOS image for AWS instances ([#3432](https://github.com/kubermatic/kubermatic/issues/3432))
- vSphere VMs are cleaned up on ISO failure. ([#3474](https://github.com/kubermatic/kubermatic/issues/3474))

### Misc

- updated Prometheus to v2.9.2 ([#3348](https://github.com/kubermatic/kubermatic/issues/3348))
- Draining of nodes now times out after 2h ([#3354](https://github.com/kubermatic/kubermatic/issues/3354))
- the API stops creating an initial node deployment for new cluster for KubeAdm providers. ([#3373](https://github.com/kubermatic/kubermatic/issues/3373))
- More details are shown when using `kubectl get machine/machineset/machinedeployment` ([#3377](https://github.com/kubermatic/kubermatic/issues/3377))
- Pod AntiAffinity and PDBs were added to the Kubermatic control plane components and the monitoring stack to spread them out if possible and reduce the chance of unavailability ([#3400](https://github.com/kubermatic/kubermatic/issues/3400))
- Support for Kubernetes 1.11.10 was added ([#3429](https://github.com/kubermatic/kubermatic/issues/3429))




## v2.10.0

### Kubermatic core

* **ACTION REQUIRED:** The config option `Values.kubermatic.rbac` changed to `Values.kubermatic.masterController` ([#3051](https://github.com/kubermatic/kubermatic/pull/3051))
* The user cluster controller manager was added. It is deployed within the cluster namespace in the seed and takes care of reconciling all resources that are inside the user cluster
* Add feature gate to enable etcd corruption check ([#2460](https://github.com/kubermatic/kubermatic/pull/2460))
* Kubernetes 1.10 was removed as officially supported version from Kubermatic as it's EOL ([#2712](https://github.com/kubermatic/kubermatic/pull/2712))
* Add short names to the ClusterAPI CRDs to allow using `kubectl get md` for `machinedeployments`, `kubectl get ms` for `machinesets` and `kubectl get ma` to get `machines` ([#2718](https://github.com/kubermatic/kubermatic/pull/2718))
* Update canal to v2.6.12, Kubernetes Dashboard to v1.10.1 and replace kube-dns with CoreDNS 1.3.1 ([#2985](https://github.com/kubermatic/kubermatic/pull/2985))
* Update Vertical Pod Autoscaler to 0.5 ([#3143](https://github.com/kubermatic/kubermatic/pull/3143))
* Avoid the name "kubermatic" for cloud provider resources visible by end users ([#3152](https://github.com/kubermatic/kubermatic/pull/3152))
* In order to provide Grafana dashboards for user cluster resource usage, the node-exporter is now deployed by default as an addon into user clusters. ([#3089](https://github.com/kubermatic/kubermatic/pull/3089))
* Make the default AMI's for AWS instances configurable via the datacenters.yaml ([#3169](https://github.com/kubermatic/kubermatic/pull/3169))
* Vertical Pod Autoscaler is not deployed by default anymore ([#2805](https://github.com/kubermatic/kubermatic/pull/2805))
* Initial node deployments are now created inside the same API call as the cluster, fixing spurious issues where the creation didn't happen ([#2989](https://github.com/kubermatic/kubermatic/pull/2989))
* Errors when reconciling MachineDeployments and MachineSets will now result in an event on the object ([#2923](https://github.com/kubermatic/kubermatic/pull/2923))
* Filter out not valid VM types for azure provider ([#2736](https://github.com/kubermatic/kubermatic/pull/2736))
* Mark cluster upgrades as restricted if kubelet version is incompatible. ([#2976](https://github.com/kubermatic/kubermatic/pull/2976))
* Enable automatic detection of the OpenStack BlockStorage API version within the cloud config ([#3112](https://github.com/kubermatic/kubermatic/pull/3112))
* Add the ContainerLinuxUpdateOperator to all clusters that use ContainerLinux nodes ([#3239](https://github.com/kubermatic/kubermatic/pull/3239))
* The trust-device-path cloud config property of Openstack clusters can be configured via datacenters.yaml. ([#3265](https://github.com/kubermatic/kubermatic/pull/3265))
* Set AntiAffinity for pods to prevent situations where the API servers of all clusters got scheduled on a single node ([#3269](https://github.com/kubermatic/kubermatic/pull/3269))
* Set resource requests & limits for all addons ([#3270](https://github.com/kubermatic/kubermatic/pull/3270))
* Add Kubernetes v1.14.1 to the list of supported versions ([#3273](https://github.com/kubermatic/kubermatic/pull/3273))
* A small amount of resources gets reserved on each node for the Kubelet and system services ([#3298](https://github.com/kubermatic/kubermatic/pull/3298))
* Update etcd to v3.3.12 ([#3288](https://github.com/kubermatic/kubermatic/pull/3288))
* Update the metrics-server to v0.3.2 ([#3289](https://github.com/kubermatic/kubermatic/pull/3289))
* Update the user cluster Prometheus to v2.9.1 ([#3287](https://github.com/kubermatic/kubermatic/pull/3287))
* It is now possible to scale MachineDeployments and MachineSets via `kubectl scale` ([#3277](https://github.com/kubermatic/kubermatic/pull/3277))

### Dashboard

* The color scheme of the Dashboard was changed
* It is now possible to edit the project name in UI ([#1003](https://github.com/kubermatic/dashboard/pull/1003))
* Made Nodes and Node Deployments statuses more accurate ([#1016](https://github.com/kubermatic/dashboard/pull/1016))
* Redesign DigitalOcean sizes and OpenStack flavors option pickers ([#1021](https://github.com/kubermatic/dashboard/pull/1021))
* Smoother operation on bad network connection thanks to changes in asset caching. ([#1030](https://github.com/kubermatic/dashboard/pull/1030))
* Added a flag allowing to change the default number of nodes created with clusters. ([#1032](https://github.com/kubermatic/dashboard/pull/1032))
* Setting openstack tags for instances is possible via UI now. ([#1038](https://github.com/kubermatic/dashboard/pull/1038))
* Allowed Node Deployment naming. ([#1039](https://github.com/kubermatic/dashboard/pull/1039))
* Adding multiple owners to a project is possible via UI now. ([#1042](https://github.com/kubermatic/dashboard/pull/1042))
* Allowed specifying kubelet version for Node Deployments. ([#1047](https://github.com/kubermatic/dashboard/pull/1047))
* Events related to the Nodes are now displayed in the Node Deployment details view. ([#1054](https://github.com/kubermatic/dashboard/pull/1054))
* Fixed reload behaviour of openstack setting fields. ([#1056](https://github.com/kubermatic/dashboard/pull/1056))
* Fixed a bug with the missing version in the footer. ([#1067](https://github.com/kubermatic/dashboard/pull/1067))
* Project owners are now visible in project list view . ([#1082](https://github.com/kubermatic/dashboard/pull/1082))
* Added possibility to assign labels to nodes. ([#1101](https://github.com/kubermatic/dashboard/pull/1101))
* Updated AWS instance types. ([#1122](https://github.com/kubermatic/dashboard/pull/1122))
* Fixed display number of replicas if the field is empty (0 replicas). ([#1126](https://github.com/kubermatic/dashboard/pull/1126))
* Added an option to include custom links into the application. ([#1131](https://github.com/kubermatic/dashboard/pull/1131))
* Remove AWS instance types t3.nano & t3.micro as they are too small to schedule any workload on them ([#1138](https://github.com/kubermatic/dashboard/pull/1138))
* Redesigned the application sidebar. ([#1173](https://github.com/kubermatic/dashboard/pull/1173))

### Logging & Monitoring stack

* Update fluent-bit to 1.0.6 ([#3222](https://github.com/kubermatic/kubermatic/pull/3222))
* Add elasticsearch-exporter to logging stack to improve monitoring ([#2773](https://github.com/kubermatic/kubermatic/pull/2773))
* New alerts for cert-manager created certificates about to expire ([#2787](https://github.com/kubermatic/kubermatic/pull/2787))
* Add blackbox-exporter chart ([#2954](https://github.com/kubermatic/kubermatic/pull/2954))
* Update Elasticsearch to 6.6.2 ([#3062](https://github.com/kubermatic/kubermatic/pull/3062))
* Add Grafana dashboards for kubelet metrics ([#3081](https://github.com/kubermatic/kubermatic/pull/3081))
* Prometheus was updated to 2.8.1 (Alertmanager 0.16.2), Grafana was updated to 6.1.3 ([#3163](https://github.com/kubermatic/kubermatic/pull/3163))
* Alertmanager PVC size is configurable ([#3199](https://github.com/kubermatic/kubermatic/pull/3199))
* Add lifecycle hooks to the Elasticsearch StatefulSet to make starting/stopping more graceful ([#2933](https://github.com/kubermatic/kubermatic/pull/2933))
* Pod annotations are no longer logged in Elasticsearch ([#2959](https://github.com/kubermatic/kubermatic/pull/2959))
* Improve Prometheus backups in high traffic environments ([#3047](https://github.com/kubermatic/kubermatic/pull/3047))
* Fix VolumeSnapshotLocations for Ark configuration ([#3076](https://github.com/kubermatic/kubermatic/pull/3076))
* node-exporter is not exposed on all host interfaces anymore ([#3085](https://github.com/kubermatic/kubermatic/pull/3085))
* Improve Kibana usability by auto-provisioning index patterns ([#3099](https://github.com/kubermatic/kubermatic/pull/3099))
* Configurable Prometheus backup timeout to accommodate larger seed clusters ([#3223](https://github.com/kubermatic/kubermatic/pull/3223))

### Other

* **ACTION REQUIRED:** update from Ark 0.10 to Velero 0.11 ([#3077](https://github.com/kubermatic/kubermatic/pull/3077))
* Replace hand written go tcp proxy with Envoy within the nodeport-proxy ([#2916](https://github.com/kubermatic/kubermatic/pull/2916))
* cert-manager was updated to 0.7.0, Dex was updated to 2.15.0,Minio was updated to RELEASE.2019-04-09T01-22-30Z ([#3163](https://github.com/kubermatic/kubermatic/pull/3163))
* update nginx-ingress-controller to 0.24.1 ([#3200](https://github.com/kubermatic/kubermatic/pull/3200))
* Allow scheduling Helm charts using affinities, node selectors and tolerations for more stable clusters ([#3155](https://github.com/kubermatic/kubermatic/pull/3155))
* Helm charts: Define configurable resource constraints ([#3012](https://github.com/kubermatic/kubermatic/pull/3012))
* improve Helm charts metadata to make Helm-based workflows easier and aid in cluster updates ([#3221](https://github.com/kubermatic/kubermatic/pull/3221))
* dex keys expirations can now be configured in helm chart ([#3301](https://github.com/kubermatic/kubermatic/pull/3301))
* Update the nodeport-proxy Envoy to v1.10 ([#3274](https://github.com/kubermatic/kubermatic/pull/3274))

## Bugfixes

* Fixed invalid variable caching in Grafana dashboards ([#2792](https://github.com/kubermatic/kubermatic/pull/2792))
* Migrations are now executed only after the leader lease was acquired ([#3276](https://github.com/kubermatic/kubermatic/pull/3276))




# Kubermatic 2.9

## v2.9.3

- Errors when reconciling MachineDeployments and MachineSets will now result in an event on the object ([#2930](https://github.com/kubermatic/kubermatic/issues/2930))
- Missing permissions have been added to the kube-state-metrics ClusterRole ([#2978](https://github.com/kubermatic/kubermatic/issues/2978))
- Fixed invalid variable caching in Grafana dashboards ([#2992](https://github.com/kubermatic/kubermatic/issues/2992))
- Kibana is automatically initialized in new installations. ([#2995](https://github.com/kubermatic/kubermatic/issues/2995))
- Updated machine controller to v1.1.0 ([#3028](https://github.com/kubermatic/kubermatic/issues/3028))




## v2.9.2

* The cleanup of services of type LoadBalancer on cluster deletion was fixed and re-enabled ([#2780](https://github.com/kubermatic/kubermatic/pull/2780))
* The Kubernetes Dashboard addon was updated to 1.10.1 ([#2848](https://github.com/kubermatic/kubermatic/pull/2848))
* Joining of nodes via the BYO functionality was fixed ([#2835](https://github.com/kubermatic/kubermatic/pull/2835))
* It is now possible to configure whether Openstack security groups for LoadBalancers should be managed by Kubernetes, [check the sample `datacenters.yaml` in the docs for details](https://docs.kubermatic.io/installation/install_kubermatic/_manual/#defining-the-datacenters) ([#2878](https://github.com/kubermatic/kubermatic/pull/2878))
* A bug that resulted in clusters being twice in the UI overview got resolved ([#1088](https://github.com/kubermatic/dashboard/pull/1088))
* A bug that could cause the image of a NodeDeployment to be set to the default when the NodeDeployment gets edited got resolved ([#1076](https://github.com/kubermatic/dashboard/pull/1076))
* A bug that caused the version of the UI to not be shown in the footer got resolved ([#1096](https://github.com/kubermatic/dashboard/pull/1096))
* A bug that caused updating and deleting of NodeDeployments in the NodeDeployment details page not to work got resolved ([#1076](https://github.com/kubermatic/dashboard/pull/1076))
* The NodeDeployment detail view now correctly displays the node datacenter instead of the seed datacenter ([#1094](https://github.com/kubermatic/dashboard/pull/1094))
* Support for Kubernetes 1.11.8, 1.12.6, 1.13.3 and 1.13.4 was added ([#2894](https://github.com/kubermatic/kubermatic/pull/2894))




## v2.9.1

* The Docker version used for all new machines with CoreOS or Ubuntu has a fix for CVE-2019-573. It s advised to roll over all your worker nodes to make sure that new version is used
* It is now possible to name NodeDeployments
* A bug that caused duplicate top level keys in the values.example.yaml got fixed
* A bug that made it impossible to choose a subnet on Openstack after a network was chosen got fixed
* Scraping of 1.13 user cluster Schedulers and Controller manager now works
* Scraping of the seed clusters Scheduler and Controller manager now works
* A bug that caused spurious failures when applying the cert-manager chart was resolved
* NodeDeployment events are now shown in the UI
* It is now possible to configure the Kubernetes version of a NodeDeployment in the UI




## v2.9.0

### Supported Kubernetes versions

- `1.11.5-7`
- `1.12.3-5`
- `1.13.0-2`


### Cloud Provider

- Added support for PersistentVolumes on **Hetzner Cloud** ([#2613](https://github.com/kubermatic/kubermatic/issues/2613))
- Openstack Floating IPs will now be de-allocated from your project if they were allocated during node creation ([#2675](https://github.com/kubermatic/kubermatic/issues/2675))

### Dashboard

- It is now possible to edit the project name in UI. ([#1003](https://github.com/kubermatic/dashboard/issues/1003))
- Machine Networks for VSphere can now be set in the UI ([#829](https://github.com/kubermatic/dashboard/issues/829))
- VSphere: Setting a dedicated VSphere user for cloud provider functionalities is now possible. ([#834](https://github.com/kubermatic/dashboard/issues/834))
- Fixed that the cluster upgrade link did not appear directly when the details page is loaded ([#836](https://github.com/kubermatic/dashboard/issues/836))
- Kubeconfig can now be shared via a generated link from the UI ([#857](https://github.com/kubermatic/dashboard/issues/857))
  - See https://docs.kubermatic.io/advanced/oidc_auth/ for more information.
- Fixed duplicated SSH keys in summary view during cluster creation. ([#879](https://github.com/kubermatic/dashboard/issues/879))
- On project change, the user will stay on the same page, if he has the corresponding rights. ([#889](https://github.com/kubermatic/dashboard/issues/889))
- Fixed issues with caching the main page. ([#893](https://github.com/kubermatic/dashboard/issues/893))
- Nodes are now being managed as NodeDeployments, this allows to easily change settings for a group of Nodes. ([#949](https://github.com/kubermatic/dashboard/issues/949))
- Removed Container Runtime selection, which is no longer supported. ([#828](https://github.com/kubermatic/dashboard/issues/828))
- Menu entries will be disabled as long as selected project is not in active state.
- Selected project state icon was added in the project selector and in the list view.
- Input field inside add project dialog will be automatically focused after opening dialog.
- After adding new project user will be redirected to project list ([#808](https://github.com/kubermatic/dashboard/issues/808))
- Notifications timeout is now 10s.
- Close and copy to clipboard actions are available on notifications. ([#798](https://github.com/kubermatic/dashboard/issues/798))
- Provider-specific data will now be fetched without re-sending credentials. ([#814](https://github.com/kubermatic/dashboard/issues/814))
- Various minor visual improvements

### Misc

- Added support for Kubernetes v1.13
- Kubermatic now supports Kubernetes 1.12 ([#2132](https://github.com/kubermatic/kubermatic/issues/2132))
- The startup time for new clusters was improved ([#2148](https://github.com/kubermatic/kubermatic/issues/2148))
- The EOL Kubernetes 1.9 is no longer supported ([#2252](https://github.com/kubermatic/kubermatic/issues/2252))
- S3 metrics exporter has been moved out of the kubermatic chart into its own chart ([#2256](https://github.com/kubermatic/kubermatic/issues/2256))
- Displaying the terms of service can now be toggled in values.yaml ([#2277](https://github.com/kubermatic/kubermatic/issues/2277))
- **ACTION REQUIRED:** added a new command line flag to API server that accepts a set of key=value pairs that enables/disables various features. Existing `enable-prometheus-endpoint` flag is deprecated, the users should use `-feature-gates=PrometheusEndpoint=true` instead. ([#2278](https://github.com/kubermatic/kubermatic/issues/2278))
- etcd readiness check timeouts have been increased ([#2312](https://github.com/kubermatic/kubermatic/issues/2312))
- Removed unused fields from cloud specs exposed in the API ([#2314](https://github.com/kubermatic/kubermatic/issues/2314))
- Kubermatic now validates nodes synchronously ([#2340](https://github.com/kubermatic/kubermatic/issues/2340))
- Kubermatic now manages Nodes as group via the NodeGroup feature ([#2357](https://github.com/kubermatic/kubermatic/issues/2357))
- Components will no longer be shown as as unhealthy when only some replicas are up ([#2358](https://github.com/kubermatic/kubermatic/issues/2358))
- Kubernetes API servers can now be used with OpenID authentication
  - **ACTION REQUIRED:** to enable the OpenID for kubernetes API server the users must set `-feature-gates=OpenIDConnectTokens=true` and provide `-oidc-issuer-url`, `-oidc-issuer-client-id` when running the controller. ([#2370](https://github.com/kubermatic/kubermatic/issues/2370))
- **ACTION REQUIRED:** Resource limits for control plane containers have been increased. This might require additional resources for the seed cluster ([#2395](https://github.com/kubermatic/kubermatic/issues/2395))
  - Kubernetes API server: 4Gi RAM, 2 CPU
  - Kubernetes Controller Manager: 2Gi RAM, 2 CPU
  - Kubernetes scheduler: 512Mi RAM, 1 CPU
  - CoreDNS: 128Mi RAM, 0.1 CPU
  - etcd: 2Gi RAM, 2 CPU
  - kube state metrics: 1Gi, 0.1 CPU
  - OpenVPN: 128Mi RAM, 0.1 CPU
  - Prometheus: 1Gi RAM, 0.1 CPU
- **ACTION_REQUIRED:** Kubermatic CustomResourceDefinitions have been extracted out of the helm chart. This requires the execution of the `charts/kubermatic/migrate/migrate-kubermatic-chart.sh` script in case the CRD's where installed without the `"helm.sh/resource-policy": keep` annotation. ([#2459](https://github.com/kubermatic/kubermatic/issues/2459))
- Control plane components are no longer logging at debug level ([#2471](https://github.com/kubermatic/kubermatic/issues/2471))
- Experimental support for VerticalPodAutoscaler has been added. The VPA resources use the PodUpdatePolicy=initial ([#2505](https://github.com/kubermatic/kubermatic/issues/2505))
- Added 1.11.6 & 1.12.4 to supported Kubernetes versions ([#2537](https://github.com/kubermatic/kubermatic/issues/2537))
- It's now possible to rename a project ([#2588](https://github.com/kubermatic/kubermatic/issues/2588))
- It is now possible for a user to select whether PVCs/PVs and/or LBs should be cleaned up when deleting the cluster. ([#2604](https://github.com/kubermatic/kubermatic/issues/2604))
- Credentials for Docker Hub are no longer necessary. ([#2605](https://github.com/kubermatic/kubermatic/issues/2605))
- Added support for Heptio Ark-based backups ([#2617](https://github.com/kubermatic/kubermatic/issues/2617))
- Running `kubectl get cluster` in a seed now shows some more details ([#2622](https://github.com/kubermatic/kubermatic/issues/2622))
- Kubernetes 1.10 was removed as officially supported version from Kubermatic as its EOL ([#2712](https://github.com/kubermatic/kubermatic/issues/2712))
- Updated machine controller to v0.10.5 ([#2490](https://github.com/kubermatic/kubermatic/issues/2490))
- Updated dex to 2.12.0 ([#2318](https://github.com/kubermatic/kubermatic/issues/2318))
- Updated nginx-ingress-controller to v0.22.0 ([#2668](https://github.com/kubermatic/kubermatic/issues/2668))
- **ACTION REQUIRED:** Updated cert-manager to v0.6.0 (see https://cert-manager.readthedocs.io/en/latest/admin/upgrading/index.html) ([#2674](https://github.com/kubermatic/kubermatic/issues/2674))

### Monitoring

- Version v1.11.0 - 1.11.3 Clusters will no longer gather `rest_*` metrics from the controller-manager due to a [bug in kubernetes](https://github.com/kubernetes/kubernetes/pull/68530) ([#2020](https://github.com/kubermatic/kubermatic/issues/2020))
- Enabled scraping of user cluster resources ([#2149](https://github.com/kubermatic/kubermatic/issues/2149))
- Prometheus is now scraping user clustersNew `kubermatic-controller-manager` flag `monitoring-scrape-annotation-prefix` ([#2219](https://github.com/kubermatic/kubermatic/issues/2219))
- UserCluster Prometheus: decreased storage.tsdb.retention to 1h ([#2246](https://github.com/kubermatic/kubermatic/issues/2246))
- Add datacenter label to kubermatic_cluster_info metric ([#2248](https://github.com/kubermatic/kubermatic/issues/2248))
- Fixed the trigger condition for `EtcdInsufficientMembers` alert ([#2262](https://github.com/kubermatic/kubermatic/issues/2262))
- **ACTION REQUIRED:** move the metrics-server into the seed cluster. The metrics-server addon must be removed from the list of addons to install. ([#2320](https://github.com/kubermatic/kubermatic/issues/2320))
- ArkNoRecentBackup alert does not trigger on backups that are not part of a schedule ([#2351](https://github.com/kubermatic/kubermatic/issues/2351))
- fluentd has been replaced with fluentbit ([#2469](https://github.com/kubermatic/kubermatic/issues/2469))
- Cluster Prometheus resource requests and limits are now configurable in cluster resource ([#2576](https://github.com/kubermatic/kubermatic/issues/2576))
- Alerts for for control-plane components now reside in cluster namespaces ([#2583](https://github.com/kubermatic/kubermatic/issues/2583))
- Updated kube-state-metrics to 1.5.0 ([#2627](https://github.com/kubermatic/kubermatic/issues/2627))
- Updated Prometheus to v2.6.0 ([#2597](https://github.com/kubermatic/kubermatic/issues/2597))
- Updated alertmanager to v0.16 ([#2661](https://github.com/kubermatic/kubermatic/issues/2661))
- Updated Grafana to v5.4.3 ([#2662](https://github.com/kubermatic/kubermatic/issues/2662))
- Updated node-exporter to v0.17 (**note: breaking changes to metric names might require updates to customized dashboards**) ([#2666](https://github.com/kubermatic/kubermatic/issues/2666))
- Updated Minio to RELEASE.2019-01-16T21-44-08Z ([#2667](https://github.com/kubermatic/kubermatic/issues/2667))
- metrics-server will use 2 replicas ([#2707](https://github.com/kubermatic/kubermatic/issues/2707))

### Security

- The admin token can no longer be read through the Kubermatic API. ([#2105](https://github.com/kubermatic/kubermatic/issues/2105))
- Communicating with cloud providers through the project APIs no longer requires providing additional credentials. ([#2180](https://github.com/kubermatic/kubermatic/issues/2180))
- Kubernetes will be automatically updated to versions that contain a fix for CVE-2018-1002105 ([#2478](https://github.com/kubermatic/kubermatic/issues/2478))

### Bugfixes

- Missing upgrade paths for K8S 1.10 and 1.11 have been added. ([#2159](https://github.com/kubermatic/kubermatic/issues/2159))
- Fixed migration of users from older versions of Kubermatic ([#2294](https://github.com/kubermatic/kubermatic/issues/2294))
- Updated machine-controller to `v0.9.9`Fixed a bug in the machine-migration that caused cloud provider instances to not be properly identified anymore ([#2307](https://github.com/kubermatic/kubermatic/issues/2307))
- Fixd missing permissions in kube-state-metrics ClusterRole ([#2366](https://github.com/kubermatic/kubermatic/issues/2366))
- Missing ca-certificates have been added to s3-exporter image ([#2464](https://github.com/kubermatic/kubermatic/issues/2464))
- Adedd missing configmap checksums to kubermatic-controller-manager chart ([#2492](https://github.com/kubermatic/kubermatic/issues/2492))
- cloud-config files are now properly escaped ([#2498](https://github.com/kubermatic/kubermatic/issues/2498))
- SSH keys can no longer be added with duplicate names ([#2499](https://github.com/kubermatic/kubermatic/issues/2499))
- Fixed an issue with kubelets being unreachable by the apiserver on some OS configurations. ([#2522](https://github.com/kubermatic/kubermatic/issues/2522))
- Timestamp format has been unified throughout the Kubermatic API. ([#2534](https://github.com/kubermatic/kubermatic/issues/2534))
- Updated cert-manager to fix an issue which caused re-issuing of a certificate via the http01 challenge to fail ([#2658](https://github.com/kubermatic/kubermatic/issues/2658))
- Nodes and NodeDeployments can no longer be configured to provision kubelets at versions incompatible with the control plane. ([#2665](https://github.com/kubermatic/kubermatic/issues/2665))




# Kubermatic 2.8

## v2.8.6

- Added support for Kubernetes v1.13 ([#2628](https://github.com/kubermatic/kubermatic/issues/2628))
- Fixed reconciling of deep objects ([#2630](https://github.com/kubermatic/kubermatic/issues/2630))




## v2.8.5

- Added Kubernetes 1.11.6 and 1.12.4 to supported versions ([#2538](https://github.com/kubermatic/kubermatic/issues/2538))




## v2.8.4

- Fixed an issue with kubelets being unreachable by the apiserver on some OS configurations. ([#2522](https://github.com/kubermatic/kubermatic/issues/2522))




## v2.8.3

### Supported Kubernetes versions

- `1.10.11`
- `1.11.5`
- `1.12.3`

### Misc

- Kubermatic now validates nodes synchronously ([#2340](https://github.com/kubermatic/kubermatic/issues/2340))
- Components will no longer be shown as as unhealthy when only some replicas are up ([#2358](https://github.com/kubermatic/kubermatic/issues/2358))
- Disabled debug logs on control plane components ([#2471](https://github.com/kubermatic/kubermatic/issues/2471))
- Kubernetes will be automatically updated to versions that contain a fix for CVE-2018-1002105 ([#2478](https://github.com/kubermatic/kubermatic/issues/2478))
- Updated the machine-controller to v0.10.3 ([#2479](https://github.com/kubermatic/kubermatic/issues/2479))

### Bugfixes

- Fixed missing permissions in kube-state-metrics ClusterRole ([#2366](https://github.com/kubermatic/kubermatic/issues/2366))




## v2.8.2

- Fixed migration of users from older versions of Kubermatic ([#2294](https://github.com/kubermatic/kubermatic/issues/2294))
- Fixed a bug in the machine-migration that caused cloud provider instances to not be properly identified anymore ([#2307](https://github.com/kubermatic/kubermatic/issues/2307))
- Increased etcd readiness check timeout ([#2312](https://github.com/kubermatic/kubermatic/issues/2312))
- Updated machine-controller to v0.9.9




## v2.8.1

### Misc

- Prometheus is now scraping user clusters ([#2219](https://github.com/kubermatic/kubermatic/issues/2219))
- Updated the Kubermatic dashboard to v1.0.2 ([#2263](https://github.com/kubermatic/kubermatic/issues/2263))
- Update machine controller to v0.9.8 ([#2275](https://github.com/kubermatic/kubermatic/issues/2275))

### Dashboard

- Removed Container Runtime selection, which is no longer supported. ([#828](https://github.com/kubermatic/dashboard/issues/828))
- Various minor visual improvements




## v2.8.0

### Supported Kubernetes versions

- `1.9.0` - `1.9.10`
- `1.10.0` - `1.10.8`
- `1.11.0` - `1.11.3`
- `1.12.0` - `1.12.1`

### Major changes

- Implemented user/project management
- Old clusters will be automatically migrated to each user's default project ([#1829](https://github.com/kubermatic/kubermatic/issues/1829))
- Kubermatic now supports Kubernetes 1.12 ([#2132](https://github.com/kubermatic/kubermatic/issues/2132))

### Dashboard

- The UI has been reworked for the new user/project management
- Fixed error appearing when trying to change selected OS ([#699](https://github.com/kubermatic/dashboard/issues/699))
- Openstack: fixed an issue, where list of tenants wouldn't get loaded when returning from summary page ([#705](https://github.com/kubermatic/dashboard/issues/705))
- Fixed confirmation of cluster deletion ([#718](https://github.com/kubermatic/dashboard/issues/718))
- Fixed the link to Kubernetes dashboard ([#740](https://github.com/kubermatic/dashboard/issues/740))
- Openstack: show selected image in cluster creation summary ([#698](https://github.com/kubermatic/dashboard/issues/698))
- vSphere: custom cluster vnet can now be selected ([#708](https://github.com/kubermatic/dashboard/issues/708))
- Openstack: the list of available networks and floating IP pools will be loaded from the API ([#737](https://github.com/kubermatic/dashboard/issues/737))
- Dashboard metrics can now be collected by Prometheus ([#678](https://github.com/kubermatic/dashboard/issues/678))
- Redesigned cluster creation summary page ([#688](https://github.com/kubermatic/dashboard/issues/688))
- Default template images for Openstack and vSphere are now taken from datacenter configuration ([#689](https://github.com/kubermatic/dashboard/issues/689))
- Fixed cluster settings view for Openstack ([#746](https://github.com/kubermatic/dashboard/issues/746))
- "Upgrade Cluster" link is no longer available for clusters that have no updates available or are not ready ([#750](https://github.com/kubermatic/dashboard/issues/750))
- Fixed initial nodes data being lost when the browser tab was closed right after cluster creation ([#796](https://github.com/kubermatic/dashboard/issues/796))
- Google Analytics code can now be optionally added by the administrator ([#742](https://github.com/kubermatic/dashboard/issues/742))
- OpenStack tenant can now be either chosen from dropdown or typed in by hand ([#759](https://github.com/kubermatic/dashboard/issues/759))
- vSphere: Network can now be selected from a list ([#771](https://github.com/kubermatic/dashboard/issues/771))
- Login token is now removed from URL for security reasons ([#790](https://github.com/kubermatic/dashboard/issues/790))
- `Admin` button has been removed from `Certificates and Keys` panel as it allowed to copy the admin token into the clipboard. Since this is a security concern we decided to remove this functionality. ([#800](https://github.com/kubermatic/dashboard/issues/800))
- Notifications timeout is now 10s
- Close and copy to clipboard actions are available on notifications. ([#798](https://github.com/kubermatic/dashboard/issues/798))
- Provider-specific data will now be fetched without re-sending credentials. ([#814](https://github.com/kubermatic/dashboard/issues/814))
- Various minor fixes and improvements

### Bugfixes

- Kubernetes aggregation layer now uses a dedicated CA ([#1787](https://github.com/kubermatic/kubermatic/issues/1787))
- fixed DNS/scheduler/controller-manager alerts in Prometheus ([#1908](https://github.com/kubermatic/kubermatic/issues/1908))
- fixed bad rules.yaml format for Prometheus ([#1924](https://github.com/kubermatic/kubermatic/issues/1924))
- Add missing RoleBinding for bootstrap tokens created with `kubeadm token create` ([#1943](https://github.com/kubermatic/kubermatic/issues/1943))
- Fixed handling of very long user IDs ([#2075](https://github.com/kubermatic/kubermatic/issues/2075))
- The API server will redact sensitive data from its legacy API responses. ([#2079](https://github.com/kubermatic/kubermatic/issues/2079)), ([#2087](https://github.com/kubermatic/kubermatic/issues/2087))
- Missing upgrade paths for K8S 1.10 and 1.11 have been added. ([#2159](https://github.com/kubermatic/kubermatic/issues/2159))

### Misc

- Added a controller for static ip address management ([#1616](https://github.com/kubermatic/kubermatic/issues/1616))
- Activated kubelet certificate rotation feature flags ([#1771](https://github.com/kubermatic/kubermatic/issues/1771))
- Made s3-exporter endpoint configurable ([#1772](https://github.com/kubermatic/kubermatic/issues/1772))
- etcd StatefulSet uses default timings again ([#1776](https://github.com/kubermatic/kubermatic/issues/1776))
- Breaking change: basic auth for kibana/grafana/prometheus/alertmanager has been replaced with oAuth ([#1808](https://github.com/kubermatic/kubermatic/issues/1808))
- Added a controller which steers control plane traffic to the kubelets via VPN.  ([#1817](https://github.com/kubermatic/kubermatic/issues/1817))
- Fixed a memory leak which occurs when using credentials for a container registry. ([#1850](https://github.com/kubermatic/kubermatic/issues/1850))
- Combined ImagePullSecrets im the Kubermatic chart ([#1877](https://github.com/kubermatic/kubermatic/issues/1877))
- Include cluster name as label on each pod ([#1891](https://github.com/kubermatic/kubermatic/issues/1891))
- Ark-based seed-cluster backup infrastructure ([#1894](https://github.com/kubermatic/kubermatic/issues/1894))
- Add AntiAffinity to the control plane pods to prevent scheduling of the same kind pod on the same node. ([#1895](https://github.com/kubermatic/kubermatic/issues/1895))
- Enabled etcd auto-compaction ([#1932](https://github.com/kubermatic/kubermatic/issues/1932))
- etcd in user cluster namespaces is defragmented every 3 hours ([#1935](https://github.com/kubermatic/kubermatic/issues/1935))
- DNS names are now used inside the cluster namespaces, Scoped to the cluster namespace ([#1959](https://github.com/kubermatic/kubermatic/issues/1959))
- Increased kubectl timeouts on AWS  ([#1983](https://github.com/kubermatic/kubermatic/issues/1983))
- Support for Kubernetes v1.8 has been dropped. The control planes of all clusters running 1.8 will be automatically updated ([#2013](https://github.com/kubermatic/kubermatic/issues/2013))
- OpenVPN status is now a part of cluster health ([#2038](https://github.com/kubermatic/kubermatic/issues/2038))
- Improved detection of user-cluster apiserver health on startup ([#2052](https://github.com/kubermatic/kubermatic/issues/2052))
- Kubermatic now uses the types from the [cluster api project](https://github.com/kubernetes-sigs/cluster-api) to manage nodes ([#2056](https://github.com/kubermatic/kubermatic/issues/2056))
- CPU&Memory limit for the Kubermatic controller manager deployment has been increased ([#2081](https://github.com/kubermatic/kubermatic/issues/2081))
- controller-manager and its controllers will no longer run with cluster-admin permissions ([#2096](https://github.com/kubermatic/kubermatic/issues/2096))
- PodDisruptionBudget is now configured for the API server deployment ([#2098](https://github.com/kubermatic/kubermatic/issues/2098))
- The kubermatic-master chart has been merged into the main kubermatic chart ([#2103](https://github.com/kubermatic/kubermatic/issues/2103))
- Version v1.11.0 - 1.11.3 Clusters will no longer gather `rest_*` metrics from the controller-manager due to a [bug in kubernetes](https://github.com/kubernetes/kubernetes/pull/68530) ([#2020](https://github.com/kubermatic/kubermatic/issues/2020))
- Communicating with cloud providers through the non-project APIs no longer requires providing additional credentials. ([#2156](https://github.com/kubermatic/kubermatic/issues/2156))
- Communicating with cloud providers through the project APIs no longer requires providing additional credentials. ([#2227](https://github.com/kubermatic/kubermatic/issues/2227))
- Updated dashboard to v1.0.1 ([#2228](https://github.com/kubermatic/kubermatic/issues/2228))
- Updated kubernetes-dashboard addon to 1.10.0 ([#1874](https://github.com/kubermatic/kubermatic/issues/1874))
- Updated nginx ingress controller to 0.18.0 ([#1800](https://github.com/kubermatic/kubermatic/issues/1800))
- Updated etcd to v3.3.9 ([#1961](https://github.com/kubermatic/kubermatic/issues/1961))
- Updated machine-controller to v0.9.5 ([#2224](https://github.com/kubermatic/kubermatic/issues/2224))
- updated cert-manager to 0.4.1 ([#1925](https://github.com/kubermatic/kubermatic/issues/1925))
- Updated Prometheus to v2.3.2 ([#1830](https://github.com/kubermatic/kubermatic/issues/1830))
- Updated dex to 2.11.0 ([#1986](https://github.com/kubermatic/kubermatic/issues/1986))
- Updated kube-proxy addon to match the cluster version ([#2017](https://github.com/kubermatic/kubermatic/issues/2017))

### Monitoring

- Grafana dashboards now use the latest kubernetes-mixin dashboards. ([#1705](https://github.com/kubermatic/kubermatic/issues/1705))
- nginx ingress controller metrics are now scraped ([#1777](https://github.com/kubermatic/kubermatic/issues/1777))
- annotations will be used instead of labels for the nginx-ingress Prometheus configuration ([#1823](https://github.com/kubermatic/kubermatic/issues/1823))
- `KubePersistentVolumeFullInFourDays` will only be predicted when there is at least 6h of historical data available ([#1862](https://github.com/kubermatic/kubermatic/issues/1862))
- reorganized Grafana dashboards, including etcd dashboard ([#1775](https://github.com/kubermatic/kubermatic/issues/1775))
- customizations of Grafana dashboard providers, datasources and dashboards themselves are now easier ([#1812](https://github.com/kubermatic/kubermatic/issues/1812))
- new Prometheus and Kubernetes Volumes dashboards ([#1838](https://github.com/kubermatic/kubermatic/issues/1838))
- Prometheus in the seed cluster can now be customized by extending the Helm chart's `values.yaml` ([#1801](https://github.com/kubermatic/kubermatic/issues/1801))
- Prometheus alerts can now be customized in cluster namespaces ([#1831](https://github.com/kubermatic/kubermatic/issues/1831))
- Added a way to customize scraping configs for in-cluster-namespace-prometheuses ([#1837](https://github.com/kubermatic/kubermatic/issues/1837))




# Kubermatic 2.7

## v2.7.8

### Supported Kubernetes versions

- `1.10.11`
- `1.11.5`

### Major changes

- Communicating with cloud providers APIs no longer requires providing additional credentials. ([#2151](https://github.com/kubermatic/kubermatic/issues/2151))
- Updated the kubermatic dashboard to v0.38.0 ([#2165](https://github.com/kubermatic/kubermatic/issues/2165))
  - Provider-specific data will now be fetched without re-sending credentials. ([#806](https://github.com/kubermatic/dashboard/issues/806))
- Kubernetes will be automatically updated to versions that contain a fix for CVE-2018-1002105 and v1.8, v1.9 cluster creation is now dropped ([#2487](https://github.com/kubermatic/kubermatic/issues/2487))




## v2.7.7

### Misc

- Removed functionality to copy the admin token in the dashboard ([#2083](https://github.com/kubermatic/kubermatic/issues/2083))




## v2.7.6

### Misc

- Various minor fixes and improvements




## v2.7.5

### Bugfixes

- Fixed handling of very long user IDs ([#2070](https://github.com/kubermatic/kubermatic/issues/2070))




## v2.7.4


### Bugfixes

- Updated machine controller to `v0.7.23`: write permissions on vSphere datacenters are no longer needed. ([#2069](https://github.com/kubermatic/kubermatic/issues/2069))




## v2.7.3


### Misc

- kube-proxy addon was updated to match the cluster version [#2019](https://github.com/kubermatic/kubermatic/issues/2019)




## v2.7.2

### Monitoring

- `KubePersistentVolumeFullInFourDays` will only be predicted when there is at least 6h of historical data available ([#1862](https://github.com/kubermatic/kubermatic/issues/1862))

### Misc

- Updated machine-controller to v0.7.22 ([#1999](https://github.com/kubermatic/kubermatic/issues/1999))




## v2.7.1

### Bugfixes

- fixed DNS/scheduler/controller-manager alerts in Prometheus ([#1908](https://github.com/kubermatic/kubermatic/issues/1908))
- fix bad rules.yaml format for Prometheus ([#1924](https://github.com/kubermatic/kubermatic/issues/1924))
- Add missing RoleBinding for bootstrap tokens created with `kubeadm token create` ([#1943](https://github.com/kubermatic/kubermatic/issues/1943))
- Fix bug with endless resource updates being triggered due to a wrong comparison ([#1964](https://github.com/kubermatic/kubermatic/issues/1964))
- Fix escaping of special characters in the cloud-config ([#1976](https://github.com/kubermatic/kubermatic/issues/1976))

### Misc

- Update kubernetes-dashboard addon to 1.10.0 ([#1874](https://github.com/kubermatic/kubermatic/issues/1874))
- Update machine-controller to v0.7.21 ([#1975](https://github.com/kubermatic/kubermatic/issues/1975))




## v2.7.0

### Bugfixes

- Fixed a rare issue with duplicate entries on the list of nodes ([#1391](https://github.com/kubermatic/kubermatic/issues/1391))
- Fixed deletion of old etcd backups ([#1394](https://github.com/kubermatic/kubermatic/issues/1394))
- Fix deadlock during backup cleanup when the etcd of the cluster never reached a healthy state. ([#1612](https://github.com/kubermatic/kubermatic/issues/1612))
- Use dedicated CA for Kubernetes aggregation layer ([#1787](https://github.com/kubermatic/kubermatic/issues/1787))

### Cloud Provider

- Non-ESXi vsphere hosts are now supported ([#1306](https://github.com/kubermatic/kubermatic/issues/1306))
- VSphere target folder will be properly cleaned up on cluster deletion. ([#1314](https://github.com/kubermatic/kubermatic/issues/1314))
- Fixed floating IP defaulting on openstack ([#1332](https://github.com/kubermatic/kubermatic/issues/1332))
- Azure: added multi-AZ node support ([#1354](https://github.com/kubermatic/kubermatic/issues/1354))
- Fixed premature logout from vsphere API ([#1373](https://github.com/kubermatic/kubermatic/issues/1373))
- Image templates can now be configured in datacenter.yaml for Openstack and vSphere ([#1397](https://github.com/kubermatic/kubermatic/issues/1397))
- AWS: allow multiple clusters per subnet/VPC ([#1481](https://github.com/kubermatic/kubermatic/issues/1481))
- In a VSphere DC is is now possible to set a `infra_management_user` which when set will automatically be used for everything except the cloud provider functionality for all VSphere clusters in that DC. ([#1592](https://github.com/kubermatic/kubermatic/issues/1592))
- Always allocate public IP on new machines when using Azure ([#1644](https://github.com/kubermatic/kubermatic/issues/1644))
- Add missing cloud provider flags on the apiserver and controller-manager for azure ([#1646](https://github.com/kubermatic/kubermatic/issues/1646))
- Azure: fixed minor issue with seed clusters running on Azure ([#1657](https://github.com/kubermatic/kubermatic/issues/1657))
- Create AvailabilitySet for Azure clusters and set it for each machine ([#1661](https://github.com/kubermatic/kubermatic/issues/1661))
- OpenStack LoadBalancer manage-security-groups setting is set into cluster's cloud-config for Kubernetes versions where https://github.com/kubernetes/kubernetes/issues/58145 is fixed. ([#1720](https://github.com/kubermatic/kubermatic/issues/1720))

### Dashboard

- Fixed cluster settings view for Openstack ([#746](https://github.com/kubermatic/dashboard/issues/746))
- Fixed error appearing when trying to change selected OS ([#699](https://github.com/kubermatic/dashboard/issues/699))
- Openstack: fixed an issue, where list of tenants wouldn't get loaded when returning from summary page ([#705](https://github.com/kubermatic/dashboard/issues/705))
- Fixed confirmation of cluster deletion ([#718](https://github.com/kubermatic/dashboard/issues/718))
- Fixed the link to Kubernetes dashboard ([#740](https://github.com/kubermatic/dashboard/issues/740))
- vSphere: custom cluster vnet can now be selected ([#708](https://github.com/kubermatic/dashboard/issues/708))
- Openstack: the list of available networks and floating IP pools will be loaded from the API ([#737](https://github.com/kubermatic/dashboard/issues/737))
- Dashboard metrics can now be collected by Prometheus ([#678](https://github.com/kubermatic/dashboard/issues/678))
- Redesigned cluster creation summary page ([#688](https://github.com/kubermatic/dashboard/issues/688))
- Default template images for Openstack and vSphere are now taken from datacenter configuration ([#689](https://github.com/kubermatic/dashboard/issues/689))
- Various minor fixes and improvements

### Misc

- Control plane can now reach the nodes via VPN ([#1234](https://github.com/kubermatic/kubermatic/issues/1234))
- Addons in kubermatic charts can now be specified as a list ([#1304](https://github.com/kubermatic/kubermatic/issues/1304))
- Added support for Kubernetes 1.8.14, 1.9.8, 1.9.9, 1.10.4 and 1.10.5 ([#1348](https://github.com/kubermatic/kubermatic/issues/1348))
- Add support for Kubernetes 1.9.10, 1.10.6 and 1.11.1 ([#1712](https://github.com/kubermatic/kubermatic/issues/1712))
- Enabled Mutating/Validating Admission Webhooks for K8S 1.9+ ([#1352](https://github.com/kubermatic/kubermatic/issues/1352))
- Update addon manager to v0.1.0 ([#1363](https://github.com/kubermatic/kubermatic/issues/1363))
- Master components can now talk to cluster DNS ([#1379](https://github.com/kubermatic/kubermatic/issues/1379))
- Non-default IP can now be used for cluster DNS ([#1393](https://github.com/kubermatic/kubermatic/issues/1393))
- SSH key pair can now be detached from a cluster ([#1395](https://github.com/kubermatic/kubermatic/issues/1395))
- Removed Kubermatic API v2 ([#1409](https://github.com/kubermatic/kubermatic/issues/1409))
- Added EFK stack in seed clusters ([#1430](https://github.com/kubermatic/kubermatic/issues/1430))
- Fixed some issues with eleasticsearch ([#1484](https://github.com/kubermatic/kubermatic/issues/1484))
- Master components will now talk to the apiserver over secure port ([#1486](https://github.com/kubermatic/kubermatic/issues/1486))
- Added support for Kubernetes version 1.11.0 ([#1493](https://github.com/kubermatic/kubermatic/issues/1493))
- Clients will now talk to etcd over TLS ([#1495](https://github.com/kubermatic/kubermatic/issues/1495))
- Communication between apiserver and etcd is now encrypted ([#1496](https://github.com/kubermatic/kubermatic/issues/1496))
- With the introduction of Kubermatic's addon manager, the K8S addon manager's deployments will be automatically cleaned up on old setups ([#1513](https://github.com/kubermatic/kubermatic/issues/1513))
- controller-manager will now automatically restart on backup config change ([#1548](https://github.com/kubermatic/kubermatic/issues/1548))
- The control plane now has its own DNS resolver ([#1549](https://github.com/kubermatic/kubermatic/issues/1549))
- apiserver will now automatically restart on master-files change ([#1552](https://github.com/kubermatic/kubermatic/issues/1552))
- Add missing reconciling of the OpenVPN config inside the user cluster ([#1605](https://github.com/kubermatic/kubermatic/issues/1605))
- Add pod anti-affinity for the etcd StatefulSet ([#1607](https://github.com/kubermatic/kubermatic/issues/1607))
- Add PodDisruptionBudget for the etcd StatefulSet ([#1608](https://github.com/kubermatic/kubermatic/issues/1608))
- Add support for configuring component settings(Replicas & Resources) via the cluster object ([#1636](https://github.com/kubermatic/kubermatic/issues/1636))
- Update nodeport-proxy to v1.2 ([#1640](https://github.com/kubermatic/kubermatic/issues/1640))
- Added  access to the private quay.io repos from the kubermatic helm template ([#1652](https://github.com/kubermatic/kubermatic/issues/1652))
- the correct default StorageClass is now installed into the user cluster via an extra addon ([#1670](https://github.com/kubermatic/kubermatic/issues/1670))
- Update machine-controller to v0.7.18 ([#1708](https://github.com/kubermatic/kubermatic/issues/1708))
- Add possibility to override the seed DNS name for a given node datacenter via the datacenters.yaml ([#1715](https://github.com/kubermatic/kubermatic/issues/1715))
- Heapster is replaced by metrics-server. ([#1730](https://github.com/kubermatic/kubermatic/issues/1730))
- Combine the two existing CA secrets into a single one ([#1732](https://github.com/kubermatic/kubermatic/issues/1732))
- It is now possible to customize user cluster configmaps/secrets via a `MutatingAdmissionWebhook` ([#1740](https://github.com/kubermatic/kubermatic/issues/1740))
- Make s3-exporter endpoint configurable ([#1772](https://github.com/kubermatic/kubermatic/issues/1772))
- Update nginx ingress controller to 0.18.0 ([#1800](https://github.com/kubermatic/kubermatic/issues/1800))

### Monitoring

- Fixed metric name for addon controller ([#1323](https://github.com/kubermatic/kubermatic/issues/1323))
- Error metrics are now collected for Kubermatic API endpoints ([#1376](https://github.com/kubermatic/kubermatic/issues/1376))
- Prometheus is now a Statefulset ([#1399](https://github.com/kubermatic/kubermatic/issues/1399))
- Alert Manager is now a Statefulset ([#1414](https://github.com/kubermatic/kubermatic/issues/1414))
- Fixed job labels for recording rules and alerts ([#1415](https://github.com/kubermatic/kubermatic/issues/1415))
- Added official etcd alerts ([#1417](https://github.com/kubermatic/kubermatic/issues/1417))
- Added an S3 exporter for metrics ([#1482](https://github.com/kubermatic/kubermatic/issues/1482))
- Added alert rule for machines which stuck in deletion ([#1606](https://github.com/kubermatic/kubermatic/issues/1606))
- The customer cluster Prometheus inside its namespace alerts on its own now. ([#1703](https://github.com/kubermatic/kubermatic/issues/1703))
- Add kube-state-metrics to the cluster namespace ([#1716](https://github.com/kubermatic/kubermatic/issues/1716))
- Scrape nginx ingress controller metrics ([#1777](https://github.com/kubermatic/kubermatic/issues/1777))
- use annotations instead of labels for the nginx-ingress Prometheus configuration ([#1823](https://github.com/kubermatic/kubermatic/issues/1823))




# Kubermatic 2.6

## v2.6.17

### Supported Kubernetes versions

- `1.10.11`

### Bugfixes

- Fixed handling of very long user IDs ([#2086](https://github.com/kubermatic/kubermatic/issues/2086))

### Misc

- Enabled the usage of Heapster for the HorizontalPodAutoscaler ([#2199](https://github.com/kubermatic/kubermatic/issues/2199))
- Kubernetes will be automatically updated to versions that contain a fix for CVE-2018-1002105 and v1.8, v1.9 cluster creation is now dropped ([#2497](https://github.com/kubermatic/kubermatic/issues/2497))




## v2.6.16

- Updated machine-controller to v0.7.18 ([#1709](https://github.com/kubermatic/kubermatic/issues/1709))
- Added support for Kubernetes 1.8.14, 1.9.8, 1.9.9, 1.9.10, 1.10.4, 1.10.5 and 1.10.6 ([#1710](https://github.com/kubermatic/kubermatic/issues/1710))




## v2.6.15

- Added addon for default StorageClass depending on a cloud provider ([#1697](https://github.com/kubermatic/kubermatic/issues/1697))




## v2.6.14

### Cloud Provider

- Azure: fixed minor issue with seed clusters running on Azure ([#1657](https://github.com/kubermatic/kubermatic/issues/1657))

### Misc

- Updated machine-controller to v0.7.17 ([#1677](https://github.com/kubermatic/kubermatic/issues/1677))




## v2.6.13

- Minor fixes for seed clusters running on Azure ([#1646](https://github.com/kubermatic/kubermatic/issues/1646))




## v2.6.11

### Cloud Provider

- Azure: public IPs will always be allocated on new machines ([#1644](https://github.com/kubermatic/kubermatic/issues/1644))

### Misc

- Updated nodeport-proxy to v1.2 ([#1640](https://github.com/kubermatic/kubermatic/issues/1640))




## v2.6.10

- Updated machine-controller to v0.7.14 ([#1635](https://github.com/kubermatic/kubermatic/issues/1635))




## v2.6.9

- controller-manager will now automatically restart on backup config change ([#1548](https://github.com/kubermatic/kubermatic/issues/1548))
- apiserver will now automatically restart on master-files change ([#1552](https://github.com/kubermatic/kubermatic/issues/1552))




## v2.6.8

- Minor fixes and improvements




## v2.6.7

- With the introduction of Kubermatic's addon manager, the K8S addon manager's deployments will be automatically cleaned up on old setups ([#1513](https://github.com/kubermatic/kubermatic/issues/1513))




## v2.6.6

- AWS: multiple clusters per subnet/VPC are now allowed ([#1481](https://github.com/kubermatic/kubermatic/issues/1481))




## v2.6.5

### Bugfixes

- Fixed a rare issue with duplicate entries on the list of nodes ([#1391](https://github.com/kubermatic/kubermatic/issues/1391))
- Fixed deletion of old etcd backups ([#1394](https://github.com/kubermatic/kubermatic/issues/1394))

### Cloud Provider

- Image templates can now be configured in datacenter.yaml for Openstack and vSphere ([#1397](https://github.com/kubermatic/kubermatic/issues/1397))

### Dashboard

- Minor visual improvements ([#684](https://github.com/kubermatic/dashboard/issues/684))
- The node list will no longer be expanded when clicking on an IP ([#676](https://github.com/kubermatic/dashboard/issues/676))
- Openstack: the tenant can now be picked from a list loaded from the API ([#679](https://github.com/kubermatic/dashboard/issues/679))
- Added a button to easily duplicate an existing node ([#675](https://github.com/kubermatic/dashboard/issues/675))
- A note has been added to the footer identifying whether the dashboard is a part of a demo system ([#682](https://github.com/kubermatic/dashboard/issues/682))
- Enabled CoreOS on Openstack ([#673](https://github.com/kubermatic/dashboard/issues/673))
- cri-o has been disabled ([#670](https://github.com/kubermatic/dashboard/issues/670))
- Node deletion can now be confirmed by pressing enter ([#672](https://github.com/kubermatic/dashboard/issues/672))

### Misc

- Non-default IP can now be used for cluster DNS ([#1393](https://github.com/kubermatic/kubermatic/issues/1393))

### Monitoring

- Error metrics are now collected for Kubermatic API endpoints ([#1376](https://github.com/kubermatic/kubermatic/issues/1376))




## v2.6.3

### Cloud Provider

- Fixed floating IP defaulting on openstack ([#1332](https://github.com/kubermatic/kubermatic/issues/1332))
- Azure: added multi-AZ node support ([#1354](https://github.com/kubermatic/kubermatic/issues/1354))
- Fixed premature logout from vsphere API ([#1373](https://github.com/kubermatic/kubermatic/issues/1373))

### Misc

- Control plane can now reach the nodes via VPN ([#1234](https://github.com/kubermatic/kubermatic/issues/1234))
- Enabled Mutating/Validating Admission Webhooks for K8S 1.9+ ([#1352](https://github.com/kubermatic/kubermatic/issues/1352))
- Updated addon manager to v0.1.0 ([#1363](https://github.com/kubermatic/kubermatic/issues/1363))
- Update machine-controller to v0.7.5 ([#1374](https://github.com/kubermatic/kubermatic/issues/1374))




## v2.6.2

- Minor fixes and improvements for Openstack support




## v2.6.1

### Cloud Provider

- Non-ESXi vsphere hosts are now supported ([#1306](https://github.com/kubermatic/kubermatic/issues/1306))
- VSphere target folder will be properly cleaned up on cluster deletion. ([#1314](https://github.com/kubermatic/kubermatic/issues/1314))

### Misc

- Addons in kubermatic charts can now be specified as a list ([#1304](https://github.com/kubermatic/kubermatic/issues/1304))
- Updated machine-controller to v0.7.3 ([#1311](https://github.com/kubermatic/kubermatic/issues/1311))

### Monitoring

- Fixed metric name for addon controller ([#1323](https://github.com/kubermatic/kubermatic/issues/1323))




## v2.6.0

### Bugfixes

- Cluster IPv6 addresses will be ignored on systems on which they are available ([#1017](https://github.com/kubermatic/kubermatic/issues/1017))
- Fixed an issue with duplicate users being sometimes created ([#990](https://github.com/kubermatic/kubermatic/issues/990))

### Cloud Provider

- Added Azure support ([#1200](https://github.com/kubermatic/kubermatic/issues/1200))
- Openstack: made cluster resource cleanup idempotent ([#961](https://github.com/kubermatic/kubermatic/issues/961))

### Misc

- Updated prometheus operator to v0.19.0 ([#1014](https://github.com/kubermatic/kubermatic/issues/1014))
- Updated dex to v2.10.0 ([#1052](https://github.com/kubermatic/kubermatic/issues/1052))
- etcd operator has been replaced with a `StatefulSet` ([#1065](https://github.com/kubermatic/kubermatic/issues/1065))
- Nodeport range is now configurable ([#1084](https://github.com/kubermatic/kubermatic/issues/1084))
- Bare-metal provider has been removed ([#1087](https://github.com/kubermatic/kubermatic/issues/1087))
- Introduced addon manager ([#1152](https://github.com/kubermatic/kubermatic/issues/1152))
- etcd data of user clusters can now be automatically backed up ([#1170](https://github.com/kubermatic/kubermatic/issues/1170))
- Updated machine-controller to v0.7.2 ([#1227](https://github.com/kubermatic/kubermatic/issues/1227))
- etcd disk size can now be configured ([#1301](https://github.com/kubermatic/kubermatic/issues/1301))
- Updated kube-state-metrics to v1.3.1 ([#933](https://github.com/kubermatic/kubermatic/issues/933))
- Added the ability to blacklist a cluster from reconciliation by the cluster-controller ([#936](https://github.com/kubermatic/kubermatic/issues/936))
- Allow disabling TLS verification in offline environments ([#968](https://github.com/kubermatic/kubermatic/issues/968))
- Updated nginx-ingress to v0.14.0 ([#983](https://github.com/kubermatic/kubermatic/issues/983))
- Kubernetes can now automatically allocate a nodeport if the default nodeport range is unavailable ([#987](https://github.com/kubermatic/kubermatic/issues/987))
- Updated nodeport-proxy to v1.1 ([#988](https://github.com/kubermatic/kubermatic/issues/988))
- Added support for Kubernetes v1.10.2 ([#989](https://github.com/kubermatic/kubermatic/issues/989))
- Various other fixes and improvements

### Monitoring

- Added alerts for kubermatic master components being down ([#1031](https://github.com/kubermatic/kubermatic/issues/1031))
- Massive amount of general improvements to alerting
