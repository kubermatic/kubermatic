package v1

import (
	internalinterfaces "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// Addons returns a AddonInformer.
	Addons() AddonInformer
	// Clusters returns a ClusterInformer.
	Clusters() ClusterInformer
	// Projects returns a ProjectInformer.
	Projects() ProjectInformer
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

// Addons returns a AddonInformer.
func (v *version) Addons() AddonInformer {
	return &addonInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Clusters returns a ClusterInformer.
func (v *version) Clusters() ClusterInformer {
	return &clusterInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Projects returns a ProjectInformer.
func (v *version) Projects() ProjectInformer {
	return &projectInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Users returns a UserInformer.
func (v *version) Users() UserInformer {
	return &userInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// UserSSHKeies returns a UserSSHKeyInformer.
func (v *version) UserSSHKeies() UserSSHKeyInformer {
	return &userSSHKeyInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}
