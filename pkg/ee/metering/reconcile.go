// +build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2021 Loodse GmbH

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

package metering

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileMeteringResources reconciles the metering related resources.
func ReconcileMeteringResources(ctx context.Context, client ctrlruntimeclient.Client, namespace string,
	cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, log *zap.SugaredLogger) error {

	if err := persistentVolumeClaimCreator(ctx, client, seed); err != nil {
		return fmt.Errorf("failed to reconcile metering pvc: %v", err)
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, []reconciling.NamedServiceAccountCreatorGetter{
		serviceAccountCreator(),
	}, metav1.NamespaceSystem, client); err != nil {
		return fmt.Errorf("failed to reconcile metering ServiceAccounts: %v", err)
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, []reconciling.NamedClusterRoleBindingCreatorGetter{
		clusterRoleBindingCreator(cfg.Namespace),
	}, "", client); err != nil {
		return fmt.Errorf("failed to reconcile metering ClusterRoleBindings: %v", err)
	}

	if err := reconciling.ReconcileCronJobs(ctx, []reconciling.NamedCronJobCreatorGetter{
		cronJobCreator(seed.Name),
	}, namespace, client); err != nil {
		return fmt.Errorf("failed to reconcile cronjpbs: %v", err)
	}

	if err := reconciling.ReconcileDeployments(ctx, []reconciling.NamedDeploymentCreatorGetter{
		deploymentCreator(seed),
	}, metav1.NamespaceSystem, client); err != nil {
		return fmt.Errorf("failed to reconcile VPA Deployments: %v", err)
	}

	return nil
}
