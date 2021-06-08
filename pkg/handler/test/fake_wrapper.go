/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package test

import (
	"encoding/json"
	"io"
	"sort"
	"testing"

	"github.com/go-test/deep"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
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
		return k[i].Name < k[j].Name
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
		originalMarshalled, _ := json.Marshal(k)
		expectedMarshalled, _ := json.Marshal(expected)
		t.Logf("Original:\n---\n%s\n---\n", string(originalMarshalled))
		t.Logf("expected:\n---\n%s\n---\n", string(expectedMarshalled))
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

// NewServiceAccountV1SliceWrapper wraps []apiv1.ServiceAccount
// to provide convenient methods for tests
type NewServiceAccountV1SliceWrapper []apiv1.ServiceAccount

// Sort sorts the collection by name
func (k NewServiceAccountV1SliceWrapper) Sort() {
	sort.Slice(k, func(i, j int) bool {
		return k[i].Name > (k[j].Name)
	})
}

// DecodeOrDie reads and decodes json data from the reader
func (k *NewServiceAccountV1SliceWrapper) DecodeOrDie(r io.Reader, t *testing.T) *NewServiceAccountV1SliceWrapper {
	t.Helper()
	dec := json.NewDecoder(r)
	err := dec.Decode(k)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// EqualOrDie compares whether expected collection is equal to the actual one
func (k NewServiceAccountV1SliceWrapper) EqualOrDie(expected NewServiceAccountV1SliceWrapper, t *testing.T) {
	t.Helper()
	if diff := deep.Equal(k, expected); diff != nil {
		t.Errorf("actual slice is different that the expected one. Diff: %v", diff)
	}
}

// NewServiceAccountTokenV1SliceWrapper wraps []apiv1.ServiceAccountToken
// to provide convenient methods for tests
type NewServiceAccountTokenV1SliceWrapper []apiv1.PublicServiceAccountToken

// Sort sorts the collection by name
func (k NewServiceAccountTokenV1SliceWrapper) Sort() {
	sort.Slice(k, func(i, j int) bool {
		return k[i].Name < (k[j].Name)
	})
}

// DecodeOrDie reads and decodes json data from the reader
func (k *NewServiceAccountTokenV1SliceWrapper) DecodeOrDie(r io.Reader, t *testing.T) *NewServiceAccountTokenV1SliceWrapper {
	t.Helper()
	dec := json.NewDecoder(r)
	err := dec.Decode(k)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// EqualOrDie compares whether expected collection is equal to the actual one
func (k NewServiceAccountTokenV1SliceWrapper) EqualOrDie(expected NewServiceAccountTokenV1SliceWrapper, t *testing.T) {
	t.Helper()
	if diff := deep.Equal(k, expected); diff != nil {
		t.Errorf("actual slice is different that the expected one. Diff: %v", diff)
	}
}

// NewAddonSliceWrapper wraps []apiv1.Addon
// to provide convenient methods for tests
type NewAddonSliceWrapper []apiv1.Addon

// Sort sorts the collection by CreationTimestamp
func (k NewAddonSliceWrapper) Sort() {
	sort.Slice(k, func(i, j int) bool {
		return k[i].CreationTimestamp.Before(k[j].CreationTimestamp)
	})
}

// DecodeOrDie reads and decodes json data from the reader
func (k *NewAddonSliceWrapper) DecodeOrDie(r io.Reader, t *testing.T) *NewAddonSliceWrapper {
	t.Helper()
	dec := json.NewDecoder(r)
	err := dec.Decode(k)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// EqualOrDie compares whether expected collection is equal to the actual one
func (k NewAddonSliceWrapper) EqualOrDie(expected NewAddonSliceWrapper, t *testing.T) {
	t.Helper()
	if diff := deep.Equal(k, expected); diff != nil {
		t.Errorf("actual slice is different that the expected one. Diff: %v", diff)
	}
}

// NewRoleNameSliceWrapper wraps []apiv1.RoleName
// to provide convenient methods for tests
type NewRoleNameSliceWrapper []apiv1.RoleName

// Sort sorts the collection by CreationTimestamp
func (k NewRoleNameSliceWrapper) Sort() {
	sort.Slice(k, func(i, j int) bool {
		return k[i].Name < (k[j].Name)
	})
}

// DecodeOrDie reads and decodes json data from the reader
func (k *NewRoleNameSliceWrapper) DecodeOrDie(r io.Reader, t *testing.T) *NewRoleNameSliceWrapper {
	t.Helper()
	dec := json.NewDecoder(r)
	err := dec.Decode(k)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// EqualOrDie compares whether expected collection is equal to the actual one
func (k NewRoleNameSliceWrapper) EqualOrDie(expected NewRoleNameSliceWrapper, t *testing.T) {
	t.Helper()
	if diff := deep.Equal(k, expected); diff != nil {
		t.Errorf("actual slice is different that the expected one. Diff: %v", diff)
	}
}

// NewConstraintTemplateV1SliceWrapper wraps []apiv1.ConstraintTemplate
// to provide convenient methods for tests
type NewConstraintTemplateV1SliceWrapper []apiv2.ConstraintTemplate

// DecodeOrDie reads and decodes json data from the reader
func (k *NewConstraintTemplateV1SliceWrapper) DecodeOrDie(r io.Reader, t *testing.T) *NewConstraintTemplateV1SliceWrapper {
	t.Helper()
	dec := json.NewDecoder(r)
	err := dec.Decode(k)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// Sort sorts the collection by Name
func (k NewConstraintTemplateV1SliceWrapper) Sort() {
	sort.Slice(k, func(i, j int) bool {
		return k[i].Name < (k[j].Name)
	})
}

// EqualOrDie compares whether expected collection is equal to the actual one
func (k NewConstraintTemplateV1SliceWrapper) EqualOrDie(expected NewConstraintTemplateV1SliceWrapper, t *testing.T) {
	t.Helper()
	if diff := deep.Equal(k, expected); diff != nil {
		t.Errorf("actual slice is different that the expected one. Diff: %v", diff)
	}
}

// NewConstraintsSliceWrapper wraps []apiv2.Constraints
// to provide convenient methods for tests
type NewConstraintsSliceWrapper []apiv2.Constraint

// DecodeOrDie reads and decodes json data from the reader
func (k *NewConstraintsSliceWrapper) DecodeOrDie(r io.Reader, t *testing.T) *NewConstraintsSliceWrapper {
	t.Helper()
	dec := json.NewDecoder(r)
	err := dec.Decode(k)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// Sort sorts the collection by Name
func (k NewConstraintsSliceWrapper) Sort() {
	sort.Slice(k, func(i, j int) bool {
		return k[i].Name < (k[j].Name)
	})
}

// EqualOrDie compares whether expected collection is equal to the actual one
func (k NewConstraintsSliceWrapper) EqualOrDie(expected NewConstraintsSliceWrapper, t *testing.T) {
	t.Helper()
	if diff := deep.Equal(k, expected); diff != nil {
		t.Errorf("actual slice is different that the expected one. Diff: %v", diff)
	}
}

// NodeDeploymentSliceWrapper wraps []apiv1.NodeDeployment
// to provide convenient methods for tests
type NodeDeploymentSliceWrapper []apiv1.NodeDeployment

// Sort sorts the collection by CreationTimestamp
func (k NodeDeploymentSliceWrapper) Sort() {
	sort.Slice(k, func(i, j int) bool {
		return k[i].Name < k[j].Name
	})
}

// DecodeOrDie reads and decodes json data from the reader
func (k *NodeDeploymentSliceWrapper) DecodeOrDie(r io.Reader, t *testing.T) *NodeDeploymentSliceWrapper {
	t.Helper()
	dec := json.NewDecoder(r)
	err := dec.Decode(k)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// EqualOrDie compares whether expected collection is equal to the actual one
func (k NodeDeploymentSliceWrapper) EqualOrDie(expected NodeDeploymentSliceWrapper, t *testing.T) {
	t.Helper()
	if diff := deep.Equal(k, expected); diff != nil {
		t.Errorf("actual slice is different that the expected one. Diff: %v", diff)
	}
}

// NewClusterTemplateSliceWrapper wraps []apiv2.ClusterTemplate
// to provide convenient methods for tests
type NewClusterTemplateSliceWrapper []apiv2.ClusterTemplate

// Sort sorts the collection by Name
func (k NewClusterTemplateSliceWrapper) Sort() {
	sort.Slice(k, func(i, j int) bool {
		return k[i].Name < k[j].Name
	})
}

// DecodeOrDie reads and decodes json data from the reader
func (k *NewClusterTemplateSliceWrapper) DecodeOrDie(r io.Reader, t *testing.T) *NewClusterTemplateSliceWrapper {
	t.Helper()
	dec := json.NewDecoder(r)
	err := dec.Decode(k)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// EqualOrDie compares whether expected collection is equal to the actual one
func (k NewClusterTemplateSliceWrapper) EqualOrDie(expected NewClusterTemplateSliceWrapper, t *testing.T) {
	t.Helper()
	if diff := deep.Equal(k, expected); diff != nil {
		t.Errorf("actual slice is different that the expected one. Diff: %v", diff)
	}
}
