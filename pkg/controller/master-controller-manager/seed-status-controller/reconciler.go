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

package seedstatuscontroller

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// SeedKubeconfigUnavailableReason is the reason for the SeedConditionKubeconfigValid
	// in case the kubeconfig does not exist and no client can be constructed.
	SeedKubeconfigUnavailableReason = "KubeconfigUnavailable"
	// SeedKubeconfigUnavailableReason is the reason for the SeedConditionKubeconfigValid
	// in case the seed cluster was not yet prepared by the admin to be a seed (i.e. a
	// manual step is missing). This condition is not used anymore since KKP 2.21.
	SeedClusterUninitializedReason = "ClusterUninitialized"
	// SeedKubeconfigInvalidReason is the reason for the SeedConditionKubeconfigValid
	// in case no functioning client could be constructed using the given kubeconfig.
	SeedKubeconfigInvalidReason = "KubeconfigInvalid"
	// SeedKubeconfigValidReason is the reason for the SeedConditionKubeconfigValid
	// in case everything is OK.
	SeedKubeconfigValidReason = "KubeconfigValid"
)

// Reconciler watches the seed status and updates the phase, versions
// and the SeedConditionKubeconfigValid condition.
type Reconciler struct {
	ctrlruntimeclient.Client

	seedKubeconfigGetter provider.SeedKubeconfigGetter
	seedClientGetter     provider.SeedClientGetter
	log                  *zap.SugaredLogger
	recorder             events.EventRecorder
	versions             kubermatic.Versions
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := r.log.With("seed", request.Name)
	logger.Debug("Reconciling")

	seed := &kubermaticv1.Seed{}
	if err := r.Get(ctx, request.NamespacedName, seed); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get seed: %w", err)
	}

	if err := r.reconcile(ctx, logger, seed); err != nil {
		r.recorder.Eventf(seed, nil, corev1.EventTypeWarning, "ReconcilingFailed", "Reconciling", err.Error())
		return reconcile.Result{}, fmt.Errorf("failed to reconcile seed %s: %w", seed.Name, err)
	}

	// if the seed kubeconfig is invalid, try again later in case a temporary problem occurred
	if seed.Status.HasConditionValue(kubermaticv1.SeedConditionKubeconfigValid, corev1.ConditionFalse) {
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// Requeue after a minute because we want to keep the cluster count somewhat up-to-date;
	// it's not important enough to keep a permanent watch on Cluster objects in the seed cluster though.
	return reconcile.Result{RequeueAfter: 1 * time.Minute}, nil
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, seed *kubermaticv1.Seed) error {
	if seed.DeletionTimestamp != nil {
		return util.UpdateSeedStatus(ctx, r, seed, func(s *kubermaticv1.Seed) {
			s.Status.Phase = kubermaticv1.SeedTerminatingPhase
		})
	}

	return util.UpdateSeedStatus(ctx, r, seed, func(s *kubermaticv1.Seed) {
		r.updateKubeconfigValidCondition(ctx, log, s)
		r.updateVersions(ctx, log, s)
		r.updateClusters(ctx, log, s)

		// make sure to update the phase last, so it can depend on the previously updated conditions
		s.Status.Phase = getSeedPhase(seed)
	})
}

func (r *Reconciler) updateKubeconfigValidCondition(ctx context.Context, log *zap.SugaredLogger, seed *kubermaticv1.Seed) {
	cond := kubermaticv1.SeedConditionKubeconfigValid

	// check that we have a kubeconfig
	client, err := r.seedClientGetter(seed)
	if err != nil {
		log.Debugw("failed to create client for seed", zap.Error(err))
		util.SetSeedCondition(seed, cond, corev1.ConditionFalse, SeedKubeconfigUnavailableReason, err.Error())

		return
	}

	// Check that we can at least read namespaces; we choose the seed's
	// namespace because ultimately this is where we guaranteed need to
	// have permissions.
	ns := corev1.Namespace{}
	key := types.NamespacedName{Name: seed.Namespace}
	if err := client.Get(ctx, key, &ns); err != nil {
		log.Errorw("Failed to retrieve KKP namespace", "namespace", seed.Namespace, zap.Error(err))
		util.SetSeedCondition(seed, cond, corev1.ConditionFalse, SeedKubeconfigInvalidReason, err.Error())
		return
	}

	// mark kubeconfig as working
	util.SetSeedCondition(seed, cond, corev1.ConditionTrue, SeedKubeconfigValidReason, "")
}

func (r *Reconciler) updateVersions(ctx context.Context, log *zap.SugaredLogger, seed *kubermaticv1.Seed) {
	seed.Status.Versions.Kubermatic = r.versions.GitVersion

	kubeconfig, err := r.seedKubeconfigGetter(seed)
	if err != nil {
		// this error is already reflected in the KubeconfigValid condition
		return
	}

	client, err := discovery.NewDiscoveryClientForConfig(kubeconfig)
	if err != nil {
		log.Errorw("Failed to create discovery client", zap.Error(err))
		return
	}

	version, err := client.ServerVersion()
	if err != nil {
		log.Errorw("Failed to query cluster version", zap.Error(err))
		return
	}

	seed.Status.Versions.Cluster = version.String()
}

func (r *Reconciler) updateClusters(ctx context.Context, log *zap.SugaredLogger, seed *kubermaticv1.Seed) {
	client, err := r.seedClientGetter(seed)
	if err != nil {
		// this error is already reflected in the KubeconfigValid condition
		return
	}

	clusters := &metav1.PartialObjectMetadataList{}
	clusters.SetGroupVersionKind(kubermaticv1.SchemeGroupVersion.WithKind("ClusterList"))
	if err := client.List(ctx, clusters); err != nil {
		log.Errorw("Failed to count users clusters", zap.Error(err))
		return
	}

	seed.Status.Clusters = len(clusters.Items)
}

func getSeedPhase(seed *kubermaticv1.Seed) kubermaticv1.SeedPhase {
	if _, ok := seed.Annotations[common.SkipReconcilingAnnotation]; ok {
		return kubermaticv1.SeedPausedPhase
	}

	KubeconfigValid := getConditionStatus(seed, kubermaticv1.SeedConditionKubeconfigValid)
	resourcesReconciled := getConditionStatus(seed, kubermaticv1.SeedConditionResourcesReconciled)

	if KubeconfigValid == corev1.ConditionTrue && resourcesReconciled == corev1.ConditionTrue {
		return kubermaticv1.SeedHealthyPhase
	}

	if KubeconfigValid == corev1.ConditionFalse {
		return kubermaticv1.SeedInvalidPhase
	}

	// KubeconfigValid=Unknown should never happen, as this controller just set it earlier

	if resourcesReconciled == corev1.ConditionFalse {
		return kubermaticv1.SeedUnhealthyPhase
	}

	return ""
}

func getConditionStatus(seed *kubermaticv1.Seed, conditionType kubermaticv1.SeedConditionType) corev1.ConditionStatus {
	condition, exists := seed.Status.Conditions[conditionType]
	if !exists {
		return corev1.ConditionUnknown
	}

	return condition.Status
}
