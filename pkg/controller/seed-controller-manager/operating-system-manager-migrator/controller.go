/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package operatingsystemmanagermigrator

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	kubermaticpred "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	"k8c.io/machine-controller/sdk/providerconfig"
	osmresources "k8c.io/operating-system-manager/pkg/controllers/osc/resources"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kkp-operating-system-manager-migrator"
)

// UserClusterClientProvider provides functionality to get a user cluster client.
type UserClusterClientProvider interface {
	GetClient(ctx context.Context, c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

type Reconciler struct {
	ctrlruntimeclient.Client
	log *zap.SugaredLogger

	workerNameLabelSelector       labels.Selector
	workerName                    string
	recorder                      events.EventRecorder
	userClusterConnectionProvider UserClusterClientProvider
	versions                      kubermatic.Versions
}

func Add(
	mgr manager.Manager,
	userClusterConnectionProvider UserClusterClientProvider,
	log *zap.SugaredLogger,
	workerName string,
	numWorkers int,
	versions kubermatic.Versions,
) error {
	workerSelector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return fmt.Errorf("failed to build worker-name selector: %w", err)
	}

	reconciler := &Reconciler{
		Client:                        mgr.GetClient(),
		log:                           log.Named(ControllerName),
		workerNameLabelSelector:       workerSelector,
		workerName:                    workerName,
		recorder:                      mgr.GetEventRecorder(ControllerName),
		userClusterConnectionProvider: userClusterConnectionProvider,
		versions:                      versions,
	}

	_, err = builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{}, builder.WithPredicates(workerlabel.Predicate(workerName), withEventFilter())).
		Build(reconciler)

	return err
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if cluster.DeletionTimestamp != nil {
		// Cluster is queued for deletion; no action required
		r.log.Debugw("Cluster is queued for deletion; no action required", "cluster", cluster.Name)
		return reconcile.Result{}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := util.ClusterReconcileWrapper(
		ctx,
		r,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionOperatingSystemManagerMigratorControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, cluster)
		},
	)

	if result == nil || err != nil {
		result = &reconcile.Result{}
	}

	if err != nil {
		r.recorder.Eventf(cluster, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
	}

	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// To migrate the cluster, we simply have to iterate through all the machine deployments and set the annotation `k8c.io/operating-system-profile`
	// to the name of the operating system profile.
	userClusterClient, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get usercluster client: %w", err)
	}

	mdList := &clusterv1alpha1.MachineDeploymentList{}
	if err := userClusterClient.List(ctx, mdList); err != nil {
		return nil, fmt.Errorf("failed to list existing MachineDeployments: %w", err)
	}

	for _, md := range mdList.Items {
		if md.DeletionTimestamp != nil {
			// MachineDeployment is queued for deletion; no action required
			r.log.Debugw("MachineDeployment is queued for deletion; no action required", "machineDeployment", md.Name)
			continue
		}

		if md.Annotations == nil {
			md.Annotations = make(map[string]string)
		}

		if _, ok := md.Annotations[osmresources.MachineDeploymentOSPAnnotation]; !ok {
			config, err := providerconfig.GetConfig(md.Spec.Template.Spec.ProviderSpec)
			if err != nil {
				return nil, err
			}
			md.Annotations[osmresources.MachineDeploymentOSPAnnotation] = fmt.Sprintf("osp-%s", config.OperatingSystem)
		}

		if err := userClusterClient.Update(ctx, &md); err != nil {
			return nil, fmt.Errorf("failed to update MachineDeployment: %w", err)
		}
	}

	// If we have reached this point, we have successfully migrated the cluster. Update the cluster and set EnableOperatingSystemManager to nil.
	//nolint:staticcheck
	cluster.Spec.EnableOperatingSystemManager = nil
	if err := r.Update(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to update cluster: %w", err)
	}

	// The predicate will prevent us from reconciling this cluster again.
	r.recorder.Eventf(cluster, nil, corev1.EventTypeNormal, "ClusterMigrated", "Reconciling", "Cluster has been migrated from the legacy machine-controller userdata to operating-system-manager")
	return nil, nil
}

func withEventFilter() predicate.Funcs {
	return kubermaticpred.Factory(func(o ctrlruntimeclient.Object) bool {
		cluster, ok := o.(*kubermaticv1.Cluster)
		if !ok {
			return false
		}
		// Only reconcile clusters that have OSM explicitly set to false.
		//nolint:staticcheck
		return cluster.Spec.EnableOperatingSystemManager != nil && !*cluster.Spec.EnableOperatingSystemManager
	})
}
