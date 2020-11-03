# Cluster DNS

**Author**: Alvaro Aleman (@alvaroaleman), Henrik Schmidt (@mrIncompetent), Matthias Loibl (@metalmatze)

**Status**: Draft proposal; Prototype in progress.

*short description of the topic*
This change is enabling master cluster components (apiserver, scheduler, contoller-manager) to talk to the DNS servers that are running inside the user-cluster itself. This will allow those master cluster components to talk via service names to services running inside that user-cluster.

## Motivation and Background

*What is the background and why do we want to deploy it*

The current approach doesn’t allow the apiserver to resolve service names that are running within the cluster, as we don’t tell it to use the DNS server inside the cluster.
This breaks a couple of use cases, for example:

* extension apiservers
* dynamic volume provisioning with a custom provisioner (kube-cluster)
* admisison webhooks

*Access from the master components to the pod & service network*

The apiserver already has access to the pod & service network using a custom vpn implementation https://github.com/kubermatic/pod-network-bridge
This might be replaced by openvpn in the future.

## Implementation

*How to implement the idea*

As we want to switch to another DNS, we can not use the seed cluster's DNS anymore. Currently we rely on using that DNS in the following places:

apiserver deployment: `--etcd-servers=http://etcd-cluster-client:2379`
controller-manager deployment: `--master=http://apiserver:8080`
machine-controller: `-master=http://apiserver:8080`
scheduler: `--master=http://apiserver:8080`
openvpn sidecar in apiserver deployment

Instead of relying on the seed clusters DNS, we use the cluster-ips to circumvent the usage of DNS. This works due to the fact, that we have CIDRs in the seed and user clusters that don’t overlap.
For accessing etcd without dns and without using pod IPs a new service `etcd-client` is created which is non-headless (as opposed to existing service `etcd`).

From the places mentioned above, only the kubernetes master components (apiserver, controller-manager, scheduler) need to be changed. Others could be changed as well to have streamlined manifests.

The master components of the user-cluster need to be configured to use the deployed DNS service within the user cluster.
This cluster-IP of the dns service is predefined as `10.10.10.10`.
Pods can be configured using "spec.dnsConfig" mentioned in https://raw.githubusercontent.com/kubernetes/website/master/docs/concepts/services-networking/custom-dns.yaml

## Task & effort

*Specify the tasks and the effort in days*

* Replace service references with cluster-ip master templates - 1d
* Give all master components access to the user clusters pod & service network - 1d
* Configure master components to use user cluster DNS - 0,5d
* Validate using - 1d
  * e2e tests
  * deployment of an extension apiserver
  * deployment of an admission controller
