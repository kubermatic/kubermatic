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
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
)

// Tenant builds only the fields KKP applies to a management-cluster Tenant.
func Tenant(cluster *kubermaticv1.Cluster, defaultTenantSpec *runtime.RawExtension) (*unstructured.Unstructured, error) {
	tenant := &unstructured.Unstructured{}
	tenant.SetGroupVersionKind(KubelbTenantGVK)
	tenant.SetName(cluster.Name)
	tenant.SetLabels(map[string]string{
		TenantClusterNameLabelKey:         cluster.Name,
		TenantClusterExternalNameLabelKey: cluster.Status.Address.ExternalName,
		TenantProjectIDLabelKey:           cluster.Labels[kubermaticv1.ProjectIDLabelKey],
	})

	if defaultTenantSpec != nil {
		raw, err := json.Marshal(defaultTenantSpec)
		if err != nil {
			return nil, fmt.Errorf("failed to encode project default tenant spec: %w", err)
		}
		var spec map[string]any
		if err := json.Unmarshal(raw, &spec); err != nil {
			return nil, fmt.Errorf("failed to decode project default tenant spec: %w", err)
		}
		if len(spec) > 0 {
			tenant.Object["spec"] = spec
		}
	}

	return tenant, nil
}
