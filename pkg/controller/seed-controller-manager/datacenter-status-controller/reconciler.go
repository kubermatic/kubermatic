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

package datacenterstatuscontroller

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v3/pkg/kubernetes"
	"k8c.io/kubermatic/v3/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler watches the seed status and updates the phase, versions
// and the SeedConditionKubeconfigValid condition.
type Reconciler struct {
	seedClient ctrlruntimeclient.Client
	log        *zap.SugaredLogger
	recorder   record.EventRecorder
	versions   kubermatic.Versions
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := r.log.With("datacenter", request.Name)
	logger.Debug("Reconciling")

	datacenter := &kubermaticv1.Datacenter{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, datacenter); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get datacenter: %w", err)
	}

	if err := r.reconcile(ctx, logger, datacenter); err != nil {
		r.recorder.Event(datacenter, corev1.EventTypeWarning, "ReconcilingFailed", err.Error())
		return reconcile.Result{}, fmt.Errorf("failed to reconcile datacenter %s: %w", datacenter.Name, err)
	}

	// Requeue after a minute because we want to keep the cluster count somewhat up-to-date;
	// it's not important enough to keep a permanent watch on Cluster objects in the seed cluster though.
	return reconcile.Result{RequeueAfter: 1 * time.Minute}, nil
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, datacenter *kubermaticv1.Datacenter) error {
	if datacenter.DeletionTimestamp != nil {
		return nil
	}

	clusters := &metav1.PartialObjectMetadataList{}
	clusters.SetGroupVersionKind(kubermaticv1.SchemeGroupVersion.WithKind("ClusterList"))

	if err := r.seedClient.List(ctx, clusters); err != nil {
		return fmt.Errorf("failed to count users clusters: %w", err)
	}

	return kuberneteshelper.UpdateDatacenterStatus(ctx, r.seedClient, datacenter, func(s *kubermaticv1.Datacenter) {
		datacenter.Status.Clusters = len(clusters.Items)
	})
}
