# Kubermatic 2.7

- [v2.7.0](#v270)
- [v2.7.1](#v271)
- [v2.7.2](#v272)
- [v2.7.3](#v273)
- [v2.7.4](#v274)
- [v2.7.5](#v275)
- [v2.7.6](#v276)
- [v2.7.7](#v277)
- [v2.7.8](#v278)

## [v2.7.8](https://github.com/kubermatic/kubermatic/releases/tag/v2.7.8)

### Supported Kubernetes versions

- `1.10.11`
- `1.11.5`

### Major changes

- Communicating with cloud providers APIs no longer requires providing additional credentials. ([#2151](https://github.com/kubermatic/kubermatic/issues/2151))
- Updated the kubermatic dashboard to v0.38.0 ([#2165](https://github.com/kubermatic/kubermatic/issues/2165))
  - Provider-specific data will now be fetched without re-sending credentials. ([#806](https://github.com/kubermatic/dashboard/issues/806))
- Kubernetes will be automatically updated to versions that contain a fix for CVE-2018-1002105 and v1.8, v1.9 cluster creation is now dropped ([#2487](https://github.com/kubermatic/kubermatic/issues/2487))


## [v2.7.7](https://github.com/kubermatic/kubermatic/releases/tag/v2.7.7)

### Misc

- Removed functionality to copy the admin token in the dashboard ([#2083](https://github.com/kubermatic/kubermatic/issues/2083))


## [v2.7.6](https://github.com/kubermatic/kubermatic/releases/tag/v2.7.6)

### Misc

- Various minor fixes and improvements


## [v2.7.5](https://github.com/kubermatic/kubermatic/releases/tag/v2.7.5)

### Bugfixes

- Fixed handling of very long user IDs ([#2070](https://github.com/kubermatic/kubermatic/issues/2070))


## [v2.7.4](https://github.com/kubermatic/kubermatic/releases/tag/v2.7.4)


### Bugfixes

- Updated machine controller to `v0.7.23`: write permissions on vSphere datacenters are no longer needed. ([#2069](https://github.com/kubermatic/kubermatic/issues/2069))


## [v2.7.3](https://github.com/kubermatic/kubermatic/releases/tag/v2.7.3)


### Misc

- kube-proxy addon was updated to match the cluster version [#2019](https://github.com/kubermatic/kubermatic/issues/2019)


## [v2.7.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.7.2)

### Monitoring

- `KubePersistentVolumeFullInFourDays` will only be predicted when there is at least 6h of historical data available ([#1862](https://github.com/kubermatic/kubermatic/issues/1862))

### Misc

- Updated machine-controller to v0.7.22 ([#1999](https://github.com/kubermatic/kubermatic/issues/1999))


## [v2.7.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.7.1)

### Bugfixes

- fixed DNS/scheduler/controller-manager alerts in Prometheus ([#1908](https://github.com/kubermatic/kubermatic/issues/1908))
- fix bad rules.yaml format for Prometheus ([#1924](https://github.com/kubermatic/kubermatic/issues/1924))
- Add missing RoleBinding for bootstrap tokens created with `kubeadm token create` ([#1943](https://github.com/kubermatic/kubermatic/issues/1943))
- Fix bug with endless resource updates being triggered due to a wrong comparison ([#1964](https://github.com/kubermatic/kubermatic/issues/1964))
- Fix escaping of special characters in the cloud-config ([#1976](https://github.com/kubermatic/kubermatic/issues/1976))

### Misc

- Update kubernetes-dashboard addon to 1.10.0 ([#1874](https://github.com/kubermatic/kubermatic/issues/1874))
- Update machine-controller to v0.7.21 ([#1975](https://github.com/kubermatic/kubermatic/issues/1975))


## [v2.7.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.7.0)

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
- SSH key pair can now be detached from a cluster ([#1395](https://github.com/kubermatic/kubermatic/issues/1395))
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
- Alert Manager is now a Statefulset ([#1414](https://github.com/kubermatic/kubermatic/issues/1414))
- Fixed job labels for recording rules and alerts ([#1415](https://github.com/kubermatic/kubermatic/issues/1415))
- Added official etcd alerts ([#1417](https://github.com/kubermatic/kubermatic/issues/1417))
- Added an S3 exporter for metrics ([#1482](https://github.com/kubermatic/kubermatic/issues/1482))
- Added alert rule for machines which stuck in deletion ([#1606](https://github.com/kubermatic/kubermatic/issues/1606))
- The customer cluster Prometheus inside its namespace alerts on its own now. ([#1703](https://github.com/kubermatic/kubermatic/issues/1703))
- Add kube-state-metrics to the cluster namespace ([#1716](https://github.com/kubermatic/kubermatic/issues/1716))
- Scrape nginx ingress controller metrics ([#1777](https://github.com/kubermatic/kubermatic/issues/1777))
- use annotations instead of labels for the nginx-ingress Prometheus configuration ([#1823](https://github.com/kubermatic/kubermatic/issues/1823))
