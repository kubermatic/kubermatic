# Title of the Proposal: Ingress for user clusters

**Author**: Alvaro Aleman (@alvaroaleman)

**Status**: Draft proposal

## Goal

The goal of this proposal is to describe a way to make ingress for Openshift user clusters work without any further configuration.

## Non-Goals

* Setting up certificates for an ingress controller, Openshift creates self-signed ones. Getting valid certificates can be a follow-up.
* Solve Ingress for Kubernetes clusters, this can be a follow-up.

## Motivation and Background

Openshift assumes there is a [wildcard DNS set up that points to all nodes that running an ingress controller](https://docs.openshift.com/container-platform/4.1/installing/installing_vsphere/installing-vsphere.html#installation-dns-user-infra_installing-vsphere)


## Implementation

In order to make Ingress work for Openshift by default, we need a wildcard DNS set up in a way that traffic
directed at that DNS name will end up at an Ingress Controller running for the given cluster.

The proposed implementation looks like this:

* The human Kubermatic operator must provide a DNS domain per seed under which kubermatic will set up a wildcard domain per cluster
* [External DNS](https://github.com/kubernetes-sigs/external-dns/) must be set up in the seed and configured
	with the admin's provider of choice
* Kubermatic runs a controller that manages a [`DNSEndpoint`](https://github.com/kubernetes-sigs/external-dns/blob/f763d2a4139746abd775c61642cb9e776b387ba6/docs/contributing/crd-source.md) custom resource and
    * Sets its `DNSName` to `*.<<clusterid>>.<<DNS_ZONE>>`
    * Synchronizes the `target` property of it with the ready nodes in the cluster

## Alternatives considered

### Operating an authoritative DNS server per seed

This would work but is a component we ourselves must make highly available and make sure its reachable. By using
`External DNS`, this can still be done by using the `CoreDNS` provider, but its also possible to choose from a
wide variety of DNS-As-A-Service providers.

### Using a service of type LoadBalancer

Depending on the implementation, services of type LoadBalancer may not have an associated DNS record. So
in order to support this, we would have to create the DNS entry and consequently support any possible DNS
provider.

Furthermore, services of type LoadBalancer are not available everywhere.

### Using a service of type LoadBalancer and creating a CNAME record per usercluster, pointing to it

This still adds the complexity of managing a DNS resolver while adding the requirement for support
of services of type LoadBalancer.

### Proxying the traffic from the seed to the usercluster

Proxying the traffic from the seed to the usercluster would allow us to get away with creating a DNS entry
only once during Kubermatic setup and then managing an ingress controller that does name-based virtual
hosting to redirect it to the usercluster by using the appropriate VPN.

This has some drawbacks:

* Proxying it from the seed into the usercluster will introduce a decent bit of latency and may impose
	a big load onto the seed clusters network
* All userclusters use the same CIDR range for the VPN, so in order for this to work correctly we would
	have to manage NAT rules inside the network namespace of the ingress controller running in the seed
