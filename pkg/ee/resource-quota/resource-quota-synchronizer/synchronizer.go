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

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	kubermaticresources "k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// This controller syncs the ResourceQuotas from the master cluster to the seed clusters.
	ControllerName = "kkp-resource-quota-synchronizer"
)

type reconciler struct {
	log          *zap.SugaredLogger
	masterClient ctrlruntimeclient.Client
	seedClients  map[string]ctrlruntimeclient.Client
	recorder     record.EventRecorder
}

func Add(masterMgr manager.Manager,
	seedManagers map[string]manager.Manager,
	log *zap.SugaredLogger,
) error {
	log = log.Named(ControllerName)
	r := &reconciler{
		log:          log,
		masterClient: masterMgr.GetClient(),
		seedClients:  map[string]ctrlruntimeclient.Client{},
		recorder:     masterMgr.GetEventRecorderFor(ControllerName),
	}

	c, err := controller.New(ControllerName, masterMgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()
	}

	// Watch for changes to ResourceQuota
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.ResourceQuota{}}, &handler.EnqueueRequestForObject{},
		predicate.ByNamespace(kubermaticresources.KubermaticNamespace)); err != nil {
		return fmt.Errorf("failed to watch resource quotas: %w", err)
	}

	return nil
}

func resourceQuotaCreatorGetter(rq *kubermaticv1.ResourceQuota) reconciling.NamedKubermaticV1ResourceQuotaCreatorGetter {
	return func() (string, reconciling.KubermaticV1ResourceQuotaCreator) {
		return rq.Name, func(c *kubermaticv1.ResourceQuota) (*kubermaticv1.ResourceQuota, error) {
			c.Name = rq.Name
			c.Spec = rq.Spec
			c.Status.GlobalUsage = rq.Status.GlobalUsage
			return c, nil
		}
	}
}

// Reconcile reconciles the resource quotas in the master cluster and syncs them to all seeds.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("resource", request.Name)
	log.Debug("Processing")

	err := r.reconcile(ctx, log, request)
	if err != nil {
		log.Errorw("ReconcilingError", zap.Error(err))
	}

	return reconcile.Result{}, err
}

func (r *reconciler) syncAllSeeds(log *zap.SugaredLogger, resourceQuota *kubermaticv1.ResourceQuota, action func(seedClient ctrlruntimeclient.Client, resourceQuota *kubermaticv1.ResourceQuota) error) error {
	for seedName, seedClient := range r.seedClients {
		log := log.With("seed", seedName)

		log.Debug("Reconciling resourceQuota with seed")

		err := action(seedClient, resourceQuota)
		if err != nil {
			return fmt.Errorf("failed syncing resourceQuota %s for seed %s: %w", resourceQuota.Name, seedName, err)
		}
		log.Debug("Reconciled resourceQuota with seed")
	}
	return nil
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, request reconcile.Request) error {
	resourceQuota := &kubermaticv1.ResourceQuota{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, resourceQuota); err != nil {
		return ctrlruntimeclient.IgnoreNotFound(err)
	}

	// handling deletion
	if !resourceQuota.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, resourceQuota); err != nil {
			return fmt.Errorf("handling deletion of resourceQuota: %w", err)
		}
		return nil
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.masterClient, resourceQuota, apiv1.ResourceQuotaSeedCleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	resourceQuotaCreatorGetters := []reconciling.NamedKubermaticV1ResourceQuotaCreatorGetter{
		resourceQuotaCreatorGetter(resourceQuota),
	}

	return r.syncAllSeeds(log, resourceQuota, func(seedClient ctrlruntimeclient.Client, resourceQuota *kubermaticv1.ResourceQuota) error {
		return reconciling.ReconcileKubermaticV1ResourceQuotas(ctx, resourceQuotaCreatorGetters, kubermaticresources.KubermaticNamespace, seedClient)
	})
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, resourceQuota *kubermaticv1.ResourceQuota) error {
	// if finalizer not set to master ResourceQuota then return
	if !kuberneteshelper.HasFinalizer(resourceQuota, apiv1.ResourceQuotaSeedCleanupFinalizer) {
		return nil
	}

	err := r.syncAllSeeds(log, resourceQuota, func(seedClient ctrlruntimeclient.Client, resourceQuota *kubermaticv1.ResourceQuota) error {
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

	return kuberneteshelper.TryRemoveFinalizer(ctx, r.masterClient, resourceQuota, apiv1.ResourceQuotaSeedCleanupFinalizer)
}
