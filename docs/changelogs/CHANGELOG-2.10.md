# Kubermatic 2.10

- [v2.10.0](#v2100)
- [v2.10.1](#v2101)
- [v2.10.2](#v2102)
- [v2.10.3](#v2103)

## [v2.10.3](https://github.com/kubermatic/kubermatic/releases/tag/v2.10.3)

- Kubernetes 1.11 which is end-of-life has been removed. ([#4031](https://github.com/kubermatic/kubermatic/issues/4031))
- Kubernetes 1.12 which is end-of-life has been removed. ([#4065](https://github.com/kubermatic/kubermatic/issues/4065))
- Kubernetes versions affected by CVE-2019-11247 and CVE-2019-11249 have been dropped ([#4066](https://github.com/kubermatic/kubermatic/issues/4066))
- Kubernetes versions affected by CVE-2019-9512 and CVE-2019-9514 have been dropped ([#4113](https://github.com/kubermatic/kubermatic/issues/4113))
- updated Envoy to 1.11.1 ([#4075](https://github.com/kubermatic/kubermatic/issues/4075))


## [v2.10.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.10.2)

### Misc

- Updated Dashboard to v1.2.2 ([#3553](https://github.com/kubermatic/kubermatic/issues/3553))
  - Missing parameters for OIDC providers have been added. ([#1273](https://github.com/kubermatic/dashboard/issues/1273))
  - `containerRuntimeVersion` and `kernelVersion` are now displayed on NodeDeployment detail page ([#1217](https://github.com/kubermatic/dashboard/issues/1217))
  - Fixed changing default OpenStack image on Operating System change ([#1218](https://github.com/kubermatic/dashboard/issues/1218))
  - The OIDC provider URL is now configurable via "oidc_provider_url" variable. ([#1224](https://github.com/kubermatic/dashboard/issues/1224))
- Insecure Kubernetes versions v1.13.6 and v1.14.2 have been disabled. ([#3554](https://github.com/kubermatic/kubermatic/issues/3554))


## [v2.10.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.10.1)

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


## [v2.10.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.10.0)

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
