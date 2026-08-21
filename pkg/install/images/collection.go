/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package images

import (
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

// RefType tells whether a collected reference is a container image or an OCI
// Helm chart reference that mirror-images / list-images would copy.
type RefType string

const (
	RefTypeImage     RefType = "image"
	RefTypeHelmChart RefType = "helm-chart"
)

// CollectedRef is a container image or OCI Helm chart reference together with
// the deduplicated set of sources it was discovered in.
//
// Source labels use a "<kind>:<name>" vocabulary:
//   - "reconciler:<name>"          - Go reconciler factory (e.g. "reconciler:etcd")
//   - "addon:<name>"               - rendered addon manifest (e.g. "addon:canal")
//   - "etcd-backup:store" / "etcd-backup:delete"
//   - "chart:<name>"               - vendored Helm chart under --charts-directory
//   - "application:<chart>@<ver>"  - system/default/external catalog application
//   - "config:mirrorImages"        - KubermaticConfig spec.mirrorImages
//   - "static"                     - hardcoded extras (web terminal)
type CollectedRef struct {
	Ref     string
	Type    RefType
	Sources sets.Set[string]
}

// Collection aggregates refs, deduplicating by ref string and merging the
// source labels of duplicates.
type Collection struct {
	refs map[string]*CollectedRef
}

func NewCollection() *Collection {
	return &Collection{refs: map[string]*CollectedRef{}}
}

// Insert adds a ref with one source label. If the ref already exists, the
// source is merged into the existing entry. An entry already typed as a Helm
// chart is never downgraded to a plain image by a later Insert.
func (c *Collection) Insert(ref string, t RefType, source string) {
	if ref == "" {
		return
	}

	if existing, ok := c.refs[ref]; ok {
		existing.Sources.Insert(source)
		// a helm-chart ref must stay a helm-chart even if a later insert sees it as an image
		if existing.Type == RefTypeHelmChart {
			return
		}
		if t == RefTypeHelmChart {
			existing.Type = RefTypeHelmChart
		}
		return
	}

	entry := &CollectedRef{
		Ref:     ref,
		Type:    t,
		Sources: sets.New(source),
	}
	c.refs[ref] = entry
}

// InsertAll adds many refs that share a type and a single source label.
func (c *Collection) InsertAll(refs []string, t RefType, source string) {
	for _, ref := range refs {
		c.Insert(ref, t, source)
	}
}

// Merge folds another collection into this one, merging source labels.
func (c *Collection) Merge(other *Collection) {
	if other == nil {
		return
	}
	for _, entry := range other.refs {
		for source := range entry.Sources {
			c.Insert(entry.Ref, entry.Type, source)
		}
	}
}

// FilterPrefix drops entries whose ref does not begin with prefix. When prefix
// is empty the collection is left unchanged.
func (c *Collection) FilterPrefix(prefix string) {
	if prefix == "" {
		return
	}
	for ref := range c.refs {
		if !strings.HasPrefix(ref, prefix) {
			delete(c.refs, ref)
		}
	}
}

// RefList returns all ref strings, sorted. Use this for the mirror/archive
// path that only cares about which refs to copy.
func (c *Collection) RefList() []string {
	refs := make([]string, 0, len(c.refs))
	for ref := range c.refs {
		refs = append(refs, ref)
	}
	slices.Sort(refs)
	return refs
}

// Sorted returns all entries sorted by ref, each with a sorted Sources slice,
// for deterministic output.
func (c *Collection) Sorted() []CollectedRef {
	refs := c.RefList()
	out := make([]CollectedRef, 0, len(refs))
	for _, ref := range refs {
		entry := c.refs[ref]
		out = append(out, CollectedRef{
			Ref:     entry.Ref,
			Type:    entry.Type,
			Sources: sets.New(sets.List(entry.Sources)...),
		})
	}
	return out
}
