# Kubermatic 2.6

- [v2.6.0](#v260)
- [v2.6.1](#v261)
- [v2.6.2](#v262)
- [v2.6.3](#v263)
- [v2.6.4](#v264)
- [v2.6.5](#v265)
- [v2.6.6](#v266)
- [v2.6.7](#v267)
- [v2.6.8](#v268)
- [v2.6.9](#v269)
- [v2.6.10](#v2610)
- [v2.6.11](#v2611)
- [v2.6.12](#v2612)
- [v2.6.13](#v2613)
- [v2.6.14](#v2614)
- [v2.6.15](#v2615)
- [v2.6.16](#v2616)
- [v2.6.17](#v2617)

## [v2.6.17](https://github.com/kubermatic/kubermatic/releases/tag/v2.6.17)

### Supported Kubernetes versions

- `1.10.11`

### Bugfixes

- Fixed handling of very long user IDs ([#2086](https://github.com/kubermatic/kubermatic/issues/2086))

### Misc

- Enabled the usage of Heapster for the HorizontalPodAutoscaler ([#2199](https://github.com/kubermatic/kubermatic/issues/2199))
- Kubernetes will be automatically updated to versions that contain a fix for CVE-2018-1002105 and v1.8, v1.9 cluster creation is now dropped ([#2497](https://github.com/kubermatic/kubermatic/issues/2497))


## [v2.6.16](https://github.com/kubermatic/kubermatic/releases/tag/v2.6.16)

- Updated machine-controller to v0.7.18 ([#1709](https://github.com/kubermatic/kubermatic/issues/1709))
- Added support for Kubernetes 1.8.14, 1.9.8, 1.9.9, 1.9.10, 1.10.4, 1.10.5 and 1.10.6 ([#1710](https://github.com/kubermatic/kubermatic/issues/1710))


## [v2.6.15](https://github.com/kubermatic/kubermatic/releases/tag/v2.6.15)

- Added addon for default StorageClass depending on a cloud provider ([#1697](https://github.com/kubermatic/kubermatic/issues/1697))


## [v2.6.14](https://github.com/kubermatic/kubermatic/releases/tag/v2.6.14)

### Cloud Provider

- Azure: fixed minor issue with seed clusters running on Azure ([#1657](https://github.com/kubermatic/kubermatic/issues/1657))

### Misc

- Updated machine-controller to v0.7.17 ([#1677](https://github.com/kubermatic/kubermatic/issues/1677))


## [v2.6.13](https://github.com/kubermatic/kubermatic/releases/tag/v2.6.13)

- Minor fixes for seed clusters running on Azure ([#1646](https://github.com/kubermatic/kubermatic/issues/1646))


## [v2.6.11](https://github.com/kubermatic/kubermatic/releases/tag/v2.6.11)

### Cloud Provider

- Azure: public IPs will always be allocated on new machines ([#1644](https://github.com/kubermatic/kubermatic/issues/1644))

### Misc

- Updated nodeport-proxy to v1.2 ([#1640](https://github.com/kubermatic/kubermatic/issues/1640))


## [v2.6.10](https://github.com/kubermatic/kubermatic/releases/tag/v2.6.10)

- Updated machine-controller to v0.7.14 ([#1635](https://github.com/kubermatic/kubermatic/issues/1635))


## [v2.6.9](https://github.com/kubermatic/kubermatic/releases/tag/v2.6.9)

- controller-manager will now automatically restart on backup config change ([#1548](https://github.com/kubermatic/kubermatic/issues/1548))
- apiserver will now automatically restart on master-files change ([#1552](https://github.com/kubermatic/kubermatic/issues/1552))


## [v2.6.8](https://github.com/kubermatic/kubermatic/releases/tag/v2.6.8)

- Minor fixes and improvements


## [v2.6.7](https://github.com/kubermatic/kubermatic/releases/tag/v2.6.7)

- With the introduction of Kubermatic's addon manager, the K8S addon manager's deployments will be automatically cleaned up on old setups ([#1513](https://github.com/kubermatic/kubermatic/issues/1513))


## [v2.6.6](https://github.com/kubermatic/kubermatic/releases/tag/v2.6.6)

- AWS: multiple clusters per subnet/VPC are now allowed ([#1481](https://github.com/kubermatic/kubermatic/issues/1481))


## [v2.6.5](https://github.com/kubermatic/kubermatic/releases/tag/v2.6.5)

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


## [v2.6.3](https://github.com/kubermatic/kubermatic/releases/tag/v2.6.3)

### Cloud Provider

- Fixed floating IP defaulting on openstack ([#1332](https://github.com/kubermatic/kubermatic/issues/1332))
- Azure: added multi-AZ node support ([#1354](https://github.com/kubermatic/kubermatic/issues/1354))
- Fixed premature logout from vsphere API ([#1373](https://github.com/kubermatic/kubermatic/issues/1373))

### Misc

- Control plane can now reach the nodes via VPN ([#1234](https://github.com/kubermatic/kubermatic/issues/1234))
- Enabled Mutating/Validating Admission Webhooks for K8S 1.9+ ([#1352](https://github.com/kubermatic/kubermatic/issues/1352))
- Updated addon manager to v0.1.0 ([#1363](https://github.com/kubermatic/kubermatic/issues/1363))
- Update machine-controller to v0.7.5 ([#1374](https://github.com/kubermatic/kubermatic/issues/1374))


## [v2.6.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.6.2)

- Minor fixes and improvements for Openstack support


## [v2.6.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.6.1)

### Cloud Provider

- Non-ESXi vsphere hosts are now supported ([#1306](https://github.com/kubermatic/kubermatic/issues/1306))
- VSphere target folder will be properly cleaned up on cluster deletion. ([#1314](https://github.com/kubermatic/kubermatic/issues/1314))

### Misc

- Addons in kubermatic charts can now be specified as a list ([#1304](https://github.com/kubermatic/kubermatic/issues/1304))
- Updated machine-controller to v0.7.3 ([#1311](https://github.com/kubermatic/kubermatic/issues/1311))

### Monitoring

- Fixed metric name for addon controller ([#1323](https://github.com/kubermatic/kubermatic/issues/1323))


## [v2.6.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.6.0)

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
