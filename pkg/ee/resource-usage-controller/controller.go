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

package resourceusagecontroller

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	controllerpredicate "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	machinevalidation "k8c.io/kubermatic/v2/pkg/ee/validation/machine"
	"k8c.io/kubermatic/v2/pkg/machine/accelerator"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	"k8c.io/machine-controller/sdk/providerconfig"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/events"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrlruntimepredicate "sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const controllerName = "resource_usage_controller"

type reconciler struct {
	log                    *zap.SugaredLogger
	seedClient             ctrlruntimeclient.Client
	userClient             ctrlruntimeclient.Client
	clusterName            string
	kubeVirtInfraNamespace string
	caBundle               *certificates.CABundle
	recorder               events.EventRecorder
	clusterIsPaused        userclustercontrollermanager.IsPausedChecker
	controllerVersion      string
	acceleratorAccounting  bool
	clock                  clock.Clock
}

func Add(log *zap.SugaredLogger, seedMgr, userMgr manager.Manager, clusterName, kubeVirtInfraNamespace, controllerVersion string, acceleratorAccounting bool, caBundle *certificates.CABundle,
	clusterIsPaused userclustercontrollermanager.IsPausedChecker) error {
	log = log.Named(controllerName)

	r := &reconciler{
		log:                    log,
		seedClient:             seedMgr.GetClient(),
		userClient:             userMgr.GetClient(),
		clusterName:            clusterName,
		kubeVirtInfraNamespace: kubeVirtInfraNamespace,
		caBundle:               caBundle,
		recorder:               userMgr.GetEventRecorder(controllerName),
		clusterIsPaused:        clusterIsPaused,
		controllerVersion:      controllerVersion,
		acceleratorAccounting:  acceleratorAccounting,
		clock:                  clock.RealClock{},
	}

	bldr := builder.ControllerManagedBy(userMgr).
		Named(controllerName).
		For(&clusterv1alpha1.Machine{}, builder.WithPredicates(controllerpredicate.ByNamespace(metav1.NamespaceSystem)))
	if acceleratorAccounting {
		bldr.WatchesRawSource(source.Kind(
			seedMgr.GetCache(),
			&kubermaticv1.ResourceQuota{},
			handler.TypedEnqueueRequestsFromMapFunc(r.mapResourceQuotaToRequest),
			resourceQuotaAccountingChangedPredicate(),
		)).WatchesRawSource(source.Kind(
			seedMgr.GetCache(),
			&kubermaticv1.Cluster{},
			handler.TypedEnqueueRequestsFromMapFunc(r.mapClusterToRequest),
			controllerpredicate.TypedByName[*kubermaticv1.Cluster](clusterName),
			clusterAccountingRelevantChangePredicate(),
		))
	}

	_, err := bldr.Build(r)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		if r.isAcceleratorAccountingHeartbeatRequest(request) {
			// Keep the heartbeat loop alive so accounting resumes after an unpause
			// even when no Machine or ResourceQuota event occurs in the meantime.
			return reconcile.Result{RequeueAfter: resources.AcceleratorAccountingHeartbeatInterval}, nil
		}
		return reconcile.Result{}, nil
	}

	log := r.log.With("resource", request)
	log.Debug("reconciling")

	machines := &clusterv1alpha1.MachineList{}
	if err := r.userClient.List(ctx, machines); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get machines: %w", err)
	}

	cluster := &kubermaticv1.Cluster{}
	if err = r.seedClient.Get(ctx, types.NamespacedName{
		Name: r.clusterName,
	}, cluster); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}

	var resourceQuota *kubermaticv1.ResourceQuota
	if r.acceleratorAccounting {
		resourceQuota, err = r.activeAcceleratorAccountingResourceQuota(ctx, cluster)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	err = r.reconcile(ctx, cluster, machines, resourceQuota)
	if err != nil {
		r.recorder.Eventf(cluster, nil, corev1.EventTypeWarning, "ClusterResourceUsageReconcileFailed", "Reconciling", err.Error())
		return reconcile.Result{}, err
	}

	if resourceQuota != nil && r.isAcceleratorAccountingHeartbeatRequest(request) {
		return reconcile.Result{RequeueAfter: resources.AcceleratorAccountingHeartbeatInterval}, nil
	}

	return reconcile.Result{}, nil
}

func (r *reconciler) isAcceleratorAccountingHeartbeatRequest(request reconcile.Request) bool {
	// Machine events retain their normal namespaced request keys and remain event-driven.
	// The Seed ResourceQuota and Cluster watches map to this one cluster-scoped key, which
	// alone owns the periodic heartbeat so the work does not scale with Machine count.
	return r.acceleratorAccounting && request.Namespace == "" && request.Name == r.clusterName
}

func (r *reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster, machines *clusterv1alpha1.MachineList, resourceQuota *kubermaticv1.ResourceQuota) error {
	resourceUsage := kubermaticv1.NewResourceDetails(resource.Quantity{}, resource.Quantity{}, resource.Quantity{})
	for _, machine := range machines.Items {
		resourceDetails, err := machinevalidation.GetMachineResourceUsage(ctx, r.userClient, r.kubeVirtInfraNamespace, &machine, r.caBundle)
		if err != nil {
			return fmt.Errorf("error getting machine resource usage for machine %q: %w", machine.Name, err)
		}

		resourceUsage.CPU.Add(*resourceDetails.CPU())
		resourceUsage.Memory.Add(*resourceDetails.Memory())
		resourceUsage.Storage.Add(*resourceDetails.Storage())
	}

	var accountingStatus *kubermaticv1.ClusterAcceleratorAccountingStatus
	if resourceQuota != nil {
		resourceUsage.Accelerators, accountingStatus = r.acceleratorUsageAndStatus(cluster.Name, resourceQuota, machines)
	}

	return util.UpdateClusterStatus(ctx, r.seedClient, cluster, func(c *kubermaticv1.Cluster) {
		c.Status.ResourceUsage = resourceUsage
		c.Status.AcceleratorAccounting = accountingStatus
	})
}

func (r *reconciler) acceleratorUsageAndStatus(clusterName string, resourceQuota *kubermaticv1.ResourceQuota, machines *clusterv1alpha1.MachineList) ([]kubermaticv1.AcceleratorQuota, *kubermaticv1.ClusterAcceleratorAccountingStatus) {
	usage := &kubermaticv1.ResourceDetails{}
	var machinesWithoutFootprint int32
	var machinesWithInvalidFootprint int32
	var machinesWithUnsupportedFootprintSchema int32

	for i := range machines.Items {
		machine := &machines.Items[i]
		encodedFootprint, hasFootprint := machine.Annotations[accelerator.FootprintAnnotationKey]

		providerConfig, err := providerconfig.GetConfig(machine.Spec.ProviderSpec)
		if err != nil {
			if hasFootprint {
				machinesWithInvalidFootprint++
			}
			continue
		}
		if providerConfig.CloudProvider != providerconfig.CloudProviderKubeVirt {
			if hasFootprint {
				machinesWithInvalidFootprint++
			}
			continue
		}
		if !hasFootprint {
			machinesWithoutFootprint++
			continue
		}

		footprint, err := accelerator.Decode(encodedFootprint)
		if err != nil {
			machinesWithInvalidFootprint++
			if errors.Is(err, accelerator.ErrUnsupportedSchemaVersion) {
				machinesWithUnsupportedFootprintSchema++
			}
			continue
		}

		if footprint.Provider != accelerator.ProviderKubeVirt {
			machinesWithInvalidFootprint++
			continue
		}

		if len(footprint.Resources) > 0 {
			kubermaticv1.AddAcceleratorUsage(usage, []kubermaticv1.AcceleratorQuota{{
				Provider:  footprint.Provider,
				Resources: footprint.Resources,
			}})
		}
	}

	quotaDigest := kubermaticv1.AcceleratorQuotaDigestFor(resourceQuota.Spec.Quota.Accelerators)
	status := &kubermaticv1.ClusterAcceleratorAccountingStatus{
		ObservedQuotaDigest:          quotaDigest,
		FootprintSchemaVersion:       accelerator.SchemaVersionV1Alpha1,
		ControllerVersion:            r.controllerVersion,
		ObservedAt:                   metav1.NewTime(r.clock.Now().UTC()),
		MachinesWithoutFootprint:     machinesWithoutFootprint,
		MachinesWithInvalidFootprint: machinesWithInvalidFootprint,
	}

	globalStatus := resourceQuota.Status.GlobalAcceleratorAccounting
	if globalStatus != nil {
		status.ObservedAccountingRevision = globalStatus.ObservedAccountingRevision
	}
	if globalStatus == nil || globalStatus.ObservedAccountingRevision == "" || globalStatus.ObservedQuotaDigest == "" {
		status.Blockers = append(status.Blockers, acceleratorAccountingBlocker(
			kubermaticv1.AcceleratorAccountingBlockerTypeMissingReport,
			"the synchronized ResourceQuota does not contain a master-issued accounting revision and quota digest",
			clusterName,
			0,
		))
	} else if globalStatus.ObservedQuotaDigest != quotaDigest {
		status.Blockers = append(status.Blockers, acceleratorAccountingBlocker(
			kubermaticv1.AcceleratorAccountingBlockerTypeQuotaDigestMismatch,
			"the synchronized accounting quota digest does not match the accelerator quota spec",
			clusterName,
			0,
		))
	}

	if machinesWithoutFootprint > 0 {
		status.Blockers = append(status.Blockers, acceleratorAccountingBlocker(
			kubermaticv1.AcceleratorAccountingBlockerTypeLegacyMachines,
			"KubeVirt Machines without a trusted accelerator footprint must be recreated",
			clusterName,
			machinesWithoutFootprint,
		))
	}
	if machinesWithUnsupportedFootprintSchema > 0 {
		status.Blockers = append(status.Blockers, acceleratorAccountingBlocker(
			kubermaticv1.AcceleratorAccountingBlockerTypeUnsupportedFootprintSchema,
			"KubeVirt Machines use an unsupported accelerator footprint schema",
			clusterName,
			machinesWithUnsupportedFootprintSchema,
		))
	}
	if invalidFootprints := machinesWithInvalidFootprint - machinesWithUnsupportedFootprintSchema; invalidFootprints > 0 {
		status.Blockers = append(status.Blockers, acceleratorAccountingBlocker(
			kubermaticv1.AcceleratorAccountingBlockerTypeInvalidFootprints,
			"KubeVirt Machines have malformed footprints or footprints that do not match their provider",
			clusterName,
			invalidFootprints,
		))
	}

	status.Ready = len(status.Blockers) == 0
	return usage.Accelerators, status
}

func acceleratorAccountingBlocker(blockerType kubermaticv1.AcceleratorAccountingBlockerType, message, clusterName string, count int32) kubermaticv1.AcceleratorAccountingBlocker {
	return kubermaticv1.AcceleratorAccountingBlocker{
		Type:        blockerType,
		Message:     message,
		ClusterName: clusterName,
		Count:       count,
	}
}

func (r *reconciler) activeAcceleratorAccountingResourceQuota(ctx context.Context, cluster *kubermaticv1.Cluster) (*kubermaticv1.ResourceQuota, error) {
	if cluster.Spec.Cloud.Kubevirt == nil {
		return nil, nil
	}

	projectID := cluster.Labels[kubermaticv1.ProjectIDLabelKey]
	if projectID == "" {
		return nil, nil
	}

	resourceQuotas := &kubermaticv1.ResourceQuotaList{}
	if err := r.seedClient.List(ctx, resourceQuotas, ctrlruntimeclient.MatchingLabels{
		kubermaticv1.ResourceQuotaSubjectNameLabelKey: projectID,
		kubermaticv1.ResourceQuotaSubjectKindLabelKey: kubermaticv1.ProjectSubjectKind,
	}); err != nil {
		return nil, fmt.Errorf("failed to list ResourceQuotas for project %q: %w", projectID, err)
	}

	var activeResourceQuota *kubermaticv1.ResourceQuota
	for i := range resourceQuotas.Items {
		resourceQuota := &resourceQuotas.Items[i]
		if !acceleratorAccountingActiveForProject(resourceQuota, projectID) {
			continue
		}
		if activeResourceQuota != nil {
			return nil, fmt.Errorf("multiple active accelerator accounting ResourceQuotas found for project %q", projectID)
		}
		activeResourceQuota = resourceQuota
	}

	return activeResourceQuota, nil
}

func acceleratorAccountingActiveForProject(resourceQuota *kubermaticv1.ResourceQuota, projectID string) bool {
	return resourceQuota != nil &&
		projectID != "" &&
		resourceQuota.Spec.Subject.Kind == kubermaticv1.ProjectSubjectKind &&
		resourceQuota.Spec.Subject.Name == projectID &&
		resourceQuota.Labels[kubermaticv1.ResourceQuotaSubjectNameLabelKey] == projectID &&
		resourceQuota.Labels[kubermaticv1.ResourceQuotaSubjectKindLabelKey] == kubermaticv1.ProjectSubjectKind &&
		resourceQuota.DeletionTimestamp.IsZero() &&
		resourceQuota.Annotations[resources.AcceleratorAccountingEnabledAnnotation] == resources.AcceleratorAccountingEnabledAnnotationValue
}

func (r *reconciler) mapResourceQuotaToRequest(ctx context.Context, resourceQuota *kubermaticv1.ResourceQuota) []reconcile.Request {
	cluster := &kubermaticv1.Cluster{}
	if err := r.seedClient.Get(ctx, types.NamespacedName{Name: r.clusterName}, cluster); err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to get Cluster %q while mapping ResourceQuota event: %w", r.clusterName, err))
		return nil
	}

	projectID := cluster.Labels[kubermaticv1.ProjectIDLabelKey]
	if projectID == "" || resourceQuota.Spec.Subject.Kind != kubermaticv1.ProjectSubjectKind || resourceQuota.Spec.Subject.Name != projectID ||
		resourceQuota.Labels[kubermaticv1.ResourceQuotaSubjectNameLabelKey] != projectID ||
		resourceQuota.Labels[kubermaticv1.ResourceQuotaSubjectKindLabelKey] != kubermaticv1.ProjectSubjectKind {
		return nil
	}

	return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: r.clusterName}}}
}

func (r *reconciler) mapClusterToRequest(_ context.Context, _ *kubermaticv1.Cluster) []reconcile.Request {
	return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: r.clusterName}}}
}

func resourceQuotaAccountingChangedPredicate() ctrlruntimepredicate.TypedPredicate[*kubermaticv1.ResourceQuota] {
	return ctrlruntimepredicate.TypedFuncs[*kubermaticv1.ResourceQuota]{
		CreateFunc: func(event.TypedCreateEvent[*kubermaticv1.ResourceQuota]) bool { return true },
		UpdateFunc: func(e event.TypedUpdateEvent[*kubermaticv1.ResourceQuota]) bool {
			if e.ObjectOld == nil || e.ObjectNew == nil {
				return false
			}

			oldRevision, oldDigest := acceleratorAccountingPair(e.ObjectOld.Status.GlobalAcceleratorAccounting)
			newRevision, newDigest := acceleratorAccountingPair(e.ObjectNew.Status.GlobalAcceleratorAccounting)
			return e.ObjectOld.Spec.Subject != e.ObjectNew.Spec.Subject ||
				!kubermaticv1.AcceleratorQuotasEqual(e.ObjectOld.Spec.Quota.Accelerators, e.ObjectNew.Spec.Quota.Accelerators) ||
				e.ObjectOld.Labels[kubermaticv1.ResourceQuotaSubjectNameLabelKey] != e.ObjectNew.Labels[kubermaticv1.ResourceQuotaSubjectNameLabelKey] ||
				e.ObjectOld.Labels[kubermaticv1.ResourceQuotaSubjectKindLabelKey] != e.ObjectNew.Labels[kubermaticv1.ResourceQuotaSubjectKindLabelKey] ||
				e.ObjectOld.Annotations[resources.AcceleratorAccountingEnabledAnnotation] != e.ObjectNew.Annotations[resources.AcceleratorAccountingEnabledAnnotation] ||
				(e.ObjectOld.DeletionTimestamp == nil) != (e.ObjectNew.DeletionTimestamp == nil) ||
				oldRevision != newRevision || oldDigest != newDigest
		},
		DeleteFunc:  func(event.TypedDeleteEvent[*kubermaticv1.ResourceQuota]) bool { return true },
		GenericFunc: func(event.TypedGenericEvent[*kubermaticv1.ResourceQuota]) bool { return false },
	}
}

func acceleratorAccountingPair(status *kubermaticv1.ResourceQuotaGlobalAcceleratorAccountingStatus) (kubermaticv1.AcceleratorAccountingRevision, kubermaticv1.AcceleratorQuotaDigest) {
	if status == nil {
		return "", ""
	}
	return status.ObservedAccountingRevision, status.ObservedQuotaDigest
}

func clusterAccountingRelevantChangePredicate() ctrlruntimepredicate.TypedPredicate[*kubermaticv1.Cluster] {
	return ctrlruntimepredicate.TypedFuncs[*kubermaticv1.Cluster]{
		CreateFunc: func(event.TypedCreateEvent[*kubermaticv1.Cluster]) bool { return true },
		UpdateFunc: func(e event.TypedUpdateEvent[*kubermaticv1.Cluster]) bool {
			if e.ObjectOld == nil || e.ObjectNew == nil {
				return false
			}

			return e.ObjectOld.Labels[kubermaticv1.ProjectIDLabelKey] != e.ObjectNew.Labels[kubermaticv1.ProjectIDLabelKey] ||
				(e.ObjectOld.Spec.Cloud.Kubevirt == nil) != (e.ObjectNew.Spec.Cloud.Kubevirt == nil) ||
				(e.ObjectOld.DeletionTimestamp == nil) != (e.ObjectNew.DeletionTimestamp == nil)
		},
		DeleteFunc:  func(event.TypedDeleteEvent[*kubermaticv1.Cluster]) bool { return false },
		GenericFunc: func(event.TypedGenericEvent[*kubermaticv1.Cluster]) bool { return false },
	}
}
