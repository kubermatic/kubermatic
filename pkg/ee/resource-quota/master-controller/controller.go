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
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	k8cequality "k8c.io/kubermatic/sdk/v2/apis/equality"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const ControllerName = "kkp-master-resource-quota-controller"

type reconciler struct {
	masterClient ctrlruntimeclient.Client
	log          *zap.SugaredLogger
	recorder     events.EventRecorder
	seedClients  map[string]ctrlruntimeclient.Client
	now          func() time.Time
	newRevision  func() kubermaticv1.AcceleratorAccountingRevision
}

func Add(mgr manager.Manager,
	seedManagers map[string]manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
) error {
	reconciler := &reconciler{
		log:          log.Named(ControllerName),
		recorder:     mgr.GetEventRecorder(ControllerName),
		masterClient: mgr.GetClient(),
		seedClients:  map[string]ctrlruntimeclient.Client{},
	}

	bldr := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.ResourceQuota{})

	for seedName, seedManager := range seedManagers {
		reconciler.seedClients[seedName] = seedManager.GetClient()

		bldr.WatchesRawSource(source.Kind(
			seedManager.GetCache(),
			&kubermaticv1.ResourceQuota{},
			&handler.TypedEnqueueRequestForObject[*kubermaticv1.ResourceQuota]{},
			seedResourceQuotaChangedPredicate(),
		))
	}

	_, err := bldr.Build(reconciler)

	return err
}

// seedResourceQuotaChangedPredicate keeps the Seed watch focused on fields owned
// by the Seed-side accounting controller. Global status is mirrored from the
// master and must not enqueue the master controller again on every heartbeat.
func seedResourceQuotaChangedPredicate() predicate.TypedPredicate[*kubermaticv1.ResourceQuota] {
	return predicate.TypedFuncs[*kubermaticv1.ResourceQuota]{
		CreateFunc: func(event.TypedCreateEvent[*kubermaticv1.ResourceQuota]) bool { return true },
		UpdateFunc: func(e event.TypedUpdateEvent[*kubermaticv1.ResourceQuota]) bool {
			if e.ObjectOld == nil || e.ObjectNew == nil {
				return false
			}

			return !k8cequality.Semantic.DeepEqual(e.ObjectOld.Status.LocalUsage, e.ObjectNew.Status.LocalUsage) ||
				!k8cequality.Semantic.DeepEqual(e.ObjectOld.Status.LocalAcceleratorAccounting, e.ObjectNew.Status.LocalAcceleratorAccounting) ||
				e.ObjectOld.Spec.Subject != e.ObjectNew.Spec.Subject ||
				e.ObjectOld.Labels[kubermaticv1.ResourceQuotaSubjectNameLabelKey] != e.ObjectNew.Labels[kubermaticv1.ResourceQuotaSubjectNameLabelKey] ||
				e.ObjectOld.Labels[kubermaticv1.ResourceQuotaSubjectKindLabelKey] != e.ObjectNew.Labels[kubermaticv1.ResourceQuotaSubjectKindLabelKey] ||
				(e.ObjectOld.DeletionTimestamp == nil) != (e.ObjectNew.DeletionTimestamp == nil)
		},
		DeleteFunc:  func(event.TypedDeleteEvent[*kubermaticv1.ResourceQuota]) bool { return true },
		GenericFunc: func(event.TypedGenericEvent[*kubermaticv1.ResourceQuota]) bool { return false },
	}
}

// Reconcile calculates project-wide resource usage and accelerator accounting readiness.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Reconciling")

	resourceQuota := &kubermaticv1.ResourceQuota{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, resourceQuota); err != nil {
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
	// skip reconcile if resourceQuota is in delete state
	if !resourceQuota.DeletionTimestamp.IsZero() {
		log.Debug("resource quota is in deletion, skipping")
		return 0, nil
	}

	// for all related resource quotas on seeds, calculate global usage
	globalUsage := kubermaticv1.NewResourceDetails(resource.Quantity{}, resource.Quantity{}, resource.Quantity{})
	acceleratorAccountingActive := resourceQuota.Annotations[resources.AcceleratorAccountingEnabledAnnotation] == resources.AcceleratorAccountingEnabledAnnotationValue
	seedResourceQuotas := make(map[string]*kubermaticv1.ResourceQuota, len(r.seedClients))
	seedErrors := make(map[string]error)
	for seed, seedClient := range r.seedClients {
		seedResourceQuota := &kubermaticv1.ResourceQuota{}
		err := seedClient.Get(ctx, types.NamespacedName{Namespace: resourceQuota.Namespace, Name: resourceQuota.Name},
			seedResourceQuota)
		if err != nil {
			seedErrors[seed] = err
			continue
		}
		seedResourceQuotas[seed] = seedResourceQuota
		localUsage := seedResourceQuota.Status.LocalUsage
		if localUsage.CPU != nil {
			globalUsage.CPU.Add(*localUsage.CPU)
		}
		if localUsage.Memory != nil {
			globalUsage.Memory.Add(*localUsage.Memory)
		}
		if localUsage.Storage != nil {
			globalUsage.Storage.Add(*localUsage.Storage)
		}
		if acceleratorAccountingActive {
			kubermaticv1.AddAcceleratorUsage(globalUsage, localUsage.Accelerators)
		}
	}

	seedReadError := aggregateSeedReadErrors(seedErrors)
	if seedReadError != nil {
		// Preserve the last complete usage snapshot if a configured Seed cannot be
		// read. Accelerator readiness still records that Seed as unreachable below.
		globalUsage = resourceQuota.Status.GlobalUsage.DeepCopy()
	}
	globalAccounting, requeueAfter := r.globalAcceleratorAccounting(resourceQuota, seedResourceQuotas, seedErrors)
	if err := r.ensureGlobalStatus(ctx, log, resourceQuota, globalUsage, globalAccounting); err != nil {
		return 0, err
	}
	if seedReadError != nil {
		return 0, seedReadError
	}

	return requeueAfter, nil
}

func aggregateSeedReadErrors(seedErrors map[string]error) error {
	seedNames := make([]string, 0, len(seedErrors))
	for seed, err := range seedErrors {
		if !apierrors.IsNotFound(err) {
			seedNames = append(seedNames, seed)
		}
	}
	if len(seedNames) == 0 {
		return nil
	}

	sort.Strings(seedNames)
	seedReadErrors := make([]error, 0, len(seedNames))
	for _, seed := range seedNames {
		seedReadErrors = append(seedReadErrors, fmt.Errorf("seed %q: %w", seed, seedErrors[seed]))
	}
	return fmt.Errorf("error getting ResourceQuota from configured Seeds: %w", errors.Join(seedReadErrors...))
}

func (r *reconciler) ensureGlobalStatus(ctx context.Context, log *zap.SugaredLogger, resourceQuota *kubermaticv1.ResourceQuota,
	globalUsage *kubermaticv1.ResourceDetails, globalAccounting *kubermaticv1.ResourceQuotaGlobalAcceleratorAccountingStatus) error {
	if k8cequality.Semantic.DeepEqual(*globalUsage, resourceQuota.Status.GlobalUsage) &&
		k8cequality.Semantic.DeepEqual(globalAccounting, resourceQuota.Status.GlobalAcceleratorAccounting) {
		log.Debugw("global usage for resource quota is the same, not updating",
			"cpu", globalUsage.CPU.String(),
			"memory", globalUsage.Memory.String(),
			"storage", globalUsage.Storage.String())
		return nil
	}
	log.Debugw("global usage for resource quota needs update",
		"cpu", globalUsage.CPU.String(),
		"memory", globalUsage.Memory.String(),
		"storage", globalUsage.Storage.String())

	return util.UpdateResourceQuotaStatus(ctx, r.masterClient, resourceQuota, func(rq *kubermaticv1.ResourceQuota) {
		rq.Status.GlobalUsage = *globalUsage
		rq.Status.GlobalAcceleratorAccounting = globalAccounting
	})
}

func (r *reconciler) globalAcceleratorAccounting(
	resourceQuota *kubermaticv1.ResourceQuota,
	seedResourceQuotas map[string]*kubermaticv1.ResourceQuota,
	seedErrors map[string]error,
) (*kubermaticv1.ResourceQuotaGlobalAcceleratorAccountingStatus, time.Duration) {
	if resourceQuota.Annotations[resources.AcceleratorAccountingEnabledAnnotation] != resources.AcceleratorAccountingEnabledAnnotationValue {
		return nil, 0
	}

	now := r.currentTime()
	digest := kubermaticv1.AcceleratorQuotaDigestFor(resourceQuota.Spec.Quota.Accelerators)
	previous := resourceQuota.Status.GlobalAcceleratorAccounting
	transitioning := previous == nil || previous.ObservedAccountingRevision == "" || previous.ObservedQuotaDigest != digest
	var revision kubermaticv1.AcceleratorAccountingRevision
	if !transitioning {
		revision = previous.ObservedAccountingRevision
	} else {
		revision = r.nextRevision()
	}

	status := &kubermaticv1.ResourceQuotaGlobalAcceleratorAccountingStatus{
		ActivationPhase:            kubermaticv1.AcceleratorAccountingPhaseActivating,
		ObservedAccountingRevision: revision,
		ObservedQuotaDigest:        digest,
		Ready:                      true,
	}

	if len(r.seedClients) == 0 {
		status.ActivationPhase = kubermaticv1.AcceleratorAccountingPhaseReady
		if !transitioning && !previous.ObservedAt.IsZero() {
			refreshAt := previous.ObservedAt.Add(resources.AcceleratorAccountingHeartbeatInterval)
			if refreshAt.After(now) {
				status.ObservedAt = previous.ObservedAt
				return status, refreshAt.Sub(now)
			}
		}
		status.ObservedAt = metav1.NewTime(now)
		return status, resources.AcceleratorAccountingHeartbeatInterval
	}

	var oldestObservedAt time.Time
	var nextExpiry time.Time
	seedNames := make([]string, 0, len(r.seedClients))
	for seed := range r.seedClients {
		seedNames = append(seedNames, seed)
	}
	sort.Strings(seedNames)
	for _, seed := range seedNames {
		seedResourceQuota, found := seedResourceQuotas[seed]
		if !found {
			status.Ready = false
			message := "the ResourceQuota is missing from the configured Seed"
			if err := seedErrors[seed]; err != nil && !apierrors.IsNotFound(err) {
				message = fmt.Sprintf("the configured Seed cannot be reached: %v", err)
			}
			status.Blockers = append(status.Blockers, kubermaticv1.AcceleratorAccountingBlocker{
				Type:     kubermaticv1.AcceleratorAccountingBlockerTypeUnreachableSeed,
				Message:  message,
				SeedName: seed,
			})
			continue
		}

		localStatus := seedResourceQuota.Status.LocalAcceleratorAccounting
		if localStatus == nil {
			status.Ready = false
			status.Blockers = append(status.Blockers, kubermaticv1.AcceleratorAccountingBlocker{
				Type:     kubermaticv1.AcceleratorAccountingBlockerTypeMissingReport,
				Message:  "the configured Seed has not published accelerator accounting status",
				SeedName: seed,
			})
			continue
		}

		accepted := true
		switch {
		case localStatus.ObservedAccountingRevision != revision:
			accepted = false
			status.Blockers = append(status.Blockers, kubermaticv1.AcceleratorAccountingBlocker{
				Type:     kubermaticv1.AcceleratorAccountingBlockerTypeRevisionMismatch,
				Message:  "the configured Seed has not observed the current accounting revision",
				SeedName: seed,
			})
		case localStatus.ObservedQuotaDigest != digest:
			accepted = false
			status.Blockers = append(status.Blockers, kubermaticv1.AcceleratorAccountingBlocker{
				Type:     kubermaticv1.AcceleratorAccountingBlockerTypeQuotaDigestMismatch,
				Message:  "the configured Seed has not observed the current accelerator quota",
				SeedName: seed,
			})
		}
		if !accepted {
			status.Ready = false
			continue
		}

		status.LegacyMachinesWithoutFootprint += localStatus.LegacyMachinesWithoutFootprint
		status.MachinesWithInvalidFootprint += localStatus.MachinesWithInvalidFootprint
		observedAt := localStatus.ObservedAt.Time
		if observedAt.IsZero() {
			// A transient child bootstrap blocker such as NewCluster legitimately has
			// no observation time yet. Treat a missing timestamp as stale only when
			// the Seed incorrectly claims that the report is already ready.
			if localStatus.Ready {
				status.Blockers = append(status.Blockers, kubermaticv1.AcceleratorAccountingBlocker{
					Type:     kubermaticv1.AcceleratorAccountingBlockerTypeStaleHeartbeat,
					Message:  "the configured Seed accelerator accounting heartbeat is stale",
					SeedName: seed,
				})
			}
		} else {
			if oldestObservedAt.IsZero() || observedAt.Before(oldestObservedAt) {
				oldestObservedAt = observedAt
			}
			expiresAt := observedAt.Add(resources.AcceleratorAccountingHeartbeatTimeout)
			if nextExpiry.IsZero() || expiresAt.Before(nextExpiry) {
				nextExpiry = expiresAt
			}
			if !expiresAt.After(now) {
				status.Blockers = append(status.Blockers, kubermaticv1.AcceleratorAccountingBlocker{
					Type:     kubermaticv1.AcceleratorAccountingBlockerTypeStaleHeartbeat,
					Message:  "the configured Seed accelerator accounting heartbeat is stale",
					SeedName: seed,
				})
			}
		}
		if !localStatus.Ready {
			if len(localStatus.Blockers) == 0 {
				status.Blockers = append(status.Blockers, kubermaticv1.AcceleratorAccountingBlocker{
					Type:     kubermaticv1.AcceleratorAccountingBlockerTypeMissingReport,
					Message:  "the configured Seed accelerator accounting report is not ready",
					SeedName: seed,
				})
			} else {
				for _, blocker := range localStatus.Blockers {
					blocker.SeedName = seed
					status.Blockers = append(status.Blockers, blocker)
				}
			}
		}
	}

	if !oldestObservedAt.IsZero() {
		status.ObservedAt = metav1.NewTime(oldestObservedAt)
	}
	status.Ready = status.Ready && len(status.Blockers) == 0
	status.ActivationPhase = acceleratorAccountingPhase(status.Ready, transitioning, status.Blockers)
	return status, masterAccountingRequeueAfter(now, nextExpiry)
}

func acceleratorAccountingPhase(ready, transitioning bool, blockers []kubermaticv1.AcceleratorAccountingBlocker) kubermaticv1.AcceleratorAccountingPhase {
	if ready {
		return kubermaticv1.AcceleratorAccountingPhaseReady
	}
	if transitioning {
		return kubermaticv1.AcceleratorAccountingPhaseActivating
	}

	for _, blocker := range blockers {
		switch blocker.Type {
		case kubermaticv1.AcceleratorAccountingBlockerTypeMissingReport,
			kubermaticv1.AcceleratorAccountingBlockerTypeRevisionMismatch,
			kubermaticv1.AcceleratorAccountingBlockerTypeQuotaDigestMismatch,
			kubermaticv1.AcceleratorAccountingBlockerTypeNewCluster:
			continue
		default:
			return kubermaticv1.AcceleratorAccountingPhaseBlocked
		}
	}
	return kubermaticv1.AcceleratorAccountingPhaseActivating
}

func (r *reconciler) currentTime() time.Time {
	if r.now != nil {
		return r.now().UTC()
	}
	return time.Now().UTC()
}

func (r *reconciler) nextRevision() kubermaticv1.AcceleratorAccountingRevision {
	if r.newRevision != nil {
		return r.newRevision()
	}
	return kubermaticv1.AcceleratorAccountingRevision(uuid.NewString())
}

func masterAccountingRequeueAfter(now, expiresAt time.Time) time.Duration {
	if expiresAt.IsZero() || !expiresAt.After(now) {
		return resources.AcceleratorAccountingHeartbeatInterval
	}
	return expiresAt.Sub(now)
}
