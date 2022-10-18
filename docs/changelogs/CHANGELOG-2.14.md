# Kubermatic 2.14

- [v2.14.0](#v2140)
- [v2.14.1](#v2141)
- [v2.14.2](#v2142)
- [v2.14.3](#v2143)
- [v2.14.4](#v2144)
- [v2.14.5](#v2145)
- [v2.14.6](#v2146)
- [v2.14.7](#v2147)
- [v2.14.8](#v2148)
- [v2.14.9](#v2149)
- [v2.14.10](#v21410)
- [v2.14.11](#v21411)
- [v2.14.12](#v21412)
- [v2.14.13](#v21413)

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
[general upgrade guidelines](https://docs.kubermatic.com/kubermatic/v2.14/upgrading/) for more
information on how to limit the impact of such changes during KKP upgrades.

### Misc

- ACTION REQUIRED: Migrate from google_containers to k8s.gcr.io Docker registry ([#5986](https://github.com/kubermatic/kubermatic/issues/5986))


## [v2.14.6](https://github.com/kubermatic/kubermatic/releases/tag/v2.14.6)

### Bugfixes

- Fix creation of RHEL8 machines ([#5951](https://github.com/kubermatic/kubermatic/issues/5951))

### Misc

- Allow custom envvar definitions for Dex to be passed via the `oauth` chart, key in `values.yaml` is `dex.env` ([#5847](https://github.com/kubermatic/kubermatic/issues/5847))
- Provide a way of skipping Certificate cert-manager resources in `oauth` and `kubermatic` charts ([#5972](https://github.com/kubermatic/kubermatic/issues/5972))


## [v2.14.5](https://github.com/kubermatic/kubermatic/releases/tag/v2.14.5)

### Bugfixes

- Fallback to in-tree cloud provider for OTC ([#5778](https://github.com/kubermatic/kubermatic/issues/5778))


## [v2.14.4](https://github.com/kubermatic/kubermatic/releases/tag/v2.14.4)

### Bugfixes

- fix flaky TestGCPDiskTypes ([#5693](https://github.com/kubermatic/kubermatic/issues/5693))
- fix changing user/password for OpenStack cluster credentials ([#5691](https://github.com/kubermatic/kubermatic/issues/5691))
- fix componentsOverride affecting default values when reconciling clusters ([#5704](https://github.com/kubermatic/kubermatic/issues/5704))
- fix typo in prometheus chart ([#5726](https://github.com/kubermatic/kubermatic/issues/5726))

### Misc

- Allow to configure Velero plugin InitContainers ([#5718](https://github.com/kubermatic/kubermatic/issues/5718), [#5719](https://github.com/kubermatic/kubermatic/issues/5719))
- addons/csi: add nodeplugin for flatcar linux ([#5701](https://github.com/kubermatic/kubermatic/issues/5701))


## [v2.14.3](https://github.com/kubermatic/kubermatic/releases/tag/v2.14.3)

- Added Kubernetes v1.16.13, and removed v1.16.2-9 in default version configuration ([#5659](https://github.com/kubermatic/kubermatic/issues/5659))
- Added Kubernetes v1.17.9, and removed v1.17.0-5 in default version configuration ([#5664](https://github.com/kubermatic/kubermatic/issues/5664))
- Added Kubernetes v1.18.6, and removed v1.18.2 in default version configuration ([#5673](https://github.com/kubermatic/kubermatic/issues/5673))


## [v2.14.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.14.2)

### Bugfixes

- Fix Kubermatic operator not to specify unsupported `dynamic-datacenter` flag in CE mode. ([#5615](https://github.com/kubermatic/kubermatic/issues/5615))
- Fix Seed validation for Community Edition. ([#5619](https://github.com/kubermatic/kubermatic/issues/5619))
- Fix Subnetworks for GCP, because the network filtering was wrong. ([#5632](https://github.com/kubermatic/kubermatic/pull/5632))
- Fix label for nodeport-proxy when deployed with the operator. ([#5612](https://github.com/kubermatic/kubermatic/pull/5612))

### Misc

- Change default number of replicas for seed and master controller manager to one. ([#5620](https://github.com/kubermatic/kubermatic/issues/5620))
- Remove empty Docker secret for Kubermatic Operator CE Helm chart. ([#5618](https://github.com/kubermatic/kubermatic/pull/5618))


## [v2.14.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.14.1)

- Added missing Flatcar Linux handling in API ([#5368](https://github.com/kubermatic/kubermatic/issues/5368))
- Fixed nodes sometimes not having the correct distribution label applied. ([#5437](https://github.com/kubermatic/kubermatic/issues/5437))
- Fixed missing Kubermatic Prometheus metrics. ([#5505](https://github.com/kubermatic/kubermatic/issues/5505))


## [v2.14.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.14.0)

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
