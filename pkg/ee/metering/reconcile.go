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

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileMeteringResources reconciles the metering related resources.
func ReconcileMeteringResources(ctx context.Context, client ctrlruntimeclient.Client, seed *kubermaticv1.Seed) error {

	if err := persistentVolumeClaimCreator(ctx, client, seed); err != nil {
		return fmt.Errorf("failed to reconcile metering PVC: %v", err)
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, []reconciling.NamedServiceAccountCreatorGetter{
		serviceAccountCreator(),
	}, resources.KubermaticNamespace, client); err != nil {
		return fmt.Errorf("failed to reconcile metering ServiceAccounts: %v", err)
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, []reconciling.NamedClusterRoleBindingCreatorGetter{
		clusterRoleBindingCreator(resources.KubermaticNamespace),
	}, "", client); err != nil {
		return fmt.Errorf("failed to reconcile metering ClusterRoleBindings: %v", err)
	}

	modifiers := []reconciling.ObjectModifier{
		common.VolumeRevisionLabelsModifierFactory(ctx, client),
	}
	if err := reconciling.ReconcileCronJobs(ctx, []reconciling.NamedCronJobCreatorGetter{
		cronJobCreator(seed.Name),
	}, resources.KubermaticNamespace, client, modifiers...); err != nil {
		return fmt.Errorf("failed to reconcile metering CronJob: %v", err)
	}

	if err := reconciling.ReconcileDeployments(ctx, []reconciling.NamedDeploymentCreatorGetter{
		deploymentCreator(seed),
	}, resources.KubermaticNamespace, client, modifiers...); err != nil {
		return fmt.Errorf("failed to reconcile metering Deployment: %v", err)
	}

	return nil
}
