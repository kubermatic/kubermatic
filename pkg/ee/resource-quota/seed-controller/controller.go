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

package seedcontroller

import (
	"context"
	"fmt"
	"reflect"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticpred "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kubermaticresources "k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var ControllerName = "kkp-resource-quota-seed-controller"

type reconciler struct {
	log                     *zap.SugaredLogger
	workerNameLabelSelector labels.Selector
	recorder                record.EventRecorder
	seedClient              ctrlruntimeclient.Client
}

func Add(mgr manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	numWorkers int,
) error {
	workerSelector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return fmt.Errorf("failed to build worker-name selector: %w", err)
	}

	reconciler := &reconciler{
		log:                     log.Named(ControllerName),
		workerNameLabelSelector: workerSelector,
		recorder:                mgr.GetEventRecorderFor(ControllerName),
		seedClient:              mgr.GetClient(),
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.Cluster{}},
		enqueueResourceQuota(reconciler.seedClient, reconciler.log),
		workerlabel.Predicates(workerName),
		withClusterEventFilter(),
	); err != nil {
		return fmt.Errorf("failed to create watch for clusters: %w", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.ResourceQuota{}},
		&handler.EnqueueRequestForObject{},
		kubermaticpred.ByNamespace(kubermaticresources.KubermaticNamespace),
	); err != nil {
		return fmt.Errorf("failed to create watch for seed resource quotas: %w", err)
	}

	return nil
}

// Reconcile calculates the resource usage for a resource quota and sets the local usage.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Reconciling")

	resourceQuota := &kubermaticv1.ResourceQuota{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, resourceQuota); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get resource quota %q: %w", resourceQuota.Name, err)
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

	projectIdReq, err := labels.NewRequirement(kubermaticv1.ProjectIDLabelKey, selection.Equals, []string{resourceQuota.Spec.Subject.Name})
	if err != nil {
		return fmt.Errorf("error creating project id req: %w", err)
	}

	clusterList := &kubermaticv1.ClusterList{}
	if err := r.seedClient.List(ctx, clusterList,
		&ctrlruntimeclient.ListOptions{LabelSelector: r.workerNameLabelSelector.Add(*projectIdReq)}); err != nil {
		return fmt.Errorf("failed listing clusters: %w", err)
	}

	localUsage := kubermaticv1.NewResourceDetails(resource.Quantity{}, resource.Quantity{}, resource.Quantity{})
	for _, cluster := range clusterList.Items {
		if cluster.Status.ResourceUsage != nil {
			clusterUsage := cluster.Status.ResourceUsage
			if clusterUsage.CPU != nil {
				localUsage.CPU.Add(*clusterUsage.CPU)
			}
			if clusterUsage.Memory != nil {
				localUsage.Memory.Add(*clusterUsage.Memory)
			}
			if clusterUsage.Storage != nil {
				localUsage.Storage.Add(*clusterUsage.Storage)
			}
		}
	}

	if err = r.ensureLocalUsage(ctx, log, resourceQuota, localUsage); err != nil {
		return err
	}

	return nil
}

func (r *reconciler) ensureLocalUsage(ctx context.Context, log *zap.SugaredLogger, resourceQuota *kubermaticv1.ResourceQuota,
	localUsage *kubermaticv1.ResourceDetails) error {
	if reflect.DeepEqual(localUsage, resourceQuota.Status.LocalUsage) {
		log.Debugw("local usage is for resource quota is the same, not updating",
			"cpu", localUsage.CPU.String(),
			"memory", localUsage.Memory.String(),
			"storage", localUsage.Storage.String())
		return nil
	}
	log.Debugw("local usage for resource quota needs update",
		"cpu", localUsage.CPU.String(),
		"memory", localUsage.Memory.String(),
		"storage", localUsage.Storage.String())

	oldResourceQuota := resourceQuota.DeepCopy()

	resourceQuota.Status.LocalUsage = *localUsage
	if err := r.seedClient.Patch(ctx, resourceQuota, ctrlruntimeclient.MergeFrom(oldResourceQuota)); err != nil {
		return fmt.Errorf("failed to patch resource quota %q: %w", resourceQuota.Name, err)
	}

	return nil
}

func withClusterEventFilter() predicate.Predicate {
	return predicate.Funcs{
		// when cluster is created, no point to calculate yet as the machines are not created
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldCluster, ok := e.ObjectOld.(*kubermaticv1.Cluster)
			if !ok {
				return false
			}
			newCluster, ok := e.ObjectNew.(*kubermaticv1.Cluster)
			if !ok {
				return false
			}
			return !reflect.DeepEqual(oldCluster.Status.ResourceUsage, newCluster.Status.ResourceUsage)
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func enqueueResourceQuota(client ctrlruntimeclient.Client, log *zap.SugaredLogger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request

		clusterLabels := a.GetLabels()
		projectId, ok := clusterLabels[kubermaticv1.ProjectIDLabelKey]
		if !ok {
			log.Debugw("cluster does not have `project-id` label, skipping", "cluster", a.GetName())
		}
		subjectNameReq, err := labels.NewRequirement(kubermaticv1.ResourceQuotaSubjectNameLabelKey, selection.Equals, []string{projectId})
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("error creating subject name req: %w", err))
		}

		resourceQuotaList := &kubermaticv1.ResourceQuotaList{}

		if err := client.List(context.Background(), resourceQuotaList,
			&ctrlruntimeclient.ListOptions{Namespace: kubermaticresources.KubermaticNamespace,
				LabelSelector: labels.NewSelector().Add(*subjectNameReq)}); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list resourceQuotas: %w", err))
		}

		for _, rq := range resourceQuotaList.Items {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      rq.Name,
				Namespace: rq.Namespace,
			}})
		}
		return requests
	})
}
