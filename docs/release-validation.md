# Release validations

This document provides a non-exhaustive minimal list of functionality that must be validated prior
to creating a new relase. All these tests must be executed with every supported minor Kuberntes version.


1. Conformance tests on all supported providers (AWS, Openstack, Digitalocean, VSphere, Azure, Hetzner)
1. Cloud provider functionality `service type LoadBalancer` on all providers that support it (AWS, Openstack, Azure)
1. Cloud provider functionality `PersistentVolume` on all providers that support it (AWS, Openstack, Vsphere, Azure)
1. HorizontalPodAutoscaler
