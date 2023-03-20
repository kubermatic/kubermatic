/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package nodeversioncontroller

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	"k8c.io/api/v2/pkg/semver"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "kkp-node-version-controller"
)

// reconciler watches nodes inside the user cluster and
// updates the oldestNodeVersion field in the cluster status.
type reconciler struct {
	log               *zap.SugaredLogger
	seedClient        ctrlruntimeclient.Client
	userClusterClient ctrlruntimeclient.Client
	clusterName       string
}

func Add(ctx context.Context, log *zap.SugaredLogger, seedMgr, userMgr manager.Manager, clusterName string) error {
	r := &reconciler{
		log:               log,
		seedClient:        seedMgr.GetClient(),
		userClusterClient: userMgr.GetClient(),
		clusterName:       clusterName,
	}
	c, err := controller.New(controllerName, userMgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &corev1.Node{}}, controllerutil.EnqueueConst("")); err != nil {
		return fmt.Errorf("failed to establish watch for nodes: %w", err)
	}

	return nil
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.log.Debug("Reconciling")

	err := r.reconcile(ctx)
	if err != nil {
		r.log.Errorw("Reconciling failed", zap.Error(err))
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context) error {
	nodes := &corev1.NodeList{}
	if err := r.userClusterClient.List(ctx, nodes); err != nil {
		return fmt.Errorf("failed to get nodes: %w", err)
	}

	var oldestVersion *semver.Semver
	for _, node := range nodes.Items {
		parsed, err := semver.NewSemver(node.Status.NodeInfo.KubeletVersion)
		if err != nil {
			return fmt.Errorf("failed to get nodes: %w", err)
		}

		if oldestVersion == nil || parsed.LessThan(oldestVersion) {
			oldestVersion = parsed
		}
	}

	// parse the version again to reliably get rid of the "v" prefix, otherwise
	// this could lead to reconcile loops; our semver types are weird when marshalled.
	if oldestVersion != nil {
		parsed, err := semver.NewSemver(oldestVersion.String())
		if err != nil {
			return fmt.Errorf("failed to parse version %v: %w", oldestVersion, err)
		}

		oldestVersion = parsed
	}

	cluster := &kubermaticv1.Cluster{}
	if err := r.seedClient.Get(ctx, types.NamespacedName{Name: r.clusterName}, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("failed to get cluster %q: %w", r.clusterName, err)
	}

	if cluster.Status.Versions.OldestNodeVersion != oldestVersion {
		r.log.Infow("Determined new oldest node version", "version", oldestVersion)

		return kuberneteshelper.UpdateClusterStatus(ctx, r.seedClient, cluster, func(cluster *kubermaticv1.Cluster) {
			cluster.Status.Versions.OldestNodeVersion = oldestVersion
		})
	}

	return nil
}
