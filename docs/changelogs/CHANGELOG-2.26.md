# Kubermatic 2.26

- [v2.26.0](#v2260)
- [v2.26.1](#v2261)
- [v2.26.2](#v2262)
- [v2.26.3](#v2263)
- [v2.26.4](#v2264)
- [v2.26.5](#v2265)
- [v2.26.6](#v2266)
- [v2.26.7](#v2267)
- [v2.26.8](#v2268)
- [v2.26.9](#v2269)
- [v2.26.10](#v22610)

## v2.26.10

**GitHub release: [v2.26.10](https://github.com/kubermatic/kubermatic/releases/tag/v2.26.10)**

### New Features

- KubeLB: KKP defaulting will now enable KubeLB for a cluster if it's enforced at the datacenter level ([#14748](https://github.com/kubermatic/kubermatic/pull/14748))

### Design

- Fix clickable documentation links in hints for disabled checkboxes ([#7434](https://github.com/kubermatic/dashboard/pull/7434))

### Bugfixes

- Fix validation error when switching expose strategy from Tunneling to LoadBalancer by clearing tunnelingAgentIP automatically ([#7422](https://github.com/kubermatic/dashboard/pull/7422))
- KubeLB: Fix a bug where enforcement on a datacenter was not enabling KubeLB for the user clusters in the dashboard ([#7453](https://github.com/kubermatic/dashboard/pull/7453))
- List all OpenStack networks in the UI wizard during cluster creation ([#7437](https://github.com/kubermatic/dashboard/pull/7437))
- Shows custom disk fields when a custom disk is configured in the Machine Deployment edit dialog ([#7415](https://github.com/kubermatic/dashboard/pull/7415))

### Updates

- Update machine-controller(MC) to [v1.60.2](https://github.com/kubermatic/machine-controller/releases/tag/v1.60.2) ([#14744](https://github.com/kubermatic/kubermatic/pull/14744))
- Update operating-system-manager(OSM) to [v1.6.7](https://github.com/kubermatic/operating-system-manager/releases/tag/v1.6.7) ([#14795](https://github.com/kubermatic/kubermatic/pull/14795))
- Update to Go 1.23.10 ([#14666](https://github.com/kubermatic/kubermatic/pull/14666),[#7449](https://github.com/kubermatic/dashboard/pull/7449))

### Cleanup

- By default the oauth2-proxy disables Dex's approval screen now. To return to the old behaviour, set `approval_prompt = "force"` for each IAP deployment in your Helm values.yaml ([#14751](https://github.com/kubermatic/kubermatic/pull/14751))


## v2.26.9

**GitHub release: [v2.26.9](https://github.com/kubermatic/kubermatic/releases/tag/v2.26.9)**

### Bugfixes

- Correctly mounts the custom CA bundle ConfigMap to fix reconciliation failures in custom CA environments ([#14575](https://github.com/kubermatic/kubermatic/pull/14575))
- Fix `--skip-seed-validation` flag on the KKP installer ([#14590](https://github.com/kubermatic/kubermatic/pull/14590))
- Fix a bug that caused network policies to not be removed from the KubeVirt infra cluster ([#14639](https://github.com/kubermatic/kubermatic/pull/14639))
- Fix a bug where CSI snapshot validating webhook was being deployed even if the CSI drivers are disabled for a cluster. When the CSI driver is disabled after cluster creation the both mentioned resources will be cleaned up now ([#14466](https://github.com/kubermatic/kubermatic/pull/14466))
- KubeLB: CCM will adjust the tenant kubeconfig to use apiserver endpoint and CA certificate from the management kubeconfig that is provided to KKP at the seed/datacenter level ([#14522](https://github.com/kubermatic/kubermatic/pull/14522))
- Use infra management user credentials (if configured) to fetch data for vSphere ([#7397](https://github.com/kubermatic/dashboard/pull/7397))


## v2.26.8

**GitHub release: [v2.26.8](https://github.com/kubermatic/kubermatic/releases/tag/v2.26.8)**

### ACTION REQUIRED

- Update cert-manager to v1.16.5. In the cert-manager values.yaml, following updates should be done ([#14400](https://github.com/kubermatic/kubermatic/pull/14400))
    - update  `webhook.replicas` to `webhook.replicaCount`
    - update  `cainjector.replicas` to `webhook.replicaCount`
    - remove `webhook.injectAPIServerCA`

### Supported Kubernetes versions

- Add 1.31.8/1.30.12 to the list of supported Kubernetes releases ([#14395](https://github.com/kubermatic/kubermatic/pull/14395))

### Bugfixes

- Fix a bug for KubeLB where disabling the ingress class for a user cluster was not working ([#14396](https://github.com/kubermatic/kubermatic/pull/14396))
- Add role prioritization: Update logic to return the highest-priority role for members with multiple roles ([#7272](https://github.com/kubermatic/dashboard/pull/7272))
- Add special characters restriction on Inputs and escape values to avoid rendering as HTML ([#7288](https://github.com/kubermatic/dashboard/pull/7288))

### Updates

- Security: Update Cilium to 1.15.16 because the previous version is affected by CVE-2025-32793 ([#14435](https://github.com/kubermatic/kubermatic/pull/14435))
- Update oauth2-proxy to v7.8.2 ([#14392](https://github.com/kubermatic/kubermatic/pull/14392))
- Update OSM version to [v1.6.5](https://github.com/kubermatic/operating-system-manager/releases/tag/v1.6.5) ([#14413](https://github.com/kubermatic/kubermatic/pull/14413))
- Update KubeLB CCM to [v1.1.4](https://docs.kubermatic.com/kubelb/v1.1/release-notes/#v114) ([#14365](https://github.com/kubermatic/kubermatic/pull/14365))

## v2.26.7

**GitHub release: [v2.26.7](https://github.com/kubermatic/kubermatic/releases/tag/v2.26.7)**

### Updates

- Add 1.31.7/1.30.11 to the list of supported Kubernetes releases ([#14291](https://github.com/kubermatic/kubermatic/pull/14291))
- Update etcd to 3.5.17 for all supported Kubernetes releases ([#14337](https://github.com/kubermatic/kubermatic/pull/14337))
- Update OSM to [1.6.4](https://github.com/kubermatic/operating-system-manager/releases/tag/v1.6.4) ([#14333](https://github.com/kubermatic/kubermatic/pull/14333))

## v2.26.6

**GitHub release: [v2.26.6](https://github.com/kubermatic/kubermatic/releases/tag/v2.26.6)**


### Breaking Changes

- VSphere credentials are now handled properly. For existing usercluster this will change the credentials in machine-controller and osm to `infraManagementUser` and  `infraManagementPassword` instead of `username` and `password` when specified. The latter one was always mounted to the before mentioned depl- Edge Provider: Fix a bug where clusters were stuck in `creating` phase due to wrongfully waiting for Machine Controller's health status ([#14257](https://github.com/kubermatic/kubermatic/pull/14257))

### Bugfixes

- Fix a Go panic when using git-source in Applications ([#14231](https://github.com/kubermatic/kubermatic/pull/14231))
- Fix an issue where the CBSL status was not updating due to the missing cluster-backup-storage-controller in the master controller manager ([#14255](https://github.com/kubermatic/kubermatic/pull/14255))
- Update Dashboard API to use correct OSP which is selected while creating a cluster ([#7217](https://github.com/kubermatic/dashboard/pull/7217))

### Updates

- Security: Update nginx-ingress-controller to 1.11.5, fixing CVE-2025-1097, CVE-2025-1098, CVE-2025-1974, CVE-2025-24513, CVE-2025-24514 ([#14275](https://github.com/kubermatic/kubermatic/pull/14275))

## v2.26.5

**GitHub release: [v2.26.5](https://github.com/kubermatic/kubermatic/releases/tag/v2.26.5)**

### Supported Kubernetes versions

- Add 1.31.5/1.30.9/1.29.13 to the list of supported Kubernetes releases ([#14069](https://github.com/kubermatic/kubermatic/pull/14069))

### New Features

- Add KubeVirt DS in the charts repo to generate images for the mirrored images command ([#14064](https://github.com/kubermatic/kubermatic/pull/14064))

### Bugfixes

- Fix a bug where ca-bundle was not being used to communicate to minio for metering ([#14072](https://github.com/kubermatic/kubermatic/pull/14072))
- Fix node label overwriting issue with the initial Machine Deployment ([#14033](https://github.com/kubermatic/kubermatic/pull/14033))
- Include KubeVirt CCM and Fluent-Bit images in the mirror-images command ([#14063](https://github.com/kubermatic/kubermatic/pull/14063))
- Fix datacenter creation for Edge provider ([#7165](https://github.com/kubermatic/dashboard/pull/7165))
- Fix wrong GCP machine deployment values in Edit Machine Deployment dialog ([#7169](https://github.com/kubermatic/dashboard/pull/7169))
- In the cluster backup feature, fix the issue with restoring a backup that includes all namespaces and add the option to restore all namespaces from a backup ([#7168](https://github.com/kubermatic/dashboard/pull/7168))
- VSphere: fix a bug where updating replicas of machine deployments was causing machine rotation ([#7130](https://github.com/kubermatic/dashboard/pull/7130))

### Updates

- Update go-git to 5.13.0 [CVE-2025-21613, CVE-2025-21614] ([#14151](https://github.com/kubermatic/kubermatic/pull/14151))

## v2.26.4

**GitHub release: [v2.26.4](https://github.com/kubermatic/kubermatic/releases/tag/v2.26.4)**

### Bugfixes

- Fix Seed-MLA's Prometheus trying to scrape Kopia job Pods from Velero ([#14007](https://github.com/kubermatic/kubermatic/pull/14007))
- Ignore reading the storage classes from the infra cluster and only roll out the ones from the seed object ([#14023](https://github.com/kubermatic/kubermatic/pull/14023))

### Updates

- Update OSM to 1.6.2 ([#14025](https://github.com/kubermatic/kubermatic/pull/14025))

## v2.26.3

**GitHub release: [v2.26.3](https://github.com/kubermatic/kubermatic/releases/tag/v2.26.3)**

### New Features

- The download archives on GitHub now include the dependencies of all included Helm charts ([#13954](https://github.com/kubermatic/kubermatic/pull/13954))
- Update KubeVirt-CSI-Driver-Operator version ([#13996](https://github.com/kubermatic/kubermatic/pull/13996))
- KubeLB: enable gateway API and use load balancer class values will now be picked from the datacenter configuration during cluster creation ([#7055](https://github.com/kubermatic/dashboard/pull/7055))

### Design

- Add search field in options dropdown of autocomplete element ([#7069](https://github.com/kubermatic/dashboard/pull/7069))

### Bugfixes

- [EE] Fix ClusterBackupStorageLocation sync on remote seed clusters ([#13955](https://github.com/kubermatic/kubermatic/pull/13955))
- [EE] Fix kubeLB cleanup not being performed when clusters are deleted ([#13960](https://github.com/kubermatic/kubermatic/pull/13960))
- Disable `/metrics` endpoint in master/seed MLA and user cluster MLA charts for Grafana ([#13939](https://github.com/kubermatic/kubermatic/pull/13939))
- Do not add `InTree*Unregister` feature gates to clusters on Kubernetes 1.30+ ([#13983](https://github.com/kubermatic/kubermatic/pull/13983))
- Kubelb: rely only on cluster spec for `enable-gateway-api` and `use-loadbalancer-class` flags for KubeLB CCM ([#13947](https://github.com/kubermatic/kubermatic/pull/13947))
- Mount correct `ca-bundle` ConfigMap in kubermatic-seed-controller-manager Deployment on dedicated master/seed environments ([#13938](https://github.com/kubermatic/kubermatic/pull/13938))
- Remove redundant storage classes from OpenStack CSI addon ([#13920](https://github.com/kubermatic/kubermatic/pull/13920))
- Remove storage classes filtration in KubeVirt Namespaced mode ([#13985](https://github.com/kubermatic/kubermatic/pull/13985))
- The created RBAC Role for the csi-driver now grants get for VirtualMachineInstances ([#13967](https://github.com/kubermatic/kubermatic/pull/13967))
- The rollback revision is now set explicit to the last deployed revision when a helm release managed by an application installation fails and a rollback is executed to avoid errors when the last deployed revision is not the current minus 1 and history limit is set to 1 ([#13953](https://github.com/kubermatic/kubermatic/pull/13953))
- Fix an issue where selecting "Backup All Namespaces" in the create backup/schedule dialog for cluster backups caused new namespaces to be excluded ([#7037](https://github.com/kubermatic/dashboard/pull/7037))
- Fix list images in kubevirt and tinkerbell ([#7049](https://github.com/kubermatic/dashboard/pull/7049))
- Make `Domain` field optional when using application credentials for Openstack provider ([#7044](https://github.com/kubermatic/dashboard/pull/7044))

### Updates

- Update KubeVirt CSI Driver operator to v0.4.1 in KKP ([#14011](https://github.com/kubermatic/kubermatic/pull/14011))
- Update MC version to [v.1.60.1](https://github.com/kubermatic/machine-controller/releases/tag/v1.60.1) ([#14016](https://github.com/kubermatic/kubermatic/pull/14016))

### Cleanup

- CentOS removed as a supported operating system ([#13917](https://github.com/kubermatic/kubermatic/pull/13917))
- Mark domain as optional field for OpenStack preset ([#13948](https://github.com/kubermatic/kubermatic/pull/13948))
- Removal of CentOS as a supported OS since it has reached EOL ([#7039](https://github.com/kubermatic/dashboard/pull/7039))

## v2.26.2

**GitHub release: [v2.26.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.26.2)**

### Bugfixes

- Fix Cluster Backup feature not provisioning Velero inside user clusters ([#13901](https://github.com/kubermatic/kubermatic/pull/13901))
- Fix a bug where `groups` scope was missing in authentication request for kubernetes-dashboard ([#7014](https://github.com/kubermatic/dashboard/pull/7014))
- Fix initial sync for CustomOperatingSystemProfiles when creating new user clusters (follow-up to #13831) ([#13895](https://github.com/kubermatic/kubermatic/pull/13895))
- Update Cluster-Backup Velero CRDs ([#13901](https://github.com/kubermatic/kubermatic/pull/13901))

### Updates

- Update OpenStack CCM to 1.30.2 / 1.31.2 ([#13899](https://github.com/kubermatic/kubermatic/pull/13899))
- Update kubevirt CSI driver to commit 35836e0c8b68d9916d29a838ea60cdd3fc6199cf ([#13896](https://github.com/kubermatic/kubermatic/pull/13896))

### Miscellaneous

- Add e2e tests for the Cluster Backup feature ([#13901](https://github.com/kubermatic/kubermatic/pull/13901))

## v2.26.1

**GitHub release: [v2.26.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.26.1)**

### ACTION REQUIRED

- A regression in 2.26.0 started overriding the `floatingIPPool` fields of OpenStack Clusters with the default external network. If you are using a floating IP pool that is not the default external network, you might have to update `Cluster` objects manually after upgrading KKP to set the correct floating IP pool again ([#13834](https://github.com/kubermatic/kubermatic/pull/13834))

### New Features

- Bump KubeVirt CSI Driver Operator to support zone-aware topologies ([#13833](https://github.com/kubermatic/kubermatic/pull/13833))
- Support `ZoneAndRegionEnable` field in the CCM cloud config ([#13876](https://github.com/kubermatic/kubermatic/pull/13876))
- Setup KubeVirt network controller in the seed-controller-manager. ([#13858](https://github.com/kubermatic/kubermatic/pull/13858))
- Support for Kube-OVN subnet and VPCs for KubeVirt ([#6941](https://github.com/kubermatic/dashboard/pull/6941))

### Bugfixes

- Fix cluster credentials not being synced into cluster namespaces whenever a Secret is updated in the KKP namespace ([#13819](https://github.com/kubermatic/kubermatic/pull/13819))
- Fix seed controller panic while creating `nodeport-proxy-envoy` deployment for user clusters ([#13835](https://github.com/kubermatic/kubermatic/pull/13835))
- Refactor Cluster Backups ([#13807](https://github.com/kubermatic/kubermatic/pull/13807))
    - The controllers for this feature now run in the master-controller-manager and usercluster-controller-manager instead of the seed-controller-manager
    - Fix ClusterBackupStorageLocations not being synchronized from the master to seed clusters.
- [EE] Fix Cluster Backups failing because of empty label selectors ([#6971](https://github.com/kubermatic/dashboard/pull/6971))
- KubeVirt: use infra namespace from datacenter configuration, if specified ([#6964](https://github.com/kubermatic/dashboard/pull/6964))

### Updates

- Security: Update Cilium to 1.14.16 / 1.15.10 because the previous versions are affected by CVE-2024-47825 ([#13832](https://github.com/kubermatic/kubermatic/pull/13832))
- Update AWS cloud-controller-manager to 1.31.1 ([#13838](https://github.com/kubermatic/kubermatic/pull/13838))
- Support KubeVirt VolumeBindingMode in the tenant storage class ([#13821](https://github.com/kubermatic/kubermatic/pull/13821))

### Miscellaneous

- CustomOperatingSystemProfiles are now applied more consistently and earlier when creating new user clusters ([#13831](https://github.com/kubermatic/kubermatic/pull/13831))


## v2.26.0

**GitHub release: [v2.26.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.26.0)**

Before upgrading, make sure to read the [general upgrade guidelines](https://docs.kubermatic.com/kubermatic/v2.26/installation/upgrading/). Consider tweaking `seedControllerManager.maximumParallelReconciles` to ensure user cluster reconciliations will not cause resource exhaustion on seed clusters. A [full upgrade guide is available from the official documentation](https://docs.kubermatic.com/kubermatic/v2.26/installation/upgrading/upgrade-from-2.25-to-2.26/).

### Action Required

- Update to controller-runtime 0.19 / Kubernetes 1.31 dependencies ([#13621](https://github.com/kubermatic/kubermatic/pull/13621))
    - [EE] ConstraintTemplates now correctly mark the `spec.targets[].code` field as required, making it necessary to update ConstraintTemplates to the new schema. Please refer to the migration guide for more information.
- Extend web terminal options for dashboard ([#13323](https://github.com/kubermatic/kubermatic/pull/13323))
    - Introduce `WebTerminalOptions` in KubermaticSettings to configure web terminal options for the dashboard.
    - The field `enableWebTerminal` in KubermaticSettings has been deprecated in favor of `webTerminalOptions.enabled`. Please use webTerminalOptions instead
- Update Seed-MLA Alertmanager to v0.27.0; this removes the v1 API endpoints that were deprecated since 2019 ([#13264](https://github.com/kubermatic/kubermatic/pull/13264))
- Add gzip support for etcd snapshots ([#13365](https://github.com/kubermatic/kubermatic/pull/13365))
    - etcd snapshots are now gzip-compressed before being uploaded to the backup storage.
    - The default backup store container (`spec.seedController.backupStoreContainer` in the `KubermaticConfiguration` needs to upload `/backup/snapshot.db.gz` instead of `/backup/snapshot.db`; if you have customized the store container, please adjust your scripting accordingly. The `BACKUP_TO_CREATE` env variable also now contains the filename with an additional `.gz` ending.
- Update nginx-ingress-controller to 1.10.0; this release includes following breaking changes ([#13269](https://github.com/kubermatic/kubermatic/pull/13269))
    - Does not support chroot image (this will be fixed on a future minor patch release)
    - Dropped Opentracing and zipkin modules, just Opentelemetry is supported as of this release
    - Dropped support for PodSecurityPolicy
    - Dropped support for GeoIP (legacy), only GeoIP2 is supported
    - The automatically generated `NetworkPolicy` from nginx 1.9.3 is now disabled by default, refer to https://github.com/kubernetes/ingress-nginx/pull/10238 for more information
- Update cert-manager to 1.14.4; setting feature gates works slightly differently now, please consult https://cert-manager.io/docs/releases/upgrading/upgrading-1.12-1.13 for more information ([#13273](https://github.com/kubermatic/kubermatic/pull/13273))
- Updated helm-exporter to 1.2.16 and switch to using the upstream Helm chart; you must `helm delete` the old release before installing the new chart ([#13275](https://github.com/kubermatic/kubermatic/pull/13275))
- Update Dex to 2.39.1; the validation of username and password in the LDAP connector is much more strict now. Dex uses the [EscapeFilter](https://pkg.go.dev/gopkg.in/ldap.v1#EscapeFilter) function to check for special characters in credentials and prevent injections by denying such requests ([#13270](https://github.com/kubermatic/kubermatic/pull/13270))
- Update oauth2-proxy to 7.6.0; this release introduces a change to how auth routes are evaluated using the flags skip-auth-route/skip-auth-regex. The new behaviour uses the regex you specify to evaluate the full path including query parameters. For more details please read the detailed description in https://github.com/oauth2-proxy/oauth2-proxy/issues/2271 ([#13271](https://github.com/kubermatic/kubermatic/pull/13271))
- Remove OpenVPN as means to connect control planes and cluster nodes. Existing user cluster must be migrated to Konnectivity before upgrading ([#13316](https://github.com/kubermatic/kubermatic/pull/13316))
- Update KubeLB integration to support v1.1.0 ([#13661](https://github.com/kubermatic/kubermatic/pull/13661))
    - [EE] If you are using KubeLB, before upgrading to KKP 2.26, please upgrade KubeLB management cluster to [v1.1.0](https://docs.kubermatic.com/kubelb/v1.1/installation/management-cluster). This is required for KKP integration of KubeLB to be functional.
    - KubeLB integration has been upgraded to support KubeLB v1.1
    - Options to enable Gateway API and Load Balancer class have been added at seed and cluster level
- Automated migration from machine-controller user data to OSM ([#13659](https://github.com/kubermatic/kubermatic/pull/13659))
    - KKP will perform automated migrations for clusters that are using machine-controller user data to OSM
    - Migration from machine-controller user data to OSM is automated. Users can scale up/down their machines, and there won't be any hindrance. However, existing machines/nodes using MC user data will not be rotated. This is by design to avoid unnecessary node rotations, but this can also lead to a drift between the cloud-config for new and old machines. It is recommended, not mandatory, to either rotate the machines one by one or rotate the machine deployment as a whole following https://docs.kubermatic.com/kubermatic/v2.26/cheat-sheets/rollout-machinedeployment/
- Separate container image tag/tag-suffix can be set for KKP UI & KKP API ([#13274](https://github.com/kubermatic/kubermatic/pull/13274))
    - If custom image tag/tag-suffix is being used for KKP UI & the admin desires to use the same (or different) custom tag/tag-suffix for the Kubermatic API image as well, then it needs to be explicitly set in the `KubermaticConfiguration.spec.api.dockerTag/dockerTagSuffix` otherwise the default tag for the KKP version will be used
- Initial applications are created in the namespace specified in the application specification instead of `kube-system` namespace. This doesn't affect any existing clusters and only applies to newly created clusters. Users are not affected and no action is required from their side ([#13746](https://github.com/kubermatic/kubermatic/pull/13746))

### API Changes

- Bump Seed MLA Loki and Promtail ([#13281](https://github.com/kubermatic/kubermatic/pull/13281))
    - Update Seed-MLA Loki to 2.9.6; this Helm chart version now uses a slightly different configuration syntax, please change `.loki.config` into `.loki.loki`.
    - Update Seed-MLA Promtail to 2.9.3
- Add `spec.componentsOverride.operatingSystemManager` to allow overriding OSM settings and resources ([#13285](https://github.com/kubermatic/kubermatic/pull/13285))
- Loadbalancer provider (lb-provider) & loadbalancer method (lb-method) can be configured at the datacenter for openstack provider ([#13574](https://github.com/kubermatic/kubermatic/pull/13574))
- Operating System Manager is now mandatory to create a functional cluster since machine-controller user-data plugins have been removed (https://github.com/kubermatic/machine-controller/pull/1789). Thus, the Operating System Manager is now always enabled for the user clusters ([#13381](https://github.com/kubermatic/kubermatic/pull/13381))
- Webhook backend support for user cluster's apiserver audit logs ([#13436](https://github.com/kubermatic/kubermatic/pull/13436))
- Update blackbox-exporter to v0.25.0; the `proxy_connect_header` configuration structure has been changed to match Prometheus (see [PR](https://github.com/prometheus/blackbox_exporter/pull/1008)), update your `values.yaml` if you configured this option ([#13266](https://github.com/kubermatic/kubermatic/pull/13266))

### Supported Kubernetes Versions

- Add 1.30.3/1.29.7/1.28.12 to the list of supported Kubernetes releases ([#13517](https://github.com/kubermatic/kubermatic/pull/13517))
- Add Kubernetes 1.30 to EKS/AKS versions, remove 1.24, 1.25 and 1.26 from AKS ([#13443](https://github.com/kubermatic/kubermatic/pull/13443))
- Add support for Kubernetes 1.27.13, 1.28.9 and 1.29.4 (fixes CVE-2024-3177) ([#13297](https://github.com/kubermatic/kubermatic/pull/13297))
- Add Support for Kubernetes 1.30 ([#13314](https://github.com/kubermatic/kubermatic/pull/13314))
- Add support for Kubernetes 1.31 ([#13593](https://github.com/kubermatic/kubermatic/pull/13593))
- Remove support for new Kubernetes 1.27 clusters. Existing clusters can still be reconciled, but must be upgraded before upgrading to KKP 2.27 ([#13710](https://github.com/kubermatic/kubermatic/pull/13710))
- Add support for Kubernetes v1.31.1, v1.30.5, v1.29.9, v1.28.14 ([#13773](https://github.com/kubermatic/kubermatic/pull/13773))
    - Add 1.31, remove 1.27 from the list of supported Kubernetes releases on AKS and EKS

#### Supported Versions

- 1.28.9
- 1.28.14
- 1.29.4
- 1.29.9
- 1.30.5
- 1.31.1

### Cloud Providers

#### Anexia

- Update Anexia CCM to 1.5.6 ([#13501](https://github.com/kubermatic/kubermatic/pull/13501))

#### AWS

- Fix AWS nodes connectivity issue to the Metadata Service when using Cilium as the CNI (this impacted most visibly the EBS CSI driver not functioning correctly) ([#13554](https://github.com/kubermatic/kubermatic/pull/13554))
- Update AWS CCM to v1.27.9, v1.28.9, v1.29.6, v1.30.3 ([#13495](https://github.com/kubermatic/kubermatic/pull/13495))

#### Azure

- Fix `csi` Addon not applying cleanly on Azure user clusters that were created with KKP <= 2.24 ([#13250](https://github.com/kubermatic/kubermatic/pull/13250))
- Fix an issue with Azure support that prevented successful provisioning of user clusters on some Azure locations ([#13405](https://github.com/kubermatic/kubermatic/pull/13405))
- Fix Azure CCM not being reconciled because of labelling changes ([#13334](https://github.com/kubermatic/kubermatic/pull/13334))
- The azuredisk/azurefile CSI addons have been replaced with manifests based on the upstream Helm chart ([#13514](https://github.com/kubermatic/kubermatic/pull/13514))
- Update Azure CCM / cloud node manager to 1.27.18, 1.28.10, 1.29.8, 1.30.4 ([#13496](https://github.com/kubermatic/kubermatic/pull/13496))
- Change Azure load balancer SKU default value to Standard ([#13328](https://github.com/kubermatic/kubermatic/pull/13328))

#### DigitalOcean

- Update Digitalocean CCM to v0.1.54 ([#13497](https://github.com/kubermatic/kubermatic/pull/13497))

#### GCP

- Update GCP CCM to 30.0.0, 29.0.0 ([#13510](https://github.com/kubermatic/kubermatic/pull/13510))

#### Hetzner

- Update Hetzner CCM to 1.20.0 ([#13500](https://github.com/kubermatic/kubermatic/pull/13500))

#### KubeVirt

- Allow to use generic namespace name for KubeVirt in single namespace mode ([#13614](https://github.com/kubermatic/kubermatic/pull/13614))
- Kubevirt provider waits for the etcdbackups to get deleted before removing the namespace, when a cluster is deleted ([#13635](https://github.com/kubermatic/kubermatic/pull/13635))
- Allow the deployment of Kubevirt user clusters in the single namespace of the infrastructure cluster ([#13552](https://github.com/kubermatic/kubermatic/pull/13552))

#### OpenStack

- Explicitly configure OpenStack CCM with floating IP pool configured for user cluster instead of defaulting to first external network available ([#12975](https://github.com/kubermatic/kubermatic/pull/12975))
- Update OpenStack CCM to 1.30.0 ([#13498](https://github.com/kubermatic/kubermatic/pull/13498))
- Enable OpenStack config drive from seed datacenter ([#13656](https://github.com/kubermatic/kubermatic/pull/13656))
- The OpenStack provider is now reconciling user cluster cloud resources on a regular basis ([#13191](https://github.com/kubermatic/kubermatic/pull/13191))

#### VMware Cloud Director

- Upgrade VCD CSI Driver to v1.6.0 ([#13706](https://github.com/kubermatic/kubermatic/pull/13706))
    - Volume expansion has been enabled in the default storage class

#### VSphere

- `cloud-config` handling for CCM/CSI was moved from machine-controller to KKP and cleaned up; adding `Global.ip-family` field to vSphere CSI cloud-config ([#13603](https://github.com/kubermatic/kubermatic/pull/13603))
- Update vSphere CCM to 1.30.1 ([#13499](https://github.com/kubermatic/kubermatic/pull/13499))


### New Features

- Improve compatibility with cluster-autoscaler 1.27.1+: Pods using temporary volumes are now marked as evictable ([#13180](https://github.com/kubermatic/kubermatic/pull/13180))
-  Add insecure/HTTP flags to the Helm sources in the ApplicationDefinitions ([#13406](https://github.com/kubermatic/kubermatic/pull/13406))
    - Add `insecure` and `useHTTP` options to Helm sources in `ApplicationDefinitions`. This allows to configure a plaintext or self-signed connection to an `oci://...` registry.
    - `https://localhost` and `oci://localhost` URLs are now forbidden in `ApplicationDefinitions`. Since `localhost` would refer to the usercluster-controller-manager Pod, no such URLs should exist and the impact of this change should be non-existent
- Add `AddonReconciledSuccessfully` condition / `Phase` to addons ([#13257](https://github.com/kubermatic/kubermatic/pull/13257))
    - Add new `AddonReconciledSuccessfully` condition to Addon resources.
    - Add `Phase` (New/Healthy/Unhealthy) to Addon resources (for informational purpose only, integrations should rely on the individual condition statuses)
- Bump Metering to 1.2.1 ([#13185](https://github.com/kubermatic/kubermatic/pull/13185))
    - Add `format` to metering report configuration, allowing to generate JSON files instead of CSV.
    - Add `cloud-provider`, `datacenter` and `cluster-owner` columns to the generated metering reports.
- A new option to customize non-essential fields in Presets ([#13672](https://github.com/kubermatic/kubermatic/pull/13672))
- Add `AllowedOperatingSystems` option for the project. This can be used to limit the allowed operating systems for KKP projects ([#13442](https://github.com/kubermatic/kubermatic/pull/13442))
- Add `displayName` for applications, this is the name displayed on the UI ([#13331](https://github.com/kubermatic/kubermatic/pull/13331))
- Add Canal CNI version v3.27.3 ([#13239](https://github.com/kubermatic/kubermatic/pull/13239))
- Add new `kubermatic_cluster_owner` metric on seed clusters, with `cluster_name` and `user` labels ([#13194](https://github.com/kubermatic/kubermatic/pull/13194))
- Add new admin option to enable/disable etcd backups ([#13355](https://github.com/kubermatic/kubermatic/pull/13355))
- Allow to specify extra annotations for the Dex ingress ([#13188](https://github.com/kubermatic/kubermatic/pull/13188))
- Introduce annotation configuration for the dashboard in `KubermaticSettings`. A List of protected and hidden annotations can now be configured for the dashboard ([#13668](https://github.com/kubermatic/kubermatic/pull/13668))
- Introduce Cilium 1.15.3 and mitigate CVE-2024-28860 and CVE-2024-28248 in 1.14.9 and 1.13.14 ([#13241](https://github.com/kubermatic/kubermatic/pull/13241))
- KKP resources in the `kubermatic.k8c.io` API Group can be annotated with `policy.k8c.io/prevent-deletion` to make the kubermatic-webhook reject any delete attempt (even by cluster-admins). This is meant as a last resort mechanism to prevent accidental deletions by admins during maintenance on a KKP system ([#13284](https://github.com/kubermatic/kubermatic/pull/13284))
- Monitoring: introduce `signout_redirect_url` field to configure the URL to redirect the user to after signing out from Grafana ([#13313](https://github.com/kubermatic/kubermatic/pull/13313))
- Support for configuring `apiserver` service type for the user clusters ([#13562](https://github.com/kubermatic/kubermatic/pull/13562))
- Support for default and enforced applications for user clusters ([#13644](https://github.com/kubermatic/kubermatic/pull/13644))
- The image tag in the included `mla/minio-lifecycle-mgr` helm chart has been pinned from `latest` to `RELEASE.2024-03-13T23-51-57Z` ([#13199](https://github.com/kubermatic/kubermatic/pull/13199))
- Add Baremetal Provider ([#13414](https://github.com/kubermatic/kubermatic/pull/13414))
    - Add Tinkerbell Support in KKP's baremetal provider ([#13570](https://github.com/kubermatic/kubermatic/pull/13570))
- Add support for Ubuntu 24.04 ([#13815](https://github.com/kubermatic/kubermatic/pull/13815))

### Bugfixes

- Minor fixes to the veloro chart ([#13516](https://github.com/kubermatic/kubermatic/pull/13516))
    - Adds the label `name: nodeAgent` to the Velero `DaemonSet` pods.
    - The secret `velero-restic-credentials` is renamed to `velero-repo-credentials`
- `local` command in KKP installer does not check / wait for DNS anymore ([#13620](https://github.com/kubermatic/kubermatic/pull/13620))
- Add `displayName` and `scope` columns for printing the cluster templates; `kubectl get clustertemplates` will now show the actual display name and scope for the cluster templates ([#13419](https://github.com/kubermatic/kubermatic/pull/13419))
- Add images for metering prometheus to mirror-images ([#13503](https://github.com/kubermatic/kubermatic/pull/13503))
- Add images for velero and kubeLB to mirrored images list ([#13192](https://github.com/kubermatic/kubermatic/pull/13192))
- Add automated retry for Applications stuck in "pending-install" due to an ongoing bug in helm ([#13301](https://github.com/kubermatic/kubermatic/pull/13301))
- All Helm charts now use a plain semver (without leading "v") as their `version`, allowing for easier integration with Flux and other tools that do not allow leading "v" (like Helm does). Git tags and container image tags are not affected by this change ([#13268](https://github.com/kubermatic/kubermatic/pull/13268))
- The cluster-autoscaler addon now works based on the namespace instead of cluster names; all MachineDeployments in the `kube-system` namespace are scaled ([#13202](https://github.com/kubermatic/kubermatic/pull/13202))
- Deduplicate alerts in alertmanager ([#13569](https://github.com/kubermatic/kubermatic/pull/13569))
- Default storage class addon will be removed if the CSI driver (csi addon) is disabled for user cluster ([#13445](https://github.com/kubermatic/kubermatic/pull/13445))
- Enable local command for Enterprise Edition in the KKP installer ([#13333](https://github.com/kubermatic/kubermatic/pull/13333))
- Fix #13393 where externally deployed Velero CRDs are removed automatically from user user cluster ([#13396](https://github.com/kubermatic/kubermatic/pull/13396))
- Fix a bug where unrequired `cloud-config` secret was being propagated to the user clusters ([#13366](https://github.com/kubermatic/kubermatic/pull/13366))
- Fix Envoy image configured for nodeport proxy not being used for the seed's Envoy deployment ([#13225](https://github.com/kubermatic/kubermatic/pull/13225))
- Fix high CPU usage in master-controller-manager ([#13209](https://github.com/kubermatic/kubermatic/pull/13209))
- Fix increased reconcile rate for ClusterBackupStorageLocation objects on seed clusters ([#13218](https://github.com/kubermatic/kubermatic/pull/13218))
- Fix KubermaticConfiguration getting deleted when a Seed on a shared master/seed cluster is deleted ([#13585](https://github.com/kubermatic/kubermatic/pull/13585))
- Fix missing registry overwrites for cluster-backup (Velero) images, kubevirt CSI images and KubeOne jobs ([#13435](https://github.com/kubermatic/kubermatic/pull/13435))
- Fix mla-gateway Pods not reacting to renewed certificates ([#13472](https://github.com/kubermatic/kubermatic/pull/13472))
- Fix null pointer exception that occurred while KKP controllers checked whether the CSI addon is in use or not ([#13369](https://github.com/kubermatic/kubermatic/pull/13369))
- Fix runbook URL for Prometheus alerting rules ([#13657](https://github.com/kubermatic/kubermatic/pull/13657))
- Fix stale caches: After an etcd restore, all control plane components of a usercluster are now automatically restarted. A new annotation `kubermatic.k8c.io/last-restart` on Cluster objects can be used to trigger a full rolllout of a usercluster's control plane ([#13441](https://github.com/kubermatic/kubermatic/pull/13441))
- Fix telemetry agent container images not starting up ([#13289](https://github.com/kubermatic/kubermatic/pull/13289))
- Fix usercluster-ctrl-mgr spamming oldest node version in its logs ([#13440](https://github.com/kubermatic/kubermatic/pull/13440))
- Fix VPA admission-controller PDB blocking evictions ([#13515](https://github.com/kubermatic/kubermatic/pull/13515))
- Improve helm repository prefix handling for system applications; only prepend `oci://` prefix if it doesn't already exist in the specified URL ([#13336](https://github.com/kubermatic/kubermatic/pull/13336))
- Installer does not validate iap client_secrets for grafana and alertmanager the same way it does for encryption_key ([#13315](https://github.com/kubermatic/kubermatic/pull/13315))
- Restore missing bgpconfigurations CRD in Canal 3.27 ([#13505](https://github.com/kubermatic/kubermatic/pull/13505))
- Update Canal 3.27 to 3.27.4 and Canal 3.28 to 3.28.1 ([#13625](https://github.com/kubermatic/kubermatic/pull/13625))
- When the cluster-backup feature is enabled, KKP will now reconcile a ConfigMap in the `velero` namespace in user clusters. This ConfigMap is used to configure the restore helper image in order to apply KKP's image rewriting mechanism ([#13471](https://github.com/kubermatic/kubermatic/pull/13471))
- Fix an issue which prohibited users to specify custom values for Cilium system application ([#13276](https://github.com/kubermatic/kubermatic/pull/13276))
- Allow `ingressClassName` configuration in IAP ([#13716](https://github.com/kubermatic/kubermatic/pull/13716))
- Add kv-infra-namespace flag to usercluster-controller ([#13768](https://github.com/kubermatic/kubermatic/pull/13768))
- Fix failure to migrate Cilium `ApplicationInstallations` to new `valuesBlock` field ([#13736](https://github.com/kubermatic/kubermatic/pull/13736))
- Fix reconciling loop when resetting Application values to an empty value ([#13741](https://github.com/kubermatic/kubermatic/pull/13741))
- Fix TOML/YAML configuration mixup in the IAP Helm chart ([#13776](https://github.com/kubermatic/kubermatic/pull/13776))
- Fix vSphere CCM/CSI images (pre 1.28 clusters will now use a Kubermatic-managed mirror on quay.io for the images) ([#13720](https://github.com/kubermatic/kubermatic/pull/13720))
- Nvidia-gpu-operator Application now configures a name override to be installable in the default `nvidia-gpu-operator` namespace ([#13766](https://github.com/kubermatic/kubermatic/pull/13766))
- Only applicable if custom update rules in `KubermaticConfiguration.spec.versions.updates` were defined:* Custom update rules with `automaticNodeUpdate: true` and `automatic` either absent or explicitly set to "false" will be treated as automatic update rule.* All existing user clusters with a version matching the "from" version constraint of such a rule will be automatically updated to the configured target version.* New user clusters can not be created with a version matching the "from" version constraint of such a rule ([#13709](https://github.com/kubermatic/kubermatic/pull/13709))


### Updates

- Update `kubermatic/util` to Alpine 3.19 ([#13187](https://github.com/kubermatic/kubermatic/pull/13187))
- Bump Seed MLA Grafana to 10.4 ([#13223](https://github.com/kubermatic/kubermatic/pull/13223))
    - Update seed-MLA Grafana to 10.4.1
    - Update seed-MLA Grafana dashboards: more consistent styling, do not use deprecated Chart panels anymore
    - Remove all custom Grafana plugins (`grafana-piechart-panel`, `farski-blendstat-panel`, `michaeldmoore-multistat-panel` and `vonage-status-panel`): most are deprecated or soon defunct and none of the KKP dashboard use any of these panel types
- Bump usercluster/metering Prometheus to 2.51.1 ([#13306](https://github.com/kubermatic/kubermatic/pull/13306))
- Apply OCI labels to all KKP container images ([#13210](https://github.com/kubermatic/kubermatic/pull/13210))
    - Improve labels on KKP container images.
    - Update container images to Alpine 3.19
- Update MLA Alertmanager Proxy helm chart ([#13222](https://github.com/kubermatic/kubermatic/pull/13222))
    - Update Alertmanager Authorization Envoy to v1.29.2
    - Improve alertmanager-proxy Helm Chart: do not require root permissions, drop capabilities and make logging/ports configurable
- Allows KKP administrator to interface thanos query with thanos-sidecar to get full benefit of using thanos ([#13482](https://github.com/kubermatic/kubermatic/pull/13482))
- Remove support for Canal 3.8 ([#13506](https://github.com/kubermatic/kubermatic/pull/13506))
- Security: update nginx-ingress-controller to 1.11.2 (fixes CVE-2024-7646) ([#13600](https://github.com/kubermatic/kubermatic/pull/13600))
- Update `kube-state-metrics` addon to v2.13.0 ([#13599](https://github.com/kubermatic/kubermatic/pull/13599))
- Update cert-manager Helm chart to 1.15.1 ([#13494](https://github.com/kubermatic/kubermatic/pull/13494))
- Update cluster-autoscaler addon to 1.30.1, 1.29.3, 1.28.5, 1.27.8 ([#13507](https://github.com/kubermatic/kubermatic/pull/13507))
- Update configmap-reload to 0.12.0; container image is now pulled from `ghcr.io/jimmidyson/configmap-reload` instead of Docker Hub ([#13265](https://github.com/kubermatic/kubermatic/pull/13265))
- Update flatcar-linux-update-operator to 0.9.0 ([#13666](https://github.com/kubermatic/kubermatic/pull/13666))
- Update Helm version used by KKP to 3.14.3 ([#13244](https://github.com/kubermatic/kubermatic/pull/13244))
- Update Karma to v1.120 ([#13277](https://github.com/kubermatic/kubermatic/pull/13277))
- Update kube-dependencies to 0.29.3 ([#13186](https://github.com/kubermatic/kubermatic/pull/13186))
- Update kube-state-metrics to v2.12 ([#13278](https://github.com/kubermatic/kubermatic/pull/13278))
- Update node-exporter to v1.7.0 ([#13279](https://github.com/kubermatic/kubermatic/pull/13279))
- Update Prometheus to v2.51.1 ([#13280](https://github.com/kubermatic/kubermatic/pull/13280))
- Update usercluster kube-state-metrics to 2.12.0 ([#13307](https://github.com/kubermatic/kubermatic/pull/13307))
- Update Velero to v1.14.0 ([#13473](https://github.com/kubermatic/kubermatic/pull/13473))
- Update KubeLB to v1.1.2 ([#13809](https://github.com/kubermatic/kubermatic/pull/13809))
- Update oauth2-proxy to 7.7.0 ([#13788](https://github.com/kubermatic/kubermatic/pull/13788))
- Update to Go 1.23.2 ([#13789](https://github.com/kubermatic/kubermatic/pull/13789))
- Bump machine-controller to 1.60.0, OSM to 1.6.0 ([#13815](https://github.com/kubermatic/kubermatic/pull/13815))
- Add support for KubeVirt provider network ([#13791](https://github.com/kubermatic/kubermatic/pull/13791))

### Cleanup

- Add SecurityContext to KKP operator/controller-manager containers, including OSM and machine-controller ([#13282](https://github.com/kubermatic/kubermatic/pull/13282))
- Addon conditions now contain the KKP version that has last successfully reconciled the addon (similar to the Cluster conditions) ([#13519](https://github.com/kubermatic/kubermatic/pull/13519))
- Addons reconciliation is triggered more consistently for changes to Cluster objects, reducing the overall number of unnecessary addon reconciliations ([#13252](https://github.com/kubermatic/kubermatic/pull/13252))
- Fix misleading errors about undeploying the cluster-backup components from newly created user clusters ([#13403](https://github.com/kubermatic/kubermatic/pull/13403))
- Replace custom Velero Helm chart with a wrapper around the official upstream chart ([#13488](https://github.com/kubermatic/kubermatic/pull/13488))
- Replace kubernetes.io/ingress.class annotation with ingressClassName spec field ([#13549](https://github.com/kubermatic/kubermatic/pull/13549))
- S3-Exporter does not run with root permissions and does not leak credentials via CLI flags anymore ([#13226](https://github.com/kubermatic/kubermatic/pull/13226))
- Etcd container images are now loaded from registry.k8s.io instead of gcr.io/etcd-development ([#13726](https://github.com/kubermatic/kubermatic/pull/13726))

### Deprecation

- Add `spec.componentsOverride.coreDNS` to Cluster objects, deprecate `spec.clusterNetwork.coreDNSReplicas` in favor of the new `spec.componentsOverride.coreDNS.replicas` field ([#13409](https://github.com/kubermatic/kubermatic/pull/13409))
- Cilium kubeProxyReplacement values `strict`, `partial`, `probe`, and `disabled` have been deprecated, please use true or false instead ([#13291](https://github.com/kubermatic/kubermatic/pull/13291))
- Add support for Canal 3.28, deprecate Canal 3.25 ([#13504](https://github.com/kubermatic/kubermatic/pull/13504))
- Remove deprecated Cilium and Hubble KKP Addons, as Cilium CNI is managed by Applications ([#13229](https://github.com/kubermatic/kubermatic/pull/13229))
- The field `values` in ApplicationInstallation and `defaultValues` in ApplicationDefinition were deprecated in KKP 2.25 and will be removed in KKP 2.27+ ([#13747](https://github.com/kubermatic/kubermatic/pull/13747))

### Miscellaneous

- Compatibility of addons is now automatically tested against previous KKP releases to prevent addons failing to change immutable fields ([#13256](https://github.com/kubermatic/kubermatic/pull/13256))
- Fix metrics-server: correct networkpolicy port for metrics-server ([#13438](https://github.com/kubermatic/kubermatic/pull/13438))
- Metering CronJobs now use a `metering-` prefix; older jobs are automatically removed ([#13200](https://github.com/kubermatic/kubermatic/pull/13200))
- Reduce number of Helm upgrades in application-installation-controller by tracking changes to Helm chart version, values and templated manifests ([#13121](https://github.com/kubermatic/kubermatic/pull/13121))
- Add dynamic base id to envoy agent on the user cluster ([#13261](https://github.com/kubermatic/kubermatic/pull/13261))
- Utility container images like `kubermatic/util` or `kubermatic/http-prober` are now built automatically on CI instead of relying on developer intervention ([#13189](https://github.com/kubermatic/kubermatic/pull/13189))
- `kubermatic.io/initial-cni-values-request` is now included in the default hidden annotations list for the dashboard ([#13764](https://github.com/kubermatic/kubermatic/pull/13764))

### Dashboard and API

#### Cloud Providers

##### VSphere

- VSphere: Support for assigning VMs to VM groups ([#6774](https://github.com/kubermatic/dashboard/pull/6774))

#### New Features

- Support for annotations ([#6809](https://github.com/kubermatic/dashboard/pull/6809))
    - Dashboard now supports managing annotations for clusters, machine deployments, and nodes.
    - Admin settings have been introduced for annotations. Admins can hide annotations or mark them as protected/read-only
- Default/Enforced applications in the cluster wizard ([#6794](https://github.com/kubermatic/dashboard/pull/6794))
    - Highest semantic version is selected by default for applications on the dashboard
    - Default/Enforced applications are now marked and visible for user clusters
- `DisplayName` for applications is used on the UI ([#6663](https://github.com/kubermatic/dashboard/pull/6663))
- Add a `yaml` block field to add additional parameters to the `config` for the backup storage location ([#6738](https://github.com/kubermatic/dashboard/pull/6738))
- Add basic support for displaying OpenNebula machine deployments ([#6270](https://github.com/kubermatic/dashboard/pull/6270))
- Add enable/disable etcd backups feature option in admin settings ([#6681](https://github.com/kubermatic/dashboard/pull/6681))
- Add new static labels option in admin settings ([#6735](https://github.com/kubermatic/dashboard/pull/6735))
- Add Baremetal provider and Tinkerbell Support ([#6765](https://github.com/kubermatic/dashboard/pull/6765)) and ([#6764](https://github.com/kubermatic/dashboard/pull/6764))
- Add the grafana orgId parameter to Grafana UI link in dashboard ([#6617](https://github.com/kubermatic/dashboard/pull/6617))
- Audit logging backend webhook configuration for cluster and datacenter ([#6781](https://github.com/kubermatic/dashboard/pull/6781))
- Cluster Backup: CA bundle and Prefix configuration for backup storage ([#6682](https://github.com/kubermatic/dashboard/pull/6682))
- Display the used preset name on the cluster detail page ([#6705](https://github.com/kubermatic/dashboard/pull/6705))
- Enable editing allowed IP ranges for NodePorts ([#6783](https://github.com/kubermatic/dashboard/pull/6783))
- Support for configuring internet access for the web terminal ([#6668](https://github.com/kubermatic/dashboard/pull/6668))
- Support for enabling/disabling operating systems for machines in user clusters at the project level ([#6723](https://github.com/kubermatic/dashboard/pull/6723))
- Update KubeLB integration to support enabling/disabling gateway API and load balancer class ([#6810](https://github.com/kubermatic/dashboard/pull/6810))
- Admin panel settings for applications ([#6787](https://github.com/kubermatic/dashboard/pull/6787))
    - Admins can now manage applications using admin panel
    - Application can be marked as default or enforced using dashboard
- Support Kube-OVN provider networks for VPCs and Subnets ([#6915](https://github.com/kubermatic/dashboard/pull/6915))

#### Bugfixes

- Adjust the preset domain field to accept emails ([#6690](https://github.com/kubermatic/dashboard/pull/6690))
- Fix a bug where CNI was always being defaulted to Cilium irrespective of what was configured in the cluster template or default cluster template ([#6708](https://github.com/kubermatic/dashboard/pull/6708))
- Fix an issue where the cursor in web terminal kept jumping to the beginning due to sizing issue ([#6799](https://github.com/kubermatic/dashboard/pull/6799))
- Fix template value for machine deployments in edit mode ([#6669](https://github.com/kubermatic/dashboard/pull/6669))
- Fix the pagination in project members table ([#6741](https://github.com/kubermatic/dashboard/pull/6741))
- Fix TLS errors in the admin page when using a custom CA for the metering object store ([#6752](https://github.com/kubermatic/dashboard/pull/6752))
- Grant admin all owner privileges on all projects ([#6754](https://github.com/kubermatic/dashboard/pull/6754))
- Resolve conflict in determining available Kubernetes versions where upgrades where possible in `Cluster` object but not via the Dashboard ([#6651](https://github.com/kubermatic/dashboard/pull/6651))
- Support for eBPF proxy mode when the CNI plugin is none ([#6757](https://github.com/kubermatic/dashboard/pull/6757))
- Fix CNI plugin defaulting for Edge cloud provider ([#6878](https://github.com/kubermatic/dashboard/pull/6878))
- Fix default CNI application values in cluster wizard ([#6884](https://github.com/kubermatic/dashboard/pull/6884))
- Select correct template value when editing MD of VCD provider ([#6927](https://github.com/kubermatic/dashboard/pull/6927))

#### Updates

- KKP API is now built using Go 1.23.2 ([#6924](https://github.com/kubermatic/dashboard/pull/6924))
- Update to Angular version 17 ([#6639](https://github.com/kubermatic/dashboard/pull/6639))
- Update web-terminal image to v0.9.1 ([#6890](https://github.com/kubermatic/dashboard/pull/6890))

#### Cleanup

- The dialog for changelog has been removed in favor of an external URL that points to relevant changelogs ([#6631](https://github.com/kubermatic/dashboard/pull/6631))
- The option to disable the operating system manager on cluster creation has been removed ([#6683](https://github.com/kubermatic/dashboard/pull/6683))

#### Miscellaneous

- Migrate to MDC-based Angular Material Components ([#6685](https://github.com/kubermatic/dashboard/pull/6685))
