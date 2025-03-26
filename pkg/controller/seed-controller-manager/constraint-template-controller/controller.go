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

package constrainttemplatecontroller

import (
	"context"
	"fmt"

	v1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// This controller syncs the kubermatic constraint templates to gatekeeper constraint templates on the user cluster.
	ControllerName = "kkp-constraint-template-controller"

	// cleanupFinalizer indicates that synced gatekeeper Constraint Templates on user cluster need cleanup.
	cleanupFinalizer = "kubermatic.k8c.io/cleanup-gatekeeper-constraint-templates"
)

// UserClusterClientProvider provides functionality to get a user cluster client.
type UserClusterClientProvider interface {
	GetClient(ctx context.Context, c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

type reconciler struct {
	log                       *zap.SugaredLogger
	workerNameLabelSelector   labels.Selector
	recorder                  record.EventRecorder
	seedClient                ctrlruntimeclient.Client
	userClusterClientProvider UserClusterClientProvider
	userClusterClients        map[string]ctrlruntimeclient.Client
}

func Add(
	mgr manager.Manager,
	clientProvider UserClusterClientProvider,
	log *zap.SugaredLogger,
	workerName string,
	numWorkers int,
) error {
	workerSelector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return fmt.Errorf("failed to build worker-name selector: %w", err)
	}

	reconciler := &reconciler{
		log:                       log.Named(ControllerName),
		workerNameLabelSelector:   workerSelector,
		recorder:                  mgr.GetEventRecorderFor(ControllerName),
		seedClient:                mgr.GetClient(),
		userClusterClientProvider: clientProvider,
		userClusterClients:        map[string]ctrlruntimeclient.Client{},
	}

	_, err = builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.ConstraintTemplate{}).
		Watches(&kubermaticv1.Cluster{}, enqueueAllConstraintTemplates(reconciler.seedClient, reconciler.log), builder.WithPredicates(workerlabel.Predicate(workerName))).
		Build(reconciler)

	return err
}

// Reconcile reconciles the kubermatic constraint template on the seed cluster to all user clusters
// which have opa integration enabled.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("constrainttemplate", request.Name)
	log.Debug("Reconciling")

	constraintTemplate := &kubermaticv1.ConstraintTemplate{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, constraintTemplate); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("constraint template not found, returning")
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get constraint template %s: %w", constraintTemplate.Name, err)
	}

	err := r.reconcile(ctx, log, constraintTemplate)
	if err != nil {
		r.recorder.Event(constraintTemplate, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, constraintTemplate *kubermaticv1.ConstraintTemplate) error {
	if constraintTemplate.DeletionTimestamp != nil {
		if !kuberneteshelper.HasFinalizer(constraintTemplate, cleanupFinalizer) {
			return nil
		}

		err := r.syncAllClusters(ctx, log, constraintTemplate, func(userClusterClient ctrlruntimeclient.Client, ct *kubermaticv1.ConstraintTemplate) error {
			err := userClusterClient.Delete(ctx, &v1.ConstraintTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: constraintTemplate.Name,
				},
			})

			return ctrlruntimeclient.IgnoreNotFound(err)
		})
		if err != nil {
			return err
		}

		return kuberneteshelper.TryRemoveFinalizer(ctx, r.seedClient, constraintTemplate, cleanupFinalizer)
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.seedClient, constraintTemplate, cleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	ctReconcilerFactories := []reconciling.NamedGatekeeperConstraintTemplateReconcilerFactory{
		constraintTemplateReconcilerFactory(constraintTemplate),
	}

	return r.syncAllClusters(ctx, log, constraintTemplate, func(userClusterClient ctrlruntimeclient.Client, ct *kubermaticv1.ConstraintTemplate) error {
		return reconciling.ReconcileGatekeeperConstraintTemplates(ctx, ctReconcilerFactories, "", userClusterClient)
	})
}

func (r *reconciler) syncAllClusters(
	ctx context.Context,
	log *zap.SugaredLogger,
	constraintTemplate *kubermaticv1.ConstraintTemplate,
	action func(userClusterClient ctrlruntimeclient.Client, ct *kubermaticv1.ConstraintTemplate) error,
) error {
	clusterList, err := r.getClustersForConstraintTemplate(ctx, constraintTemplate)
	if err != nil {
		return fmt.Errorf("failed listing clusters: %w", err)
	}

	for _, userCluster := range clusterList.Items {
		clusterLog := log.With("cluster", userCluster.Name)

		if userCluster.Spec.Pause {
			clusterLog.Debug("Cluster is paused, skipping")
			continue
		}

		// if the control plane is not healthy, we cannot possibly create a functioning usercluster client
		if !userCluster.Status.ExtendedHealth.ControlPlaneHealthy() {
			clusterLog.Debug("Cluster control-plane is not healthy yet, skipping")
			continue
		}

		if userCluster.Spec.OPAIntegration == nil || !userCluster.Spec.OPAIntegration.Enabled {
			clusterLog.Debug("Cluster does not integrate with OPA, skipping")
			continue
		}

		// Get user cluster client from map, if it does not exist yet, create it
		var err error
		userClusterClient, ok := r.userClusterClients[userCluster.Name]
		if !ok {
			userClusterClient, err = r.userClusterClientProvider.GetClient(ctx, &userCluster)
			if err != nil {
				return fmt.Errorf("error getting client for cluster %s: %w", userCluster.Name, err)
			}
			r.userClusterClients[userCluster.Name] = userClusterClient
		}

		err = action(userClusterClient, constraintTemplate)
		if err != nil {
			return fmt.Errorf("failed syncing constraint template for cluster %s: %w", userCluster.Name, err)
		}
		clusterLog.Debug("Reconciled constraint template with cluster")
	}

	return nil
}

func constraintTemplateReconcilerFactory(kubeCT *kubermaticv1.ConstraintTemplate) reconciling.NamedGatekeeperConstraintTemplateReconcilerFactory {
	return func() (string, reconciling.GatekeeperConstraintTemplateReconciler) {
		return kubeCT.Name, func(ct *v1.ConstraintTemplate) (*v1.ConstraintTemplate, error) {
			ct.Name = kubeCT.Name
			ct.Spec = v1.ConstraintTemplateSpec{
				CRD:     kubeCT.Spec.CRD,
				Targets: kubeCT.Spec.Targets,
			}

			return ct, nil
		}
	}
}

func enqueueAllConstraintTemplates(client ctrlruntimeclient.Client, log *zap.SugaredLogger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request

		cluster, ok := a.(*kubermaticv1.Cluster)
		if !ok {
			err := fmt.Errorf("object was not a cluster but a %T", a)
			log.Error(err)
			utilruntime.HandleError(err)
			return nil
		}
		if cluster.Spec.OPAIntegration == nil || !cluster.Spec.OPAIntegration.Enabled {
			return nil
		}

		ctList := &kubermaticv1.ConstraintTemplateList{}
		if err := client.List(ctx, ctList); err != nil {
			log.Error(err)
			utilruntime.HandleError(fmt.Errorf("failed to list constraint templates: %w", err))
		}
		for _, ct := range ctList.Items {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
				Name: ct.Name,
			}})
		}
		return requests
	})
}
