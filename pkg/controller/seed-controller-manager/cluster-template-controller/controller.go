/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package clustertemplatecontroller

import (
	"context"
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/types"

	"go.uber.org/zap"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "cluster_template_controller"

	finalizer = kubermaticapiv1.SeedClusterTemplateInstanceFinalizer
)

type reconciler struct {
	log                     *zap.SugaredLogger
	workerNameLabelSelector labels.Selector
	workerName              string
	recorder                record.EventRecorder
	namespace               string
	seedClient              ctrlruntimeclient.Client
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	namespace string,
	numWorkers int) error {

	workerSelector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return fmt.Errorf("failed to build worker-name selector: %v", err)
	}

	reconciler := &reconciler{
		log:                     log.Named(ControllerName),
		workerNameLabelSelector: workerSelector,
		workerName:              workerName,
		recorder:                mgr.GetEventRecorderFor(ControllerName),
		namespace:               namespace,
		seedClient:              mgr.GetClient(),
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.ClusterTemplateInstance{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to create watch for seed cluster template instance: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.Cluster{}},
		enqueueClusterTemplateInstances(),
		workerlabel.Predicates(workerName),
	); err != nil {
		return fmt.Errorf("failed to create watch for clusters: %w", err)
	}

	return nil
}

// Reconcile reconciles the kubermatic cluster template instance in the seed cluster
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {

	log := r.log.With("request", request)
	log.Debug("Reconciling")

	instance := &kubermaticv1.ClusterTemplateInstance{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, instance); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get cluster template instance %s: %w", instance.Name, ctrlruntimeclient.IgnoreNotFound(err))
	}

	err := r.reconcile(ctx, instance, log)
	if err != nil {
		log.Errorw("ReconcilingError", zap.Error(err))
		r.recorder.Eventf(instance, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return reconcile.Result{}, err

}

func (r *reconciler) reconcile(ctx context.Context, instance *kubermaticv1.ClusterTemplateInstance, log *zap.SugaredLogger) error {
	var remove = true
	var add = false
	clusterTemplateLabelSelector := ctrlruntimeclient.MatchingLabels{kubernetes.ClusterTemplateInstanceLabelKey: instance.Name}

	clusterList := &kubermaticv1.ClusterList{}
	if err := r.seedClient.List(ctx, clusterList, &ctrlruntimeclient.ListOptions{LabelSelector: r.workerNameLabelSelector}, clusterTemplateLabelSelector); err != nil {
		return fmt.Errorf("failed listing clusters: %w", err)
	}

	// deletion
	if !instance.DeletionTimestamp.IsZero() {
		if !kuberneteshelper.HasFinalizer(instance, finalizer) {
			return nil
		}

		if err := r.patchFinalizer(ctx, instance, remove); err != nil {
			return err
		}

		return nil
	}

	// initialization
	if !kuberneteshelper.HasFinalizer(instance, finalizer) {
		if err := r.patchFinalizer(ctx, instance, add); err != nil {
			return err
		}
	}

	nc := len(clusterList.Items)
	currentNumberOfClusters := int64(nc)
	desiredNumberOfClusters := instance.Spec.Replicas

	switch {
	case currentNumberOfClusters == desiredNumberOfClusters:
		return nil
	case currentNumberOfClusters > desiredNumberOfClusters:
		toDelete := currentNumberOfClusters - desiredNumberOfClusters
		if err := r.deleteClusters(ctx, clusterList, toDelete, log); err != nil {
			log.Errorf("failed to delete clusters %v", err)
			return err
		}
	case currentNumberOfClusters < desiredNumberOfClusters:
		toCreate := desiredNumberOfClusters - currentNumberOfClusters
		if err := r.createClusters(ctx, instance, toCreate, currentNumberOfClusters, log); err != nil {
			log.Errorf("failed to create clusters %v", err)
			return err
		}
	}

	return nil
}

func (r *reconciler) patchFinalizer(ctx context.Context, instance *kubermaticv1.ClusterTemplateInstance, remove bool) error {
	oldInstance := instance.DeepCopy()

	kuberneteshelper.AddFinalizer(instance, finalizer)

	if remove {
		kuberneteshelper.RemoveFinalizer(instance, finalizer)
	}

	if err := r.seedClient.Patch(ctx, instance, ctrlruntimeclient.MergeFrom(oldInstance)); err != nil {
		return fmt.Errorf("failed to update cluster template instance %s finalizer: %v", instance.Name, err)
	}

	return nil
}

func (r *reconciler) deleteClusters(ctx context.Context, clusterList *kubermaticv1.ClusterList, numberOfClusters int64, log *zap.SugaredLogger) error {
	log.Debugf("delete %d clusters", numberOfClusters)
	if numberOfClusters > 0 {
		clusters := []*kubermaticv1.Cluster{}

		for _, cluster := range clusterList.Items {
			clusters = append(clusters, cluster.DeepCopy())
		}
		// delete always with the highest index
		sortClustersByIndexedName(clusters)

		for index, cluster := range clusters {
			if err := r.seedClient.Delete(ctx, cluster); err != nil {
				return err
			}
			if int64(index+1) == numberOfClusters {
				break
			}
		}
	}
	return nil
}

func sortClustersByIndexedName(clusters []*kubermaticv1.Cluster) {
	sort.SliceStable(clusters, func(i, j int) bool {
		mi, mj := clusters[i], clusters[j]
		return mi.Spec.HumanReadableName > mj.Spec.HumanReadableName
	})
}

func (r *reconciler) createClusters(ctx context.Context, instance *kubermaticv1.ClusterTemplateInstance, numberOfClusters, index int64, log *zap.SugaredLogger) error {

	log.Debugf("create clusters from template %s, number of clusters: %d", instance.Spec.ClusterTemplateID, numberOfClusters)

	template := &kubermaticv1.ClusterTemplate{}
	if err := r.seedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: instance.Spec.ClusterTemplateID}, template); err != nil {
		return fmt.Errorf("failed to get template %s: %w", instance.Spec.ClusterTemplateID, err)
	}

	// This is temporary cluster with cloud spec from the template.
	// It holds credential for the new cluster
	partialCluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: template.Name,
		},
	}
	partialCluster.Spec = template.Spec

	for i := 0; i < int(numberOfClusters); i++ {

		newCluster := genNewCluster(template, instance, r.workerName, int(index)+i)

		// Here partialCluster is used to copy credentials to the new cluster
		err := resources.CopyCredentials(resources.NewCredentialsData(context.Background(), partialCluster, r.seedClient), newCluster)
		if err != nil {
			return fmt.Errorf("failed to get credentials: %v", err)
		}
		if err := kubernetesprovider.CreateOrUpdateCredentialSecretForCluster(ctx, r.seedClient, newCluster); err != nil {
			return err
		}
		kuberneteshelper.AddFinalizer(newCluster, kubermaticapiv1.CredentialsSecretsCleanupFinalizer)

		if err := r.seedClient.Create(ctx, newCluster); err != nil {
			return err
		}
	}

	return nil
}

func genNewCluster(template *kubermaticv1.ClusterTemplate, instance *kubermaticv1.ClusterTemplateInstance, workerName string, index int) *kubermaticv1.Cluster {

	newCluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:        rand.String(10),
			Labels:      template.ClusterLabels,
			Annotations: template.Annotations,
		},
		Status: kubermaticv1.ClusterStatus{},
	}

	if newCluster.Labels == nil {
		newCluster.Labels = map[string]string{}
	}

	if len(workerName) > 0 {
		newCluster.Labels[kubermaticv1.WorkerNameLabelKey] = workerName
	}
	newCluster.Labels[kubermaticv1.ProjectIDLabelKey] = instance.Spec.ProjectID
	newCluster.Labels[kubernetes.ClusterTemplateInstanceLabelKey] = instance.Name
	newCluster.Spec = template.Spec

	newCluster.Spec.HumanReadableName = fmt.Sprintf("%s-%d", newCluster.Spec.HumanReadableName, index)
	newCluster.Status.UserEmail = template.Annotations[kubermaticv1.ClusterTemplateUserAnnotationKey]

	return newCluster
}

func enqueueClusterTemplateInstances() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request

		clusterLabels := a.GetLabels()
		if clusterLabels != nil {
			instanceName, ok := clusterLabels[kubernetes.ClusterTemplateInstanceLabelKey]
			if ok && instanceName != "" {
				requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
					Name: instanceName,
				}})
			}
		}

		return requests
	})
}
