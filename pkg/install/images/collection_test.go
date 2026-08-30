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
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"
)

func TestCollectionInsertDedup(t *testing.T) {
	c := NewCollection()
	c.Insert("registry.example/foo:1.0", RefTypeImage, "reconciler:foo")
	c.Insert("registry.example/foo:1.0", RefTypeImage, "addon:bar")

	entries := c.Sorted()
	if len(entries) != 1 {
		t.Fatalf("expected 1 deduped entry, got %d", len(entries))
	}

	want := []string{"addon:bar", "reconciler:foo"}
	if got := sets.List(entries[0].Sources); !reflect.DeepEqual(got, want) {
		t.Errorf("expected sources %v, got %v", want, got)
	}
}

func TestCollectionHelmChartNotDowngraded(t *testing.T) {
	c := NewCollection()
	// a chart ref is seen first as a helm-chart
	c.Insert("registry.example/charts/cilium:1.18.10", RefTypeHelmChart, "application:cilium@1.18.10")
	// later the same ref shows up as a plain image; it must stay a helm-chart
	c.Insert("registry.example/charts/cilium:1.18.10", RefTypeImage, "reconciler:weird")

	entries := c.Sorted()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Type != RefTypeHelmChart {
		t.Errorf("helm-chart entry was downgraded to %q", entries[0].Type)
	}
}

func TestCollectionUpgradeToHelmChart(t *testing.T) {
	c := NewCollection()
	c.Insert("registry.example/charts/x:1.0", RefTypeImage, "reconciler:x")
	c.Insert("registry.example/charts/x:1.0", RefTypeHelmChart, "application:x@1.0")

	entries := c.Sorted()
	if entries[0].Type != RefTypeHelmChart {
		t.Errorf("expected type upgraded to helm-chart, got %q", entries[0].Type)
	}
}

func TestCollectionMerge(t *testing.T) {
	a := NewCollection()
	a.Insert("a:1", RefTypeImage, "reconciler:a")
	a.Insert("shared:1", RefTypeImage, "reconciler:a")

	b := NewCollection()
	b.Insert("b:1", RefTypeImage, "addon:b")
	b.Insert("shared:1", RefTypeImage, "addon:shared")

	a.Merge(b)

	refs := a.RefList()
	want := []string{"a:1", "b:1", "shared:1"}
	if !reflect.DeepEqual(refs, want) {
		t.Errorf("RefList = %v, want %v", refs, want)
	}

	for _, entry := range a.Sorted() {
		if entry.Ref == "shared:1" {
			want := []string{"addon:shared", "reconciler:a"}
			if got := sets.List(entry.Sources); !reflect.DeepEqual(got, want) {
				t.Errorf("merged sources = %v, want %v", got, want)
			}
		}
	}
}

func TestCollectionFilterPrefix(t *testing.T) {
	c := NewCollection()
	c.Insert("quay.io/foo:1", RefTypeImage, "reconciler:foo")
	c.Insert("docker.io/bar:1", RefTypeImage, "reconciler:bar")
	c.FilterPrefix("quay.io")

	refs := c.RefList()
	if !reflect.DeepEqual(refs, []string{"quay.io/foo:1"}) {
		t.Errorf("FilterPrefix left %v, want only [quay.io/foo:1]", refs)
	}
}

func TestCollectionDeterministicOrdering(t *testing.T) {
	c := NewCollection()
	c.Insert("z:1", RefTypeImage, "src:z")
	c.Insert("a:1", RefTypeImage, "src:a")
	c.Insert("m:1", RefTypeImage, "src:m")

	first := c.RefList()
	// run again to ensure stability across calls
	second := c.RefList()
	if !reflect.DeepEqual(first, second) {
		t.Errorf("RefList not stable: %v vs %v", first, second)
	}

	want := []string{"a:1", "m:1", "z:1"}
	if !reflect.DeepEqual(first, want) {
		t.Errorf("RefList = %v, want sorted %v", first, want)
	}
}

func TestCollectionInsertEmptyRef(t *testing.T) {
	c := NewCollection()
	c.Insert("", RefTypeImage, "src:empty")
	if got := c.RefList(); len(got) != 0 {
		t.Errorf("empty ref should be ignored, got %v", got)
	}
}
