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
	"sort"
	"time"

	"go.uber.org/zap"

	k8cequality "k8c.io/kubermatic/sdk/v2/apis/equality"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/machine/accelerator"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var ControllerName = "kkp-resource-quota-seed-controller"

type reconciler struct {
	log               *zap.SugaredLogger
	workerName        string
	recorder          events.EventRecorder
	seedClient        ctrlruntimeclient.Client
	controllerVersion string
	now               func() time.Time
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	numWorkers int,
	controllerVersion string,
) error {
	reconciler := &reconciler{
		log:               log.Named(ControllerName),
		workerName:        workerName,
		recorder:          mgr.GetEventRecorder(ControllerName),
		seedClient:        mgr.GetClient(),
		controllerVersion: controllerVersion,
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.ResourceQuota{}).
		Watches(&kubermaticv1.Cluster{}, enqueueResourceQuota(reconciler.seedClient, reconciler.log, workerName), builder.WithPredicates(workerlabel.Predicate(workerName), withClusterEventFilter())).
		Build(reconciler)

	return err
}

// Reconcile calculates Seed-local resource usage and accelerator accounting readiness.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Reconciling")

	resourceQuota := &kubermaticv1.ResourceQuota{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, resourceQuota); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("resource quota not found, might be deleted: %w", err)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get resource quota: %w", err)
	}

	requeueAfter, err := r.reconcile(ctx, resourceQuota, log)
	if err != nil {
		r.recorder.Eventf(resourceQuota, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
	}

	return reconcile.Result{RequeueAfter: requeueAfter}, err
}

func (r *reconciler) reconcile(ctx context.Context, resourceQuota *kubermaticv1.ResourceQuota, log *zap.SugaredLogger) (time.Duration, error) {
	// If the controller is in worker-name mode, ignore all non-Cluster-RQ's
	// (i.e. all RQ's that span multiple clusters), as it makes no sense to
	// update an RQ's status with data that spans only a subset of subjects.
	// As of now, only project RQ's exist and so there is no single-cluster-RQ.
	if r.workerName != "" /* resourceQuota.Spec.Subject.Kind != "cluster" */ {
		log.Debug("Ignoring request because worker-name is set.")
		return 0, nil
	}

	// skip reconcile if resourceQuota is in delete state
	if !resourceQuota.DeletionTimestamp.IsZero() {
		log.Debug("resource quota is in deletion, skipping")
		return 0, nil
	}

	projectIDReq, err := labels.NewRequirement(kubermaticv1.ProjectIDLabelKey, selection.Equals, []string{resourceQuota.Spec.Subject.Name})
	if err != nil {
		return 0, fmt.Errorf("error creating project id req: %w", err)
	}

	clusterList := &kubermaticv1.ClusterList{}
	if err := r.seedClient.List(ctx, clusterList,
		&ctrlruntimeclient.ListOptions{LabelSelector: labels.NewSelector().Add(*projectIDReq)}); err != nil {
		return 0, fmt.Errorf("failed listing clusters: %w", err)
	}

	localUsage := kubermaticv1.NewResourceDetails(resource.Quantity{}, resource.Quantity{}, resource.Quantity{})
	acceleratorAccountingActive := resourceQuota.Annotations[resources.AcceleratorAccountingEnabledAnnotation] == resources.AcceleratorAccountingEnabledAnnotationValue
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
			if acceleratorAccountingActive && cluster.Spec.Cloud.Kubevirt != nil {
				kubermaticv1.AddAcceleratorUsage(localUsage, clusterUsage.Accelerators)
			}
		}
	}

	localAccounting, requeueAfter := r.localAcceleratorAccounting(resourceQuota, clusterList.Items)

	if err = r.ensureLocalStatus(ctx, log, resourceQuota, localUsage, localAccounting); err != nil {
		return 0, err
	}

	return requeueAfter, nil
}

func (r *reconciler) ensureLocalStatus(ctx context.Context, log *zap.SugaredLogger, resourceQuota *kubermaticv1.ResourceQuota,
	localUsage *kubermaticv1.ResourceDetails, localAccounting *kubermaticv1.ResourceQuotaLocalAcceleratorAccountingStatus) error {
	if k8cequality.Semantic.DeepEqual(localUsage, resourceQuota.Status.LocalUsage) &&
		k8cequality.Semantic.DeepEqual(localAccounting, resourceQuota.Status.LocalAcceleratorAccounting) {
		log.Debugw("local usage for resource quota is the same, not updating",
			"cpu", localUsage.CPU.String(),
			"memory", localUsage.Memory.String(),
			"storage", localUsage.Storage.String())
		return nil
	}
	log.Debugw("local usage for resource quota needs update",
		"cpu", localUsage.CPU.String(),
		"memory", localUsage.Memory.String(),
		"storage", localUsage.Storage.String())

	return util.UpdateResourceQuotaStatus(ctx, r.seedClient, resourceQuota, func(rq *kubermaticv1.ResourceQuota) {
		rq.Status.LocalUsage = *localUsage
		rq.Status.LocalAcceleratorAccounting = localAccounting
	})
}

func (r *reconciler) localAcceleratorAccounting(resourceQuota *kubermaticv1.ResourceQuota, clusters []kubermaticv1.Cluster) (*kubermaticv1.ResourceQuotaLocalAcceleratorAccountingStatus, time.Duration) {
	if resourceQuota.Annotations[resources.AcceleratorAccountingEnabledAnnotation] != resources.AcceleratorAccountingEnabledAnnotationValue {
		return nil, 0
	}

	now := r.currentTime()
	status := &kubermaticv1.ResourceQuotaLocalAcceleratorAccountingStatus{}
	if global := resourceQuota.Status.GlobalAcceleratorAccounting; global != nil {
		status.ObservedAccountingRevision = global.ObservedAccountingRevision
		status.ObservedQuotaDigest = global.ObservedQuotaDigest
	}

	relevantClusters := make([]kubermaticv1.Cluster, 0, len(clusters))
	for i := range clusters {
		if clusters[i].Spec.Cloud.Kubevirt != nil {
			relevantClusters = append(relevantClusters, clusters[i])
		}
	}
	sort.Slice(relevantClusters, func(i, j int) bool {
		return relevantClusters[i].Name < relevantClusters[j].Name
	})

	if resourceQuota.Status.GlobalAcceleratorAccounting == nil {
		status.Blockers = append(status.Blockers, kubermaticv1.AcceleratorAccountingBlocker{
			Type:    kubermaticv1.AcceleratorAccountingBlockerTypeMissingReport,
			Message: "the master accelerator accounting revision has not reached this Seed",
		})
		return status, resources.AcceleratorAccountingHeartbeatInterval
	}

	if len(relevantClusters) == 0 {
		if previous := resourceQuota.Status.LocalAcceleratorAccounting; previous != nil &&
			previous.ObservedAccountingRevision == status.ObservedAccountingRevision &&
			previous.ObservedQuotaDigest == status.ObservedQuotaDigest &&
			!previous.ObservedAt.IsZero() {
			refreshAt := previous.ObservedAt.Add(resources.AcceleratorAccountingHeartbeatInterval)
			if refreshAt.After(now) {
				status.ObservedAt = previous.ObservedAt
				status.Ready = true
				return status, refreshAt.Sub(now)
			}
		}
		status.ObservedAt = metav1.NewTime(now)
		status.Ready = true
		return status, resources.AcceleratorAccountingHeartbeatInterval
	}

	var oldestObservedAt time.Time
	var nextExpiry time.Time
	for i := range relevantClusters {
		cluster := &relevantClusters[i]
		observation := r.observeClusterAcceleratorAccounting(status, cluster, now)
		status.LegacyMachinesWithoutFootprint += observation.legacyMachines
		status.MachinesWithInvalidFootprint += observation.invalidMachines
		status.Blockers = append(status.Blockers, observation.blockers...)
		if !observation.observedAt.IsZero() && (oldestObservedAt.IsZero() || observation.observedAt.Before(oldestObservedAt)) {
			oldestObservedAt = observation.observedAt
		}
		if !observation.expiresAt.IsZero() && (nextExpiry.IsZero() || observation.expiresAt.Before(nextExpiry)) {
			nextExpiry = observation.expiresAt
		}
	}

	if !oldestObservedAt.IsZero() {
		status.ObservedAt = metav1.NewTime(oldestObservedAt)
	}
	status.Ready = len(status.Blockers) == 0
	return status, accountingRequeueAfter(now, nextExpiry)
}

type clusterAcceleratorAccountingObservation struct {
	legacyMachines  int32
	invalidMachines int32
	observedAt      time.Time
	expiresAt       time.Time
	blockers        []kubermaticv1.AcceleratorAccountingBlocker
}

func (r *reconciler) observeClusterAcceleratorAccounting(
	expected *kubermaticv1.ResourceQuotaLocalAcceleratorAccountingStatus,
	cluster *kubermaticv1.Cluster,
	now time.Time,
) clusterAcceleratorAccountingObservation {
	result := clusterAcceleratorAccountingObservation{}
	clusterStatus := cluster.Status.AcceleratorAccounting
	if clusterStatus == nil {
		result.blockers = append(result.blockers, clusterAccountingBlocker(
			kubermaticv1.AcceleratorAccountingBlockerTypeNewCluster,
			"the KubeVirt cluster has not published accelerator accounting status",
			cluster.Name,
		))
		return result
	}
	if clusterStatus.ObservedAccountingRevision != expected.ObservedAccountingRevision {
		result.blockers = append(result.blockers, clusterAccountingBlocker(
			kubermaticv1.AcceleratorAccountingBlockerTypeRevisionMismatch,
			"the KubeVirt cluster has not observed the current accounting revision",
			cluster.Name,
		))
		return result
	}
	if clusterStatus.ObservedQuotaDigest != expected.ObservedQuotaDigest {
		result.blockers = append(result.blockers, clusterAccountingBlocker(
			kubermaticv1.AcceleratorAccountingBlockerTypeQuotaDigestMismatch,
			"the KubeVirt cluster has not observed the current accelerator quota",
			cluster.Name,
		))
		return result
	}

	result.legacyMachines = clusterStatus.MachinesWithoutFootprint
	result.invalidMachines = clusterStatus.MachinesWithInvalidFootprint
	result.observedAt = clusterStatus.ObservedAt.Time
	result.expiresAt = result.observedAt.Add(resources.AcceleratorAccountingHeartbeatTimeout)
	switch {
	case clusterStatus.FootprintSchemaVersion != accelerator.SchemaVersionV1Alpha1:
		result.blockers = append(result.blockers, clusterAccountingBlocker(
			kubermaticv1.AcceleratorAccountingBlockerTypeUnsupportedFootprintSchema,
			fmt.Sprintf("the KubeVirt cluster reports unsupported footprint schema %q", clusterStatus.FootprintSchemaVersion),
			cluster.Name,
		))
	case clusterStatus.ControllerVersion == "" || (r.controllerVersion != "" && clusterStatus.ControllerVersion != r.controllerVersion):
		result.blockers = append(result.blockers, clusterAccountingBlocker(
			kubermaticv1.AcceleratorAccountingBlockerTypeIncompatibleControllerVersion,
			fmt.Sprintf("the KubeVirt cluster reports controller version %q; expected %q", clusterStatus.ControllerVersion, r.controllerVersion),
			cluster.Name,
		))
	}
	if result.observedAt.IsZero() || !result.expiresAt.After(now) {
		result.blockers = append(result.blockers, clusterAccountingBlocker(
			kubermaticv1.AcceleratorAccountingBlockerTypeStaleHeartbeat,
			"the KubeVirt cluster accelerator accounting heartbeat is stale",
			cluster.Name,
		))
	}
	if !clusterStatus.Ready {
		if len(clusterStatus.Blockers) == 0 {
			result.blockers = append(result.blockers, clusterAccountingBlocker(
				kubermaticv1.AcceleratorAccountingBlockerTypeMissingReport,
				"the KubeVirt cluster accelerator accounting report is not ready",
				cluster.Name,
			))
		} else {
			for _, blocker := range clusterStatus.Blockers {
				blocker.ClusterName = cluster.Name
				result.blockers = append(result.blockers, blocker)
			}
		}
	}
	return result
}

func clusterAccountingBlocker(blockerType kubermaticv1.AcceleratorAccountingBlockerType, message, clusterName string) kubermaticv1.AcceleratorAccountingBlocker {
	return kubermaticv1.AcceleratorAccountingBlocker{
		Type:        blockerType,
		Message:     message,
		ClusterName: clusterName,
	}
}

func (r *reconciler) currentTime() time.Time {
	if r.now != nil {
		return r.now().UTC()
	}
	return time.Now().UTC()
}

func accountingRequeueAfter(now, expiresAt time.Time) time.Duration {
	if expiresAt.IsZero() || !expiresAt.After(now) {
		return resources.AcceleratorAccountingHeartbeatInterval
	}
	return expiresAt.Sub(now)
}

func withClusterEventFilter() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			cluster, ok := e.Object.(*kubermaticv1.Cluster)
			return ok && cluster.Spec.Cloud.Kubevirt != nil
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
			return !reflect.DeepEqual(oldCluster.Status.ResourceUsage, newCluster.Status.ResourceUsage) ||
				!reflect.DeepEqual(oldCluster.Status.AcceleratorAccounting, newCluster.Status.AcceleratorAccounting) ||
				oldCluster.Labels[kubermaticv1.ProjectIDLabelKey] != newCluster.Labels[kubermaticv1.ProjectIDLabelKey] ||
				(oldCluster.Spec.Cloud.Kubevirt == nil) != (newCluster.Spec.Cloud.Kubevirt == nil)
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func enqueueResourceQuota(client ctrlruntimeclient.Client, log *zap.SugaredLogger, workerName string) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request

		clusterLabels := a.GetLabels()
		projectID, ok := clusterLabels[kubermaticv1.ProjectIDLabelKey]
		if !ok {
			log.Debugw("cluster does not have `project-id` label, skipping", "cluster", a.GetName())
			return requests
		}

		subjectNameReq, err := labels.NewRequirement(kubermaticv1.ResourceQuotaSubjectNameLabelKey, selection.Equals, []string{projectID})
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("error creating subject name req: %w", err))
			return requests
		}

		subjectKindReq, err := labels.NewRequirement(kubermaticv1.ResourceQuotaSubjectKindLabelKey, selection.Equals, []string{kubermaticv1.ProjectSubjectKind})
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("error creating subject name req: %w", err))
			return requests
		}

		resourceQuotaList := &kubermaticv1.ResourceQuotaList{}

		if err := client.List(ctx, resourceQuotaList,
			&ctrlruntimeclient.ListOptions{LabelSelector: labels.NewSelector().Add(*subjectKindReq, *subjectNameReq)},
		); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list resourceQuotas: %w", err))
			return requests
		}

		for _, rq := range resourceQuotaList.Items {
			// If a worker-name is given, we want to only reconcile clusters that have that label;
			// this means for multi-cluster resources (e.g. project quotas for projects), we should
			// skip them, as they will contain data for both worker-named and unnamed clusters;
			// otherwise this controller (with a worker-name) would fight another controller (without
			// a worker-name) about the current status of the resource quota.
			// As of now, only project quotas exist though.
			if workerName == "" || rq.Spec.Subject.Kind != kubermaticv1.ProjectSubjectKind {
				requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      rq.Name,
					Namespace: rq.Namespace,
				}})
			}
		}

		return requests
	})
}
