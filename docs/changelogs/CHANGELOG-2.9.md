# Kubermatic 2.9

- [v2.9.0](#v290)
- [v2.9.1](#v291)
- [v2.9.2](#v292)
- [v2.9.3](#v293)

## [v2.9.3](https://github.com/kubermatic/kubermatic/releases/tag/v2.9.3)

- Errors when reconciling MachineDeployments and MachineSets will now result in an event on the object ([#2930](https://github.com/kubermatic/kubermatic/issues/2930))
- Missing permissions have been added to the kube-state-metrics ClusterRole ([#2978](https://github.com/kubermatic/kubermatic/issues/2978))
- Fixed invalid variable caching in Grafana dashboards ([#2992](https://github.com/kubermatic/kubermatic/issues/2992))
- Kibana is automatically initialized in new installations. ([#2995](https://github.com/kubermatic/kubermatic/issues/2995))
- Updated machine controller to v1.1.0 ([#3028](https://github.com/kubermatic/kubermatic/issues/3028))


## [v2.9.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.9.2)

* The cleanup of services of type LoadBalancer on cluster deletion was fixed and re-enabled ([#2780](https://github.com/kubermatic/kubermatic/pull/2780))
* The Kubernetes Dashboard addon was updated to 1.10.1 ([#2848](https://github.com/kubermatic/kubermatic/pull/2848))
* Joining of nodes via the BYO functionality was fixed ([#2835](https://github.com/kubermatic/kubermatic/pull/2835))
* It is now possible to configure whether Openstack security groups for LoadBalancers should be managed by Kubernetes, [check the sample `datacenters.yaml` in the docs for details](https://docs.kubermatic.com/kubermatic/v2.12/installation/install_kubermatic/manual/#defining-the-datacenters) ([#2878](https://github.com/kubermatic/kubermatic/pull/2878))
* A bug that resulted in clusters being twice in the UI overview got resolved ([#1088](https://github.com/kubermatic/dashboard/pull/1088))
* A bug that could cause the image of a NodeDeployment to be set to the default when the NodeDeployment gets edited got resolved ([#1076](https://github.com/kubermatic/dashboard/pull/1076))
* A bug that caused the version of the UI to not be shown in the footer got resolved ([#1096](https://github.com/kubermatic/dashboard/pull/1096))
* A bug that caused updating and deleting of NodeDeployments in the NodeDeployment details page not to work got resolved ([#1076](https://github.com/kubermatic/dashboard/pull/1076))
* The NodeDeployment detail view now correctly displays the node datacenter instead of the seed datacenter ([#1094](https://github.com/kubermatic/dashboard/pull/1094))
* Support for Kubernetes 1.11.8, 1.12.6, 1.13.3 and 1.13.4 was added ([#2894](https://github.com/kubermatic/kubermatic/pull/2894))


## [v2.9.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.9.1)

* The Docker version used for all new machines with CoreOS or Ubuntu has a fix for CVE-2019-573. It s advised to roll over all your worker nodes to make sure that new version is used
* It is now possible to name NodeDeployments
* A bug that caused duplicate top level keys in the values.example.yaml got fixed
* A bug that made it impossible to choose a subnet on Openstack after a network was chosen got fixed
* Scraping of 1.13 user cluster Schedulers and Controller manager now works
* Scraping of the seed clusters Scheduler and Controller manager now works
* A bug that caused spurious failures when applying the cert-manager chart was resolved
* NodeDeployment events are now shown in the UI
* It is now possible to configure the Kubernetes version of a NodeDeployment in the UI


## [v2.9.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.9.0)

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
- Kubeconfig can now be shared via a generated link from the UI; see https://docs.kubermatic.com/ for more information. ([#857](https://github.com/kubermatic/dashboard/issues/857))
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
- **ACTION REQUIRED:** Updated cert-manager to v0.6.0 (see https://cert-manager.io/docs/installation/upgrading/upgrading-0.5-0.6) ([#2674](https://github.com/kubermatic/kubermatic/issues/2674))

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
- Added missing configmap checksums to kubermatic-controller-manager chart ([#2492](https://github.com/kubermatic/kubermatic/issues/2492))
- cloud-config files are now properly escaped ([#2498](https://github.com/kubermatic/kubermatic/issues/2498))
- SSH keys can no longer be added with duplicate names ([#2499](https://github.com/kubermatic/kubermatic/issues/2499))
- Fixed an issue with kubelets being unreachable by the apiserver on some OS configurations. ([#2522](https://github.com/kubermatic/kubermatic/issues/2522))
- Timestamp format has been unified throughout the Kubermatic API. ([#2534](https://github.com/kubermatic/kubermatic/issues/2534))
- Updated cert-manager to fix an issue which caused re-issuing of a certificate via the http01 challenge to fail ([#2658](https://github.com/kubermatic/kubermatic/issues/2658))
- Nodes and NodeDeployments can no longer be configured to provision kubelets at versions incompatible with the control plane. ([#2665](https://github.com/kubermatic/kubermatic/issues/2665))
