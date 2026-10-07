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
	"reflect"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestTenantDefaultSpecRepresentations(t *testing.T) {
	cluster := &kubermaticv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"}}
	decoded := &runtime.RawExtension{Object: &unstructured.Unstructured{
		Object: map[string]any{"gatewayAPI": map[string]any{"class": "eg-object"}},
	}}
	tenant, err := Tenant(cluster, decoded)
	if err != nil {
		t.Fatalf("build Tenant from decoded defaults: %v", err)
	}
	class, found, err := unstructured.NestedString(tenant.Object, "spec", "gatewayAPI", "class")
	if err != nil || !found || class != "eg-object" {
		t.Fatalf("decoded GatewayClass = %q, found=%t, error=%v", class, found, err)
	}

	raw := &runtime.RawExtension{Raw: []byte(`{"gatewayAPI":{"class":"eg-object"}}`)}
	fromRaw, err := Tenant(cluster, raw)
	if err != nil {
		t.Fatalf("build Tenant from raw defaults: %v", err)
	}
	if !reflect.DeepEqual(tenant.Object, fromRaw.Object) {
		t.Errorf("raw and decoded defaults produce different Tenants:\ndecoded: %#v\nraw: %#v", tenant.Object, fromRaw.Object)
	}
}
