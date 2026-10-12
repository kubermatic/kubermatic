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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/structured-merge-diff/v6/fieldpath"
	"sigs.k8s.io/structured-merge-diff/v6/value"
)

const legacyTenantFieldManager = "seed-controller-manager"

// TenantManagedFieldsMigrationPatch transfers legacy KKP ownership only for
// fields explicitly included in the desired Tenant. Create ownership also
// includes API defaults, so omitted legacy fields must remain with their old
// owner: their original source cannot be recovered from managedFields.
// Fields already owned by another manager are not transferred.
func TenantManagedFieldsMigrationPatch(existing, desired *unstructured.Unstructured, fieldManager string) ([]byte, error) {
	apiVersion := KubelbTenantGVK.GroupVersion().String()
	entries := existing.GetManagedFields()
	migrated := make([]metav1.ManagedFieldsEntry, 0, len(entries)+1)
	transferred := fieldpath.NewSet()
	var transferTime *metav1.Time
	for _, entry := range entries {
		if entry.Manager != legacyTenantFieldManager || entry.Operation != metav1.ManagedFieldsOperationUpdate || entry.APIVersion != apiVersion || entry.Subresource != "" || entry.FieldsType != "FieldsV1" {
			migrated = append(migrated, entry)
			continue
		}

		fields, err := tenantManagedFieldSet(entry)
		if err != nil {
			return nil, err
		}
		remaining := fieldpath.NewSet()
		owned := fieldpath.NewSet()
		fields.Iterate(func(path fieldpath.Path) {
			if isKKPTenantField(path) && tenantFieldIsSpecified(desired.Object, path) {
				owned.Insert(path)
			} else {
				remaining.Insert(path)
			}
		})
		if owned.Empty() {
			migrated = append(migrated, entry)
			continue
		}
		transferred = transferred.Union(owned)
		if transferTime == nil {
			transferTime = entry.Time
		}
		if !remaining.Empty() {
			raw, err := remaining.ToJSON()
			if err != nil {
				return nil, fmt.Errorf("failed to encode remaining legacy tenant fields: %w", err)
			}
			entry.FieldsV1 = metav1.NewFieldsV1(string(raw))
			migrated = append(migrated, entry)
		}
	}
	if transferred.Empty() {
		return nil, nil
	}
	if existing.GetResourceVersion() == "" {
		return nil, fmt.Errorf("cannot migrate tenant field ownership without a resource version")
	}

	applyIndex := -1
	for i, entry := range migrated {
		if entry.Manager != fieldManager || entry.Operation != metav1.ManagedFieldsOperationApply || entry.Subresource != "" {
			continue
		}
		// An apply manager's identity excludes APIVersion. Adding another
		// entry here would overwrite ownership that requires API conversion.
		if entry.APIVersion != apiVersion {
			return nil, fmt.Errorf("cannot migrate tenant apply ownership from API version %q", entry.APIVersion)
		}
		if applyIndex != -1 {
			return nil, fmt.Errorf("found multiple tenant apply field managers for %q", fieldManager)
		}
		fields, err := tenantManagedFieldSet(entry)
		if err != nil {
			return nil, err
		}
		transferred = transferred.Union(fields)
		applyIndex = i
	}
	if applyIndex == -1 {
		applyIndex = len(migrated)
		migrated = append(migrated, metav1.ManagedFieldsEntry{
			Manager:    fieldManager,
			Operation:  metav1.ManagedFieldsOperationApply,
			APIVersion: apiVersion,
			Time:       transferTime,
			FieldsType: "FieldsV1",
		})
	}
	raw, err := transferred.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to encode migrated tenant fields: %w", err)
	}
	migrated[applyIndex].FieldsV1 = metav1.NewFieldsV1(string(raw))

	// As in client-go's csaupgrade helper, replacing resourceVersion makes
	// concurrent writes return an API conflict instead of overwriting ownership.
	return json.Marshal([]map[string]any{
		{"op": "replace", "path": "/metadata/managedFields", "value": migrated},
		{"op": "replace", "path": "/metadata/resourceVersion", "value": existing.GetResourceVersion()},
	})
}

func tenantManagedFieldSet(entry metav1.ManagedFieldsEntry) (*fieldpath.Set, error) {
	if entry.FieldsType != "FieldsV1" || entry.FieldsV1 == nil {
		return nil, fmt.Errorf("unsupported tenant field ownership for manager %q", entry.Manager)
	}
	fields := fieldpath.NewSet()
	if err := fields.FromJSON(entry.FieldsV1.GetRawReader()); err != nil {
		return nil, fmt.Errorf("failed to decode tenant fields for manager %q: %w", entry.Manager, err)
	}
	return fields, nil
}

func isKKPTenantField(path fieldpath.Path) bool {
	if len(path) > 0 && path[0].FieldName != nil && *path[0].FieldName == "spec" {
		return true
	}
	if len(path) != 3 || path[0].FieldName == nil || *path[0].FieldName != "metadata" || path[1].FieldName == nil || *path[1].FieldName != "labels" || path[2].FieldName == nil {
		return false
	}
	switch *path[2].FieldName {
	case TenantClusterNameLabelKey, TenantClusterExternalNameLabelKey, TenantProjectIDLabelKey:
		return true
	default:
		return false
	}
}

// tenantFieldIsSpecified follows the API server's recorded ownership path in
// the desired object. Associative list entries are matched by their keys, not
// position, so adopting one GatewayClass mapping preserves omitted mappings.
func tenantFieldIsSpecified(object any, path fieldpath.Path) bool {
	if len(path) == 0 {
		return true
	}
	element := path[0]
	switch {
	case element.FieldName != nil:
		fields, ok := object.(map[string]any)
		if !ok {
			return false
		}
		child, found := fields[*element.FieldName]
		return found && tenantFieldIsSpecified(child, path[1:])
	case element.Key != nil:
		items, ok := object.([]any)
		if !ok {
			return false
		}
		for _, item := range items {
			fields, ok := item.(map[string]any)
			if !ok {
				continue
			}
			matches := true
			for _, key := range *element.Key {
				field, found := fields[key.Name]
				if !found || !value.Equals(value.NewValueInterface(field), key.Value) {
					matches = false
					break
				}
			}
			if matches && tenantFieldIsSpecified(item, path[1:]) {
				return true
			}
		}
	}
	// Tenant list fields are atomic or keyed maps. Preserve any ownership
	// using other selectors instead of guessing how to adopt it.
	return false
}
