# Kubermatic 2.15

- [v2.15.0](#v2150)
- [v2.15.1](#v2151)
- [v2.15.2](#v2152)
- [v2.15.3](#v2153)
- [v2.15.4](#v2154)
- [v2.15.5](#v2155)
- [v2.15.6](#v2156)
- [v2.15.7](#v2157)
- [v2.15.8](#v2158)
- [v2.15.9](#v2159)
- [v2.15.10](#v21510)
- [v2.15.11](#v21511)
- [v2.15.12](#v21512)
- [v2.15.13](#v21513)

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
