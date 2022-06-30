//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2020 Kubermatic GmbH

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

package synccontroller

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "group-project-binding-sync-controller"
)

type reconciler struct {
	log              *zap.SugaredLogger
	recorder         record.EventRecorder
	masterClient     ctrlruntimeclient.Client
	seedsGetter      provider.SeedsGetter
	seedClientGetter provider.SeedClientGetter
}

func Add(
	masterManager manager.Manager,
	seedsGetter provider.SeedsGetter,
	seedKubeconfigGetter provider.SeedKubeconfigGetter,
	log *zap.SugaredLogger,
	numWorkers int,
) error {
	r := &reconciler{
		log:              log.Named(ControllerName),
		recorder:         masterManager.GetEventRecorderFor(ControllerName),
		masterClient:     masterManager.GetClient(),
		seedsGetter:      seedsGetter,
		seedClientGetter: provider.SeedClientGetterFactory(seedKubeconfigGetter),
	}

	c, err := controller.New(ControllerName, masterManager, controller.Options{Reconciler: r, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.GroupProjectBinding{}},
		&handler.EnqueueRequestForObject{},
	); err != nil {
		return fmt.Errorf("failed to create watch for groupprojectbindings: %w", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.Seed{}},
		enqueueGroupProjectBindingsForSeed(r.masterClient, r.log),
	); err != nil {
		return fmt.Errorf("failed to create watch for seeds: %w", err)
	}

	return nil
}

// Reconcile reconciles Kubermatic GroupProjectBinding objects on the master cluster to all seed clusters.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)

	groupProjectBinding := &kubermaticv1.GroupProjectBinding{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, groupProjectBinding); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if !groupProjectBinding.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, groupProjectBinding); err != nil {
			return reconcile.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return reconcile.Result{}, nil
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.masterClient, groupProjectBinding, apiv1.SeedGroupProjectBindingCleanupFinalizer); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
	}

	groupProjectBindingCreatorGetters := []reconciling.NamedKubermaticV1GroupProjectBindingCreatorGetter{
		groupProjectBindingCreatorGetter(groupProjectBinding),
	}

	err := r.syncAllSeeds(log, groupProjectBinding, func(seedClusterClient ctrlruntimeclient.Client, groupProjectBinding *kubermaticv1.GroupProjectBinding) error {
		return reconciling.ReconcileKubermaticV1GroupProjectBindings(ctx, groupProjectBindingCreatorGetters, "", seedClusterClient)
	})

	if err != nil {
		r.recorder.Eventf(groupProjectBinding, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		return reconcile.Result{}, fmt.Errorf("failed to reconcile groupprojectbinding '%s': %w", groupProjectBinding.Name, err)
	}

	return reconcile.Result{}, nil
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, groupProjectBinding *kubermaticv1.GroupProjectBinding) error {
	err := r.syncAllSeeds(log, groupProjectBinding,
		func(seedClusterClient ctrlruntimeclient.Client, groupProjectBinding *kubermaticv1.GroupProjectBinding) error {
			if err := seedClusterClient.Delete(ctx, groupProjectBinding); err != nil {
				return ctrlruntimeclient.IgnoreNotFound(err)
			}

			return nil
		},
	)

	if err != nil {
		return err
	}

	return kuberneteshelper.TryRemoveFinalizer(ctx, r.masterClient, groupProjectBinding, apiv1.SeedGroupProjectBindingCleanupFinalizer)
}

type actionFunc func(seedClusterClient ctrlruntimeclient.Client, groupProjectBinding *kubermaticv1.GroupProjectBinding) error

func (r *reconciler) syncAllSeeds(log *zap.SugaredLogger, groupProjectBinding *kubermaticv1.GroupProjectBinding, action actionFunc) error {
	seedErrs := []error{}

	seeds, err := r.seedsGetter()
	if err != nil {
		return fmt.Errorf("failed to get seeds: %w", err)
	}

	for _, seed := range seeds {
		seedClient, err := r.seedClientGetter(seed)

		if err != nil {
			log.Errorf("failed to get client for seed '%s': %w", seed.Name, err)
			seedErrs = append(seedErrs, err)
			continue
		}

		if err := action(seedClient, groupProjectBinding); err != nil {
			log.Errorf("failed to sync GroupProjectBinding for seed '%s': %w", seed.Name, err)
			seedErrs = append(seedErrs, err)
			continue
		}

		log.Debugw("reconciled groupprojectbinding with seed", "seed", seed.Name)
	}

	if len(seedErrs) > 0 {
		slice := []string{}
		for _, err := range seedErrs {
			slice = append(slice, err.Error())
		}

		return fmt.Errorf("failed to sync to at least one seed: %s", strings.Join(slice, ","))
	}

	return nil
}

func enqueueGroupProjectBindingsForSeed(client ctrlruntimeclient.Client, log *zap.SugaredLogger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request

		groupProjectBindingList := &kubermaticv1.GroupProjectBindingList{}

		if err := client.List(context.Background(), groupProjectBindingList); err != nil {
			log.Error(err)
			utilruntime.HandleError(fmt.Errorf("failed to list userprojectbindings: %w", err))
		}

		for _, groupProjectBinding := range groupProjectBindingList.Items {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
				Name: groupProjectBinding.Name,
			}})
		}

		return requests
	})
}
