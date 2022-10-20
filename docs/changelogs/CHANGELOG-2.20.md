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
