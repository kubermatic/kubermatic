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
	"strings"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/machine-controller/sdk/apis/cluster/common"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	controllerName = "kkp-ccm-csi-migrator"
)

type reconciler struct {
	log             *zap.SugaredLogger
	seedClient      ctrlruntimeclient.Client
	userClient      ctrlruntimeclient.Client
	seedRecorder    events.EventRecorder
	versions        kubermatic.Versions
	clusterName     string
	clusterIsPaused userclustercontrollermanager.IsPausedChecker
}

func Add(log *zap.SugaredLogger, seedMgr, userMgr manager.Manager, versions kubermatic.Versions, clusterName string, clusterIsPaused userclustercontrollermanager.IsPausedChecker) error {
	log = log.Named(controllerName)

	r := &reconciler{
		log:             log,
		seedClient:      seedMgr.GetClient(),
		userClient:      userMgr.GetClient(),
		seedRecorder:    seedMgr.GetEventRecorder(controllerName),
		versions:        versions,
		clusterName:     clusterName,
		clusterIsPaused: clusterIsPaused,
	}

	_, err := builder.ControllerManagedBy(userMgr).
		Named(controllerName).
		Watches(&clusterv1alpha1.Machine{}, controllerutil.EnqueueConst("")).
		Build(r)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.log.Debug("Reconciling")

	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		return reconcile.Result{}, nil
	}

	cluster := &kubermaticv1.Cluster{}
	if err := r.seedClient.Get(ctx, types.NamespacedName{Name: r.clusterName}, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			r.log.Debug("cluster not found, returning")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}

	err = r.reconcile(ctx, cluster)
	if err != nil {
		r.seedRecorder.Eventf(cluster, nil, corev1.EventTypeWarning, "CCMCSIMigrationFailed", "Reconciling", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	machines := &clusterv1alpha1.MachineList{}
	if err := r.userClient.List(ctx, machines); err != nil {
		return fmt.Errorf("failed to list machines in user cluster: %w", err)
	}

	// check all the machines have been migrated
	var migrated = true
	for _, machine := range machines.Items {
		flag := getKubeletFlags(machine.Annotations)[common.ExternalCloudProviderKubeletFlag]
		if boolFlag, err := strconv.ParseBool(flag); !boolFlag || err != nil {
			migrated = false
			break
		}
	}

	// update cluster condition
	if err := controllerutil.UpdateClusterStatus(ctx, r.seedClient, cluster, func(c *kubermaticv1.Cluster) {
		conditionType := kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted
		newStatus := corev1.ConditionFalse
		reason := kubermaticv1.ReasonClusterCCMMigrationInProgress
		message := "migrating to external CCM"

		if migrated {
			newStatus = corev1.ConditionTrue
			reason = kubermaticv1.ReasonClusterCSIKubeletMigrationCompleted
			message = "external CCM/CSI migration completed"
		}

		controllerutil.SetClusterCondition(c, r.versions, conditionType, newStatus, reason, message)
	}); err != nil {
		return fmt.Errorf("failed to update cluster: %w", err)
	}

	return nil
}

// getKubeletFlags was removed too early from machine-controller in
// https://github.com/kubermatic/machine-controller/pull/1789, but is still needed here.
func getKubeletFlags(annotations map[string]string) map[common.KubeletFlags]string {
	result := map[common.KubeletFlags]string{}
	for name, value := range annotations {
		if strings.HasPrefix(name, common.KubeletFlagsGroupAnnotationPrefixV1) {
			nameFlagValue := strings.SplitN(name, "/", 2)
			if len(nameFlagValue) != 2 {
				continue
			}
			result[common.KubeletFlags(nameFlagValue[1])] = value
		}
	}
	return result
}
