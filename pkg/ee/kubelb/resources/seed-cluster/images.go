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

	"k8c.io/kubermatic/v2/pkg/resources/registry"
)

// TenantProxyImages contains images for the proxy that CCM deploys in the user cluster.
// The xDS writer uses the configured CCM image itself.
type TenantProxyImages struct {
	Envoy           string
	ShutdownManager string
}

// GetTenantProxyImages returns the KubeLB v1.5.0 proxy images with registry overrides
// applied. Deployment arguments and the mirror inventory must use the same images.
func GetTenantProxyImages(rewrite registry.ImageRewriter) (TenantProxyImages, error) {
	// Keep these defaults aligned with kubelb-ee/internal/images/images.go at the CCM release tag.
	envoy, err := rewrite("docker.io/envoyproxy/envoy:distroless-v1.36.4")
	if err != nil {
		return TenantProxyImages{}, fmt.Errorf("failed to rewrite KubeLB tenant proxy Envoy image: %w", err)
	}
	shutdownManager, err := rewrite("docker.io/envoyproxy/gateway:v1.8.3")
	if err != nil {
		return TenantProxyImages{}, fmt.Errorf("failed to rewrite KubeLB tenant proxy shutdown manager image: %w", err)
	}

	return TenantProxyImages{Envoy: envoy, ShutdownManager: shutdownManager}, nil
}
