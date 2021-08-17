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

	if err := r.createClusters(ctx, instance, log); err != nil {
		return err
	}
	return r.seedClient.Delete(ctx, instance)
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

func (r *reconciler) createClusters(ctx context.Context, instance *kubermaticv1.ClusterTemplateInstance, log *zap.SugaredLogger) error {

	log.Debugf("create clusters from template %s, number of clusters: %d", instance.Spec.ClusterTemplateID, instance.Spec.Replicas)

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

	if instance.Spec.Replicas > 0 {
		oldInstance := instance.DeepCopy()
		for i := 0; i < int(instance.Spec.Replicas); i++ {

			newCluster := genNewCluster(template, instance, r.workerName)

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
				// if error then change number of replicas
				created := int64(i + 1)
				totalReplicas := instance.Spec.Replicas
				instance.Spec.Replicas = totalReplicas - created
				if err := r.seedClient.Patch(ctx, instance, ctrlruntimeclient.MergeFrom(oldInstance)); err != nil {
					return err
				}
				return fmt.Errorf("failed to create desired number of clusters. Created %d from %d", created, totalReplicas)
			}
		}
	}

	return nil
}

func genNewCluster(template *kubermaticv1.ClusterTemplate, instance *kubermaticv1.ClusterTemplateInstance, workerName string) *kubermaticv1.Cluster {

	name := rand.String(10)
	newCluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
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

	newCluster.Spec.HumanReadableName = fmt.Sprintf("%s-%s", newCluster.Spec.HumanReadableName, name)
	newCluster.Status.UserEmail = template.Annotations[kubermaticv1.ClusterTemplateUserAnnotationKey]

	return newCluster
}
