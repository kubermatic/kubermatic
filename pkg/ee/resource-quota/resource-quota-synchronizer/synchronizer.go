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

package resourcequotasynchronizer

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// This controller syncs the ResourceQuotas from the master cluster to the seed clusters.
	ControllerName = "kkp-resource-quota-synchronizer"

	// cleanupFinalizer indicates that synced resource quota on seed clusters needs cleanup.
	cleanupFinalizer = "kubermatic.k8c.io/cleanup-seed-resource-quota"
)

type reconciler struct {
	log          *zap.SugaredLogger
	masterClient ctrlruntimeclient.Client
	seedClients  kuberneteshelper.SeedClientMap
	recorder     events.EventRecorder
}

func Add(masterMgr manager.Manager,
	seedManagers map[string]manager.Manager,
	log *zap.SugaredLogger,
) error {
	log = log.Named(ControllerName)
	r := &reconciler{
		log:          log,
		masterClient: masterMgr.GetClient(),
		seedClients:  kuberneteshelper.SeedClientMap{},
		recorder:     masterMgr.GetEventRecorder(ControllerName),
	}

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()
	}

	_, err := builder.ControllerManagedBy(masterMgr).
		Named(ControllerName).
		For(&kubermaticv1.ResourceQuota{}).
		Build(r)

	return err
}

func resourceQuotaReconcilerFactory(rq *kubermaticv1.ResourceQuota) reconciling.NamedResourceQuotaReconcilerFactory {
	return func() (string, reconciling.ResourceQuotaReconciler) {
		return rq.Name, func(c *kubermaticv1.ResourceQuota) (*kubermaticv1.ResourceQuota, error) {
			c.Name = rq.Name
			c.Labels = rq.Labels
			c.Spec = rq.Spec
			return c, nil
		}
	}
}

// Reconcile reconciles the resource quotas in the master cluster and syncs them to all seeds.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("resource", request.Name)
	log.Debug("Processing")

	resourceQuota := &kubermaticv1.ResourceQuota{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, resourceQuota); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	err := r.reconcile(ctx, log, resourceQuota)

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, resourceQuota *kubermaticv1.ResourceQuota) error {
	// handling deletion
	if !resourceQuota.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, resourceQuota); err != nil {
			return fmt.Errorf("handling deletion of resourceQuota: %w", err)
		}
		return nil
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.masterClient, resourceQuota, cleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	resourceQuotaReconcilerFactories := []reconciling.NamedResourceQuotaReconcilerFactory{
		resourceQuotaReconcilerFactory(resourceQuota),
	}

	return r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
		// ensure resource quota
		if err := reconciling.ReconcileResourceQuotas(ctx, resourceQuotaReconcilerFactories, "", seedClient); err != nil {
			return err
		}

		// ensure status
		globalUsage := resourceQuota.Status.GlobalUsage.DeepCopy()
		return util.UpdateResourceQuotaStatus(ctx, seedClient, resourceQuota, func(rq *kubermaticv1.ResourceQuota) {
			rq.Status.GlobalUsage = *globalUsage
		})
	})
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, resourceQuota *kubermaticv1.ResourceQuota) error {
	// if finalizer not set to master ResourceQuota then return
	if !kuberneteshelper.HasFinalizer(resourceQuota, cleanupFinalizer) {
		return nil
	}

	err := r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
		err := seedClient.Delete(ctx, &kubermaticv1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceQuota.Name,
				Namespace: resourceQuota.Namespace,
			},
		})

		return ctrlruntimeclient.IgnoreNotFound(err)
	})
	if err != nil {
		return err
	}

	return kuberneteshelper.TryRemoveFinalizer(ctx, r.masterClient, resourceQuota, cleanupFinalizer)
}
