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
	"reflect"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/resources"
	utilcluster "k8c.io/kubermatic/v2/pkg/util/cluster"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kkp-cluster-template-controller"
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
	numWorkers int,
) error {
	workerSelector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return fmt.Errorf("failed to build worker-name selector: %w", err)
	}

	reconciler := &reconciler{
		log:                     log.Named(ControllerName),
		workerNameLabelSelector: workerSelector,
		workerName:              workerName,
		recorder:                mgr.GetEventRecorderFor(ControllerName),
		namespace:               namespace,
		seedClient:              mgr.GetClient(),
	}

	_, err = builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.ClusterTemplateInstance{}).
		Build(reconciler)

	return err
}

// Reconcile reconciles the kubermatic cluster template instance in the seed cluster.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("templateinstance", request.Name)
	log.Debug("Reconciling")

	instance := &kubermaticv1.ClusterTemplateInstance{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, instance); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get cluster template instance %s: %w", request.NamespacedName, err)
	}

	err := r.reconcile(ctx, instance, log)
	if err != nil {
		r.recorder.Event(instance, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, instance *kubermaticv1.ClusterTemplateInstance, log *zap.SugaredLogger) error {
	// handle deletion
	if !instance.DeletionTimestamp.IsZero() {
		return nil
	}

	// create all [remaining] clusters
	if err := r.createClusters(ctx, instance, log); err != nil {
		return err
	}

	log.Info("all clusters created successfully, deleting temporary ClusterTemplateInstance")

	// now that all clusters are created, delete this temporary object
	return r.seedClient.Delete(ctx, instance)
}

func (r *reconciler) patchInstance(ctx context.Context, instance *kubermaticv1.ClusterTemplateInstance, patch func(instance *kubermaticv1.ClusterTemplateInstance)) error {
	oldInstance := instance.DeepCopy()

	patch(instance)

	if !reflect.DeepEqual(oldInstance, instance) {
		if err := r.seedClient.Patch(ctx, instance, ctrlruntimeclient.MergeFrom(oldInstance)); err != nil {
			return fmt.Errorf("failed to update cluster template instance %s: %w", instance.Name, err)
		}
	}

	return nil
}

func (r *reconciler) createClusters(ctx context.Context, instance *kubermaticv1.ClusterTemplateInstance, log *zap.SugaredLogger) error {
	if instance.Spec.Replicas > 0 {
		log.Infof("creating %d clusters", instance.Spec.Replicas)

		template := &kubermaticv1.ClusterTemplate{}
		if err := r.seedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: instance.Spec.ClusterTemplateID}, template); err != nil {
			return fmt.Errorf("failed to get template %s: %w", instance.Spec.ClusterTemplateID, err)
		}

		for i := range instance.Spec.Replicas {
			if err := r.createCluster(ctx, log, template, instance); err != nil {
				created := i
				totalReplicas := instance.Spec.Replicas

				if patchErr := r.patchInstance(ctx, instance, func(i *kubermaticv1.ClusterTemplateInstance) {
					i.Spec.Replicas = totalReplicas - created
				}); patchErr != nil {
					return fmt.Errorf("error patching cluster template instance (%w), after cluster creation fail: %w", patchErr, err)
				}

				return fmt.Errorf("failed to create desired number of clusters. Created %d of %d: %w", created, totalReplicas, err)
			}
		}
	}

	return nil
}

func (r *reconciler) createCluster(ctx context.Context, log *zap.SugaredLogger, template *kubermaticv1.ClusterTemplate, instance *kubermaticv1.ClusterTemplateInstance) error {
	// This is temporary cluster with cloud spec from the template.
	// It holds credential for the new cluster
	partialCluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: template.Name,
		},
	}
	partialCluster.Spec = template.Spec

	if instance.Annotations != nil && instance.Annotations[kubermaticv1.ClusterTemplateInstanceOwnerAnnotationKey] != "" {
		if template.Annotations == nil {
			template.Annotations = map[string]string{}
		}
		template.Annotations[kubermaticv1.ClusterTemplateUserAnnotationKey] = instance.Annotations[kubermaticv1.ClusterTemplateInstanceOwnerAnnotationKey]
	}

	newCluster := genNewCluster(template, instance, r.workerName)
	newStatus := newCluster.Status.DeepCopy()

	// Here partialCluster is used to copy credentials to the new cluster
	err := resources.CopyCredentials(resources.NewCredentialsData(ctx, partialCluster, r.seedClient), newCluster)
	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	// reuse our reconciling framework, because this is a special place where right after the Cluster
	// creation, we must set some status fields and this requires us to wait for the Cluster object
	// to appear in our caches.
	name := types.NamespacedName{Name: newCluster.Name}
	dummyCreator := func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		return newCluster, nil
	}

	log.Infof("creating cluster %s", newCluster.Name)

	if err := reconciling.EnsureNamedObject(ctx, name, dummyCreator, r.seedClient, &kubermaticv1.Cluster{}, false); err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	if err := util.UpdateClusterStatus(ctx, r.seedClient, newCluster, func(c *kubermaticv1.Cluster) {
		c.Status = *newStatus
	}); err != nil {
		return fmt.Errorf("failed to set cluster status: %w", err)
	}

	if err := r.assignSSHKeyToCluster(ctx, newCluster.Name, template.UserSSHKeys); err != nil {
		log.Errorf("failed to assign SSH key to the cluster %v", err)
	}

	return nil
}

func (r *reconciler) assignSSHKeyToCluster(ctx context.Context, clusterID string, userSSHKeys []kubermaticv1.ClusterTemplateSSHKey) error {
	if len(userSSHKeys) == 0 {
		return nil
	}
	for _, sshKey := range userSSHKeys {
		userKey := &kubermaticv1.UserSSHKey{}
		if err := r.seedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: sshKey.ID}, userKey); err != nil {
			return err
		}
		userKey.AddToCluster(clusterID)
		if err := r.seedClient.Update(ctx, userKey); err != nil {
			return err
		}
	}
	return nil
}

func genNewCluster(template *kubermaticv1.ClusterTemplate, instance *kubermaticv1.ClusterTemplateInstance, workerName string) *kubermaticv1.Cluster {
	name := utilcluster.MakeClusterName()

	newCluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      template.ClusterLabels,
			Annotations: template.Annotations,
		},
	}

	if newCluster.Labels == nil {
		newCluster.Labels = map[string]string{}
	}

	if len(workerName) > 0 {
		newCluster.Labels[kubermaticv1.WorkerNameLabelKey] = workerName
	}
	newCluster.Labels[kubermaticv1.ProjectIDLabelKey] = instance.Spec.ProjectID
	newCluster.Labels[kubermaticv1.ClusterTemplateInstanceLabelKey] = instance.Name
	newCluster.Spec = template.Spec

	newCluster.Spec.HumanReadableName = fmt.Sprintf("%s-%s", newCluster.Spec.HumanReadableName, name)
	newCluster.Status.UserEmail = template.Annotations[kubermaticv1.ClusterTemplateUserAnnotationKey]

	return newCluster
}
