package main

const (
	// All features for the kubermatic-controller-manager are defined below.

	// OpenIDAuthPlugin if enabled configures the flags on the API server to use
	// OAuth2 identity providers.
	OpenIDAuthPlugin = "OpenIDAuthPlugin"

	// VerticalPodAutoscaler if enabled the cluster-controller will enable the
	// VerticalPodAutoscaler for all control plane components
	VerticalPodAutoscaler = "VerticalPodAutoscaler"

	// EtcdDataCorruptionChecks if enabled etcd will be started with
	// --experimental-initial-corrupt-check=true +
	// --experimental-corrupt-check-time=10m
	EtcdDataCorruptionChecks = "EtcdDataCorruptionChecks"
)
