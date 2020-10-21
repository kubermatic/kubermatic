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
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	v1 "k8c.io/kubermatic/v2/pkg/api/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	machineresource "k8c.io/kubermatic/v2/pkg/resources/machine"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kubermatic_initialmachinedeployment_controller"
)

// UserClusterClientProvider provides functionality to get a user cluster client
type UserClusterClientProvider interface {
	GetClient(c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

type Reconciler struct {
	ctrlruntimeclient.Client

	ctx                           context.Context
	workerName                    string
	recorder                      record.EventRecorder
	seedGetter                    provider.SeedGetter
	userClusterConnectionProvider UserClusterClientProvider
	log                           *zap.SugaredLogger
	versions                      kubermatic.Versions
}

// Add creates a new initialmachinedeployment controller
func Add(ctx context.Context, mgr manager.Manager, numWorkers int, workerName string, seedGetter provider.SeedGetter, userClusterConnectionProvider UserClusterClientProvider, log *zap.SugaredLogger, versions kubermatic.Versions) error {
	reconciler := &Reconciler{
		Client: mgr.GetClient(),

		ctx:                           ctx,
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
		return fmt.Errorf("failed to create controller: %v", err)
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to create watch: %v", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(r.ctx, request.NamespacedName, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		r.ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionMachineDeploymentControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(cluster)
		},
	)
	if err != nil {
		r.log.Errorw("Failed to reconcile cluster", "cluster", cluster.Name, zap.Error(err))
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *Reconciler) reconcile(cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// If cluster is not healthy yet there is nothing to do.
	// If it gets healthy we'll get notified by the event. No need to requeue.
	if !cluster.Status.ExtendedHealth.AllHealthy() {
		r.log.Info("cluster not healthy")
		return nil, nil
	}

	nodeDeployment, err := r.parseNodeDeployment(cluster)
	if err != nil {
		if removeErr := r.removeAnnotation(cluster); removeErr != nil {
			return nil, fmt.Errorf("failed to remove invalid (%v) initial MachineDeployment annotation: %v", err, removeErr)
		}

		return nil, err
	}

	if nodeDeployment == nil {
		return nil, nil
	}

	userClusterClient, err := r.userClusterConnectionProvider.GetClient(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get user cluster client: %v", err)
	}

	if err := r.createInitialMachineDeployment(nodeDeployment, cluster, userClusterClient); err != nil {
		return nil, fmt.Errorf("failed to create initial MachineDeployment: %v", err)
	}

	if err := r.removeAnnotation(cluster); err != nil {
		return nil, fmt.Errorf("failed to remove initial MachineDeployment annotation: %v", err)
	}

	return nil, nil
}

func (r *Reconciler) parseNodeDeployment(cluster *kubermaticv1.Cluster) (*v1.NodeDeployment, error) {
	request := cluster.Annotations[v1.InitialMachineDeploymentRequestAnnotation]
	if request == "" {
		return nil, nil
	}

	var nodeDeployment *v1.NodeDeployment
	if err := json.Unmarshal([]byte(request), &nodeDeployment); err != nil {
		return nil, fmt.Errorf("cannot unmarshal initial MachineDeployment request: %v", err)
	}

	nodeDeployment, err := machineresource.Validate(nodeDeployment, cluster.Spec.Version.Semver())
	if err != nil {
		return nil, fmt.Errorf("initial node deployment is not valid: %v", err)
	}

	return nodeDeployment, nil
}

func (r *Reconciler) createInitialMachineDeployment(nodeDeployment *v1.NodeDeployment, cluster *kubermaticv1.Cluster, client ctrlruntimeclient.Client) error {
	datacenter, err := r.getTargetDatacenter(cluster)
	if err != nil {
		return fmt.Errorf("failed to get target datacenter: %v", err)
	}

	sshKeys, err := r.getSSHKeys(cluster)
	if err != nil {
		return fmt.Errorf("failed to get SSH keys: %v", err)
	}

	data := common.CredentialsData{
		Ctx:               r.ctx,
		KubermaticCluster: cluster,
		Client:            r,
	}

	machineDeployment, err := machineresource.Deployment(cluster, nodeDeployment, datacenter, sshKeys, data)
	if err != nil {
		return fmt.Errorf("failed to assemble MachineDeployment: %v", err)
	}

	err = client.Create(r.ctx, machineDeployment)
	if err != nil {
		// in case we created the MD before but then failed to cleanup the Cluster resource's
		// annotations, we can silently ignore AlreadyExists errors here and then re-try removing
		// the annotation afterwards
		if kerrors.IsAlreadyExists(err) {
			return nil
		}

		return err
	}

	r.recorder.Eventf(cluster, corev1.EventTypeNormal, "NodeDeploymentCreated", "Initial MachineDeployment %s has been created", machineDeployment.Name)

	return nil
}

func (r *Reconciler) getTargetDatacenter(cluster *kubermaticv1.Cluster) (*kubermaticv1.Datacenter, error) {
	seed, err := r.seedGetter()
	if err != nil {
		return nil, fmt.Errorf("failed to get current Seed cluster: %v", err)
	}

	for key, dc := range seed.Spec.Datacenters {
		if key == cluster.Spec.Cloud.DatacenterName {
			return &dc, nil
		}
	}

	return nil, fmt.Errorf("there is no datacenter named %q in Seed %q", cluster.Spec.Cloud.DatacenterName, seed.Name)
}

func (r *Reconciler) getSSHKeys(cluster *kubermaticv1.Cluster) ([]*kubermaticv1.UserSSHKey, error) {
	var keys []*kubermaticv1.UserSSHKey

	projectID := cluster.Labels[kubermaticv1.ProjectIDLabelKey]
	if projectID == "" {
		return nil, fmt.Errorf("cluster does not have a %q label", kubermaticv1.ProjectIDLabelKey)
	}

	project := &kubermaticv1.Project{}
	if err := r.Get(r.ctx, ctrlruntimeclient.ObjectKey{Name: projectID}, project); err != nil {
		return nil, fmt.Errorf("failed to get owning project %q: %v", projectID, err)
	}

	sshKeyProvider := kubernetes.NewSSHKeyProvider(nil, r)
	keys, err := sshKeyProvider.List(project, &provider.SSHKeyListOptions{ClusterName: cluster.Name})
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH keys: %v", err)
	}

	return keys, nil
}

func (r *Reconciler) removeAnnotation(cluster *kubermaticv1.Cluster) error {
	oldCluster := cluster.DeepCopy()
	delete(cluster.Annotations, v1.InitialMachineDeploymentRequestAnnotation)
	return r.Patch(r.ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}
