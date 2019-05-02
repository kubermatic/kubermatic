package restmapper

import (
	"sync"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// invalidationMapper defers loading the mapper and typer until necessary.
type invalidationMapper struct {
	newRESTMapping func() (meta.RESTMapper, error)

	// protects lastError & mapper
	lock      sync.Mutex
	lastError error
	mapper    meta.RESTMapper
}

// NewInvalidationRESTMapper is a "retry/invalidation" wrapper around an RESTMapper.
// It recreates the internal RESTMapper as soon as it hits an error & retries the mapping lookup.
// If the call fails on the second attempt as well, the error will be returned.
func NewInvalidationRESTMapper(fn func() (meta.RESTMapper, error)) meta.RESTMapper {
	m := &invalidationMapper{
		newRESTMapping: fn,
	}

	// Do not return the error - We recreate the internal mapper anyway on each call if lastError is set
	m.mapper, m.lastError = m.newRESTMapping()
	return m
}

var _ meta.RESTMapper = &invalidationMapper{}

func (m *invalidationMapper) RefreshIfNeeded() error {
	if m.lastError == nil {
		return nil
	}

	glog.V(4).Infof("Refreshing the REST mapping as we hit an error during the last lookup: %v", m.lastError)
	m.mapper, m.lastError = m.newRESTMapping()
	if m.lastError != nil {
		glog.Errorf("Failed to create a new REST mapping: %v", m.lastError)
	}

	return m.lastError
}

func (m *invalidationMapper) KindFor(resource schema.GroupVersionResource) (gvk schema.GroupVersionKind, err error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for i := 1; i <= 2; i++ {
		if err = m.RefreshIfNeeded(); err != nil {
			continue
		}

		if gvk, err = m.mapper.KindFor(resource); err != nil {
			// Triggers the mapper to be recreated
			m.lastError = err
			glog.Errorf("Failed to get the kind for %v: %v", resource, err)
			continue
		}
	}

	return
}

func (m *invalidationMapper) KindsFor(resource schema.GroupVersionResource) (gvks []schema.GroupVersionKind, err error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for i := 1; i <= 2; i++ {
		if err = m.RefreshIfNeeded(); err != nil {
			continue
		}

		if gvks, err = m.mapper.KindsFor(resource); err != nil {
			// Triggers the mapper to be recreated
			m.lastError = err
			glog.Errorf("Failed to get the kinds for %v: %v", resource, err)
			continue
		}
	}

	return
}

func (m *invalidationMapper) ResourceFor(input schema.GroupVersionResource) (gvr schema.GroupVersionResource, err error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for i := 1; i <= 2; i++ {
		if err = m.RefreshIfNeeded(); err != nil {
			continue
		}

		if gvr, err = m.mapper.ResourceFor(input); err != nil {
			// Triggers the mapper to be recreated
			m.lastError = err
			glog.Errorf("Failed to get the resource for %v: %v", input, err)
			continue
		}
	}

	return
}

func (m *invalidationMapper) ResourcesFor(input schema.GroupVersionResource) (gvrs []schema.GroupVersionResource, err error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for i := 1; i <= 2; i++ {
		if err = m.RefreshIfNeeded(); err != nil {
			continue
		}

		if gvrs, err = m.mapper.ResourcesFor(input); err != nil {
			// Triggers the mapper to be recreated
			m.lastError = err
			glog.Errorf("Failed to get the resources for %v: %v", input, err)
			continue
		}
	}

	return
}

func (m *invalidationMapper) RESTMapping(gk schema.GroupKind, versions ...string) (mapping *meta.RESTMapping, err error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for i := 1; i <= 2; i++ {
		if err = m.RefreshIfNeeded(); err != nil {
			continue
		}

		if mapping, err = m.mapper.RESTMapping(gk, versions...); err != nil {
			// Triggers the mapper to be recreated
			m.lastError = err
			glog.Errorf("Failed to get the mapping for %v: %v", gk, err)
			continue
		}
	}

	return
}

func (m *invalidationMapper) RESTMappings(gk schema.GroupKind, versions ...string) (mappings []*meta.RESTMapping, err error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for i := 1; i <= 2; i++ {
		if err = m.RefreshIfNeeded(); err != nil {
			continue
		}

		if mappings, err = m.mapper.RESTMappings(gk, versions...); err != nil {
			// Triggers the mapper to be recreated
			m.lastError = err
			glog.Errorf("Failed to get the mappings for %v: %v", gk, err)
			continue
		}
	}

	return
}

func (m *invalidationMapper) ResourceSingularizer(resource string) (singular string, err error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for i := 1; i <= 2; i++ {
		if err = m.RefreshIfNeeded(); err != nil {
			continue
		}

		if singular, err = m.mapper.ResourceSingularizer(resource); err != nil {
			// Triggers the mapper to be recreated
			m.lastError = err
			glog.Errorf("Failed to get the singular for %s: %v", resource, err)
			continue
		}
	}

	return
}
