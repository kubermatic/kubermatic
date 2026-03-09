//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2024 Kubermatic GmbH

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

	"go.uber.org/zap"

	"k8c.io/kubermatic/sdk/v2/apis/equality"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "cluster-backup-controller"

	// cleanupFinalizer indicates that CBSLs on the seed clusters need cleanup.
	cleanupFinalizer = "kubermatic.k8c.io/cleanup-seed-storage-locations"
)

type reconciler struct {
	log          *zap.SugaredLogger
	recorder     events.EventRecorder
	masterClient ctrlruntimeclient.Client
	seedClients  kuberneteshelper.SeedClientMap
}

func Add(
	masterManager manager.Manager,
	seedManagers map[string]manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
) error {
	r := &reconciler{
		log:          log.Named(ControllerName),
		recorder:     masterManager.GetEventRecorder(ControllerName),
		masterClient: masterManager.GetClient(),
		seedClients:  kuberneteshelper.SeedClientMap{},
	}

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()
	}

	_, err := builder.ControllerManagedBy(masterManager).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.ClusterBackupStorageLocation{}).
		Build(r)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)

	cbsl := &kubermaticv1.ClusterBackupStorageLocation{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, cbsl); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get storage location: %w", err)
	}

	if !cbsl.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, cbsl); err != nil {
			return reconcile.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return reconcile.Result{}, nil
	}

	cbslReconcilerFactories := []kkpreconciling.NamedClusterBackupStorageLocationReconcilerFactory{
		cbslReconcilerFactory(cbsl),
	}

	err := r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
		seedCBSL := &kubermaticv1.ClusterBackupStorageLocation{}
		if err := seedClient.Get(ctx, request.NamespacedName, seedCBSL); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to fetch storage location on seed cluster: %w", err)
		}

		// The informer can trigger a reconciliation before the cache backing the
		// master client has been updated; this can make the reconciler read
		// an old state and would replicate this old state onto seeds; if master
		// and seed are the same cluster, this would effectively overwrite the
		// change that just happened.
		// To prevent this from occurring, we check the UID and refuse to update
		// the CBSL if the UID on the seed == UID on the master.
		// Note that in this distinction cannot be made inside the creator function
		// further down, as the reconciling framework reads the current state
		// from cache and even if no changes were made (because of the UID match),
		// it would still persist the new object and might overwrite the actual,
		// new state.
		if seedCBSL.UID != "" && seedCBSL.UID == cbsl.UID {
			return nil
		}

		err := kkpreconciling.ReconcileClusterBackupStorageLocations(ctx, cbslReconcilerFactories, request.Namespace, seedClient)
		if err != nil {
			return fmt.Errorf("failed to reconcile storage location: %w", err)
		}

		if cbsl.Spec.Credential != nil {
			if err := syncCBSLCredentialSecret(ctx, r.masterClient, seedClient, cbsl); err != nil {
				return fmt.Errorf("failed to sync CBSL credential secret: %w", err)
			}
		}

		// fetch the updated CBSL from the cache
		if err := seedClient.Get(ctx, request.NamespacedName, seedCBSL); err != nil {
			return fmt.Errorf("failed to fetch storage location on seed cluster: %w", err)
		}

		if !equality.Semantic.DeepEqual(seedCBSL.Status, cbsl.Status) {
			seedCBSL.Status = cbsl.Status
			if err := seedClient.Status().Update(ctx, seedCBSL); err != nil {
				return fmt.Errorf("failed to update storage location status on seed cluster: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		r.recorder.Eventf(cbsl, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, cbsl *kubermaticv1.ClusterBackupStorageLocation) error {
	err := r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
		return ctrlruntimeclient.IgnoreNotFound(seedClient.Delete(ctx, cbsl))
	})
	if err != nil {
		return err
	}

	return kuberneteshelper.TryRemoveFinalizer(ctx, r.masterClient, cbsl, cleanupFinalizer)
}

func syncCBSLCredentialSecret(
	ctx context.Context,
	masterClient ctrlruntimeclient.Client,
	seedClient ctrlruntimeclient.Client,
	cbsl *kubermaticv1.ClusterBackupStorageLocation,
) error {
	cbslKey := types.NamespacedName{
		Name:      cbsl.Spec.Credential.Name,
		Namespace: cbsl.Namespace,
	}

	cbslSecret := &corev1.Secret{}
	if err := masterClient.Get(ctx, cbslKey, cbslSecret); err != nil {
		return fmt.Errorf("failed to get credential secret from master: %w", err)
	}

	secretReconcilerFactory := []reconciling.NamedSecretReconcilerFactory{
		secretReconcilerFactory(cbslSecret),
	}

	return reconciling.ReconcileSecrets(ctx, secretReconcilerFactory, cbslSecret.Namespace, seedClient)
}
