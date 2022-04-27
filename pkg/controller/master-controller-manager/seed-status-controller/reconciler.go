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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// SeedKubeconfigUnavailableReason is the reason for the SeedConditionValidKubeconfig
	// in case the kubeconfig does not exist and not client can be constructed.
	SeedKubeconfigUnavailableReason = "KubeconfigUnavailable"
	// SeedKubeconfigUnavailableReason is the reason for the SeedConditionValidKubeconfig
	// in case the seed cluster was not yet prepared by the admin to be a seed (i.e. a
	// manual step is missing).
	SeedClusterUninitializedReason = "ClusterUninitialized"
	// SeedKubeconfigInvalidReason is the reason for the SeedConditionValidKubeconfig
	// in case the KKP namespace could not be queried for (i.e. an error other than NotFound).
	// If a NotFound error occured instead, SeedClusterUninitializedReason is the reason
	// on the condition.
	SeedKubeconfigInvalidReason = "KubeconfigInvalid"
	// SeedKubeconfigValidReason is the reason for the SeedConditionValidKubeconfig
	// in case everything is OK.
	SeedKubeconfigValidReason = "KubeconfigValid"
)

// Reconciler watches the seed status and updates the phase, versions
// and the SeedConditionValidKubeconfig condition.
type Reconciler struct {
	ctrlruntimeclient.Client

	seedKubeconfigGetter provider.SeedKubeconfigGetter
	seedClientGetter     provider.SeedClientGetter
	log                  *zap.SugaredLogger
	recorder             record.EventRecorder
	versions             kubermatic.Versions
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := r.log.With("seed", request.Name)
	logger.Debug("Reconciling")

	seed := &kubermaticv1.Seed{}
	if err := r.Get(ctx, request.NamespacedName, seed); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get seed: %w", err)
	}

	if err := r.reconcile(ctx, logger, seed); err != nil {
		r.recorder.Event(seed, corev1.EventTypeWarning, "ReconcilingFailed", err.Error())
		return reconcile.Result{}, fmt.Errorf("failed to reconcile seed %s: %w", seed.Name, err)
	}

	// if the seed kubeconfig is invalid, try again later in case a temporary problem occurred
	if seed.Status.HasConditionValue(kubermaticv1.SeedConditionValidKubeconfig, corev1.ConditionFalse) {
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, seed *kubermaticv1.Seed) error {
	if seed.DeletionTimestamp != nil {
		return kubermaticv1helper.UpdateSeedStatus(ctx, r, seed, func(s *kubermaticv1.Seed) {
			s.Status.Phase = kubermaticv1.SeedTerminatingPhase
		})
	}

	return kubermaticv1helper.UpdateSeedStatus(ctx, r, seed, func(s *kubermaticv1.Seed) {
		r.updateValidKubeconfigCondition(ctx, log, s)
		r.updateVersions(ctx, log, s)

		// make sure to update the phase last, so it can depend on the previously updated conditions
		s.Status.Phase = getSeedPhase(seed)
	})
}

func (r *Reconciler) updateValidKubeconfigCondition(ctx context.Context, log *zap.SugaredLogger, seed *kubermaticv1.Seed) {
	cond := kubermaticv1.SeedConditionValidKubeconfig

	// check that we have a kubeconfig
	client, err := r.seedClientGetter(seed)
	if err != nil {
		log.Debugw("failed to create client for seed", zap.Error(err))
		kubermaticv1helper.SetSeedCondition(seed, cond, corev1.ConditionFalse, SeedKubeconfigUnavailableReason, err.Error())

		return
	}

	// check that the kubermatic namespace exists, the one thing admins need to do to
	// mark a cluster as intended for a seed
	ns := corev1.Namespace{}
	key := types.NamespacedName{Name: seed.Namespace}
	if err := client.Get(ctx, key, &ns); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debugw("KKP namespace does not exist", "namespace", seed.Namespace)
			reason := fmt.Sprintf("namespace %q does not exist and must be created manually", seed.Namespace)
			kubermaticv1helper.SetSeedCondition(seed, cond, corev1.ConditionFalse, SeedClusterUninitializedReason, reason)
		} else {
			log.Debugw("failed to check for KKP namespace", zap.Error(err))
			kubermaticv1helper.SetSeedCondition(seed, cond, corev1.ConditionFalse, SeedKubeconfigInvalidReason, err.Error())
		}

		return
	}

	// mark kubeconfig as working
	kubermaticv1helper.SetSeedCondition(seed, cond, corev1.ConditionTrue, SeedKubeconfigValidReason, "")
}

func (r *Reconciler) updateVersions(ctx context.Context, log *zap.SugaredLogger, seed *kubermaticv1.Seed) {
	seed.Status.Versions.Kubermatic = r.versions.Kubermatic

	kubeconfig, err := r.seedKubeconfigGetter(seed)
	if err != nil {
		// this error is already reflected in the ValidKubeconfig condition
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

func getSeedPhase(seed *kubermaticv1.Seed) kubermaticv1.SeedPhase {
	validKubeconfig := getConditionStatus(seed, kubermaticv1.SeedConditionValidKubeconfig)
	resourcesReconciled := getConditionStatus(seed, kubermaticv1.SeedConditionResourcesReconciled)

	if validKubeconfig == corev1.ConditionTrue && resourcesReconciled == corev1.ConditionTrue {
		return kubermaticv1.SeedHealthyPhase
	}

	if validKubeconfig == corev1.ConditionFalse {
		return kubermaticv1.SeedInvalidPhase
	}

	// validKubeconfig=Unknown should never happen, as this controller just set it earlier

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
