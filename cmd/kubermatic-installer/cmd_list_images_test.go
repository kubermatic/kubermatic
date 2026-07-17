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

package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"k8c.io/kubermatic/v2/pkg/install/images"
)

func TestValidateListImagesFormat(t *testing.T) {
	testCases := []struct {
		format    string
		expectErr bool
	}{
		{format: "plain", expectErr: false},
		{format: "json", expectErr: false},
		{format: "", expectErr: true},
		{format: "yaml", expectErr: true},
		{format: "PLAIN", expectErr: true},
		{format: "csv", expectErr: true},
	}

	for _, tc := range testCases {
		err := validateListImagesFormat(tc.format)
		if tc.expectErr && err == nil {
			t.Errorf("format %q expected an error, got nil", tc.format)
		}
		if !tc.expectErr && err != nil {
			t.Errorf("format %q expected no error, got %v", tc.format, err)
		}
	}
}

func newTestCollection() *images.Collection {
	c := images.NewCollection()
	c.Insert("docker.io/nginxinc/nginx-unprivileged:1.20.1-alpine", images.RefTypeImage, "reconciler:mla-gateway")
	c.Insert("quay.io/kubermatic-mirror/helm-charts/cilium:1.18.10", images.RefTypeHelmChart, "application:cilium@1.18.10")
	// shared ref discovered by two sources, exercising dedup + source merging
	c.Insert("registry.example/shared:1.0", images.RefTypeImage, "reconciler:foo")
	c.Insert("registry.example/shared:1.0", images.RefTypeImage, "addon:bar")
	return c
}

func TestPrintCollectionPlain(t *testing.T) {
	var buf bytes.Buffer
	if err := printCollection(&buf, newTestCollection(), formatPlain); err != nil {
		t.Fatalf("printCollection failed: %v", err)
	}

	// plain output: sorted refs, one per line, nothing else
	want := strings.Join([]string{
		"docker.io/nginxinc/nginx-unprivileged:1.20.1-alpine",
		"quay.io/kubermatic-mirror/helm-charts/cilium:1.18.10",
		"registry.example/shared:1.0",
	}, "\n") + "\n"

	if got := buf.String(); got != want {
		t.Errorf("plain output mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestPrintCollectionJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := printCollection(&buf, newTestCollection(), formatJSON); err != nil {
		t.Fatalf("printCollection failed: %v", err)
	}

	var entries []listImageEntry
	if err := json.Unmarshal(buf.Bytes(), &entries); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw:\n%s", err, buf.String())
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// every entry must have a ref, a type and at least one source
	for _, e := range entries {
		if e.Ref == "" {
			t.Errorf("entry with empty ref: %+v", e)
		}
		if e.Type == "" {
			t.Errorf("entry %q has empty type", e.Ref)
		}
		if len(e.Sources) == 0 {
			t.Errorf("entry %q has no sources", e.Ref)
		}
	}

	// refs are emitted sorted; assert the ordering the consumers rely on
	refs := []string{entries[0].Ref, entries[1].Ref, entries[2].Ref}
	wantRefs := []string{
		"docker.io/nginxinc/nginx-unprivileged:1.20.1-alpine",
		"quay.io/kubermatic-mirror/helm-charts/cilium:1.18.10",
		"registry.example/shared:1.0",
	}
	for i := range wantRefs {
		if refs[i] != wantRefs[i] {
			t.Errorf("entry %d ref = %q, want %q", i, refs[i], wantRefs[i])
		}
	}

	// the helm-chart entry keeps its type
	if entries[1].Type != string(images.RefTypeHelmChart) {
		t.Errorf("cilium entry type = %q, want %q", entries[1].Type, images.RefTypeHelmChart)
	}

	// the shared entry merged both sources
	if entries[2].Ref != "registry.example/shared:1.0" {
		t.Fatalf("expected shared entry at index 2, got %q", entries[2].Ref)
	}
	wantSources := []string{"addon:bar", "reconciler:foo"}
	if len(entries[2].Sources) != 2 || entries[2].Sources[0] != wantSources[0] || entries[2].Sources[1] != wantSources[1] {
		t.Errorf("shared entry sources = %v, want %v", entries[2].Sources, wantSources)
	}
}

func TestPrintCollectionInvalidFormat(t *testing.T) {
	var buf bytes.Buffer
	if err := printCollection(&buf, images.NewCollection(), "bogus"); err == nil {
		t.Errorf("expected an error for bogus format, got nil")
	}
}
