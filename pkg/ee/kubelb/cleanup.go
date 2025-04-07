//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2021 Kubermatic GmbH

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

package kubelbcontroller

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubelbmanagementresources "k8c.io/kubermatic/v2/pkg/ee/kubelb/resources/kubelb-cluster"
	kubelbseedresources "k8c.io/kubermatic/v2/pkg/ee/kubelb/resources/seed-cluster"
	kubelbuserclusterresources "k8c.io/kubermatic/v2/pkg/ee/kubelb/resources/user-cluster"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func (r *reconciler) handleKubeLBCleanup(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if err := r.ensureKubeLBSeedClusterResourcesAreRemoved(ctx, cluster.Status.NamespaceName); err != nil {
		return err
	}

	if err := r.ensureKubeLBUserClusterResourcesAreRemoved(ctx, cluster); err != nil {
		return err
	}

	if err := r.ensureKubeLBManagementClusterResourcesAreRemoved(ctx, cluster); err != nil {
		return err
	}

	return kuberneteshelper.TryRemoveFinalizer(ctx, r, cluster, CleanupFinalizer)
}

func (r *reconciler) ensureKubeLBSeedClusterResourcesAreRemoved(ctx context.Context, namespace string) error {
	for _, resource := range kubelbseedresources.ResourcesForDeletion(namespace) {
		err := r.Delete(ctx, resource)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure kubeLB resources are removed/not present on seed cluster: %w", err)
		}
	}
	return nil
}

func (r *reconciler) ensureKubeLBManagementClusterResourcesAreRemoved(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	// Ensure that kubeLB management cluster kubeconfig exists. If it doesn't, we can't continue.
	seed, err := r.seedGetter()
	if err != nil {
		return err
	}

	datacenter, found := seed.Spec.Datacenters[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return fmt.Errorf("couldn't find datacenter %q for cluster %q", cluster.Spec.Cloud.DatacenterName, cluster.Name)
	}

	// Get kubeLB management cluster client.
	kubeLBManagementClient, err := r.getKubeLBManagementClusterClient(ctx, seed, datacenter)
	if err != nil {
		return err
	}

	for _, resource := range kubelbmanagementresources.ResourcesForDeletion(cluster.Name) {
		err := kubeLBManagementClient.Delete(ctx, resource)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure kubeLB resources are removed/not present on kubelb management cluster: %w", err)
		}
	}
	return nil
}

func (r *reconciler) ensureKubeLBUserClusterResourcesAreRemoved(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	userClusterClient, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get user cluster client: %w", err)
	}

	for _, resource := range kubelbuserclusterresources.ResourcesForDeletion() {
		err := userClusterClient.Delete(ctx, resource)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure kubeLB resources are removed/not present on user cluster: %w", err)
		}
	}
	return nil
}
