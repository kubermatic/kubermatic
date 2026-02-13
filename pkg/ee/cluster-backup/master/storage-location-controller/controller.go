//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2023 Kubermatic GmbH

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

package storagelocationcontroller

import (
	"context"
	"fmt"
	"time"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/ee/cluster-backup/master/storage-location-controller/backupstore"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName   = "cluster-backup-storage-location-controller"
	CleanupFinalizer = "kubermatic.k8c.io/cleanup-credentials"

	// Regularly check S3 bucket availability.
	requeueInterval = 30 * time.Minute
)

type reconciler struct {
	client   ctrlruntimeclient.Client
	recorder events.EventRecorder
	log      *zap.SugaredLogger
}

func Add(mgr manager.Manager, numWorkers int, log *zap.SugaredLogger) error {
	reconciler := &reconciler{
		client:   mgr.GetClient(),
		recorder: mgr.GetEventRecorder(ControllerName),
		log:      log,
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.ClusterBackupStorageLocation{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Build(reconciler)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Reconciling")

	cbsl := &kubermaticv1.ClusterBackupStorageLocation{}
	if err := r.client.Get(ctx, request.NamespacedName, cbsl); err != nil {
		return reconcile.Result{}, nil
	}
	if cbsl.DeletionTimestamp != nil {
		return reconcile.Result{}, r.cleanup(ctx, cbsl)
	}

	if cbsl.Spec.Provider != "aws" {
		log.Infow("unsupported provider, skipping.", "provider", cbsl.Spec.Provider)
		return reconcile.Result{}, nil
	}
	if cbsl.Spec.Credential == nil {
		log.Info("no credentials secret reference, skipping.")
		return reconcile.Result{}, nil
	}

	err := r.reconcile(ctx, cbsl)
	if err != nil {
		r.recorder.Eventf(cbsl, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
		return reconcile.Result{}, err
	}

	return reconcile.Result{
		Requeue:      true,
		RequeueAfter: requeueInterval,
	}, nil
}

func (r *reconciler) reconcile(ctx context.Context, cbsl *kubermaticv1.ClusterBackupStorageLocation) error {
	creds, err := r.getCredentials(ctx, cbsl)
	if err != nil {
		return fmt.Errorf("failed to get the credentials secret: %w", err)
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.client, cbsl, CleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	store, err := backupstore.NewBackupStore(cbsl, creds)
	if err != nil {
		return fmt.Errorf("failed to create backup store: %w", err)
	}

	if err = store.IsValid(ctx); err != nil {
		return r.updateCBSLStatus(ctx,
			cbsl,
			velerov1.BackupStorageLocationPhaseUnavailable,
			err.Error(),
		)
	}

	return r.updateCBSLStatus(ctx,
		cbsl,
		velerov1.BackupStorageLocationPhaseAvailable,
		"ClusterBackupStoreLocation is available",
	)
}

func (r *reconciler) updateCBSLStatus(ctx context.Context, cbsl *kubermaticv1.ClusterBackupStorageLocation, phase velerov1.BackupStorageLocationPhase, message string) error {
	key := types.NamespacedName{
		Namespace: cbsl.Namespace,
		Name:      cbsl.Name,
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := r.client.Get(ctx, key, cbsl); err != nil {
			return err
		}

		updatedCBSL := cbsl.DeepCopy()
		updatedCBSL.Status.Message = message
		updatedCBSL.Status.Phase = phase
		now := metav1.Now()
		updatedCBSL.Status.LastValidationTime = &now
		// we patch anyway even if there is no changes because we want to update the LastValidationTime.
		return r.client.Status().Patch(ctx, updatedCBSL, ctrlruntimeclient.MergeFrom(cbsl))
	})
}

func (r *reconciler) cleanup(ctx context.Context, cbsl *kubermaticv1.ClusterBackupStorageLocation) error {
	creds, err := r.getCredentials(ctx, cbsl)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return kuberneteshelper.TryRemoveFinalizer(ctx, r.client, cbsl, CleanupFinalizer)
		}
		return fmt.Errorf("failed to get the credentials secret: %w", err)
	}
	if err := r.client.Delete(ctx, creds); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete credentials secret: %w", err)
	}
	return kuberneteshelper.TryRemoveFinalizer(ctx, r.client, cbsl, CleanupFinalizer)
}

func (r *reconciler) getCredentials(ctx context.Context, cbsl *kubermaticv1.ClusterBackupStorageLocation) (*corev1.Secret, error) {
	creds := &corev1.Secret{}
	key := types.NamespacedName{
		Name:      cbsl.Spec.Credential.Name,
		Namespace: cbsl.Namespace,
	}

	if err := r.client.Get(ctx, key, creds); err != nil {
		return nil, err
	}

	return creds, nil
}
