# Kubermatic 2.15

## [v2.15.0-rc.3](https://github.com/kubermatic/kubermatic/releases/tag/v2.15.0-rc.3)

- Add feature flag `etcdLauncher` around etcd-launcher ([#5973](https://github.com/kubermatic/kubermatic/issues/5973))
- Provide a way of skipping Certificate cert-manager resources ([#5962](https://github.com/kubermatic/kubermatic/issues/5962), [#5969](https://github.com/kubermatic/kubermatic/issues/5969))

## [v2.15.0-rc.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.15.0-rc.2)

- Add Kubernetes 1.16.15, 1.17.12, 1.19.2 ([#5927](https://github.com/kubermatic/kubermatic/issues/5927))
- Add `operator.kubermatic.io/skip-reconciling` annotation to Seeds to allow step-by-step seed cluster upgrades ([#5883](https://github.com/kubermatic/kubermatic/issues/5883))
- Add option to enable/disable external cluster import feature from admin settings in KKP dashboard ([#2644](https://github.com/kubermatic/dashboard/issues/2644))
- Allow controlling external cluster functionality with global settings ([#5912](https://github.com/kubermatic/kubermatic/issues/5912))
- Fix KKP Operator getting stuck in Kubernetes 1.18 clusters when reconciling Ingresses ([#5915](https://github.com/kubermatic/kubermatic/issues/5915))
- Fix creation of RHEL8 machines ([#5950](https://github.com/kubermatic/kubermatic/issues/5950))
- Fix loading of the access rights in the SSH keys view ([#2645](https://github.com/kubermatic/dashboard/issues/2645))
- Fix cluster wizard rendering in Safari ([#2661](https://github.com/kubermatic/dashboard/issues/2661))

## [v2.15.0-rc.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.15.0-rc.1)

### Supported Kubernetes Versions

* 1.16.13
* 1.16.14
* 1.16.15
* 1.17.9
* 1.17.11
* 1.17.12
* 1.18.6
* 1.18.8
* 1.19.0
* 1.19.2

### Highlights

- Add support for Kubernetes 1.19, drop Kubernetes 1.15 ([#5794](https://github.com/kubermatic/kubermatic/issues/5794))
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
- machine-controller 1.17.1 ([#5794](https://github.com/kubermatic/kubermatic/issues/5794))
- nginx-ingress-controller 0.34.1 ([#5780](https://github.com/kubermatic/kubermatic/issues/5780))
- node-exporter 1.0.1 (includes the usercluster addon) ([#5791](https://github.com/kubermatic/kubermatic/issues/5791))
- Minio RELEASE.2020-09-10T22-02-45Z ([#5854](https://github.com/kubermatic/kubermatic/issues/5854))
- Prometheus 2.20.1 ([#5781](https://github.com/kubermatic/kubermatic/issues/5781))
- Velero 1.4.2 ([#5775](https://github.com/kubermatic/kubermatic/issues/5775))




# Kubermatic 2.14

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
- kubelet sets intial machine taints via --register-with-taints ([#664](https://github.com/kubermatic/machine-controller/issues/664))
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
- Added possibility to define a default project in user settings. When a default project is choosen, the user will be automatically redirected to this project after login. Attention: One initial log in might be needed for the feature to take effect. ([#1895](https://github.com/kubermatic/dashboard-v2/issues/1895))
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
  - kubelet sets intial machine taints via --register-with-taints ([#664](https://github.com/kubermatic/machine-controller/issues/664))
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
- Fixe Seed Validation Webhook rejecting new Seeds in certain situations ([#4662](https://github.com/kubermatic/kubermatic/issues/4662))
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
- It is now possible to configure an http proxy on a Seed. This will result in the proxy being used for all control plane pods in that seed that talk to a cloudprovider and for all machines in that Seed, unless its overriden on Datacenter level. ([#4459](https://github.com/kubermatic/kubermatic/issues/4459))
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

- Openstack: A bug that could result in many securtiy groups being created when the creation of security group rules failed was fixed ([#3848](https://github.com/kubermatic/kubermatic/issues/3848))
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

- Openstack: A bug that could result in many securtiy groups being created when the creation of security group rules failed was fixed ([#3848](https://github.com/kubermatic/kubermatic/issues/3848))
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
* Configurable Prometheus backup timeout to accomodate larger seed clusters ([#3223](https://github.com/kubermatic/kubermatic/pull/3223))

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
* A bug that made it impossible to choose a subnet on Openstack after a network was choosen got fixed
* Scraping of 1.13 user cluster Schedulers and Controller manager now works
* Scraping of the seed clusters Scheduler and Controller manager now works
* A bug that caused spurious failures when appplying the cert-manager chart was resolved
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
- Experimantal support for VerticalPodAutoscaler has been added. The VPA resources use the PodUpdatePolicy=initial ([#2505](https://github.com/kubermatic/kubermatic/issues/2505))
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

- Missing upgrade paths for K8S 1.10 and 1.11 have been addded. ([#2159](https://github.com/kubermatic/kubermatic/issues/2159))
- Fixed migration of users from older versions of Kubermatic ([#2294](https://github.com/kubermatic/kubermatic/issues/2294))
- Updated machine-controller to `v0.9.9`Fixed a bug in the machine-migration that caused cloud provider instances to not be properly identified anymore ([#2307](https://github.com/kubermatic/kubermatic/issues/2307))
- Fixd missing permissions in kube-state-metrics ClusterRole ([#2366](https://github.com/kubermatic/kubermatic/issues/2366))
- Missing ca-certificates have been added to s3-exporter image ([#2464](https://github.com/kubermatic/kubermatic/issues/2464))
- Adedd missing configmap checksums to kubermatic-controller-manager chart ([#2492](https://github.com/kubermatic/kubermatic/issues/2492))
- cloud-config files are now properly escaped ([#2498](https://github.com/kubermatic/kubermatic/issues/2498))
- SSH keys can no longer be added with duplicate names ([#2499](https://github.com/kubermatic/kubermatic/issues/2499))
- Fixed an issue with kubelets being unreachable by the apiserver on some OS configurations. ([#2522](https://github.com/kubermatic/kubermatic/issues/2522))
- Timestamp format has been unified throughout the Kubermatic API. ([#2534](https://github.com/kubermatic/kubermatic/issues/2534))
- Updated cert-manager to fix an issue which caused re-issuing of a certficate via the http01 challenge to fail ([#2658](https://github.com/kubermatic/kubermatic/issues/2658))
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
- Missing upgrade paths for K8S 1.10 and 1.11 have been addded. ([#2159](https://github.com/kubermatic/kubermatic/issues/2159))

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
- etcd in user cluser namespaces is defragmented every 3 hours ([#1935](https://github.com/kubermatic/kubermatic/issues/1935))
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
- SSH keypair can now be detached from a cluster ([#1395](https://github.com/kubermatic/kubermatic/issues/1395))
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
- Alert Manger is now a Statefulset ([#1414](https://github.com/kubermatic/kubermatic/issues/1414))
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
