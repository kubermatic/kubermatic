package v1

import (
	internalinterfaces "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// Clusters returns a ClusterInformer.
	Clusters() ClusterInformer
	// Users returns a UserInformer.
	Users() UserInformer
	// UserSSHKeies returns a UserSSHKeyInformer.
	UserSSHKeies() UserSSHKeyInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// Clusters returns a ClusterInformer.
func (v *version) Clusters() ClusterInformer {
	return &clusterInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Users returns a UserInformer.
func (v *version) Users() UserInformer {
	return &userInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// UserSSHKeies returns a UserSSHKeyInformer.
func (v *version) UserSSHKeies() UserSSHKeyInformer {
	return &userSSHKeyInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}
