package test

import (
	"encoding/json"
	"io"
	"sort"
	"testing"

	"github.com/go-test/deep"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
)

// NewSSHKeyV1SliceWrapper wraps []apiv1.SSHKey
// to provide convenient methods for tests
type NewSSHKeyV1SliceWrapper []apiv1.SSHKey

// Sort sorts the collection by CreationTimestamp
func (k NewSSHKeyV1SliceWrapper) Sort() {
	sort.Slice(k, func(i, j int) bool {
		return k[i].CreationTimestamp.Before(k[j].CreationTimestamp)
	})
}

// DecodeOrDie reads and decodes json data from the reader
func (k *NewSSHKeyV1SliceWrapper) DecodeOrDie(r io.Reader, t *testing.T) *NewSSHKeyV1SliceWrapper {
	t.Helper()
	dec := json.NewDecoder(r)
	err := dec.Decode(k)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// EqualOrDie compares whether expected collection is equal to the actual one
func (k NewSSHKeyV1SliceWrapper) EqualOrDie(expected NewSSHKeyV1SliceWrapper, t *testing.T) {
	t.Helper()
	if diff := deep.Equal(k, expected); diff != nil {
		t.Errorf("actual slice is different that the expected one. Diff: %v", diff)
	}
}

// NewUserV1SliceWrapper wraps []apiv1.User
// to provide convenient methods for tests
type NewUserV1SliceWrapper []apiv1.User

// Sort sorts the collection by CreationTimestamp
func (k NewUserV1SliceWrapper) Sort() {
	sort.Slice(k, func(i, j int) bool {
		return k[i].CreationTimestamp.Before(k[j].CreationTimestamp)
	})
}

// DecodeOrDie reads and decodes json data from the reader
func (k *NewUserV1SliceWrapper) DecodeOrDie(r io.Reader, t *testing.T) *NewUserV1SliceWrapper {
	t.Helper()
	dec := json.NewDecoder(r)
	err := dec.Decode(k)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// EqualOrDie compares whether expected collection is equal to the actual one
func (k NewUserV1SliceWrapper) EqualOrDie(expected NewUserV1SliceWrapper, t *testing.T) {
	t.Helper()
	if diff := deep.Equal(k, expected); diff != nil {
		t.Errorf("actual slice is different that the expected one. Diff: %v", diff)
	}
}

// NodeV1SliceWrapper wraps []apiv1.Node
// to provide convenient methods for tests
type NodeV1SliceWrapper []apiv1.Node

// Sort sorts the collection by CreationTimestamp
func (k NodeV1SliceWrapper) Sort() {
	sort.Slice(k, func(i, j int) bool {
		return k[i].CreationTimestamp.Before(k[j].CreationTimestamp)
	})
}

// DecodeOrDie reads and decodes json data from the reader
func (k *NodeV1SliceWrapper) DecodeOrDie(r io.Reader, t *testing.T) *NodeV1SliceWrapper {
	t.Helper()
	dec := json.NewDecoder(r)
	err := dec.Decode(k)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// EqualOrDie compares whether expected collection is equal to the actual one
func (k NodeV1SliceWrapper) EqualOrDie(expected NodeV1SliceWrapper, t *testing.T) {
	t.Helper()
	if diff := deep.Equal(k, expected); diff != nil {
		t.Errorf("actual slice is different that the expected one. Diff: %v", diff)
	}
}

// NewClusterV1SliceWrapper wraps []apiv1.Cluster
// to provide convenient methods for tests
type NewClusterV1SliceWrapper []apiv1.Cluster

// Sort sorts the collection by CreationTimestamp
func (k NewClusterV1SliceWrapper) Sort() {
	sort.Slice(k, func(i, j int) bool {
		return k[i].CreationTimestamp.Before(k[j].CreationTimestamp)
	})
}

// DecodeOrDie reads and decodes json data from the reader
func (k *NewClusterV1SliceWrapper) DecodeOrDie(r io.Reader, t *testing.T) *NewClusterV1SliceWrapper {
	t.Helper()
	dec := json.NewDecoder(r)
	err := dec.Decode(k)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// EqualOrDie compares whether expected collection is equal to the actual one
func (k NewClusterV1SliceWrapper) EqualOrDie(expected NewClusterV1SliceWrapper, t *testing.T) {
	t.Helper()
	if diff := deep.Equal(k, expected); diff != nil {
		t.Errorf("actual slice is different that the expected one. Diff: %v", diff)
	}
}

// ProjectV1SliceWrapper wraps []apiv1.Project
// to provide convenient methods for tests
type ProjectV1SliceWrapper []apiv1.Project

// Sort sorts the collection by CreationTimestamp
func (k ProjectV1SliceWrapper) Sort() {
	sort.Slice(k, func(i, j int) bool {
		return k[i].CreationTimestamp.Before(k[j].CreationTimestamp)
	})
}

// DecodeOrDie reads and decodes json data from the reader
func (k *ProjectV1SliceWrapper) DecodeOrDie(r io.Reader, t *testing.T) *ProjectV1SliceWrapper {
	t.Helper()
	dec := json.NewDecoder(r)
	err := dec.Decode(k)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// EqualOrDie compares whether expected collection is equal to the actual one
func (k ProjectV1SliceWrapper) EqualOrDie(expected ProjectV1SliceWrapper, t *testing.T) {
	t.Helper()
	if diff := deep.Equal(k, expected); diff != nil {
		t.Errorf("actual slice is different that the expected one. Diff: %v", diff)
	}
}
