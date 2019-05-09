// Copied from https://github.com/openshift/cluster-network-operator/blob/master/pkg/util/k8s/dynamicrestmapper.go
package restmapper

import (
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type DynamicRESTMapper struct {
	client discovery.DiscoveryInterface

	lock     sync.RWMutex
	delegate meta.RESTMapper
}

// NewDynamicRESTMapper returns a RESTMapper that dynamically discovers resource
// types at runtime. This is in contrast to controller-manager's default RESTMapper, which
// only checks resource types at startup, and so can't handle the case of first creating a
// CRD and then creating an instance of that CRD.
func NewDynamicRESTMapper(cfg *rest.Config) (meta.RESTMapper, error) {
	client, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}

	drm := &DynamicRESTMapper{client: client}
	if err := drm.reload(); err != nil {
		return nil, err
	}
	return drm, nil
}

func (drm *DynamicRESTMapper) reload() error {
	gr, err := restmapper.GetAPIGroupResources(drm.client)
	if err != nil {
		return err
	}

	// The discovery takes some time
	newMapper := restmapper.NewDiscoveryRESTMapper(gr)

	drm.lock.Lock()
	defer drm.lock.Unlock()
	drm.delegate = newMapper
	return nil
}

// reloadOnError checks if an error indicates that the delegated RESTMapper needs to be
// reloaded, and if so, reloads it and returns true.
func (drm *DynamicRESTMapper) reloadOnError(err error) bool {
	if _, matches := err.(*meta.NoKindMatchError); !matches {
		return false
	}
	err = drm.reload()
	if err != nil {
		utilruntime.HandleError(err)
	}
	return err == nil
}

func (drm *DynamicRESTMapper) mapper() meta.RESTMapper {
	drm.lock.RLock()
	defer drm.lock.RUnlock()

	return drm.delegate
}

// KindFor takes a partial resource and returns back the single match.
// It returns an error if there are multiple matches.
func (drm *DynamicRESTMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	gvk, err := drm.mapper().KindFor(resource)
	if drm.reloadOnError(err) {
		gvk, err = drm.mapper().KindFor(resource)
	}
	return gvk, err
}

// KindsFor takes a partial resource and returns back the list of
// potential kinds in priority order.
func (drm *DynamicRESTMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	gvks, err := drm.mapper().KindsFor(resource)
	if drm.reloadOnError(err) {
		gvks, err = drm.mapper().KindsFor(resource)
	}
	return gvks, err
}

// ResourceFor takes a partial resource and returns back the single
// match. It returns an error if there are multiple matches.
func (drm *DynamicRESTMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	gvr, err := drm.mapper().ResourceFor(input)
	if drm.reloadOnError(err) {
		gvr, err = drm.mapper().ResourceFor(input)
	}
	return gvr, err
}

// ResourcesFor takes a partial resource and returns back the list of
// potential resource in priority order.
func (drm *DynamicRESTMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	gvrs, err := drm.mapper().ResourcesFor(input)
	if drm.reloadOnError(err) {
		gvrs, err = drm.mapper().ResourcesFor(input)
	}
	return gvrs, err
}

// RESTMapping identifies a preferred resource mapping for the
// provided group kind.
func (drm *DynamicRESTMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	m, err := drm.mapper().RESTMapping(gk, versions...)
	if drm.reloadOnError(err) {
		m, err = drm.mapper().RESTMapping(gk, versions...)
	}
	return m, err
}

// RESTMappings returns the RESTMappings for the provided group kind
// in a rough internal preferred order. If no kind is found, it will
// return a NoResourceMatchError.
func (drm *DynamicRESTMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	ms, err := drm.mapper().RESTMappings(gk, versions...)
	if drm.reloadOnError(err) {
		ms, err = drm.mapper().RESTMappings(gk, versions...)
	}
	return ms, err
}

// ResourceSingularizer converts a resource name from plural to
// singular (e.g., from pods to pod).
func (drm *DynamicRESTMapper) ResourceSingularizer(resource string) (singular string, err error) {
	s, err := drm.mapper().ResourceSingularizer(resource)
	if drm.reloadOnError(err) {
		s, err = drm.mapper().ResourceSingularizer(resource)
	}
	return s, err
}
