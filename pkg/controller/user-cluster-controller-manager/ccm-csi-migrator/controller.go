/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package ccmcsimigrator

import (
	"context"
	"fmt"
	"strconv"

	"go.uber.org/zap"

	"github.com/kubermatic/machine-controller/pkg/apis/cluster/common"
	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "ccm-csi-migrator"
)

type reconciler struct {
	log             *zap.SugaredLogger
	seedClient      ctrlruntimeclient.Client
	userClient      ctrlruntimeclient.Client
	seedRecorder    record.EventRecorder
	versions        kubermatic.Versions
	clusterName     string
	clusterIsPaused userclustercontrollermanager.IsPausedChecker
}

func Add(ctx context.Context, log *zap.SugaredLogger, seedMgr, userMgr manager.Manager, versions kubermatic.Versions, clusterName string, clusterIsPaused userclustercontrollermanager.IsPausedChecker) error {
	log = log.Named(controllerName)

	r := &reconciler{
		log:             log,
		seedClient:      seedMgr.GetClient(),
		userClient:      userMgr.GetClient(),
		seedRecorder:    seedMgr.GetEventRecorderFor(controllerName),
		versions:        versions,
		clusterName:     clusterName,
		clusterIsPaused: clusterIsPaused,
	}
	c, err := controller.New(controllerName, userMgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller %s: %v", controllerName, err)
	}

	// Watch for changes to Machines
	if err = c.Watch(
		&source.Kind{Type: &v1alpha1.Machine{}},
		handler.EnqueueRequestsFromMapFunc(func(o ctrlruntimeclient.Object) []reconcile.Request {
			return []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name: clusterName,
					},
				},
			}
		}),
	); err != nil {
		return fmt.Errorf("failed to establish watch for the Machines %v", err)
	}

	return nil
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		return reconcile.Result{}, nil
	}

	log := r.log.With("Request", request.NamespacedName.String())
	log.Debug("Reconciling")

	cluster := &v1.Cluster{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("cluster not found, returning")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %v", err)
	}

	err = r.reconcile(ctx, log, cluster)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.seedRecorder.Event(cluster, corev1.EventTypeWarning, "CCMCSIMigrationFailed", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, oldCluster *v1.Cluster) error {
	machines := &v1alpha1.MachineList{}
	if err := r.userClient.List(ctx, machines); err != nil {
		log.Debugw("error while listing machines in user oldCluster", zap.Error(err))
		return err
	}

	// check all the machines have been migrated
	var migrated = true
	for _, machine := range machines.Items {
		flag := common.GetKubeletFlags(&machine)[common.ExternalCloudProviderKubeletFlag]
		if boolFlag, err := strconv.ParseBool(flag); !boolFlag || err != nil {
			migrated = false
			break
		}
	}

	// if the cluster condition status has changed, update it accordingly
	newCluster := oldCluster.DeepCopy()
	if toPatch := r.ensureMigrationConditionStatus(migrated, newCluster); toPatch {
		if err := r.seedClient.Patch(ctx, newCluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
			return fmt.Errorf("failed to update cluster: %v", err)
		}
	}

	return nil
}

func (r *reconciler) ensureMigrationConditionStatus(migrated bool, cluster *v1.Cluster) bool {
	var (
		newStatus corev1.ConditionStatus
		reason    string
		message   string
	)

	if migrated {
		newStatus = corev1.ConditionTrue
		reason = v1.ReasonClusterCSIKubeletMigrationCompleted
		message = "external CCM/CSI migration completed"
	} else {
		newStatus = corev1.ConditionFalse
		reason = v1.ReasonClusterCCMMigrationInProgress
		message = "migrating to external CCM"
	}

	toPatch := !helper.ClusterConditionHasStatus(cluster, v1.ClusterConditionCSIKubeletMigrationCompleted, newStatus)
	if toPatch {
		helper.SetClusterCondition(
			cluster,
			r.versions,
			v1.ClusterConditionCSIKubeletMigrationCompleted,
			newStatus,
			reason,
			message,
		)
	}
	return toPatch
}
