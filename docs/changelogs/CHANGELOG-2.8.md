# Kubermatic 2.8

- [v2.8.0](#v280)
- [v2.8.1](#v281)
- [v2.8.2](#v282)
- [v2.8.3](#v283)
- [v2.8.4](#v284)
- [v2.8.5](#v285)
- [v2.8.6](#v286)

## [v2.8.6](https://github.com/kubermatic/kubermatic/releases/tag/v2.8.6)

- Added support for Kubernetes v1.13 ([#2628](https://github.com/kubermatic/kubermatic/issues/2628))
- Fixed reconciling of deep objects ([#2630](https://github.com/kubermatic/kubermatic/issues/2630))


## [v2.8.5](https://github.com/kubermatic/kubermatic/releases/tag/v2.8.5)

- Added Kubernetes 1.11.6 and 1.12.4 to supported versions ([#2538](https://github.com/kubermatic/kubermatic/issues/2538))


## [v2.8.4](https://github.com/kubermatic/kubermatic/releases/tag/v2.8.4)

- Fixed an issue with kubelets being unreachable by the apiserver on some OS configurations. ([#2522](https://github.com/kubermatic/kubermatic/issues/2522))


## [v2.8.3](https://github.com/kubermatic/kubermatic/releases/tag/v2.8.3)

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


## [v2.8.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.8.2)

- Fixed migration of users from older versions of Kubermatic ([#2294](https://github.com/kubermatic/kubermatic/issues/2294))
- Fixed a bug in the machine-migration that caused cloud provider instances to not be properly identified anymore ([#2307](https://github.com/kubermatic/kubermatic/issues/2307))
- Increased etcd readiness check timeout ([#2312](https://github.com/kubermatic/kubermatic/issues/2312))
- Updated machine-controller to v0.9.9


## [v2.8.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.8.1)

### Misc

- Prometheus is now scraping user clusters ([#2219](https://github.com/kubermatic/kubermatic/issues/2219))
- Updated the Kubermatic dashboard to v1.0.2 ([#2263](https://github.com/kubermatic/kubermatic/issues/2263))
- Update machine controller to v0.9.8 ([#2275](https://github.com/kubermatic/kubermatic/issues/2275))

### Dashboard

- Removed Container Runtime selection, which is no longer supported. ([#828](https://github.com/kubermatic/dashboard/issues/828))
- Various minor visual improvements


## [v2.8.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.8.0)

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
