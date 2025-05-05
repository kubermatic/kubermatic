//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2025 Kubermatic GmbH

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

package kyverno

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	userclusterresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/user-cluster"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// handleKyvernoCleanup removes all Kyverno resources from the user cluster.
func (r *reconciler) handleKyvernoCleanup(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if err := r.ensureKyvernoUserClusterResourcesAreRemoved(ctx, cluster); err != nil {
		return err
	}

	return kuberneteshelper.TryRemoveFinalizer(ctx, r, cluster, CleanupFinalizer)
}

func (r *reconciler) ensureKyvernoUserClusterResourcesAreRemoved(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	userClusterClient, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get user cluster client: %w", err)
	}

	for _, resource := range userclusterresources.ResourcesForDeletion() {
		err := userClusterClient.Delete(ctx, resource)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete resource %s: %w", resource.GetName(), err)
		}
	}

	return nil
}
