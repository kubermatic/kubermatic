package kubernetes

import (
	"errors"
)

const (
	// NamespacePrefix is the prefix for the cluster namespace
	NamespacePrefix = "cluster-"
)

var (
	// ErrAlreadyExist an error indicating that the the resource already exists
	ErrAlreadyExist = errors.New("AlreadyExist")
)

// NamespaceName returns the namespace name for a cluster
func NamespaceName(clusterName string) string {
	return NamespacePrefix + clusterName
}
