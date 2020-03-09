# Implementing Loki for the user cluster 

**Author**: Youssef Azrak (@youssefazrak)

**Status**: Draft proposal

## Goals

Implement Loki for the user-cluster nodes.


## Motivation and Background

Currently, only the seed-cluster is using Loki with agents (Promtail) being deployed to each node of the cluster as Daemonset.
We are missing all the logs from the user-cluster side.


## Implementation

We will use the same architecture: Promtail agents deployed on the user-clusters' nodes that send the traffic to the main Loki server we have.

As Loki server is already deployed in the seed-cluster, only Promtail will be setup:
* Promtail agents will get deployed to each user clusters nodes. This will use a Daemonset for the user-cluster.
* Promtail sends the logs that it scraped to the Loki service we have in the seed-cluster.

* Promtail needs to resolve "loki.logging.svc.cluster.local". At the moment DNS is configured "locally" on the user-cluster side. We are following this architecture: https://kubernetes.io/docs/tasks/administer-cluster/nodelocaldns/. Without getting into details of the DNS resolution, we can simply say the DNS requests that are not resolved by the local DNS servers get forwarded to the node's DNS config so we won't be able to get the IP of Loki's SVC.
With this in mind, I would suggest implementing a new server block in the node-local-dns Corefile config that would forward the DNS resolution of Loki's URL to the control plane DNS servers. The same goes for the DNS servers in the control plane, they should forward the request to the kube-system DNS servers. We can do it the say by adding a new server block in the Corefile.


* Regarding layer 3 connectivity, we currently tunnel all the requests initiated by the control plane to the user nodes using the OpenVPN tunnel. This is done with two sidecar containers that will basically create specific routes depending on the target (svc, pods, etc) and forward the traffic to the right interface, the tunnel in this case.
We will need to do the same from a user cluster perspective: Promtail pods will get sidecar containers that will inject the routes in the routing table and forward the traffic to the tunnel in order to secure the traffic and allow using private IP addresses instead of public IP addresses. Same for the DNS pods.

## Task & effort:
* Implement Promtail on the user-cluster.
* Configure OpenVPN to inject seed-cluster networks routes. 
* Add the sidecars to Promtail: DNAT and OpenVPN.
* Configure CoreDNS (node-local-dns pod) on user-cluster for Loki's domain forwarding.
* Add the sidcars to CoreDNS: DNAT and OpenVPN.
* Configure CoreDNS on control-plane for forwarding to seed-cluster DNS servers.
* Validate using:
  * e2e tests
  * Deploying Promtail on user-cluster nodes
  * Checking logs on Loki server
