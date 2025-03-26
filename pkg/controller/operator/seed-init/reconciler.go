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

package seedinit

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/crd"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	crdutil "k8c.io/kubermatic/v2/pkg/util/crd"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler (re)stores all components required for running a Kubermatic
// seed cluster.
type Reconciler struct {
	log              *zap.SugaredLogger
	masterClient     ctrlruntimeclient.Client
	masterRecorder   record.EventRecorder
	seedClientGetter provider.SeedClientGetter
	workerName       string
	versions         kubermaticversion.Versions
}

// Reconcile acts upon requests and will restore the state of resources
// for the given namespace. Will return an error if any API operation
// failed, otherwise will return an empty dummy Result struct.
func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("seed", request.Name)
	log.Debug("Reconciling")

	seed := &kubermaticv1.Seed{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, seed); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	err := r.reconcile(ctx, log, seed)
	if err != nil {
		r.masterRecorder.Event(seed, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, seed *kubermaticv1.Seed) error {
	if seed.DeletionTimestamp != nil {
		return nil
	}

	if seed.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName {
		log.Debugw("Seed does not have matching label", "label", kubermaticv1.WorkerNameLabelKey)
		return nil
	}

	// To allow a step-by-step migration of seed clusters, it's possible to
	// disable the operator's reconciling logic for seeds.
	if _, ok := seed.Annotations[common.SkipReconcilingAnnotation]; ok {
		log.Debug("Seed is marked as paused, skipping reconciliation")
		return nil
	}

	// Once we've down our initial setup, we never have to do anything again,
	// as the regular seed-operator takes care of keeping things up-to-date.
	if seed.Status.IsInitialized() {
		log.Debugw("Seed already has been initialized", "condition", kubermaticv1.SeedConditionClusterInitialized)
		return nil
	}

	seedClient, err := r.seedClientGetter(seed)
	if err != nil {
		return fmt.Errorf("failed to create seed cluster client: %w", err)
	}

	seedKubeconfig, err := kubernetes.GetSeedKubeconfigSecret(ctx, r.masterClient, seed)
	if err != nil {
		return fmt.Errorf("failed to get seed kubeconfig: %w", err)
	}

	// retrieve the _undefaulted_ config (which is why this cannot use the KubermaticConfigurationGetter)
	config, err := kubernetes.GetRawKubermaticConfiguration(ctx, r.masterClient, seed.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get KubermaticConfiguration: %w", err)
	}

	if err := r.createInitialCRDs(ctx, seed, seedClient, log); err != nil {
		return fmt.Errorf("failed to create CRDs: %w", err)
	}

	ns := &corev1.Namespace{}
	ns.SetName(seed.Namespace)

	if err := r.createOnSeed(ctx, ns, seedClient, log.With("namespace", ns.Name)); err != nil {
		return fmt.Errorf("failed to create KKP namespace: %w", err)
	}

	if err := r.createOnSeed(ctx, seedKubeconfig, seedClient, log.With("seed", seed.Name)); err != nil {
		return fmt.Errorf("failed to create Seed kubeconfig resource copy: %w", err)
	}

	if err := r.createOnSeed(ctx, seed, seedClient, log.With("seed", seed.Name)); err != nil {
		return fmt.Errorf("failed to create Seed resource copy: %w", err)
	}

	if err := r.createOnSeed(ctx, config, seedClient, log.With("config", config.Name)); err != nil {
		return fmt.Errorf("failed to create KubermaticConfiguration resource copy: %w", err)
	}

	if err := r.setSeedCondition(ctx, seed); err != nil {
		return fmt.Errorf("failed to update seed status: %w", err)
	}

	return nil
}

func (r *Reconciler) setSeedCondition(ctx context.Context, seed *kubermaticv1.Seed) error {
	return util.UpdateSeedStatus(ctx, r.masterClient, seed, func(s *kubermaticv1.Seed) {
		util.SetSeedCondition(
			s,
			kubermaticv1.SeedConditionClusterInitialized,
			corev1.ConditionTrue,
			"CRDsUpdated",
			"All KKP CRDs have been installed successfully.",
		)
	})
}

var errFailedToCreateCRD = errors.New("not all CRDs could be installed")

func (r *Reconciler) createInitialCRDs(ctx context.Context, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("Installing CRDs…")

	// This does not make use of the reconciling framework, as we want to
	// only perform the initial setup and not "fight" with the seed-operator,
	// when a Seed object was updated.
	groups, err := crd.Groups()
	if err != nil {
		return fmt.Errorf("failed to list CRD groups in operator: %w", err)
	}

	var overallErr error
	for _, group := range groups {
		crds, err := crd.CRDsForGroup(group)
		if err != nil {
			return fmt.Errorf("failed to list CRDs for API group %q in the operator: %w", group, err)
		}

		for _, crdObject := range crds {
			crdLog := log.With("crd", crdObject.Name)

			if crdutil.SkipCRDOnCluster(&crdObject, crdutil.SeedCluster) {
				continue
			}

			// inject the current KKP version, so the operator and other controllers
			// can react to the changed CRDs (the KKP installer does the same when
			// updating CRDs on the master cluster)
			if crdObject.Annotations == nil {
				crdObject.Annotations = map[string]string{}
			}
			crdObject.Annotations[resources.VersionLabel] = r.versions.GitVersion

			err := r.createOnSeed(ctx, &crdObject, client, crdLog)
			if err == nil {
				crdLog.Info("Created CRD")
			} else {
				crdLog.Errorw("Failed to create CRD", zap.Error(err))
				overallErr = errFailedToCreateCRD
			}
		}
	}

	return overallErr
}

func (r *Reconciler) createOnSeed(ctx context.Context, obj ctrlruntimeclient.Object, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	objCopy := obj.DeepCopyObject().(ctrlruntimeclient.Object)
	objCopy.SetResourceVersion("")
	objCopy.SetUID("")
	objCopy.SetGeneration(0)

	// Never duplicate finalizers, as we cannot ensure that a finalizer on an object gets
	// actually processed on a different cluster, as the component that owns the finalizer
	// might only run on the master cluster.
	objCopy.SetFinalizers(nil)

	err := client.Create(ctx, objCopy)

	// An AlreadyExists error can occur on shared master/seed systems.
	return ctrlruntimeclient.IgnoreAlreadyExists(err)
}
