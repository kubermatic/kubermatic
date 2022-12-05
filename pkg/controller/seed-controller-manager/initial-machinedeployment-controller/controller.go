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

package initialmachinedeployment

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	nodedeploymentmigration "k8c.io/kubermatic/v2/pkg/controller/shared/nodedeployment-migration"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kkp-initial-machinedeployment-controller"
)

// UserClusterClientProvider provides functionality to get a user cluster client.
type UserClusterClientProvider interface {
	GetClient(ctx context.Context, c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

type Reconciler struct {
	ctrlruntimeclient.Client

	workerName                    string
	recorder                      record.EventRecorder
	seedGetter                    provider.SeedGetter
	userClusterConnectionProvider UserClusterClientProvider
	log                           *zap.SugaredLogger
	versions                      kubermatic.Versions
}

// Add creates a new initialmachinedeployment controller.
func Add(ctx context.Context, mgr manager.Manager, numWorkers int, workerName string, seedGetter provider.SeedGetter, userClusterConnectionProvider UserClusterClientProvider, log *zap.SugaredLogger, versions kubermatic.Versions) error {
	reconciler := &Reconciler{
		Client: mgr.GetClient(),

		workerName:                    workerName,
		recorder:                      mgr.GetEventRecorderFor(ControllerName),
		seedGetter:                    seedGetter,
		userClusterConnectionProvider: userClusterConnectionProvider,
		log:                           log,
		versions:                      versions,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}, predicateutil.ByAnnotation(kubermaticv1.InitialMachineDeploymentRequestAnnotation, "", false)); err != nil {
		return fmt.Errorf("failed to create watch: %w", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.Name)
	log.Debug("Reconciling")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionMachineDeploymentControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, log, cluster)
		},
	)
	if err != nil {
		log.Errorw("Failed to reconcile cluster", zap.Error(err))
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// there is no annotation anymore
	request := cluster.Annotations[kubermaticv1.InitialMachineDeploymentRequestAnnotation]
	if request == "" {
		return nil, nil
	}

	// never create new machines in cluster that are in deletion
	if cluster.DeletionTimestamp != nil {
		log.Debug("cluster is in deletion, not reconciling any further")
		return nil, nil
	}

	// If cluster is not healthy yet there is nothing to do.
	// If it gets healthy we'll get notified by the event. No need to requeue.
	if !cluster.Status.ExtendedHealth.AllHealthy() {
		log.Debug("cluster not healthy")
		return nil, nil
	}

	// machine-controller webhook health is not part of the ClusterHealth, but
	// for this operation we need to ensure that the webhook is up and running
	key := types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: resources.MachineControllerWebhookDeploymentName}
	status, err := resources.HealthyDeployment(ctx, r, key, -1)
	if err != nil {
		return nil, fmt.Errorf("failed to determine machine-controller webhook's health: %w", err)
	}

	if status != kubermaticv1.HealthStatusUp {
		log.Debug("machine-controller webhook is not ready")
		return nil, nil
	}

	datacenter, err := r.getTargetDatacenter(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get target datacenter: %w", err)
	}

	machineDeployment, err := r.parseMachineDeployment(cluster, datacenter, request)
	if err != nil {
		if removeErr := r.removeAnnotation(ctx, cluster); removeErr != nil {
			return nil, fmt.Errorf("failed to remove invalid (%v) initial MachineDeployment annotation: %w", err, removeErr)
		}

		return nil, err
	}

	userClusterClient, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get user cluster client: %w", err)
	}

	if err := r.createInitialMachineDeployment(ctx, log, machineDeployment, cluster, datacenter, userClusterClient); err != nil {
		return nil, fmt.Errorf("failed to create initial MachineDeployment: %w", err)
	}

	if err := r.removeAnnotation(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to remove initial MachineDeployment annotation: %w", err)
	}

	return nil, nil
}

func (r *Reconciler) parseMachineDeployment(cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.Datacenter, request string) (*clusterv1alpha1.MachineDeployment, error) {
	machineDeployment, _, err := nodedeploymentmigration.ParseNodeOrMachineDeployment(cluster, datacenter, request)
	if err != nil {
		return nil, err
	}

	if err := ValidateMachineDeployment(machineDeployment, cluster.Spec.Version.Semver()); err != nil {
		return nil, fmt.Errorf("initial MachineDeployment is invalid: %w", err)
	}

	return machineDeployment, nil
}

// createInitialMachineDeployment takes the MD from the annotation and applies the current system
// configuration, additional labels etc. to it.
func (r *Reconciler) createInitialMachineDeployment(ctx context.Context, log *zap.SugaredLogger, machineDeployment *clusterv1alpha1.MachineDeployment, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.Datacenter, client ctrlruntimeclient.Client) error {
	sshKeys, err := r.getSSHKeys(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get SSH keys: %w", err)
	}

	machineDeployment, err = CompleteMachineDeployment(machineDeployment, cluster, datacenter, sshKeys)
	if err != nil {
		return fmt.Errorf("failed to assemble MachineDeployment: %w", err)
	}

	err = client.Create(ctx, machineDeployment)
	if err != nil {
		// in case we created the MD before but then failed to cleanup the Cluster resource's
		// annotations, we can silently ignore AlreadyExists errors here and then re-try removing
		// the annotation afterwards
		return ctrlruntimeclient.IgnoreAlreadyExists(err)
	}

	log.Info("Created initial MachineDeployment")
	r.recorder.Eventf(cluster, corev1.EventTypeNormal, "MachineDeploymentCreated", "Initial MachineDeployment %s has been created", machineDeployment.Name)

	return nil
}

func (r *Reconciler) getTargetDatacenter(cluster *kubermaticv1.Cluster) (*kubermaticv1.Datacenter, error) {
	seed, err := r.seedGetter()
	if err != nil {
		return nil, fmt.Errorf("failed to get current Seed cluster: %w", err)
	}

	for key, dc := range seed.Spec.Datacenters {
		if key == cluster.Spec.Cloud.DatacenterName {
			return &dc, nil
		}
	}

	return nil, fmt.Errorf("there is no datacenter named %q in Seed %q", cluster.Spec.Cloud.DatacenterName, seed.Name)
}

func (r *Reconciler) getSSHKeys(ctx context.Context, cluster *kubermaticv1.Cluster) ([]*kubermaticv1.UserSSHKey, error) {
	projectID := cluster.Labels[kubermaticv1.ProjectIDLabelKey]
	if projectID == "" {
		return nil, fmt.Errorf("cluster does not have a %q label", kubermaticv1.ProjectIDLabelKey)
	}

	allKeys := &kubermaticv1.UserSSHKeyList{}
	if err := r.List(ctx, allKeys); err != nil {
		return nil, fmt.Errorf("failed to list UserSSHKeys: %w", err)
	}

	keys := []*kubermaticv1.UserSSHKey{}
	for i, key := range allKeys.Items {
		if key.Spec.Project != projectID {
			continue
		}

		if !sets.NewString(key.Spec.Clusters...).Has(cluster.Name) {
			continue
		}

		keys = append(keys, &allKeys.Items[i])
	}

	return keys, nil
}

func (r *Reconciler) removeAnnotation(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	oldCluster := cluster.DeepCopy()
	delete(cluster.Annotations, kubermaticv1.InitialMachineDeploymentRequestAnnotation)
	return r.Patch(ctx, cluster, ctrlruntimeclient.MergeFromWithOptions(oldCluster, ctrlruntimeclient.MergeFromWithOptimisticLock{}))
}
