package kubernetes

const (
	// NamespacePrefix is the prefix for the cluster namespace
	NamespacePrefix = "cluster-"
)

// NamespaceName returns the namespace name for a cluster
func NamespaceName(clusterName string) string {
	return NamespacePrefix + clusterName
}
