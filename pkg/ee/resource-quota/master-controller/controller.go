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

package mastercontroller

import (
	"context"
	"fmt"
	"reflect"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const ControllerName = "kkp-master-resource-quota-controller"

type reconciler struct {
	masterClient ctrlruntimeclient.Client
	log          *zap.SugaredLogger
	recorder     record.EventRecorder
	seedClients  map[string]ctrlruntimeclient.Client
}

func Add(mgr manager.Manager,
	seedManagers map[string]manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
) error {
	reconciler := &reconciler{
		log:          log.Named(ControllerName),
		recorder:     mgr.GetEventRecorderFor(ControllerName),
		masterClient: mgr.GetClient(),
		seedClients:  map[string]ctrlruntimeclient.Client{},
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	for seedName, seedManager := range seedManagers {
		reconciler.seedClients[seedName] = seedManager.GetClient()

		resourceQuotaSource := &source.Kind{Type: &kubermaticv1.ResourceQuota{}}
		if err := resourceQuotaSource.InjectCache(mgr.GetCache()); err != nil {
			return fmt.Errorf("failed to inject cache into resourceQuotaSource for seed %s: %w", seedName, err)
		}
		if err := c.Watch(resourceQuotaSource, &handler.EnqueueRequestForObject{}, predicate.ByNamespace(resources.KubermaticNamespace)); err != nil {
			return fmt.Errorf("failed to establish watch for clusters in seed %s: %w", seedName, err)
		}
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.ResourceQuota{}}, &handler.EnqueueRequestForObject{}, predicate.ByNamespace(resources.KubermaticNamespace)); err != nil {
		return fmt.Errorf("failed to create watch for resource quota: %w", err)
	}

	return nil
}

// Reconcile calculates the resource usage for a resource quota and sets the local usage.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Reconciling")

	resourceQuota := &kubermaticv1.ResourceQuota{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, resourceQuota); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get resourceQuota %s: %w", resourceQuota.Name, err)
	}

	err := r.reconcile(ctx, resourceQuota, log)
	if err != nil {
		log.Errorw("ReconcilingError", zap.Error(err))
		r.recorder.Eventf(resourceQuota, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, resourceQuota *kubermaticv1.ResourceQuota, log *zap.SugaredLogger) error {
	// skip reconcile if resourceQuota is in delete state
	if !resourceQuota.DeletionTimestamp.IsZero() {
		log.Debug("resource quota is in deletion, skipping")
		return nil
	}

	// for all related resource quotas on seeds, calculate global usage
	globalUsage := kubermaticv1.NewResourceDetails(resource.Quantity{}, resource.Quantity{}, resource.Quantity{})
	for seed, seedClient := range r.seedClients {
		seedResourceQuota := &kubermaticv1.ResourceQuota{}
		err := seedClient.Get(ctx, types.NamespacedName{Namespace: resourceQuota.Namespace, Name: resourceQuota.Name},
			seedResourceQuota)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("error getting seed %q resource quota: %w", seed, err)
		}
		globalUsage.CPU.Add(*seedResourceQuota.Status.LocalUsage.CPU)
		globalUsage.Memory.Add(*seedResourceQuota.Status.LocalUsage.Memory)
		globalUsage.Storage.Add(*seedResourceQuota.Status.LocalUsage.Storage)
	}

	if err := r.ensureGlobalUsage(ctx, log, resourceQuota, globalUsage); err != nil {
		return err
	}

	return nil
}

func (r *reconciler) ensureGlobalUsage(ctx context.Context, log *zap.SugaredLogger, resourceQuota *kubermaticv1.ResourceQuota,
	globalUsage *kubermaticv1.ResourceDetails) error {
	if reflect.DeepEqual(*globalUsage, resourceQuota.Status.LocalUsage) {
		log.Debugw("global usage is for resource quota is the same, not updating",
			"cpu", globalUsage.CPU.String(),
			"memory", globalUsage.Memory.String(),
			"storage", globalUsage.Storage.String())
		return nil
	}
	log.Debugw("global usage for resource quota needs update",
		"cpu", globalUsage.CPU.String(),
		"memory", globalUsage.Memory.String(),
		"storage", globalUsage.Storage.String())

	oldResourceQuota := resourceQuota.DeepCopy()

	resourceQuota.Status.GlobalUsage = *globalUsage
	if err := r.masterClient.Patch(ctx, resourceQuota, ctrlruntimeclient.MergeFrom(oldResourceQuota)); err != nil {
		return fmt.Errorf("failed to patch resource quota %q: %w", resourceQuota.Name, err)
	}

	return nil
}
