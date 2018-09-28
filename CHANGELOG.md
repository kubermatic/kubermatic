
### [v2.7.7]()


**Misc:**

- Removed functionality to copy the admin token in the dashboard [#2083](https://github.com/kubermatic/kubermatic/issues/2083) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.7.6]()


**Misc:**

- Various minor fixes and improvements




### [v2.7.5]()


**Bugfix:**

- Fixed handling of very long user IDs [#2070](https://github.com/kubermatic/kubermatic/issues/2070) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.7.4]()


**Bugfix:**

- Updated machine controller to `v0.7.23`: write permissions on vSphere datacenters are no longer needed. [#2069](https://github.com/kubermatic/kubermatic/issues/2069) ([pkavajin](https://github.com/pkavajin))




### [v2.7.3]()


**Misc:**

- kube-proxy addon was updated to match the cluster version [#2019](https://github.com/kubermatic/kubermatic/issues/2019) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.7.2]()


**Monitoring:**

- `KubePersistentVolumeFullInFourDays` will only be predicted when there is at least 6h of historical data available [#1862](https://github.com/kubermatic/kubermatic/issues/1862) ([cbeneke](https://github.com/cbeneke))


**Misc:**

- Updated machine-controller to `v0.7.22` [#1999](https://github.com/kubermatic/kubermatic/issues/1999) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.7.1]()


**Bugfix:**

- fixed DNS/scheduler/controller-manager alerts in Prometheus [#1908](https://github.com/kubermatic/kubermatic/issues/1908) ([xrstf](https://github.com/xrstf))
- fix bad rules.yaml format for Prometheus [#1924](https://github.com/kubermatic/kubermatic/issues/1924) ([xrstf](https://github.com/xrstf))
- Add missing RoleBinding for bootstrap tokens created with `kubeadm token create` [#1943](https://github.com/kubermatic/kubermatic/issues/1943) ([mrIncompetent](https://github.com/mrIncompetent))
- Fix bug with endless resource updates being triggered due to a wrong comparison [#1964](https://github.com/kubermatic/kubermatic/issues/1964) ([mrIncompetent](https://github.com/mrIncompetent))
- Fix escaping of special characters in the cloud-config [#1976](https://github.com/kubermatic/kubermatic/issues/1976) ([mrIncompetent](https://github.com/mrIncompetent))


**Misc:**

- Update kubernetes-dashboard addon to `1.10.0` [#1874](https://github.com/kubermatic/kubermatic/issues/1874) ([bashofmann](https://github.com/bashofmann))
- Update machine-controller to `v0.7.21` [#1975](https://github.com/kubermatic/kubermatic/issues/1975) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.7.0]()


**Bugfix:**

- Fixed a rare issue with duplicate entries on the list of nodes [#1391](https://github.com/kubermatic/kubermatic/issues/1391) ([mrIncompetent](https://github.com/mrIncompetent))
- Fixed deletion of old etcd backups [#1394](https://github.com/kubermatic/kubermatic/issues/1394) ([mrIncompetent](https://github.com/mrIncompetent))
- Fix deadlock during backup cleanup when the etcd of the cluster never reached a healthy state. [#1612](https://github.com/kubermatic/kubermatic/issues/1612) ([mrIncompetent](https://github.com/mrIncompetent))
- Use dedicated CA for Kubernetes aggregation layer [#1787](https://github.com/kubermatic/kubermatic/issues/1787) ([mrIncompetent](https://github.com/mrIncompetent))


**Cloud Provider:**

- Non-ESXi vsphere hosts are now supported [#1306](https://github.com/kubermatic/kubermatic/issues/1306) ([alvaroaleman](https://github.com/alvaroaleman))
- VSphere target folder will be properly cleaned up on cluster deletion. [#1314](https://github.com/kubermatic/kubermatic/issues/1314) ([alvaroaleman](https://github.com/alvaroaleman))
- Fixed floating IP defaulting on openstack [#1332](https://github.com/kubermatic/kubermatic/issues/1332) ([mrIncompetent](https://github.com/mrIncompetent))
- Azure: added multi-AZ node support [#1354](https://github.com/kubermatic/kubermatic/issues/1354) ([mrIncompetent](https://github.com/mrIncompetent))
- Fixed premature logout from vsphere API [#1373](https://github.com/kubermatic/kubermatic/issues/1373) ([alvaroaleman](https://github.com/alvaroaleman))
- Image templates can now be configured in datacenter.yaml for Openstack and vSphere [#1397](https://github.com/kubermatic/kubermatic/issues/1397) ([mrIncompetent](https://github.com/mrIncompetent))
- AWS: allow multiple clusters per subnet/VPC [#1481](https://github.com/kubermatic/kubermatic/issues/1481) ([mrIncompetent](https://github.com/mrIncompetent))
- In a VSphere DC is is now possible to set a `infra_management_user` which when set will automatically be used for everything except the cloud provider functionality for all VSphere clusters in that DC.  [#1592](https://github.com/kubermatic/kubermatic/issues/1592) ([alvaroaleman](https://github.com/alvaroaleman))
- Always allocate public IP on new machines when using Azure [#1644](https://github.com/kubermatic/kubermatic/issues/1644) ([mrIncompetent](https://github.com/mrIncompetent))
- Add missing cloud provider flags on the apiserver and controller-manager for azure [#1646](https://github.com/kubermatic/kubermatic/issues/1646) ([mrIncompetent](https://github.com/mrIncompetent))
- Azure: fixed minor issue with seed clusters running on Azure [#1657](https://github.com/kubermatic/kubermatic/issues/1657) ([thz](https://github.com/thz))
- Create AvailabilitySet for Azure clusters and set it for each machine [#1661](https://github.com/kubermatic/kubermatic/issues/1661) ([mrIncompetent](https://github.com/mrIncompetent))
- OpenStack LoadBalancer manage-security-groups setting is set into cluster&#39;s cloud-config for Kubernetes versions where https://github.com/kubernetes/kubernetes/issues/58145 is fixed. [#1720](https://github.com/kubermatic/kubermatic/issues/1720) ([bashofmann](https://github.com/bashofmann))


**Misc:**

- Control plane can now reach the nodes via VPN [#1234](https://github.com/kubermatic/kubermatic/issues/1234) ([thz](https://github.com/thz))
- Addons in kubermatic charts can now be specified as a list [#1304](https://github.com/kubermatic/kubermatic/issues/1304) ([guusvw](https://github.com/guusvw))
- Added support for Kubernetes `1.8.14`, `1.9.8`, `1.9.9`, `1.10.4` and `1.10.5` [#1348](https://github.com/kubermatic/kubermatic/issues/1348) ([mrIncompetent](https://github.com/mrIncompetent))
- Enabled Mutating/Validating Admission Webhooks for K8S 1.9&#43; [#1352](https://github.com/kubermatic/kubermatic/issues/1352) ([alvaroaleman](https://github.com/alvaroaleman))
- Update addon manager to v0.1.0 [#1363](https://github.com/kubermatic/kubermatic/issues/1363) ([thz](https://github.com/thz))
- Master components can now talk to cluster DNS [#1379](https://github.com/kubermatic/kubermatic/issues/1379) ([thz](https://github.com/thz))
- Non-default IP can now be used for cluster DNS [#1393](https://github.com/kubermatic/kubermatic/issues/1393) ([glower](https://github.com/glower))
- SSH keypair can now be detached from a cluster [#1395](https://github.com/kubermatic/kubermatic/issues/1395) ([p0lyn0mial](https://github.com/p0lyn0mial))
- Removed Kubermatic API v2 [#1409](https://github.com/kubermatic/kubermatic/issues/1409) ([p0lyn0mial](https://github.com/p0lyn0mial))
- Added EFK stack in seed clusters [#1430](https://github.com/kubermatic/kubermatic/issues/1430) ([pkavajin](https://github.com/pkavajin))
- Fixed some issues with eleasticsearch [#1484](https://github.com/kubermatic/kubermatic/issues/1484) ([pkavajin](https://github.com/pkavajin))
- Master components will now talk to the apiserver over secure port [#1486](https://github.com/kubermatic/kubermatic/issues/1486) ([thz](https://github.com/thz))
- Added support for Kubernetes version 1.11.0 [#1493](https://github.com/kubermatic/kubermatic/issues/1493) ([alvaroaleman](https://github.com/alvaroaleman))
- Clients will now talk to etcd over TLS [#1495](https://github.com/kubermatic/kubermatic/issues/1495) ([mrIncompetent](https://github.com/mrIncompetent))
- Communication between apiserver and etcd is now encrypted [#1496](https://github.com/kubermatic/kubermatic/issues/1496) ([mrIncompetent](https://github.com/mrIncompetent))
- With the introduction of Kubermatic&#39;s addon manager, the K8S addon manager&#39;s deployments will be automatically cleaned up on old setups [#1513](https://github.com/kubermatic/kubermatic/issues/1513) ([mrIncompetent](https://github.com/mrIncompetent))
- controller-manager will now automatically restart on backup config change [#1548](https://github.com/kubermatic/kubermatic/issues/1548) ([bashofmann](https://github.com/bashofmann))
- The control plane now has its own DNS resolver [#1549](https://github.com/kubermatic/kubermatic/issues/1549) ([alvaroaleman](https://github.com/alvaroaleman))
- apiserver will now automatically restart on master-files change [#1552](https://github.com/kubermatic/kubermatic/issues/1552) ([cbeneke](https://github.com/cbeneke))
- Add missing reconciling of the OpenVPN config inside the user cluster [#1605](https://github.com/kubermatic/kubermatic/issues/1605) ([mrIncompetent](https://github.com/mrIncompetent))
- Add pod anti-affinity for the etcd StatefulSet [#1607](https://github.com/kubermatic/kubermatic/issues/1607) ([mrIncompetent](https://github.com/mrIncompetent))
- Add PodDisruptionBudget for the etcd StatefulSet [#1608](https://github.com/kubermatic/kubermatic/issues/1608) ([mrIncompetent](https://github.com/mrIncompetent))
- Add support for configuring component settings(Replicas &amp; Resources) via the cluster object [#1636](https://github.com/kubermatic/kubermatic/issues/1636) ([mrIncompetent](https://github.com/mrIncompetent))
- Update nodeport-proxy to v1.2 [#1640](https://github.com/kubermatic/kubermatic/issues/1640) ([mrIncompetent](https://github.com/mrIncompetent))
- Added  access to the private quay.io repos from the kubermatic helm template [#1652](https://github.com/kubermatic/kubermatic/issues/1652) ([glower](https://github.com/glower))
- the correct default StorageClass is now installed into the user cluster via an extra addon [#1670](https://github.com/kubermatic/kubermatic/issues/1670) ([glower](https://github.com/glower))
- Update machine-controller to `v0.7.18` [#1708](https://github.com/kubermatic/kubermatic/issues/1708) ([mrIncompetent](https://github.com/mrIncompetent))
- Add support for Kubernetes `1.9.10`, `1.10.6` and `1.11.1` [#1712](https://github.com/kubermatic/kubermatic/issues/1712) ([mrIncompetent](https://github.com/mrIncompetent))
- Add possibility to override the seed DNS name for a given node datacenter via the datacenters.yaml [#1715](https://github.com/kubermatic/kubermatic/issues/1715) ([mrIncompetent](https://github.com/mrIncompetent))
- Heapster is replaced by metrics-server. [#1730](https://github.com/kubermatic/kubermatic/issues/1730) ([glower](https://github.com/glower))
- Combine the two existing CA secrets into a single one [#1732](https://github.com/kubermatic/kubermatic/issues/1732) ([mrIncompetent](https://github.com/mrIncompetent))
- It is now possible to customize user cluster configmaps/secrets via a `MutatingAdmissionWebhook` [#1740](https://github.com/kubermatic/kubermatic/issues/1740) ([alvaroaleman](https://github.com/alvaroaleman))
- Make s3-exporter endpoint configurable [#1772](https://github.com/kubermatic/kubermatic/issues/1772) ([bashofmann](https://github.com/bashofmann))
- Update nginx ingress controller to 0.18.0 [#1800](https://github.com/kubermatic/kubermatic/issues/1800) ([bashofmann](https://github.com/bashofmann))


**Monitoring:**

- Fixed metric name for addon controller [#1323](https://github.com/kubermatic/kubermatic/issues/1323) ([alvaroaleman](https://github.com/alvaroaleman))
- Error metrics are now collected for Kubermatic API endpoints [#1376](https://github.com/kubermatic/kubermatic/issues/1376) ([pkavajin](https://github.com/pkavajin))
- Prometheus is now a Statefulset [#1399](https://github.com/kubermatic/kubermatic/issues/1399) ([metalmatze](https://github.com/metalmatze))
- Alert Manger is now a Statefulset [#1414](https://github.com/kubermatic/kubermatic/issues/1414) ([metalmatze](https://github.com/metalmatze))
- Fixed job labels for recording rules and alerts [#1415](https://github.com/kubermatic/kubermatic/issues/1415) ([metalmatze](https://github.com/metalmatze))
- Added official etcd alerts [#1417](https://github.com/kubermatic/kubermatic/issues/1417) ([metalmatze](https://github.com/metalmatze))
- Added an S3 exporter for metrics [#1482](https://github.com/kubermatic/kubermatic/issues/1482) ([alvaroaleman](https://github.com/alvaroaleman))
- Added alert rule for machines which stuck in deletion [#1606](https://github.com/kubermatic/kubermatic/issues/1606) ([mrIncompetent](https://github.com/mrIncompetent))
- The customer cluster Prometheus inside its namespace alerts on its own now. [#1703](https://github.com/kubermatic/kubermatic/issues/1703) ([metalmatze](https://github.com/metalmatze))
- Add kube-state-metrics to the cluster namespace [#1716](https://github.com/kubermatic/kubermatic/issues/1716) ([mrIncompetent](https://github.com/mrIncompetent))
- Scrape nginx ingress controller metrics [#1777](https://github.com/kubermatic/kubermatic/issues/1777) ([bashofmann](https://github.com/bashofmann))
- use annotations instead of labels for the nginx-ingress Prometheus configuration [#1823](https://github.com/kubermatic/kubermatic/issues/1823) ([xrstf](https://github.com/xrstf))


**Dashboard:**

- Fixed cluster settings view for Openstack [#746](https://github.com/kubermatic/dashboard-v2/issues/746) ([kgroschoff](https://github.com/kgroschoff))
- Fixed error appearing when trying to change selected OS [#699](https://github.com/kubermatic/dashboard-v2/issues/699) ([kgroschoff](https://github.com/kgroschoff))
- Openstack: fixed an issue, where list of tenants wouldn&#39;t get loaded when returning from summary page [#705](https://github.com/kubermatic/dashboard-v2/issues/705) ([kgroschoff](https://github.com/kgroschoff))
- Fixed confirmation of cluster deletion [#718](https://github.com/kubermatic/dashboard-v2/issues/718) ([kgroschoff](https://github.com/kgroschoff))
- Fixed the link to Kubernetes dashboard [#740](https://github.com/kubermatic/dashboard-v2/issues/740) ([guusvw](https://github.com/guusvw))
- vSphere: custom cluster vnet can now be selected [#708](https://github.com/kubermatic/dashboard-v2/issues/708) ([kgroschoff](https://github.com/kgroschoff))
- Openstack: the list of available networks and floating IP pools will be loaded from the API [#737](https://github.com/kubermatic/dashboard-v2/issues/737) ([j3ank](https://github.com/j3ank))
- Dashboard metrics can now be collected by Prometheus [#678](https://github.com/kubermatic/dashboard-v2/issues/678) ([pkavajin](https://github.com/pkavajin))
- Redesigned cluster creation summary page [#688](https://github.com/kubermatic/dashboard-v2/issues/688) ([kgroschoff](https://github.com/kgroschoff))
- Default template images for Openstack and vSphere are now taken from datacenter configuration [#689](https://github.com/kubermatic/dashboard-v2/issues/689) ([kgroschoff](https://github.com/kgroschoff))
- Various minor fixes and improvements




### [v2.6.16]()


- Updated machine-controller to `v0.7.18` [#1709](https://github.com/kubermatic/kubermatic/issues/1709) ([mrIncompetent](https://github.com/mrIncompetent))
- Added support for Kubernetes `1.8.14`, `1.9.8`, `1.9.9`, `1.9.10`, `1.10.4`, `1.10.5` and `1.10.6` [#1710](https://github.com/kubermatic/kubermatic/issues/1710) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.6.15]()


- Added addon for default StorageClass depending on a cloud provider [#1697](https://github.com/kubermatic/kubermatic/issues/1697) ([glower](https://github.com/glower))




### [v2.6.14]()

**Cloud Provider:**

- Azure: fixed minor issue with seed clusters running on Azure [#1657](https://github.com/kubermatic/kubermatic/issues/1657) ([thz](https://github.com/thz))


**Misc:**

- Updated machine-controller to `v0.7.17` [#1677](https://github.com/kubermatic/kubermatic/issues/1677) ([thz](https://github.com/thz))




### [v2.6.13]()


- Minor fixes for seed clusters running on Azure [#1646](https://github.com/kubermatic/kubermatic/issues/1646) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.6.11]()


**Cloud Provider:**

- Azure: public IPs will always be allocated on new machines [#1644](https://github.com/kubermatic/kubermatic/issues/1644) ([mrIncompetent](https://github.com/mrIncompetent))


**Misc:**

- Updated nodeport-proxy to v1.2 [#1640](https://github.com/kubermatic/kubermatic/issues/1640) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.6.10]()


- Updated machine-controller to v0.7.14 [#1635](https://github.com/kubermatic/kubermatic/issues/1635) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.6.9]()


- controller-manager will now automatically restart on backup config change [#1548](https://github.com/kubermatic/kubermatic/issues/1548) ([bashofmann](https://github.com/bashofmann))
- apiserver will now automatically restart on master-files change [#1552](https://github.com/kubermatic/kubermatic/issues/1552) ([cbeneke](https://github.com/cbeneke))




### [v2.6.8]()


- Minor fixes and improvements




### [v2.6.7]()


**Misc:**

- With the introduction of Kubermatic&#39;s addon manager, the K8S addon manager&#39;s deployments will be automatically cleaned up on old setups [#1513](https://github.com/kubermatic/kubermatic/issues/1513) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.6.6]()


- AWS: multiple clusters per subnet/VPC are now allowed [#1481](https://github.com/kubermatic/kubermatic/issues/1481) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.6.5]()


**Bugfix:**

- Fixed a rare issue with duplicate entries on the list of nodes [#1391](https://github.com/kubermatic/kubermatic/issues/1391) ([mrIncompetent](https://github.com/mrIncompetent))
- Fixed deletion of old etcd backups [#1394](https://github.com/kubermatic/kubermatic/issues/1394) ([mrIncompetent](https://github.com/mrIncompetent))


**Cloud Provider:**

- Image templates can now be configured in datacenter.yaml for Openstack and vSphere [#1397](https://github.com/kubermatic/kubermatic/issues/1397) ([mrIncompetent](https://github.com/mrIncompetent))


**Misc:**

- Non-default IP can now be used for cluster DNS [#1393](https://github.com/kubermatic/kubermatic/issues/1393) ([glower](https://github.com/glower))


**Monitoring:**

- Error metrics are now collected for Kubermatic API endpoints [#1376](https://github.com/kubermatic/kubermatic/issues/1376) ([pkavajin](https://github.com/pkavajin))


**Dashboard:**

- Minor visual improvements [#684](https://github.com/kubermatic/dashboard-v2/issues/684) ([kgroschoff](https://github.com/kgroschoff))
- The node list will no longer be expanded when clicking on an IP [#676](https://github.com/kubermatic/dashboard-v2/issues/676) ([kgroschoff](https://github.com/kgroschoff))
- Openstack: the tenant can now be picked from a list loaded from the API [#679](https://github.com/kubermatic/dashboard-v2/issues/679) ([kgroschoff](https://github.com/kgroschoff))
- Added a button to easily duplicate an existing node [#675](https://github.com/kubermatic/dashboard-v2/issues/675) ([kgroschoff](https://github.com/kgroschoff))
- A note has been added to the footer identifying whether the dashboard is a part of a demo system [#682](https://github.com/kubermatic/dashboard-v2/issues/682) ([kgroschoff](https://github.com/kgroschoff))
- Enabled CoreOS on Openstack [#673](https://github.com/kubermatic/dashboard-v2/issues/673) ([kgroschoff](https://github.com/kgroschoff))
- cri-o has been disabled [#670](https://github.com/kubermatic/dashboard-v2/issues/670) ([kgroschoff](https://github.com/kgroschoff))
- Node deletion can now be confirmed by pressing enter [#672](https://github.com/kubermatic/dashboard-v2/issues/672) ([kgroschoff](https://github.com/kgroschoff))




### [v2.6.3]()


**Cloud Provider:**

- Fixed floating IP defaulting on openstack [#1332](https://github.com/kubermatic/kubermatic/issues/1332) ([mrIncompetent](https://github.com/mrIncompetent))
- Azure: added multi-AZ node support [#1354](https://github.com/kubermatic/kubermatic/issues/1354) ([mrIncompetent](https://github.com/mrIncompetent))
- Fixed premature logout from vsphere API [#1373](https://github.com/kubermatic/kubermatic/issues/1373) ([alvaroaleman](https://github.com/alvaroaleman))


**Misc:**

- Control plane can now reach the nodes via VPN [#1234](https://github.com/kubermatic/kubermatic/issues/1234) ([thz](https://github.com/thz))
- Enabled Mutating/Validating Admission Webhooks for K8S 1.9&#43; [#1352](https://github.com/kubermatic/kubermatic/issues/1352) ([alvaroaleman](https://github.com/alvaroaleman))
- Updated addon manager to `v0.1.0` [#1363](https://github.com/kubermatic/kubermatic/issues/1363) ([thz](https://github.com/thz))
- Update machine-controller to `v0.7.5` [#1374](https://github.com/kubermatic/kubermatic/issues/1374) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.6.2]()


- Minor fixes and improvements for Openstack support




### [v2.6.1]()


**Cloud Provider:**

- Non-ESXi vsphere hosts are now supported [#1306](https://github.com/kubermatic/kubermatic/issues/1306) ([alvaroaleman](https://github.com/alvaroaleman))
- VSphere target folder will be properly cleaned up on cluster deletion. [#1314](https://github.com/kubermatic/kubermatic/issues/1314) ([alvaroaleman](https://github.com/alvaroaleman))


**Misc:**

- Addons in kubermatic charts can now be specified as a list [#1304](https://github.com/kubermatic/kubermatic/issues/1304) ([guusvw](https://github.com/guusvw))
- Updated machine-controller to `v0.7.3` [#1311](https://github.com/kubermatic/kubermatic/issues/1311) ([mrIncompetent](https://github.com/mrIncompetent))


**Monitoring:**

- Fixed metric name for addon controller [#1323](https://github.com/kubermatic/kubermatic/issues/1323) ([alvaroaleman](https://github.com/alvaroaleman))

### [v2.6.0]()


**Bugfix:**

- Cluster IPv6 addresses will be ignored on systems on which they are available [#1017](https://github.com/kubermatic/kubermatic/issues/1017) ([mrIncompetent](https://github.com/mrIncompetent))
- Fixed an issue with duplicate users being sometimes created [#990](https://github.com/kubermatic/kubermatic/issues/990) ([mrIncompetent](https://github.com/mrIncompetent))


**Cloud Provider:**

- Added Azure support [#1200](https://github.com/kubermatic/kubermatic/issues/1200) ([kdomanski](https://github.com/kdomanski))
- Openstack: made cluster resource cleanup idempotent [#961](https://github.com/kubermatic/kubermatic/issues/961) ([mrIncompetent](https://github.com/mrIncompetent))


**Misc:**

- Updated prometheus operator to `v0.19.0` [#1014](https://github.com/kubermatic/kubermatic/issues/1014) ([mrIncompetent](https://github.com/mrIncompetent))
- Updated dex to `v2.10.0` [#1052](https://github.com/kubermatic/kubermatic/issues/1052) ([mrIncompetent](https://github.com/mrIncompetent))
- etcd operator has been replaced with a `StatefulSet` [#1065](https://github.com/kubermatic/kubermatic/issues/1065) ([mrIncompetent](https://github.com/mrIncompetent))
- Nodeport range is now configurable [#1084](https://github.com/kubermatic/kubermatic/issues/1084) ([mrIncompetent](https://github.com/mrIncompetent))
- Bare-metal provider has been removed [#1087](https://github.com/kubermatic/kubermatic/issues/1087) ([mrIncompetent](https://github.com/mrIncompetent))
- Introduced addon manager [#1152](https://github.com/kubermatic/kubermatic/issues/1152) ([mrIncompetent](https://github.com/mrIncompetent))
- etcd data of user clusters can now be automatically backed up [#1170](https://github.com/kubermatic/kubermatic/issues/1170) ([alvaroaleman](https://github.com/alvaroaleman))
- Updated machine-controller to `v0.7.2` [#1227](https://github.com/kubermatic/kubermatic/issues/1227) ([mrIncompetent](https://github.com/mrIncompetent))
- etcd disk size can now be configured [#1301](https://github.com/kubermatic/kubermatic/issues/1301) ([mrIncompetent](https://github.com/mrIncompetent))
- Updated kube-state-metrics to `v1.3.1` [#933](https://github.com/kubermatic/kubermatic/issues/933) ([metalmatze](https://github.com/metalmatze))
- Added the ability to blacklist a cluster from reconciliation by the cluster-controller [#936](https://github.com/kubermatic/kubermatic/issues/936) ([mrIncompetent](https://github.com/mrIncompetent))
- Allow disabling TLS verification in offline environments [#968](https://github.com/kubermatic/kubermatic/issues/968) ([mrIncompetent](https://github.com/mrIncompetent))
- Updated nginx-ingress to `v0.14.0` [#983](https://github.com/kubermatic/kubermatic/issues/983) ([metalmatze](https://github.com/metalmatze))
- Kubernetes can now automatically allocate a nodeport if the default nodeport range is unavailable [#987](https://github.com/kubermatic/kubermatic/issues/987) ([mrIncompetent](https://github.com/mrIncompetent))
- Updated nodeport-proxy to `v1.1` [#988](https://github.com/kubermatic/kubermatic/issues/988) ([mrIncompetent](https://github.com/mrIncompetent))
- Added support for Kubernetes `v1.10.2` [#989](https://github.com/kubermatic/kubermatic/issues/989) ([mrIncompetent](https://github.com/mrIncompetent))
- Various other fixes and improvements


**Monitoring:**

- Added alerts for kubermatic master components being down [#1031](https://github.com/kubermatic/kubermatic/issues/1031) ([metalmatze](https://github.com/metalmatze))
- Massive amount of general improvements to alerting
