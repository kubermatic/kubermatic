//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

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

package prometheus

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling/modifier"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Name = "metering-prometheus"
)

// ReconcilePrometheus reconciles the prometheus instance used as a datasource for metering.
func ReconcilePrometheus(ctx context.Context, client ctrlruntimeclient.Client, scheme *runtime.Scheme, getRegistry registry.ImageRewriter, seed *kubermaticv1.Seed) error {
	seedOwner := modifier.Ownership(seed, "", scheme)
	revisionLabelsModifier := modifier.RelatedRevisionsLabels(ctx, client)

	if err := reconciling.ReconcileServiceAccounts(ctx, []reconciling.NamedServiceAccountReconcilerFactory{
		prometheusServiceAccount(),
	}, seed.Namespace, client, seedOwner); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAccount: %w", err)
	}

	if err := reconciling.ReconcileClusterRoles(ctx, []reconciling.NamedClusterRoleReconcilerFactory{
		prometheusClusterRole(),
	}, "", client); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRole: %w", err)
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, []reconciling.NamedClusterRoleBindingReconcilerFactory{
		prometheusClusterRoleBinding(seed.Namespace),
	}, "", client); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBindings: %w", err)
	}

	if err := reconciling.ReconcileConfigMaps(ctx, []reconciling.NamedConfigMapReconcilerFactory{
		prometheusConfigMap(),
	}, seed.Namespace, client, seedOwner); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMap: %w", err)
	}

	if err := reconciling.ReconcileStatefulSets(ctx, []reconciling.NamedStatefulSetReconcilerFactory{
		PrometheusStatefulSet(getRegistry, seed),
	}, seed.Namespace, client, seedOwner, revisionLabelsModifier); err != nil {
		return fmt.Errorf("failed to reconcile StatefuleSet: %w", err)
	}

	if err := reconciling.ReconcileServices(ctx, []reconciling.NamedServiceReconcilerFactory{
		prometheusService(),
	}, seed.Namespace, client); err != nil {
		return fmt.Errorf("failed to reconcile Service: %w", err)
	}

	return nil
}
