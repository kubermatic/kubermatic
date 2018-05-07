package v1

import (
	internalinterfaces "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// Prometheuses returns a PrometheusInformer.
	Prometheuses() PrometheusInformer
	// ServiceMonitors returns a ServiceMonitorInformer.
	ServiceMonitors() ServiceMonitorInformer
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

// Prometheuses returns a PrometheusInformer.
func (v *version) Prometheuses() PrometheusInformer {
	return &prometheusInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// ServiceMonitors returns a ServiceMonitorInformer.
func (v *version) ServiceMonitors() ServiceMonitorInformer {
	return &serviceMonitorInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
