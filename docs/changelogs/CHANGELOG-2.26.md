# Kubermatic 2.26

- [v2.26.0](#v2260)


## v2.26.0

**GitHub release: [v2.26.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.26.0)**

Before upgrading, make sure to read the [general upgrade guidelines](https://docs.kubermatic.com/kubermatic/v2.26/installation/upgrading/). Consider tweaking `seedControllerManager.maximumParallelReconciles` to ensure user cluster reconciliations will not cause resource exhaustion on seed clusters. A [full upgrade guide is available from the official documentation](https://docs.kubermatic.com/kubermatic/v2.25/installation/upgrading/upgrade-from-2.25-to-2.26/).

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

#### Supported Versions

- 1.28.9
- 1.28.12
- 1.28.13
- 1.29.4
- 1.30.3
- 1.31.0

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
- Automatically add seed cluster podCIDR when APIServerAllowedIPRanges are set ([#13579](https://github.com/kubermatic/kubermatic/pull/13579))


### Bugfixes

- Minor fixes to the veloro chart ([#13516](https://github.com/kubermatic/kubermatic/pull/13516))
    - Adds the label `name: nodeAgent` to the Velero `DaemonSet` pods.
    - The secret `velero-restic-credentials` is renamed to `velero-repo-credentials`
- `local` command in KKP installer does not check / wait for DNS anymore ([#13620](https://github.com/kubermatic/kubermatic/pull/13620))
- Add `displayName` and `scope` columns for printing the cluster templates; `kubectl get clustertemplates` will now show the actual display name and scope for the cluster templates ([#13419](https://github.com/kubermatic/kubermatic/pull/13419))
- Add images for metering prometheus to mirror-images ([#13503](https://github.com/kubermatic/kubermatic/pull/13503))
- Add images for velero and kubeLB to mirrored images list ([#13192](https://github.com/kubermatic/kubermatic/pull/13192))
- Addressing inconsistencies in helm that lead to an Application stuck in "pending-install" ([#13301](https://github.com/kubermatic/kubermatic/pull/13301))
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
- Update KubeLB to v1.1.1 ([#13712](https://github.com/kubermatic/kubermatic/pull/13712))
- Update node-exporter to v1.7.0 ([#13279](https://github.com/kubermatic/kubermatic/pull/13279))
- Update Prometheus to v2.51.1 ([#13280](https://github.com/kubermatic/kubermatic/pull/13280))
- Update to Go 1.23.1 ([#13711](https://github.com/kubermatic/kubermatic/pull/13711))
- Update usercluster kube-state-metrics to 2.12.0 ([#13307](https://github.com/kubermatic/kubermatic/pull/13307))
- Update Velero Helm chart to v1.13.1 ([#13272](https://github.com/kubermatic/kubermatic/pull/13272))
- Update Velero to v1.14.0 ([#13473](https://github.com/kubermatic/kubermatic/pull/13473))

### Cleanup

- Add SecurityContext to KKP operator/controller-manager containers, including OSM and machine-controller ([#13282](https://github.com/kubermatic/kubermatic/pull/13282))
- Addon conditions now contain the KKP version that has last successfully reconciled the addon (similar to the Cluster conditions) ([#13519](https://github.com/kubermatic/kubermatic/pull/13519))
- Addons reconciliation is triggered more consistently for changes to Cluster objects, reducing the overall number of unnecessary addon reconciliations ([#13252](https://github.com/kubermatic/kubermatic/pull/13252))
- Fix misleading errors about undeploying the cluster-backup components from newly created user clusters ([#13403](https://github.com/kubermatic/kubermatic/pull/13403))
- Replace custom Velero Helm chart with a wrapper around the official upstream chart ([#13488](https://github.com/kubermatic/kubermatic/pull/13488))
- Replace kubernetes.io/ingress.class annotation with ingressClassName spec field ([#13549](https://github.com/kubermatic/kubermatic/pull/13549))
- S3-Exporter does not run with root permissions and does not leak credentials via CLI flags anymore ([#13226](https://github.com/kubermatic/kubermatic/pull/13226))

### Deprecation

- Add `spec.componentsOverride.coreDNS` to Cluster objects, deprecate `spec.clusterNetwork.coreDNSReplicas` in favor of the new `spec.componentsOverride.coreDNS.replicas` field ([#13409](https://github.com/kubermatic/kubermatic/pull/13409))
- Cilium kubeProxyReplacement values `strict`, `partial`, `probe`, and `disabled` have been deprecated, please use true or false instead ([#13291](https://github.com/kubermatic/kubermatic/pull/13291))
- Add support for Canal 3.28, deprecate Canal 3.25 ([#13504](https://github.com/kubermatic/kubermatic/pull/13504))
- Remove deprecated Cilium and Hubble KKP Addons, as Cilium CNI is managed by Applications ([#13229](https://github.com/kubermatic/kubermatic/pull/13229))

### Miscellaneous

- Compatibility of addons is now automatically tested against previous KKP releases to prevent addons failing to change immutable fields ([#13256](https://github.com/kubermatic/kubermatic/pull/13256))
- Fix metrics-server: correct networkpolicy port for metrics-server ([#13438](https://github.com/kubermatic/kubermatic/pull/13438))
- Metering CronJobs now use a `metering-` prefix; older jobs are automatically removed ([#13200](https://github.com/kubermatic/kubermatic/pull/13200))
- Reduce number of Helm upgrades in application-installation-controller by tracking changes to Helm chart version, values and templated manifests ([#13121](https://github.com/kubermatic/kubermatic/pull/13121))
- Add dynamic base id to envoy agent on the user cluster ([#13261](https://github.com/kubermatic/kubermatic/pull/13261))
- Utility container images like `kubermatic/util` or `kubermatic/http-prober` are now built automatically on CI instead of relying on developer intervention ([#13189](https://github.com/kubermatic/kubermatic/pull/13189))

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

#### Updates

- Update Go version to 1.23.1 ([#6839](https://github.com/kubermatic/dashboard/pull/6839))
- Update to Angular version 17 ([#6639](https://github.com/kubermatic/dashboard/pull/6639))

#### Cleanup

- The dialog for changelog has been removed in favor of an external URL that points to relevant changelogs ([#6631](https://github.com/kubermatic/dashboard/pull/6631))
- The option to disable the operating system manager on cluster creation has been removed ([#6683](https://github.com/kubermatic/dashboard/pull/6683))

#### Miscellaneous

- Migrate to MDC-based Angular Material Components ([#6685](https://github.com/kubermatic/dashboard/pull/6685))
