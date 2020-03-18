// Package nodeportproxy is responsible for reconciling a seed-cluster-wide
// proxy based on Envoy and a custom envoy-manager/lb-updater tools. They
// monitor Cluster resources and allocate a port on a shared LoadBalancer
// service to access the user cluster's control plane.
//
// Note that there is also pkg/resources/nodeportproxy/, which is a special,
// per-Cluster nodeport-proxy. The package are similar, but not similar
// enough to merge them together.
package nodeportproxy
