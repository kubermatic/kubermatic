/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package etcdrestore

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/minio/minio-go"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
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

const (
	ControllerName = "kubermatic_etcd_restore_controller"

	// FinishRestoreFinalizer indicates that the restore is rebuilding the etcd statefulset
	FinishRestoreFinalizer = "kubermatic.io/finish-restore"

	// ActiveRestoreAnnotationName is the cluster annotation that records the EtcdRestore resource that's currently
	// being restored into the cluster, if any. This is also used for mutual exclusion, i.e. to make sure that not
	// more than one EtcdRestore resource is active for the cluster at the same time.
	ActiveRestoreAnnotationName = "kubermatic.io/active-restore"
)

// Reconciler stores necessary components that are required to restore etcd backups
type Reconciler struct {
	log        *zap.SugaredLogger
	workerName string
	ctrlruntimeclient.Client
	recorder   record.EventRecorder
	versions   kubermatic.Versions
	seedGetter provider.SeedGetter
}

// Add creates a new etcd restore controller that is responsible for
// managing cluster etcd restores
func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	seedGetter provider.SeedGetter,
) error {
	log = log.Named(ControllerName)
	client := mgr.GetClient()

	reconciler := &Reconciler{
		log:        log,
		Client:     client,
		workerName: workerName,
		recorder:   mgr.GetEventRecorderFor(ControllerName),
		versions:   versions,
		seedGetter: seedGetter,
	}

	ctrlOptions := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	incompleteRestorePredicates := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			restore := e.Object.(*kubermaticv1.EtcdRestore)
			return restore.Status.Phase != kubermaticv1.EtcdRestorePhaseCompleted
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			restore := e.ObjectNew.(*kubermaticv1.EtcdRestore)
			return restore.Status.Phase != kubermaticv1.EtcdRestorePhaseCompleted
		},
	}

	return c.Watch(&source.Kind{Type: &kubermaticv1.EtcdRestore{}}, &handler.EnqueueRequestForObject{}, incompleteRestorePredicates)
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	restore := &kubermaticv1.EtcdRestore{}
	if err := r.Get(ctx, request.NamespacedName, restore); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: restore.Spec.Cluster.Name}, cluster); err != nil {
		return reconcile.Result{}, err
	}

	log = r.log.With("cluster", cluster.Name, "restore", restore.Name)

	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName {
		return reconcile.Result{}, nil
	}

	seed, err := r.seedGetter()
	if err != nil {
		return reconcile.Result{}, err
	}

	result, err := r.reconcile(ctx, log, restore, cluster, seed)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(restore, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError",
			"failed to reconcile etcd restore %q: %v", restore.Name, err)
	}

	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, restore *kubermaticv1.EtcdRestore, cluster *kubermaticv1.Cluster,
	seed *kubermaticv1.Seed) (*reconcile.Result, error) {
	if restore.Status.Phase == kubermaticv1.EtcdRestorePhaseCompleted {
		return nil, nil
	}

	log.Infof("performing etcd restore from backup %v", restore.Spec.BackupName)

	if restore.DeletionTimestamp == nil {
		if err := r.updateRestore(ctx, restore, func(restore *kubermaticv1.EtcdRestore) {
			kuberneteshelper.AddFinalizer(restore, FinishRestoreFinalizer)
		}); err != nil {
			return nil, fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	var destination *kubermaticv1.BackupDestination
	if restore.Spec.Destination != "" {
		if seed.Spec.EtcdBackupRestore == nil {
			return nil, errors.Errorf("can't find backup restore destination %q in Seed %q", restore.Spec.Destination, seed.Name)
		}
		var ok bool
		destination, ok = seed.Spec.EtcdBackupRestore.Destinations[restore.Spec.Destination]
		if !ok {
			return nil, errors.Errorf("can't find backup restore destination %q in Seed %q", restore.Spec.Destination, seed.Name)
		}
		if destination.Credentials == nil {
			return nil, errors.Errorf("credentials not set for backup destination %q in Seed %q", restore.Spec.Destination, seed.Name)
		}
	}

	// check that the backup to restore from exists and is accessible
	s3Client, bucketName, err := resources.GetEtcdRestoreS3Client(ctx, restore, true, r.Client, cluster, destination)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain S3 client: %w", err)
	}

	objectName := fmt.Sprintf("%s-%s", cluster.GetName(), restore.Spec.BackupName)
	if _, err := s3Client.StatObject(bucketName, objectName, minio.StatObjectOptions{}); err != nil {
		return nil, fmt.Errorf("could not access backup object %s: %w", objectName, err)
	}

	// before proceeding, ensure restore's namespace/name is stored in the ActiveRestoreAnnotationName cluster annotation
	// unless some other restore is already stored there
	thisRestore := fmt.Sprintf("%s/%s", restore.Namespace, restore.Name)
	activeRestore := cluster.Annotations[ActiveRestoreAnnotationName]
	if activeRestore != "" {
		if activeRestore != thisRestore {
			return &reconcile.Result{RequeueAfter: 1 * time.Minute}, nil
		}
	} else {
		if cluster.Annotations == nil {
			cluster.Annotations = map[string]string{}
		}
		cluster.Annotations[ActiveRestoreAnnotationName] = thisRestore
		if err := r.Client.Update(ctx, cluster); err != nil {
			if kerrors.IsConflict(err) {
				return &reconcile.Result{RequeueAfter: 1 * time.Minute}, nil
			}
			return nil, fmt.Errorf("error updating cluster active restore annotation: %w", err)
		}
	}

	if restore.Status.Phase == kubermaticv1.EtcdRestorePhaseStsRebuilding {
		return r.rebuildEtcdStatefulset(ctx, log, restore, cluster)
	}

	// pause cluster
	if err := r.updateCluster(ctx, cluster, func(cluster *kubermaticv1.Cluster) {
		cluster.Spec.Pause = true
	}); err != nil {
		return nil, fmt.Errorf("failed to pause cluster: %w", err)
	}

	if err := r.updateRestore(ctx, restore, func(restore *kubermaticv1.EtcdRestore) {
		restore.Status.Phase = kubermaticv1.EtcdRestorePhaseStarted
	}); err != nil {
		return nil, fmt.Errorf("failed to set EtcdRestore started phase: %w", err)
	}

	// delete etcd sts
	sts := &v1.StatefulSet{}
	err = r.Get(ctx, types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: resources.EtcdStatefulSetName}, sts)
	if err == nil {
		if err := r.Delete(ctx, sts); err != nil && !kerrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to delete etcd statefulset: %w", err)
		}
	} else if !kerrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get etcd statefulset: %w", err)
	}

	// delete PVCs
	pvcSelector, err := labels.Parse(fmt.Sprintf("%s=%s", resources.AppLabelKey, resources.EtcdStatefulSetName))
	if err != nil {
		return nil, fmt.Errorf("software bug: failed to parse etcd pvc selector: %w", err)
	}

	pvcs := &corev1.PersistentVolumeClaimList{}
	if err := r.List(ctx, pvcs, &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName, LabelSelector: pvcSelector}); err != nil {
		return nil, fmt.Errorf("failed to list pvcs (%v): %w", pvcSelector.String(), err)
	}

	for _, pvc := range pvcs.Items {
		deletePropagationForeground := metav1.DeletePropagationForeground
		delOpts := &ctrlruntimeclient.DeleteOptions{
			PropagationPolicy: &deletePropagationForeground,
		}
		if err := r.Delete(ctx, &pvc, delOpts); err != nil {
			return nil, fmt.Errorf("failed to delete pvc %v: %w", pvc.GetName(), err)
		}
	}

	if len(pvcs.Items) > 0 {
		// some PVCs still present -- wait
		return &reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if err := r.updateRestore(ctx, restore, func(restore *kubermaticv1.EtcdRestore) {
		restore.Status.Phase = kubermaticv1.EtcdRestorePhaseStsRebuilding
	}); err != nil {
		return nil, fmt.Errorf("failed to proceed to sts rebuilding phase: %w", err)
	}

	return r.rebuildEtcdStatefulset(ctx, log, restore, cluster)
}

func (r *Reconciler) rebuildEtcdStatefulset(ctx context.Context, log *zap.SugaredLogger, restore *kubermaticv1.EtcdRestore, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	log.Infof("rebuildEtcdStatefulset")

	if cluster.Spec.Pause {
		if err := r.updateCluster(ctx, cluster, func(cluster *kubermaticv1.Cluster) {
			kubermaticv1helper.SetClusterCondition(
				cluster,
				r.versions,
				kubermaticv1.ClusterConditionEtcdClusterInitialized,
				corev1.ConditionFalse,
				"",
				fmt.Sprintf("Etcd Cluster is being restored from backup %v", restore.Spec.BackupName),
			)
			cluster.Spec.Pause = false
		}); err != nil {
			return nil, fmt.Errorf("failed to reset etcd initialized status and unpause cluster: %w", err)
		}
	}

	// wait until cluster controller has recreated the etcd cluster and etcd becomes healthy again
	if !cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionEtcdClusterInitialized, corev1.ConditionTrue) {
		return &reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if err := r.updateCluster(ctx, cluster, func(cluster *kubermaticv1.Cluster) {
		delete(cluster.Annotations, ActiveRestoreAnnotationName)
	}); err != nil {
		return nil, fmt.Errorf("failed to clear cluster active restore annotation: %w", err)
	}

	if err := r.updateRestore(ctx, restore, func(restore *kubermaticv1.EtcdRestore) {
		restore.Status.Phase = kubermaticv1.EtcdRestorePhaseCompleted
		kuberneteshelper.RemoveFinalizer(restore, FinishRestoreFinalizer)
	}); err != nil {
		return nil, fmt.Errorf("failed to mark restore completed: %w", err)
	}

	return nil, nil
}

func (r *Reconciler) updateCluster(ctx context.Context, cluster *kubermaticv1.Cluster, modify func(*kubermaticv1.Cluster)) error {
	oldCluster := cluster.DeepCopy()
	modify(cluster)
	if reflect.DeepEqual(oldCluster, cluster) {
		return nil
	}
	return r.Client.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}

func (r *Reconciler) updateRestore(ctx context.Context, restore *kubermaticv1.EtcdRestore, modify func(*kubermaticv1.EtcdRestore)) error {
	oldRestore := restore.DeepCopy()
	modify(restore)
	if reflect.DeepEqual(oldRestore, restore) {
		return nil
	}
	return r.Client.Patch(ctx, restore, ctrlruntimeclient.MergeFrom(oldRestore))
}
