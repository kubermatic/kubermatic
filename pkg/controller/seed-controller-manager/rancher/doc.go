/*
Package rancher contains a controller responsible for reconciling all rancher-related
resources in the seed for Kubernetes clusters, as Rancher doesn't support openshift.

It is a no-op if the corresponding feature flag is not set on the cluster.
*/
package rancher
