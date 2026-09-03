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

package projectlabelsynchronizer

import (
	"context"
	"fmt"
	"maps"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const ControllerName = "kkp-project-label-synchronizer"

type reconciler struct {
	log                     *zap.SugaredLogger
	masterClient            ctrlruntimeclient.Client
	seedClients             kuberneteshelper.SeedClientMap
	workerNameLabelSelector labels.Selector
}

// requestFromCluster returns a reconcile.Request for the project the given
// cluster belongs to, if any.
func requestFromCluster(log *zap.SugaredLogger) handler.TypedEventHandler[*kubermaticv1.Cluster, reconcile.Request] {
	return handler.TypedEnqueueRequestsFromMapFunc(func(_ context.Context, cluster *kubermaticv1.Cluster) []reconcile.Request {
		labelValue, hasLabel := cluster.Labels[kubermaticv1.ProjectIDLabelKey]
		if !hasLabel {
			log.Debugw("Cluster has no project label", "cluster", cluster.Name)
			return nil
		}

		log.Debugw("Returning reconcile request for project", kubermaticv1.ProjectIDLabelKey, labelValue)
		return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: labelValue}}}
	})
}

func Add(
	masterManager manager.Manager,
	seedManagers map[string]manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
) error {
	workerSelector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return fmt.Errorf("failed to build worker-name selector: %w", err)
	}

	log = log.Named(ControllerName)
	r := &reconciler{
		log:                     log,
		masterClient:            masterManager.GetClient(),
		seedClients:             kuberneteshelper.SeedClientMap{},
		workerNameLabelSelector: workerSelector,
	}

	bldr := builder.ControllerManagedBy(masterManager).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Project{})

	workerNamePred := workerlabel.TypedPredicate[*kubermaticv1.Cluster](workerName)
	handler := requestFromCluster(log)

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()

		bldr.WatchesRawSource(source.Kind(
			seedManager.GetCache(),
			&kubermaticv1.Cluster{},
			handler,
			workerNamePred,
		))
	}

	_, err = bldr.Build(r)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With(kubermaticv1.ProjectIDLabelKey, request.Name)
	log.Debug("Processing")

	err := r.reconcile(ctx, log, request)

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, request reconcile.Request) error {
	project := &kubermaticv1.Project{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, project); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("Didn't find project, returning")
			return nil
		}

		return fmt.Errorf("failed to get project %s: %w", request.Name, err)
	}

	if project.Status.Phase == "" {
		log.Debug("Project has no phase set in its status, skipping reconciling")
		return nil
	}

	// Labels that clusters in this project should currently inherit. This can be
	// empty, e.g. if the project has no labels (anymore); reconciling still needs
	// to happen in that case so that previously-inherited labels get removed again.
	desiredInheritedLabels := getInheritedLabels(log, project.Labels)

	workerNameLabelSelectorRequirements, _ := r.workerNameLabelSelector.Requirements()
	projectLabelRequirement, err := labels.NewRequirement(kubermaticv1.ProjectIDLabelKey, selection.Equals, []string{project.Name})
	if err != nil {
		return fmt.Errorf("failed to construct label requirement for project: %w", err)
	}

	listOpts := &ctrlruntimeclient.ListOptions{
		LabelSelector: labels.NewSelector().Add(append(workerNameLabelSelectorRequirements, *projectLabelRequirement)...),
	}

	// We use an error aggregate to make sure we return an error if we encountered one but
	// still continue processing everything we can.
	var errs []error
	for seedName, seedClient := range r.seedClients {
		log := log.With("seed", seedName)

		unfilteredClusters := &kubermaticv1.ClusterList{}
		if err := seedClient.List(ctx, unfilteredClusters, listOpts); err != nil {
			errs = append(errs, fmt.Errorf("failed to list clusters in seed %q: %w", seedName, err))
			continue
		}

		filteredClusters := r.filterClustersByProjectID(log, project.Name, unfilteredClusters)
		for _, cluster := range filteredClusters {
			log := log.With("cluster", cluster.Name)
			changed, newClusterLabels := getLabelsForCluster(log, cluster.ObjectMeta.DeepCopy().Labels, cluster.Status.InheritedLabels, desiredInheritedLabels)
			if changed {
				oldCluster := cluster.DeepCopy()
				cluster.Labels = newClusterLabels
				log.Debug("Updating labels on cluster")
				if err := seedClient.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
					errs = append(errs, fmt.Errorf("failed to update cluster %q", cluster.Name))
				}
			} else {
				log.Debug("Labels on cluster are already up to date")
			}

			if !maps.Equal(cluster.Status.InheritedLabels, desiredInheritedLabels) {
				if err := util.UpdateClusterStatus(ctx, seedClient, cluster, func(c *kubermaticv1.Cluster) {
					c.Status.InheritedLabels = desiredInheritedLabels
				}); err != nil {
					errs = append(errs, fmt.Errorf("failed to update status on cluster %q: %w", cluster.Name, err))
				}
			}
		}
	}

	return kerrors.NewAggregate(errs)
}

func (r *reconciler) filterClustersByProjectID(
	log *zap.SugaredLogger,
	projectID string,
	clusters *kubermaticv1.ClusterList,
) []*kubermaticv1.Cluster {
	var result []*kubermaticv1.Cluster

	for idx, cluster := range clusters.Items {
		log := log.With("cluster", cluster.Name)
		if val := cluster.Labels[kubermaticv1.ProjectIDLabelKey]; val != projectID {
			log.Debugw("Ignoring cluster because it has the wrong project-id", "cluster-project-id", val)
			continue
		}

		result = append(result, &clusters.Items[idx])
	}

	return result
}

// getLabelsForCluster returns the updated set of labels for a cluster, given its
// current labels, the labels it previously inherited from its project (as recorded
// in cluster.Status.InheritedLabels), and the labels it should currently inherit.
//
// Labels present in desiredInheritedLabels are set/updated on the cluster. Labels
// that were previously inherited but are no longer desired get removed again,
// unless their value was changed on the cluster in the meantime (in which case we
// leave them alone, as someone else appears to manage that label now).
func getLabelsForCluster(
	log *zap.SugaredLogger,
	clusterLabels map[string]string,
	previouslyInheritedLabels map[string]string,
	desiredInheritedLabels map[string]string,
) (changed bool, newClusterLabels map[string]string) {
	if clusterLabels == nil {
		clusterLabels = map[string]string{}
	}
	newClusterLabels = clusterLabels

	for key, desiredValue := range desiredInheritedLabels {
		if clusterLabels[key] == desiredValue {
			log.Debugf("Label %q on cluster already has value of %q, nothing to do", key, desiredValue)
			continue
		}
		log.Debugf("Setting label %q to value %q on cluster", key, desiredValue)
		clusterLabels[key] = desiredValue
		changed = true
	}

	for key, previousValue := range previouslyInheritedLabels {
		if _, stillDesired := desiredInheritedLabels[key]; stillDesired {
			continue
		}
		if clusterLabels[key] != previousValue {
			log.Debugf("Label %q was previously inherited from the project but has been changed on the cluster, leaving it alone", key)
			continue
		}
		log.Debugf("Removing label %q from cluster, no longer set on project", key)
		delete(clusterLabels, key)
		changed = true
	}

	return changed, newClusterLabels
}

// getInheritedLabels returns the subset of projectLabels that clusters of that
// project should inherit, i.e. all labels minus the protected ones.
func getInheritedLabels(log *zap.SugaredLogger, projectLabels map[string]string) map[string]string {
	inheritedLabels := make(map[string]string)

	for key, val := range projectLabels {
		if kubermaticv1.ProtectedClusterLabels.Has(key) {
			log.Infof("Project wants to set protected label %q on cluster, skipping", key)
			continue
		}
		inheritedLabels[key] = val
	}

	return inheritedLabels
}
