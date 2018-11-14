package v1alpha1

import (
	internalinterfaces "github.com/kubermatic/kubermatic/api/pkg/client/cluster-api/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// Clusters returns a ClusterInformer.
	Clusters() ClusterInformer
	// Machines returns a MachineInformer.
	Machines() MachineInformer
	// MachineClasses returns a MachineClassInformer.
	MachineClasses() MachineClassInformer
	// MachineDeployments returns a MachineDeploymentInformer.
	MachineDeployments() MachineDeploymentInformer
	// MachineSets returns a MachineSetInformer.
	MachineSets() MachineSetInformer
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
	return &clusterInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Machines returns a MachineInformer.
func (v *version) Machines() MachineInformer {
	return &machineInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// MachineClasses returns a MachineClassInformer.
func (v *version) MachineClasses() MachineClassInformer {
	return &machineClassInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// MachineDeployments returns a MachineDeploymentInformer.
func (v *version) MachineDeployments() MachineDeploymentInformer {
	return &machineDeploymentInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// MachineSets returns a MachineSetInformer.
func (v *version) MachineSets() MachineSetInformer {
	return &machineSetInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
