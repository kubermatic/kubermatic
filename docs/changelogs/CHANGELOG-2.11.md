# Kubermatic 2.11

- [v2.11.0](#v2110)
- [v2.11.1](#v2111)
- [v2.11.2](#v2112)
- [v2.11.3](#v2113)
- [v2.11.4](#v2114)
- [v2.11.5](#v2115)
- [v2.11.6](#v2116)
- [v2.11.7](#v2117)
- [v2.11.8](#v2118)

## [v2.11.8](https://github.com/kubermatic/kubermatic/releases/tag/v2.11.8)

- End-of-Life Kubernetes v1.14 is no longer supported. ([#4989](https://github.com/kubermatic/kubermatic/issues/4989))
- Added Kubernetes v1.15.7, v1.15.9 ([#4995](https://github.com/kubermatic/kubermatic/issues/4995))


## [v2.11.7](https://github.com/kubermatic/kubermatic/releases/tag/v2.11.7)

- Kubernetes 1.13 which is end-of-life has been removed. ([#4327](https://github.com/kubermatic/kubermatic/issues/4327))
- Added Kubernetes v1.15.4 ([#4329](https://github.com/kubermatic/kubermatic/issues/4329))
- Added Kubernetes v1.14.7 ([#4330](https://github.com/kubermatic/kubermatic/issues/4330))
- A bug that prevented node Labels, Taints and Annotations from getting applied correctly was fixed. ([#4368](https://github.com/kubermatic/kubermatic/issues/4368))
- Removed K8S releases affected by CVE-2019-11253 ([#4515](https://github.com/kubermatic/kubermatic/issues/4515))
- A fix for CVE-2019-11253 for clusters that were created with a Kubernetes version < 1.14 was deployed ([#4520](https://github.com/kubermatic/kubermatic/issues/4520))
- Openstack: fixed API usage for datacenters with only one region ([#4536](https://github.com/kubermatic/kubermatic/issues/4536))


## [v2.11.6](https://github.com/kubermatic/kubermatic/releases/tag/v2.11.6)

- Fixed a bug that could cause intermittent delays when using kubectl logs/exec with `exposeStrategy: LoadBalancer` ([#4279](https://github.com/kubermatic/kubermatic/issues/4279))


## [v2.11.5](https://github.com/kubermatic/kubermatic/releases/tag/v2.11.5)

- Fix a bug that caused setup on nodes with a Kernel > 4.18 to fail ([#4180](https://github.com/kubermatic/kubermatic/issues/4180))
- Fixed fetching the list of tenants on some OpenStack configurations with one region ([#4185](https://github.com/kubermatic/kubermatic/issues/4185))
- Fixed a bug that could result in the clusterdeletion sometimes getting stuck ([#4202](https://github.com/kubermatic/kubermatic/issues/4202))


## [v2.11.4](https://github.com/kubermatic/kubermatic/releases/tag/v2.11.4)

- `kubeadm join` has been fixed for v1.15 clusters ([#4162](https://github.com/kubermatic/kubermatic/issues/4162))


## [v2.11.3](https://github.com/kubermatic/kubermatic/releases/tag/v2.11.3)

- Kubermatic Swagger API Spec is now exposed over its API server ([#3890](https://github.com/kubermatic/kubermatic/issues/3890))
- updated Envoy to 1.11.1 ([#4075](https://github.com/kubermatic/kubermatic/issues/4075))
- Kubernetes versions affected by CVE-2019-9512 and CVE-2019-9514 have been dropped ([#4118](https://github.com/kubermatic/kubermatic/issues/4118))
- Enabling the OIDC feature flag in clusters has been fixed. ([#4136](https://github.com/kubermatic/kubermatic/issues/4136))


## [v2.11.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.11.2)


- Fixed an issue where deleted project owners would come back after a while ([#4020](https://github.com/kubermatic/kubermatic/issues/4020))
- Kubernetes versions affected by CVE-2019-11247 and CVE-2019-11249 have been dropped ([#4066](https://github.com/kubermatic/kubermatic/issues/4066))
- Kubernetes 1.11 which is end-of-life has been removed. ([#4030](https://github.com/kubermatic/kubermatic/issues/4030))
- Kubernetes 1.12 which is end-of-life has been removed. ([#4067](https://github.com/kubermatic/kubermatic/issues/4067))


## [v2.11.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.11.1)

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


## [v2.11.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.11.0)

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
