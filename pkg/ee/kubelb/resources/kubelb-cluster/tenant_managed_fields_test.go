//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2026 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package resources

import (
	"encoding/json"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const testTenantFieldManager = "kkp-kubelb-controller"

func TestTenantManagedFieldsMigrationPatch(t *testing.T) {
	apiVersion := KubelbTenantGVK.GroupVersion().String()
	entry := func(manager string, operation metav1.ManagedFieldsOperationType, fields string) metav1.ManagedFieldsEntry {
		return metav1.ManagedFieldsEntry{
			Manager: manager, Operation: operation, APIVersion: apiVersion,
			FieldsType: "FieldsV1", FieldsV1: metav1.NewFieldsV1(fields),
		}
	}
	legacy := func(fields string) metav1.ManagedFieldsEntry {
		return entry(legacyTenantFieldManager, metav1.ManagedFieldsOperationUpdate, fields)
	}
	applied := func(fields string) metav1.ManagedFieldsEntry {
		return entry(testTenantFieldManager, metav1.ManagedFieldsOperationApply, fields)
	}
	otherVersion := legacy(`{"f:spec":{"f:loadBalancer":{"f:limit":{}}}}`)
	otherVersion.APIVersion = "kubelb.k8c.io/v1beta1"
	status := legacy(`{"f:status":{"f:phase":{}}}`)
	status.Subresource = "status"
	legacyApply := legacy(`{"f:spec":{"f:ingress":{"f:class":{}}}}`)
	legacyApply.Operation = metav1.ManagedFieldsOperationApply
	unknownFieldsType := legacy(`{"f:spec":{"f:ingress":{"f:class":{}}}}`)
	unknownFieldsType.FieldsType = "FutureFields"
	otherApplyVersion := applied(`{"f:spec":{"f:ingress":{"f:class":{}}}}`)
	otherApplyVersion.APIVersion = "kubelb.k8c.io/v1beta1"
	appliedStatus := applied(`{"f:status":{"f:phase":{}}}`)
	appliedStatus.Subresource = "status"
	admin := entry("kubectl-edit", metav1.ManagedFieldsOperationUpdate, `{"f:spec":{"f:loadBalancer":{"f:limit":{}}}}`)
	manager := entry("kubelb", metav1.ManagedFieldsOperationUpdate, `{"f:metadata":{"f:finalizers":{}},"f:spec":{"f:allowedDomains":{}}}`)

	tests := []struct {
		name        string
		desiredSpec string
		entries     []metav1.ManagedFieldsEntry
		want        []metav1.ManagedFieldsEntry
	}{
		{name: "no ownership"},
		{name: "unknown and unrelated ownership", entries: []metav1.ManagedFieldsEntry{admin, manager, otherVersion, status, legacyApply, unknownFieldsType, otherApplyVersion}},
		{name: "legacy metadata outside KKP labels", entries: []metav1.ManagedFieldsEntry{legacy(`{"f:metadata":{"f:annotations":{"f:example.com/note":{}},"f:labels":{".":{},"f:example.com/label":{}},"f:finalizers":{}}}`)}},
		{
			name:        "create apply owner from explicit defaults and all KKP labels",
			desiredSpec: `{"gatewayAPI":{"class":"eg-project"},"waf":{"enableTenantPolicies":false}}`,
			entries:     []metav1.ManagedFieldsEntry{legacy(`{"f:metadata":{"f:labels":{"f:kubermatic.k8c.io/cluster-name":{},"f:kubermatic.k8c.io/cluster-external-name":{},"f:kubermatic.k8c.io/cluster-project-id":{}}},"f:spec":{".":{},"f:gatewayAPI":{"f:class":{}},"f:waf":{"f:enableTenantPolicies":{}}}}`)},
			want:        []metav1.ManagedFieldsEntry{applied(`{"f:metadata":{"f:labels":{"f:kubermatic.k8c.io/cluster-name":{},"f:kubermatic.k8c.io/cluster-external-name":{},"f:kubermatic.k8c.io/cluster-project-id":{}}},"f:spec":{".":{},"f:gatewayAPI":{"f:class":{}},"f:waf":{"f:enableTenantPolicies":{}}}}`)},
		},
		{
			name:        "preserve unrelated legacy fields and all other owners",
			desiredSpec: `{"loadBalancer":{"limit":0}}`,
			entries: []metav1.ManagedFieldsEntry{
				legacy(`{"f:metadata":{"f:annotations":{"f:example.com/note":{}},"f:labels":{".":{},"f:example.com/label":{},"f:kubermatic.k8c.io/cluster-name":{}},"f:finalizers":{}},"f:spec":{"f:loadBalancer":{"f:limit":{}}},"f:status":{"f:phase":{}}}`),
				admin, manager, otherVersion, status, legacyApply, unknownFieldsType, appliedStatus,
			},
			want: []metav1.ManagedFieldsEntry{
				legacy(`{"f:metadata":{"f:annotations":{"f:example.com/note":{}},"f:labels":{".":{},"f:example.com/label":{}},"f:finalizers":{}},"f:status":{"f:phase":{}}}`),
				admin, manager, otherVersion, status, legacyApply, unknownFieldsType, appliedStatus,
				applied(`{"f:metadata":{"f:labels":{"f:kubermatic.k8c.io/cluster-name":{}}},"f:spec":{"f:loadBalancer":{"f:limit":{}}}}`),
			},
		},
		{
			name:        "union mapping entries with existing apply owner",
			desiredSpec: `{"gatewayAPI":{"classMappings":[{"source":"public","target":"eg-public"}]}}`,
			entries: []metav1.ManagedFieldsEntry{
				legacy(`{"f:spec":{"f:gatewayAPI":{"f:classMappings":{"k:{\"source\":\"public\"}":{".":{},"f:source":{},"f:target":{}}}}}}`),
				applied(`{"f:spec":{"f:gatewayAPI":{"f:classMappings":{"k:{\"source\":\"internal\"}":{".":{},"f:source":{},"f:target":{}}}},"f:waf":{"f:enableTenantPolicies":{}}}}`),
			},
			want: []metav1.ManagedFieldsEntry{
				applied(`{"f:spec":{"f:gatewayAPI":{"f:classMappings":{"k:{\"source\":\"internal\"}":{".":{},"f:source":{},"f:target":{}},"k:{\"source\":\"public\"}":{".":{},"f:source":{},"f:target":{}}}},"f:waf":{"f:enableTenantPolicies":{}}}}`),
			},
		},
		{
			name:        "legacy and apply co-ownership becomes sole KKP workflow",
			desiredSpec: `{"ingress":{"class":""}}`,
			entries:     []metav1.ManagedFieldsEntry{legacy(`{"f:spec":{"f:ingress":{"f:class":{}}}}`), applied(`{"f:spec":{"f:ingress":{"f:class":{}}}}`)},
			want:        []metav1.ManagedFieldsEntry{applied(`{"f:spec":{"f:ingress":{"f:class":{}}}}`)},
		},
		{
			name:        "retain ambiguous legacy defaults and omitted sibling fields",
			desiredSpec: `{"timeouts":{"connect":"10s"}}`,
			entries:     []metav1.ManagedFieldsEntry{legacy(`{"f:spec":{".":{},"f:allowedDomains":{},"f:timeouts":{".":{},"f:connect":{},"f:request":{}}}}`)},
			want: []metav1.ManagedFieldsEntry{
				legacy(`{"f:spec":{"f:allowedDomains":{},"f:timeouts":{"f:request":{}}}}`),
				applied(`{"f:spec":{".":{},"f:timeouts":{".":{},"f:connect":{}}}}`),
			},
		},
		{
			name:    "no project defaults preserves all legacy spec ownership",
			entries: []metav1.ManagedFieldsEntry{legacy(`{"f:metadata":{"f:labels":{"f:kubermatic.k8c.io/cluster-name":{}}},"f:spec":{".":{},"f:allowedDomains":{},"f:ingress":{"f:class":{}}}}`)},
			want: []metav1.ManagedFieldsEntry{
				legacy(`{"f:spec":{".":{},"f:allowedDomains":{},"f:ingress":{"f:class":{}}}}`),
				applied(`{"f:metadata":{"f:labels":{"f:kubermatic.k8c.io/cluster-name":{}}}}`),
			},
		},
		{
			name:        "explicit empty atomic list adopts its legacy ownership",
			desiredSpec: `{"allowedDomains":[]}`,
			entries:     []metav1.ManagedFieldsEntry{legacy(`{"f:spec":{"f:allowedDomains":{}}}`)},
			want:        []metav1.ManagedFieldsEntry{applied(`{"f:spec":{"f:allowedDomains":{}}}`)},
		},
		{
			name:        "explicit null adopts its legacy field without adopting siblings",
			desiredSpec: `{"timeouts":{"connect":null}}`,
			entries:     []metav1.ManagedFieldsEntry{legacy(`{"f:spec":{"f:timeouts":{"f:connect":{},"f:request":{}}}}`)},
			want: []metav1.ManagedFieldsEntry{
				legacy(`{"f:spec":{"f:timeouts":{"f:request":{}}}}`),
				applied(`{"f:spec":{"f:timeouts":{"f:connect":{}}}}`),
			},
		},
		{
			name:        "match mapping by source while preserving omitted entry and child",
			desiredSpec: `{"gatewayAPI":{"classMappings":[{"source":"new","target":"eg-new"},{"source":"public","target":"eg-public"}]}}`,
			entries: []metav1.ManagedFieldsEntry{
				legacy(`{"f:spec":{"f:gatewayAPI":{"f:classMappings":{"k:{\"source\":\"internal\"}":{".":{},"f:source":{},"f:target":{}},"k:{\"source\":\"public\"}":{".":{},"f:source":{},"f:target":{},"f:extra":{}}}}}}`),
			},
			want: []metav1.ManagedFieldsEntry{
				legacy(`{"f:spec":{"f:gatewayAPI":{"f:classMappings":{"k:{\"source\":\"internal\"}":{".":{},"f:source":{},"f:target":{}},"k:{\"source\":\"public\"}":{"f:extra":{}}}}}}`),
				applied(`{"f:spec":{"f:gatewayAPI":{"f:classMappings":{"k:{\"source\":\"public\"}":{".":{},"f:source":{},"f:target":{}}}}}}`),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			existing := &unstructured.Unstructured{}
			existing.SetGroupVersionKind(KubelbTenantGVK)
			existing.SetName("test-cluster")
			existing.SetResourceVersion("42")
			existing.SetManagedFields(tc.entries)
			before := existing.DeepCopy()
			desired := tenantMigrationDesired(t, tc.desiredSpec)
			desiredBefore := desired.DeepCopy()
			patch, err := TenantManagedFieldsMigrationPatch(existing, desired, testTenantFieldManager)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(existing, before) {
				t.Fatal("migration mutated its input")
			}
			if !reflect.DeepEqual(desired, desiredBefore) {
				t.Fatal("migration mutated the desired Tenant")
			}
			if tc.want == nil {
				if patch != nil {
					t.Fatalf("unexpected patch: %s", patch)
				}
				return
			}
			if patch == nil {
				t.Fatal("expected ownership migration patch")
			}
			var operations []struct {
				Op    string
				Path  string
				Value json.RawMessage
			}
			if err := json.Unmarshal(patch, &operations); err != nil {
				t.Fatal(err)
			}
			if len(operations) != 2 || operations[0].Op != "replace" || operations[0].Path != "/metadata/managedFields" || operations[1].Op != "replace" || operations[1].Path != "/metadata/resourceVersion" || string(operations[1].Value) != `"42"` {
				t.Fatalf("patch must change only ownership and guard observed resource version: %s", patch)
			}
			var got []metav1.ManagedFieldsEntry
			if err := json.Unmarshal(operations[0].Value, &got); err != nil {
				t.Fatal(err)
			}
			gotJSON, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			wantJSON, err := json.Marshal(tc.want)
			if err != nil {
				t.Fatal(err)
			}
			var gotValue, wantValue any
			if err := json.Unmarshal(gotJSON, &gotValue); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(wantJSON, &wantValue); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(gotValue, wantValue) {
				t.Fatalf("ownership mismatch:\ngot  %s\nwant %s", gotJSON, wantJSON)
			}
			existing.SetManagedFields(got)
			second, err := TenantManagedFieldsMigrationPatch(existing, desired, testTenantFieldManager)
			if err != nil {
				t.Fatal(err)
			}
			if second != nil {
				t.Fatalf("migration is not idempotent: %s", second)
			}
		})
	}
}

func TestTenantManagedFieldsMigrationPatchErrors(t *testing.T) {
	tests := []struct {
		name            string
		fields          string
		resourceVersion string
	}{
		{name: "malformed field set", fields: `{"f:spec":1}`, resourceVersion: "42"},
		{name: "missing resource version", fields: `{"f:spec":{"f:ingress":{"f:class":{}}}}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			existing := &unstructured.Unstructured{}
			existing.SetGroupVersionKind(KubelbTenantGVK)
			existing.SetResourceVersion(tc.resourceVersion)
			existing.SetManagedFields([]metav1.ManagedFieldsEntry{{
				Manager: legacyTenantFieldManager, Operation: metav1.ManagedFieldsOperationUpdate,
				APIVersion: existing.GetAPIVersion(), FieldsType: "FieldsV1", FieldsV1: metav1.NewFieldsV1(tc.fields),
			}})
			patch, err := TenantManagedFieldsMigrationPatch(existing, tenantMigrationDesired(t, `{"ingress":{"class":"eg-project"}}`), testTenantFieldManager)
			if err == nil || patch != nil {
				t.Fatalf("expected error without patch, got %s, %v", patch, err)
			}
		})
	}
}

// API versions do not form part of an apply manager's identity. Ownership from
// another version must not be overwritten by adding a second apply entry.
func TestTenantManagedFieldsMigrationPatchRejectsUnknownApplyVersion(t *testing.T) {
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(KubelbTenantGVK)
	existing.SetResourceVersion("42")
	existing.SetManagedFields([]metav1.ManagedFieldsEntry{
		{Manager: legacyTenantFieldManager, Operation: metav1.ManagedFieldsOperationUpdate,
			APIVersion: existing.GetAPIVersion(), FieldsType: "FieldsV1",
			FieldsV1: metav1.NewFieldsV1(`{"f:spec":{"f:ingress":{"f:class":{}}}}`)},
		{Manager: testTenantFieldManager, Operation: metav1.ManagedFieldsOperationApply,
			APIVersion: "kubelb.k8c.io/v1beta1", FieldsType: "FieldsV1",
			FieldsV1: metav1.NewFieldsV1(`{"f:spec":{"f:loadBalancer":{"f:limit":{}}}}`)},
	})
	before := existing.DeepCopy()
	patch, err := TenantManagedFieldsMigrationPatch(existing, tenantMigrationDesired(t, `{"ingress":{"class":"eg-project"}}`), testTenantFieldManager)
	if err == nil || patch != nil {
		t.Fatalf("expected error without patch, got %s, %v", patch, err)
	}
	if !reflect.DeepEqual(existing, before) {
		t.Fatal("migration mutated its input")
	}
}

func tenantMigrationDesired(t *testing.T, spec string) *unstructured.Unstructured {
	t.Helper()
	desired := &unstructured.Unstructured{}
	desired.SetGroupVersionKind(KubelbTenantGVK)
	desired.SetName("test-cluster")
	desired.SetLabels(map[string]string{
		TenantClusterNameLabelKey:         "test-cluster",
		TenantClusterExternalNameLabelKey: "test.example.com",
		TenantProjectIDLabelKey:           "test-project",
	})
	if spec != "" {
		var fields map[string]any
		if err := json.Unmarshal([]byte(spec), &fields); err != nil {
			t.Fatal(err)
		}
		desired.Object["spec"] = fields
	}
	return desired
}
