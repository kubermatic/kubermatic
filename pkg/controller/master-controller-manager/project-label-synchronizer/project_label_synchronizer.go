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

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
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
func requestFromCluster(log *zap.SugaredLogger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(mo ctrlruntimeclient.Object) []reconcile.Request {
		cluster, ok := mo.(*kubermaticv1.Cluster)
		if !ok {
			err := fmt.Errorf("Object was not a cluster but a %T", mo)
			log.Error(err)
			utilruntime.HandleError(err)
			return nil
		}

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
	ctx context.Context,
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

	ctrlOpts := controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, masterManager, ctrlOpts)
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()

		seedClusterWatch := &source.Kind{Type: &kubermaticv1.Cluster{}}
		if err := seedClusterWatch.InjectCache(seedManager.GetCache()); err != nil {
			return fmt.Errorf("failed to inject cache for seed %q into watch: %w", seedName, err)
		}
		if err := c.Watch(seedClusterWatch, requestFromCluster(log), workerlabel.Predicates(workerName)); err != nil {
			return fmt.Errorf("failed to watch clusters in seed %q: %w", seedName, err)
		}
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Project{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to watch projects: %w", err)
	}

	return nil
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With(kubermaticv1.ProjectIDLabelKey, request.Name)
	log.Debug("Processing")

	err := r.reconcile(ctx, log, request)
	if err != nil {
		log.Errorw("ReconcilingError", zap.Error(err))
	}
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

	if len(project.Labels) == 0 {
		log.Debug("Project has no labels, nothing to do")
		return nil
	}

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
			changed, newClusterLabels := getLabelsForCluster(log, cluster.ObjectMeta.DeepCopy().Labels, project.Labels)
			if !changed {
				log.Debug("Labels on cluster are already up to date")
				continue
			}
			oldCluster := cluster.DeepCopy()
			cluster.Labels = newClusterLabels
			log.Debug("Updating labels on cluster")
			if err := seedClient.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
				errs = append(errs, fmt.Errorf("failed to update cluster %q", cluster.Name))
			}
			if err := kuberneteshelper.UpdateClusterStatus(ctx, seedClient, cluster, func(c *kubermaticv1.Cluster) {
				c.Status.InheritedLabels = getInheritedLabels(project.Labels)
			}); err != nil {
				errs = append(errs, fmt.Errorf("failed to update status on cluster %q: %w", cluster.Name, err))
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

func getLabelsForCluster(
	log *zap.SugaredLogger,
	clusterLabels map[string]string,
	projectLabels map[string]string,
) (changed bool, newClusterLabels map[string]string) {
	// They shouldn't be nil as we skip projects without labels
	// and need a label on the cluster to associate it to a project
	// but better be safe than panicking.
	if clusterLabels == nil {
		clusterLabels = map[string]string{}
	}
	newClusterLabels = clusterLabels

	for projectLabelKey, projectLabelValue := range projectLabels {
		if kubermaticv1.ProtectedClusterLabels.Has(projectLabelKey) {
			log.Infof("Project wants to set protected label %q on cluster, skipping", projectLabelKey)
			continue
		}
		if clusterLabels[projectLabelKey] == projectLabelValue {
			log.Debugf("Label %q on cluster already has value of %q, nothing to do", projectLabelKey, projectLabelValue)
			continue
		}
		log.Debugf("Setting label %q to value %q on cluster", projectLabelKey, projectLabelValue)
		clusterLabels[projectLabelKey] = projectLabelValue
		changed = true
	}
	return
}

func getInheritedLabels(projectLabels map[string]string) map[string]string {
	inheritedLabels := make(map[string]string)

	for key, val := range projectLabels {
		if !kubermaticv1.ProtectedClusterLabels.Has(key) {
			inheritedLabels[key] = val
		}
	}

	return inheritedLabels
}
