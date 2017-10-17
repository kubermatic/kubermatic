// This file was automatically generated by informer-gen

package v1beta2

import (
	internalinterfaces "github.com/kubermatic/kubermatic/api/pkg/crd/client/seed/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// EtcdClusters returns a EtcdClusterInformer.
	EtcdClusters() EtcdClusterInformer
}

type version struct {
	internalinterfaces.SharedInformerFactory
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory) Interface {
	return &version{f}
}

// EtcdClusters returns a EtcdClusterInformer.
func (v *version) EtcdClusters() EtcdClusterInformer {
	return &etcdClusterInformer{factory: v.SharedInformerFactory}
}
