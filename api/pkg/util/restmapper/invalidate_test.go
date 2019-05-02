package restmapper

import (
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type fakeMapper struct {
	err      error
	singular string
}

func (m *fakeMapper) ResourceSingularizer(resource string) (singular string, err error) {
	return m.singular, m.err
}

func (m *fakeMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, nil
}

func (m *fakeMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	return nil, nil
}

func (m *fakeMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	return schema.GroupVersionResource{}, nil
}

func (m *fakeMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	return nil, nil
}

func (m *fakeMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	return nil, nil
}

func (m *fakeMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	return nil, nil
}

func TestInvalidationMapper_ResourceSingularizer(t *testing.T) {
	const expectedSingular = "Cluster"
	calledCreateFakeMapper := 0
	createFakeMapper := func() (meta.RESTMapper, error) {
		defer func() {
			calledCreateFakeMapper++
		}()

		// On the first calledCreateFakeMapper create a mapper which fails the ResourceSingularizer call
		if calledCreateFakeMapper == 0 {
			return &fakeMapper{err: errors.New("some fake error"), singular: ""}, nil
		}
		// After the first calledCreateFakeMapper create a mapper which succeeds the ResourceSingularizer call
		return &fakeMapper{err: nil, singular: expectedSingular}, nil
	}

	invalidationMapper := NewInvalidationRESTMapper(createFakeMapper)

	singular, err := invalidationMapper.ResourceSingularizer("Clusters")
	if err != nil {
		t.Fatalf("failed to call ResourceSingularizer: %v", err)
	}
	if singular != expectedSingular {
		t.Errorf("Expected ResourceSingularizer to return %s, but instead got %s", expectedSingular, singular)
	}

	if calledCreateFakeMapper != 2 {
		t.Errorf("Expected createFakeMapper to be called 2 times, but got %d", calledCreateFakeMapper)
	}
}
